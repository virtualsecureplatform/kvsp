package main

import (
	"debug/elf"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

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
		case "CLANG":
			path = "clang"
		case "TFHEUTIL":
			path = "tfheutil"
		case "CAHP_SIM":
			path = "cahp-sim"
		case "CAHP_RT":
			path = "../share/kvsp/cahp-rt"
		case "IYOKANL2":
			path = "iyokanl2"
		case "VSPCORE":
			path = "../share/kvsp/vsp-core.json"
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
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getCloudKey(keyFileName string) ([]byte, error) {
	// Get the path of tfheutil
	tfheutilPath, err := getPathOf("TFHEUTIL")
	if err != nil {
		return nil, err
	}

	// Run tfheutil to get the cloud key
	cloudKeyTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	defer os.Remove(cloudKeyTmpFile.Name())
	err = execCmd(
		tfheutilPath,
		[]string{"cloudkey", keyFileName, cloudKeyTmpFile.Name()})
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(cloudKeyTmpFile)
}

func encryptBit(keyFileName string, bit bool) ([]byte, error) {
	// Get the path of tfheutil
	tfheutilPath, err := getPathOf("TFHEUTIL")
	if err != nil {
		return nil, err
	}

	// Write the source binary to a temporary file
	plainTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	defer os.Remove(plainTmpFile.Name())
	if bit {
		plainTmpFile.Write([]byte{1})
	} else {
		plainTmpFile.Write([]byte{0})
	}
	plainTmpFile.Close()

	// Encrypt the binary
	encTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	defer os.Remove(encTmpFile.Name())
	err = execCmd(
		tfheutilPath,
		[]string{"enc", keyFileName, plainTmpFile.Name(), encTmpFile.Name(), strconv.Itoa(1)})
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(encTmpFile)
}

func encryptBytes(keyFileName string, plain []byte) ([]byte, error) {
	// Get the path of tfheutil
	tfheutilPath, err := getPathOf("TFHEUTIL")
	if err != nil {
		return nil, err
	}

	// Write the source binary to a temporary file
	plainTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	defer os.Remove(plainTmpFile.Name())
	plainTmpFile.Write(plain)
	plainTmpFile.Close()

	// Encrypt the binary
	encTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	defer os.Remove(encTmpFile.Name())
	err = execCmd(
		tfheutilPath,
		[]string{"enc", keyFileName, plainTmpFile.Name(), encTmpFile.Name(), strconv.Itoa(-1)})
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(encTmpFile)
}

func decryptBit(keyFileName string, enc []byte) (bool, error) {
	// Get the path of tfheutil
	tfheutilPath, err := getPathOf("TFHEUTIL")
	if err != nil {
		return false, err
	}

	// Write the source binary to a temporary file
	encTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return false, err
	}
	defer os.Remove(encTmpFile.Name())
	encTmpFile.Write(enc)
	encTmpFile.Close()

	// Decrypt the binary
	decTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return false, err
	}
	defer os.Remove(decTmpFile.Name())
	err = execCmd(
		tfheutilPath,
		[]string{"dec", keyFileName, encTmpFile.Name(), decTmpFile.Name(), strconv.Itoa(1)})
	if err != nil {
		return false, err
	}

	// Read the result
	plain, err := ioutil.ReadAll(decTmpFile)
	if err != nil {
		return false, err
	}
	if len(plain) != 1 {
		return false, errors.New("Invalid result of tfheutil")
	}

	return (plain[0] & 1) != 0, nil
}

func decryptBytes(keyFileName string, enc []byte) ([]byte, error) {
	// Get the path of tfheutil
	tfheutilPath, err := getPathOf("TFHEUTIL")
	if err != nil {
		return nil, err
	}

	// Write the source binary to a temporary file
	encTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	defer os.Remove(encTmpFile.Name())
	encTmpFile.Write(enc)
	encTmpFile.Close()

	// Decrypt the binary
	decTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	defer os.Remove(decTmpFile.Name())
	err = execCmd(
		tfheutilPath,
		[]string{"dec", keyFileName, encTmpFile.Name(), decTmpFile.Name(), strconv.Itoa(-1)})
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(decTmpFile)
}

// Parse the input as ELF and get ROM and RAM images.
func parseELF(fileName string) ([]byte, []byte, error) {
	input, err := elf.Open(fileName)
	if err != nil {
		return nil, nil, err
	}

	rom := make([]byte, 512)
	ram := make([]byte, 512)

	for _, prog := range input.Progs {
		addr := prog.ProgHeader.Vaddr
		size := prog.ProgHeader.Filesz
		if size == 0 {
			continue
		}

		var mem []byte
		if addr < 0x10000 { // ROM
			mem = rom[addr : addr+size]
		} else { // RAM
			mem = ram[addr-0x10000 : addr-0x10000+size]
		}

		reader := prog.Open()
		_, err := reader.Read(mem)
		if err != nil {
			return nil, nil, err
		}
	}

	return rom, ram, nil
}

func runIyokanl2(inputFileName, outputFileName string, args ...string) error {
	// Get the path of iyokanl2.
	iyokanl2Path, err := getPathOf("IYOKANL2")
	if err != nil {
		return err
	}

	// Get the path of VSP core.
	vspcorePath, err := getPathOf("VSPCORE")
	if err != nil {
		return err
	}

	// Set #CPUs plus 1 as #threads of iyokanl2.
	numThreads := runtime.NumCPU() + 1

	// Run the encrypted program as is.
	return execCmd(iyokanl2Path, append([]string{
		"-t", fmt.Sprint(numThreads),
		"-l", vspcorePath,
		"-i", inputFileName,
		"-o", outputFileName,
	}, args...))
}

func printRes(flags []bool, regs []uint16, ram []uint8) error {
	for i, flag := range flags {
		fmt.Printf("Flag %d : %t\n", i, flag)
	}
	fmt.Printf("\n")
	for i, reg := range regs {
		fmt.Printf("Reg %d : %d\n", i, reg)
	}
	fmt.Printf("\nRAM :\n")
	for i, b := range ram {
		fmt.Printf("%02x ", b)
		if i%16 == 15 {
			fmt.Printf("\n")
		}
	}
	fmt.Printf("\n")

	return nil
}

func printResAsJSON(flags []bool, regs []uint16, ram []uint8) error {
	ramReadable := make([]int, len(ram))
	for i, b := range ram {
		ramReadable[i] = int(b)
	}
	json, err := json.Marshal(struct {
		Flags []bool
		Regs  []uint16
		RAM   []int
	}{flags, regs, ramReadable})
	if err != nil {
		return err
	}
	os.Stdout.Write(json)

	return nil
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

	// Parse input ELF.
	rom, ram, err := parseELF(fileName)
	if err != nil {
		return err
	}

	// Create a KVSP plain request packet to write in.
	packet := KVSPPlainReqPacket{
		KVSPPlainReqPacketHeader{
			[4]byte{'K', 'V', 'S', 'P'},
			0,
			uint64(len(rom)),
			uint64(len(ram)),
		},
		rom,
		ram,
	}

	// Write the packet to a temporary file.
	reqTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	_, err = packet.WriteTo(reqTmpFile)
	if err != nil {
		return err
	}

	// Run iyokanl2 to get emulation result.
	resTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer os.Remove(reqTmpFile.Name())
	err = runIyokanl2(reqTmpFile.Name(), resTmpFile.Name(), "--plain")
	if err != nil {
		return err
	}

	// Read the result.
	resPacket := KVSPPlainResPacket{}
	_, err = resPacket.ReadFrom(resTmpFile)
	if err != nil {
		return err
	}

	// Print result
	printRes(resPacket.Flags, resPacket.Regs, resPacket.RAM)

	return nil
}

func doDec() error {
	// Parse command-line arguments.
	fs := flag.NewFlagSet("dec", flag.ExitOnError)
	var (
		shouldOutputJSON = fs.Bool("json", false, "Print results as JSON")
		keyFileName      = fs.String("k", "", "Key file name")
		inputFileName    = fs.String("i", "", "Input file name (encrypted)")
	)
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if *keyFileName == "" || *inputFileName == "" {
		return errors.New("Specify -k and -i options properly")
	}

	// Do decryption.
	reader, err := os.Open(*inputFileName)
	if err != nil {
		return err
	}
	packet := KVSPResPacket{}
	_, err = packet.ReadFrom(reader)
	if err != nil {
		return err
	}

	// Decrypt flags
	flags := make([]bool, packet.Nflags)
	for i, encFlag := range packet.Flags {
		flags[i], err = decryptBit(*keyFileName, encFlag)
		if err != nil {
			return err
		}
	}

	// Decrypt regs
	regs := make([]uint16, packet.Nregs)
	for i, encReg := range packet.Regs {
		plainReg, err := decryptBytes(*keyFileName, encReg)
		if err != nil {
			return err
		}
		regs[i] = uint16(plainReg[0]) | (uint16(plainReg[1]) << 8)
	}

	// Decrypt RAM
	ram, err := decryptBytes(*keyFileName, packet.RAM)
	if err != nil {
		return err
	}

	// Print the result as JSON
	if *shouldOutputJSON {
		err = printResAsJSON(flags, regs, ram)
	} else {
		// Print the result
		err = printRes(flags, regs, ram)
	}
	if err != nil {
		return err
	}

	return nil
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

	// Parse input ELF.
	rom, ram, err := parseELF(*inputFileName)
	if err != nil {
		return err
	}

	// Encrypt the images
	romEnc, err := encryptBytes(*keyFileName, rom)
	if err != nil {
		return err
	}
	ramEnc, err := encryptBytes(*keyFileName, ram)
	if err != nil {
		return err
	}

	// Get the cloud key from the secret one
	cloudKey, err := getCloudKey(*keyFileName)
	if err != nil {
		return err
	}

	// Create a KVSP request packet to write in
	packet := KVSPReqPacket{
		KVSPReqPacketHeader{
			[4]byte{'K', 'V', 'S', 'P'},
			0,
			uint64(len(cloudKey)),
			uint64(len(romEnc)),
			uint64(len(ramEnc)),
		},
		cloudKey,
		romEnc,
		ramEnc,
	}

	// Write the packet to the output
	writer, err := os.Create(*outputFileName)
	if err != nil {
		return err
	}
	_, err = packet.WriteTo(writer)
	if err != nil {
		return err
	}

	return nil
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

	// Get the path of tfheutil
	path, err := getPathOf("TFHEUTIL")
	if err != nil {
		fatalExit(err)
	}

	// Generate a key.
	return execCmd(path, []string{"genkey", *outputFileName})
}

func doRun() error {
	// Parse command-line arguments.
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	var (
		nClocks        = fs.Uint("c", 0, "Number of clocks to run")
		inputFileName  = fs.String("i", "", "Input file name (encrypted)")
		outputFileName = fs.String("o", "", "Output file name (encrypted)")
	)
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if *nClocks == 0 || *inputFileName == "" || *outputFileName == "" {
		return errors.New("Specify -c, -i, and -o options properly")
	}

	return runIyokanl2(*inputFileName, *outputFileName, "-c", fmt.Sprint(nClocks))
}

func printUsageAndExit() {
	fatalExitWithMsg(`
Usage:
  kvsp cc  OPTIONS...
  kvsp dec OPTIONS...
  kvsp enc OPTIONS...
  kvsp genkey OPTIONS...
  kvsp run OPTIONS...
`)
}

func main() {
	if len(os.Args) <= 1 {
		printUsageAndExit()
	}

	var err error
	switch os.Args[1] {
	case "cc":
		err = doCC()
	case "dec":
		err = doDec()
	case "enc":
		err = doEnc()
	case "genkey":
		err = doGenkey()
	case "run":
		err = doRun()
	case "debug":
		err = doDebug()
	case "emu":
		err = doEmu()
	default:
		printUsageAndExit()
	}

	if err != nil {
		fatalExit(err)
	}
}
