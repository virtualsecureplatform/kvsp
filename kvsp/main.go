package main

import (
	"debug/elf"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

var flagVerbose bool

func fatalExit(err error) {
	log.Fatal(err)
	os.Exit(1)
}

func fatalExitWithMsg(format string, args ...interface{}) {
	fatalExit(fmt.Errorf(format, args...))
}

func write16le(out []byte, val int) {
	out[0] = byte(val & 0xff)
	out[1] = byte((val >> 8) & 0xff)
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
	if path = os.Getenv("KVSP_" + strings.Replace(name, "-", "_", -1) + "_PATH"); path != "" {
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
		case "IYOKAN-BLUEPRINT-DIAMOND":
			path = "../share/kvsp/cahp-diamond.toml"
		case "IYOKAN-BLUEPRINT-EMERALD":
			path = "../share/kvsp/cahp-emerald.toml"
		case "IYOKAN-PACKET":
			path = "iyokan-packet"
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

func attachCommandLineOptions(ram []byte, cmdOptsSrc []string) error {
	// N1548 5.1.2.2.1 2
	// the string pointed to by argv[0]
	// represents the program name; argv[0][0] shall be the null character if the
	// program name is not available from the host environment.
	cmdOpts := []string{""}
	cmdOpts = append(cmdOpts, cmdOptsSrc...)
	argc := len(cmdOpts)

	// Slice for *argv.
	sargv := []int{
		// N1548 5.1.2.2.1 2
		// argv[argc] shall be a null pointer.
		0,
	}

	index := 512 - 2

	// Set **argv to RAM
	for i := len(cmdOpts) - 1; i >= 0; i-- {
		opt := append([]byte(cmdOpts[i]), 0)
		for j := len(opt) - 1; j >= 0; j-- {
			index--
			ram[index] = opt[j]
		}
		sargv = append(sargv, index)
	}
	// Align index
	if index%2 == 1 {
		index--
	}
	// Set *argv to RAM
	for _, val := range sargv {
		index -= 2
		write16le(ram[index:index+2], val)
	}
	// Save argc in RAM
	index -= 2
	write16le(ram[index:index+2], argc)
	// Save initial stack pointer in RAM
	initSP := index
	write16le(ram[512-2:512], initSP)

	return nil
}

func splitRAM(ram []byte) ([]byte, []byte, error) {
	ramA := make([]byte, 256)
	ramB := make([]byte, 256)

	for i := 0; i < 512; i++ {
		if i%2 == 1 {
			ramA[i/2] = ram[i]
		} else {
			ramB[i/2] = ram[i]
		}
	}

	return ramA, ramB, nil
}

func execCmdImpl(name string, args []string) *exec.Cmd {
	if flagVerbose {
		fmtArgs := make([]string, len(args))
		for i, arg := range args {
			fmtArgs[i] = fmt.Sprintf("'%s'", arg)
		}
		fmt.Fprintf(os.Stderr, "exec: '%s' %s\n", name, strings.Join(fmtArgs, " "))
	}

	return exec.Command(name, args...)
}

func execCmd(name string, args []string) error {
	cmd := execCmdImpl(name, args)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func outCmd(name string, args []string) (string, error) {
	out, err := execCmdImpl(name, args).Output()
	return string(out), err
}

func runIyokanPacket(args ...string) (string, error) {
	// Get the path of iyokan-packet
	path, err := getPathOf("IYOKAN-PACKET")
	if err != nil {
		return "", err
	}

	// Run
	return outCmd(path, args)
}

func runIyokan(args ...string) error {
	iyokanPath, err := getPathOf("IYOKAN")
	if err != nil {
		return err
	}

	// Run iyokan
	return execCmd(iyokanPath, args)
}

func packELF(inputFileName, outputFileName string, cmdOpts []string) error {
	if !fileExists(inputFileName) {
		return errors.New("File not found")
	}
	rom, ram, err := parseELF(inputFileName)
	if err != nil {
		return err
	}
	if err = attachCommandLineOptions(ram, cmdOpts); err != nil {
		return err
	}
	ramA, ramB, err := splitRAM(ram)
	if err != nil {
		return err
	}

	// Write ROM data
	romTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer os.Remove(romTmpFile.Name())
	_, err = romTmpFile.Write(rom)
	if err != nil {
		return err
	}

	// Write RAM A data
	ramATmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer os.Remove(ramATmpFile.Name())
	_, err = ramATmpFile.Write(ramA)
	if err != nil {
		return err
	}

	// Write RAM B data
	ramBTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer os.Remove(ramBTmpFile.Name())
	_, err = ramBTmpFile.Write(ramB)
	if err != nil {
		return err
	}

	// Pack
	_, err = runIyokanPacket("pack",
		"--out", outputFileName,
		"--rom", "rom:"+romTmpFile.Name(),
		"--ram", "ramA:"+ramATmpFile.Name(),
		"--ram", "ramB:"+ramBTmpFile.Name())
	if err != nil {
		return err
	}

	return nil
}

type plainPacketTOML struct {
	NumCycles int                    `toml:"cycles"`
	Ram       []plainPacketEntryTOML `toml:"ram"`
	Bits      []plainPacketEntryTOML `toml:"bits"`
}
type plainPacketEntryTOML struct {
	Name  string `toml:"name"`
	Size  int    `toml:"size"`
	Bytes []int  `toml:"bytes"`
}
type plainPacket struct {
	NumCycles int
	Flags     map[string]bool
	Regs      map[string]int
	Ram       []int
}

func (pkt *plainPacket) loadTOML(src string) error {
	var pktTOML plainPacketTOML
	if _, err := toml.Decode(src, &pktTOML); err != nil {
		return err
	}

	pkt.NumCycles = pktTOML.NumCycles

	// Load flags and registers
	pkt.Flags = make(map[string]bool)
	pkt.Regs = make(map[string]int)
	for _, entry := range pktTOML.Bits {
		if entry.Size == 1 { // flag
			if entry.Bytes[0] != 0 {
				pkt.Flags[entry.Name] = true
			} else {
				pkt.Flags[entry.Name] = false
			}
		} else if entry.Size == 16 { // register
			pkt.Regs[entry.Name] = entry.Bytes[0] | (entry.Bytes[1] << 8)
		} else {
			return errors.New("Invalid TOML for result packet")
		}
	}

	// Load ram
	var ramA, ramB []int
	for _, entry := range pktTOML.Ram {
		if entry.Name == "ramA" {
			ramA = entry.Bytes
		} else if entry.Name == "ramB" {
			ramB = entry.Bytes
		} else {
			return errors.New("Invalid TOML for result packet")
		}
	}
	pkt.Ram = make([]int, 512)
	for addr := range pkt.Ram {
		if addr%2 == 1 {
			pkt.Ram[addr] = ramA[addr/2]
		} else {
			pkt.Ram[addr] = ramB[addr/2]
		}
	}

	// Check if the packet is correct
	if _, ok := pkt.Flags["finflag"]; !ok {
		return errors.New("Invalid TOML for result packet: 'finflag' not found")
	}
	for i := 0; i < 16; i++ {
		name := fmt.Sprintf("reg_x%d", i)
		if _, ok := pkt.Regs[name]; !ok {
			return errors.New("Invalid TOML for result packet: '" + name + "' not found")
		}
	}

	return nil
}
func (pkt *plainPacket) print(w io.Writer) error {
	fmt.Fprintf(w, "#cycle\t%d\n", pkt.NumCycles)
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "f0\t%t\n", pkt.Flags["finflag"])
	fmt.Fprintf(w, "\n")
	for i := 0; i < 16; i++ {
		name := fmt.Sprintf("reg_x%d", i)
		fmt.Fprintf(w, "x%d\t%d\n", i, pkt.Regs[name])
	}
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "      \t 0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f")
	for addr := 0; addr < 512; addr++ {
		if addr%16 == 0 {
			fmt.Fprintf(w, "\n%06x\t", addr)
		}
		fmt.Fprintf(w, "%02x ", pkt.Ram[addr])
	}
	fmt.Fprintf(w, "\n")

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
	return execCmd(path, append(os.Args[2:],
		"-target", "cahp", "-mcpu=emerald", "-Oz", "--sysroot", cahpRtPath))
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
	// Parse command-line arguments.
	fs := flag.NewFlagSet("emu", flag.ExitOnError)
	var (
		whichCAHPCPU = fs.String("cahp-cpu", "emerald", "Which CAHP CPU you use, emerald or diamond")
	)
	err := fs.Parse(os.Args[2:])

	// Create tmp file for packing
	packedFile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer os.Remove(packedFile.Name())

	// Pack
	err = packELF(fs.Args()[0], packedFile.Name(), fs.Args()[1:])
	if err != nil {
		return err
	}

	// Create tmp file for the result
	resTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer os.Remove(resTmpFile.Name())

	// Run Iyokan in plain mode
	blueprint, err := getPathOf(fmt.Sprintf("IYOKAN-BLUEPRINT-%s", strings.ToUpper(*whichCAHPCPU)))
	if err != nil {
		return err
	}
	err = runIyokan("plain", "-i", packedFile.Name(), "-o", resTmpFile.Name(), "--blueprint", blueprint)
	if err != nil {
		return err
	}

	// Unpack the result
	result, err := runIyokanPacket("packet2toml", "--in", resTmpFile.Name())
	if err != nil {
		return err
	}

	// Parse and print the result
	var pkt plainPacket
	if err := pkt.loadTOML(result); err != nil {
		return err
	}
	pkt.print(os.Stdout)

	return nil
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

	// Create tmp file for decryption
	packedFile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer os.Remove(packedFile.Name())

	// Decrypt
	_, err = runIyokanPacket("dec",
		"--key", *keyFileName,
		"--in", *inputFileName,
		"--out", packedFile.Name())

	// Unpack
	result, err := runIyokanPacket("packet2toml", "--in", packedFile.Name())
	if err != nil {
		return err
	}

	// Parse and print the result
	var pkt plainPacket
	if err := pkt.loadTOML(result); err != nil {
		return err
	}
	pkt.print(os.Stdout)

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

	// Create tmp file for packing
	packedFile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer os.Remove(packedFile.Name())

	// Pack
	err = packELF(*inputFileName, packedFile.Name(), fs.Args())
	if err != nil {
		return err
	}

	// Encrypt
	_, err = runIyokanPacket("enc",
		"--key", *keyFileName,
		"--in", packedFile.Name(),
		"--out", *outputFileName)
	return err
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

	_, err = runIyokanPacket("genkey",
		"--type", "tfhepp",
		"--out", *outputFileName)
	return err
}

func doPlainpacket() error {
	// Parse command-line arguments.
	fs := flag.NewFlagSet("plainpacket", flag.ExitOnError)
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

	return packELF(*inputFileName, *outputFileName, fs.Args())
}

func doRun() error {
	// Parse command-line arguments.
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	var (
		nClocks        = fs.Uint("c", 0, "Number of clocks to run")
		inputFileName  = fs.String("i", "", "Input file name (encrypted)")
		outputFileName = fs.String("o", "", "Output file name (encrypted)")
		numGPU         = fs.Uint("g", 0, "Number of GPUs (Unspecify or set 0 for CPU mode)")
		whichCAHPCPU   = fs.String("cahp-cpu", "emerald", "Which CAHP CPU you use, emerald or diamond")
	)
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if *nClocks == 0 || *inputFileName == "" || *outputFileName == "" {
		return errors.New("Specify -c, -i, and -o options properly")
	}

	blueprint, err := getPathOf(fmt.Sprintf("IYOKAN-BLUEPRINT-%s", strings.ToUpper(*whichCAHPCPU)))
	if err != nil {
		return err
	}

	args := []string{
		"tfhe",
		"-i", *inputFileName,
		"-o", *outputFileName,
		"-c", fmt.Sprint(*nClocks),
		"--blueprint", blueprint,
	}
	if *numGPU > 0 {
		args = append(args, "--enable-gpu", "--gpu_num", fmt.Sprint(*numGPU))
	}
	return runIyokan(args...)
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
