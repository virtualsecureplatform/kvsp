package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var flagVerbose bool

func fatalExit(err error) {
	log.Fatal(err)
	os.Exit(1)
}

func fatalExitWithMsg(format string, args ...interface{}) {
	fatalExit(fmt.Errorf(format, args...))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getExecDir() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(execPath), nil
}

func prefixExecDir(path string) (string, error) {
	execPath, err := getExecDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(execPath, path), nil
}

func getPathOf(name string) (string, error) {
	path := ""
	relative := true

	// Check if environment variable is set in KVSP_XXX.
	if path = os.Getenv(fmt.Sprintf("KVSP_%s_PATH", name)); path != "" {
		relative = false
	} else {
		/*
			Do heuristic approach, which assumes binaries are in the current
			(this executable's) directory, and others are in ../share/kvsp.
		*/
		switch name {
		case "CAHP_RT":
			path = "../share/kvsp/cahp-rt"
		case "CAHP_SIM":
			path = "cahp-sim"
		case "CLANG":
			path = "clang"
		case "IYOKAN":
			path = "iyokan"
		case "KVSP-PACKET":
			path = "kvsp-packet"
		case "VSPCORE":
			path = "../share/kvsp/vsp-core.json"
		case "VSPCORE-WO-RAM-ROM":
			path = "../share/kvsp/vsp-core-wo-ram-rom.json"
		case "VSPCORE-WO-ROM":
			path = "../share/kvsp/vsp-core-wo-rom.json"
		default:
			return "", errors.New("Invalid name")
		}
	}

	if relative {
		newPath, err := prefixExecDir(path)
		if err != nil {
			return "", err
		}
		path = newPath
	}

	if !fileExists(path) {
		return "", fmt.Errorf("%s not found at %s", name, path)
	}

	return path, nil
}

func execCmd(name string, args []string) error {
	if flagVerbose {
		fmtArgs := make([]string, len(args))
		for i, arg := range args {
			fmtArgs[i] = fmt.Sprintf("'%s'", arg)
		}
		fmt.Fprintf(os.Stderr, "exec: '%s' %s\n", name, strings.Join(fmtArgs, " "))
	}

	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writePlainPacket(inputFileName, outputFileName string) error {
	// Get the path of kvsp-packet
	path, err := getPathOf("KVSP-PACKET")
	if err != nil {
		return err
	}

	// Run
	return execCmd(path, []string{"plain", inputFileName, outputFileName})
}

func runKVSPPacket(args ...string) error {
	// Get the path of kvsp-packet
	path, err := getPathOf("KVSP-PACKET")
	if err != nil {
		return err
	}

	// Run
	return execCmd(path, args)
}

func runIyokan(enableRAM, enableROM bool, args ...string) error {
	// Get the path of iyokan.
	iyokanPath, err := getPathOf("IYOKAN")
	if err != nil {
		return err
	}

	if enableRAM {
		args = append(args, "--enable-ram")
	}
	if enableROM {
		args = append(args, "--enable-rom", "io_romAddr:7:io_romData:32")
	}

	pathVSPCore := ""
	if enableRAM && enableROM {
		pathVSPCore, err = getPathOf("VSPCORE-WO-RAM-ROM")
	}
	if !enableRAM && enableROM {
		pathVSPCore, err = getPathOf("VSPCORE-WO-ROM")
	}
	if !enableRAM && !enableROM {
		pathVSPCore, err = getPathOf("VSPCORE")
	}
	if err != nil {
		return err
	}
	if pathVSPCore == "" {
		return errors.New("Invalid spec of RAM and ROM")
	}
	args = append(args, "-l", pathVSPCore)

	// Run iyokan
	return execCmd(iyokanPath, args)
}

func doCC() error {
	// Get the path of clang
	path, err := getPathOf("CLANG")
	if err != nil {
		return err
	}

	// Get the path of cahp-rt
	cahpRtPath, err := getPathOf("CAHP_RT")
	if err != nil {
		return err
	}

	// Run
	return execCmd(path, append(os.Args[2:], "-target", "cahp", "-Oz", "--sysroot", cahpRtPath))
}

func doDebug() error {
	// Get the path of cahp-sim
	path, err := getPathOf("CAHP_SIM")
	if err != nil {
		fatalExit(err)
	}

	// Run
	return execCmd(path, os.Args[2:])
}

func doEmu() error {
	fileName := os.Args[2]
	if !fileExists(fileName) {
		return errors.New("File not found")
	}

	// Write the packet to a temporary file.
	reqTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer os.Remove(reqTmpFile.Name())
	err = writePlainPacket(fileName, reqTmpFile.Name())
	if err != nil {
		return err
	}

	// Run iyokan to get emulation result.
	return runIyokan(true, true, "plain", "-i", reqTmpFile.Name())
}

func doDec() error {
	// Parse command-line arguments.
	fs := flag.NewFlagSet("dec", flag.ExitOnError)
	var (
		keyFileName   = fs.String("k", "", "Key file name")
		inputFileName = fs.String("i", "", "Input file name (encrypted)")
	)
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if *keyFileName == "" || *inputFileName == "" {
		return errors.New("Specify -k and -i options properly")
	}

	// Run kvsp-packet
	return runKVSPPacket("dec", *keyFileName, *inputFileName)
}

func doEnc() error {
	// Parse command-line arguments.
	fs := flag.NewFlagSet("enc", flag.ExitOnError)
	var (
		keyFileName    = fs.String("k", "", "Key file name")
		inputFileName  = fs.String("i", "", "Input file name (plain)")
		outputFileName = fs.String("o", "", "Output file name (encrypted)")
	)
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if *keyFileName == "" || *inputFileName == "" || *outputFileName == "" {
		return errors.New("Specify -k, -i, and -o options properly")
	}

	// Run kvsp-packet
	return runKVSPPacket("enc", *keyFileName, *inputFileName, *outputFileName)
}

func doGenkey() error {
	// Parse command-line arguments.
	fs := flag.NewFlagSet("genkey", flag.ExitOnError)
	var (
		outputFileName = fs.String("o", "", "Output file name")
	)
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if *outputFileName == "" {
		return errors.New("Specify -o options properly")
	}

	return runKVSPPacket("genkey", *outputFileName)
}

func doPlainpacket() error {
	// Parse command-line arguments.
	fs := flag.NewFlagSet("enc", flag.ExitOnError)
	var (
		inputFileName  = fs.String("i", "", "Input file name (plain)")
		outputFileName = fs.String("o", "", "Output file name (encrypted)")
	)
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if *inputFileName == "" || *outputFileName == "" {
		return errors.New("Specify -i, and -o options properly")
	}

	return runKVSPPacket("plain", *inputFileName, *outputFileName)
}

func doRun() error {
	// Parse command-line arguments.
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	var (
		nClocks        = fs.Uint("c", 0, "Number of clocks to run")
		inputFileName  = fs.String("i", "", "Input file name (encrypted)")
		outputFileName = fs.String("o", "", "Output file name (encrypted)")
		isGPU          = fs.Bool("g", false, "")
		nWorkers       = fs.Uint("t", 0, "Number of workers")
	)
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if *nClocks == 0 || *inputFileName == "" || *outputFileName == "" {
		return errors.New("Specify -c, -i, and -o options properly")
	}

	args := []string{
		"tfhe",
		"-i", *inputFileName,
		"-o", *outputFileName,
		"-c", fmt.Sprint(*nClocks),
	}
	if *nWorkers > 0 {
		args = append(args, "-t", fmt.Sprint(*nWorkers))
	}
	if *isGPU {
		args = append(args, "--enable-gpu")
		return runIyokan(true, true, args...)
	} else {
		return runIyokan(true, true, args...)
	}
}

func printUsageAndExit() {
	fatalExitWithMsg(`
Usage:
  kvsp cc  OPTIONS...
  kvsp debug OPTIONS...
  kvsp dec OPTIONS...
  kvsp emu OPTIONS...
  kvsp enc OPTIONS...
  kvsp genkey OPTIONS...
  kvsp run OPTIONS...
`)
}

func main() {
	if envvarVerbose := os.Getenv("KVSP_VERBOSE"); envvarVerbose == "1" {
		flagVerbose = true
	}

	if len(os.Args) <= 1 {
		printUsageAndExit()
	}

	var err error
	switch os.Args[1] {
	case "cc":
		err = doCC()
	case "debug":
		err = doDebug()
	case "dec":
		err = doDec()
	case "emu":
		err = doEmu()
	case "enc":
		err = doEnc()
	case "genkey":
		err = doGenkey()
	case "plainpacket":
		err = doPlainpacket()
	case "run":
		err = doRun()
	default:
		printUsageAndExit()
	}

	if err != nil {
		fatalExit(err)
	}
}
