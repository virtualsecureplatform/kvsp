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
	"time"

	"github.com/BurntSushi/toml"
)

var flagVerbose bool

const defaultCAHPProc = "ruby"
const defaultROMSize = 1024
const defaultRAMSize = 1024

// Flag for a list of values
// Thanks to: https://stackoverflow.com/a/28323276
type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
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
		case "IYOKAN-BLUEPRINT-RUBY":
			path = "../share/kvsp/cahp-ruby.toml"
		case "IYOKAN-BLUEPRINT-PEARL":
			path = "../share/kvsp/cahp-pearl.toml"
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
func parseELF(fileName string, romSize, ramSize uint64) ([]byte, []byte, error) {
	input, err := elf.Open(fileName)
	if err != nil {
		return nil, nil, err
	}

	rom := make([]byte, romSize)
	ram := make([]byte, ramSize)

	for _, prog := range input.Progs {
		addr := prog.ProgHeader.Vaddr
		size := prog.ProgHeader.Filesz
		if size == 0 {
			continue
		}

		var mem []byte
		if addr < 0x10000 { // ROM
			if addr+size >= romSize {
				return nil, nil, errors.New("Invalid ROM size: too small")
			}
			mem = rom[addr : addr+size]
		} else { // RAM
			if addr-0x10000+size >= ramSize {
				return nil, nil, errors.New("Invalid RAM size: too small")
			}
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

	ramSize := len(ram)
	index := ramSize - 2

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
	write16le(ram[ramSize-2:ramSize], initSP)

	return nil
}

func execCmdImpl(name string, args []string) *exec.Cmd {
	if flagVerbose {
		fmtArgs := make([]string, len(args))
		for i, arg := range args {
			fmtArgs[i] = fmt.Sprintf("'%s'", arg)
		}
		fmt.Fprintf(os.Stderr, "exec: '%s' %s\n", name, strings.Join(fmtArgs, " "))
	}

	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	return cmd
}

func execCmd(name string, args []string) error {
	cmd := execCmdImpl(name, args)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
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

func runIyokan(args0 []string, args1 []string) error {
	iyokanPath, err := getPathOf("IYOKAN")
	if err != nil {
		return err
	}

	// Run iyokan
	args := append(args0, args1...)
	return execCmd(iyokanPath, args)
}

func packELF(
	inputFileName, outputFileName string,
	cmdOpts []string,
	romSize, ramSize uint64,
) error {
	if !fileExists(inputFileName) {
		return errors.New("File not found")
	}
	rom, ram, err := parseELF(inputFileName, romSize, ramSize)
	if err != nil {
		return err
	}
	if err = attachCommandLineOptions(ram, cmdOpts); err != nil {
		return err
	}

	args := []string{
		"pack",
		"--out", outputFileName,
	}

	// Write ROM data
	romTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer os.Remove(romTmpFile.Name())
	if _, err = romTmpFile.Write(rom); err != nil {
		return err
	}
	args = append(args, "--rom", "rom:"+romTmpFile.Name())

	// Write RAM data
	ramTmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer os.Remove(ramTmpFile.Name())
	if _, err = ramTmpFile.Write(ram); err != nil {
		return err
	}
	args = append(args, "--ram", "ram:"+ramTmpFile.Name())

	// Pack
	if _, err = runIyokanPacket(args...); err != nil {
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
	mapRam := make(map[string]plainPacketEntryTOML)
	for _, entry := range pktTOML.Ram {
		if _, exists := mapRam[entry.Name]; exists {
			return errors.New("Invalid TOML data: same entry name in ram")
		}
		mapRam[entry.Name] = entry
	}
	pkt.Ram = nil
	if entry, ok := mapRam["ram"]; ok {
		if entry.Size%8 != 0 {
			return errors.New("Invalid RAM data: size is not multiple of 8")
		}
		pkt.Ram = make([]int, entry.Size/8)
		for addr := range entry.Bytes {
			pkt.Ram[addr] = entry.Bytes[addr]
		}
	} else {
		return errors.New("Invalid TOML for result packet")
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
	for addr := 0; addr < len(pkt.Ram); addr++ {
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
	args := []string{"-target", "cahp", "-mcpu=generic", "-Oz", "--sysroot", cahpRtPath}
	args = append(args, os.Args[2:]...)
	return execCmd(path, args)
}

func doDebug() error {
	// Get the path of cahp-sim
	path, err := getPathOf("CAHP_SIM")
	if err != nil {
		return err
	}

	// Run
	return execCmd(path, os.Args[2:])
}

func doEmu() error {
	// Parse command-line arguments.
	fs := flag.NewFlagSet("emu", flag.ExitOnError)
	var (
		whichCAHPCPU = fs.String("cahp-cpu", defaultCAHPProc, "Which CAHP CPU you use, ruby or pearl")
		iyokanArgs   arrayFlags
	)
	fs.Var(&iyokanArgs, "iyokan-args", "Raw arguments for Iyokan")
	err := fs.Parse(os.Args[2:])

	// Create tmp file for packing
	packedFile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer os.Remove(packedFile.Name())

	// Pack
	err = packELF(fs.Args()[0], packedFile.Name(), fs.Args()[1:], defaultROMSize, defaultRAMSize)
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
	err = runIyokan([]string{"plain", "-i", packedFile.Name(), "-o", resTmpFile.Name(), "--blueprint", blueprint}, iyokanArgs)
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
		keyFileName    = fs.String("k", "", "Secret key file name")
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
	err = packELF(*inputFileName, packedFile.Name(), fs.Args(), defaultROMSize, defaultRAMSize)
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

func doGenbkey() error {
	// Parse command-line arguments.
	fs := flag.NewFlagSet("genbkey", flag.ExitOnError)
	var (
		inputFileName  = fs.String("i", "", "Input file name (secret key)")
		outputFileName = fs.String("o", "", "Output file name (bootstrapping key)")
	)
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if *inputFileName == "" || *outputFileName == "" {
		return errors.New("Specify -i and -o options properly")
	}

	_, err = runIyokanPacket("genbkey",
		"--in", *inputFileName,
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

	return packELF(*inputFileName, *outputFileName, fs.Args(), defaultROMSize, defaultRAMSize)
}

func doRun() error {
	// Parse command-line arguments.
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	var (
		nClocks          = fs.Uint("c", 0, "Number of clocks to run")
		bkeyFileName     = fs.String("bkey", "", "Bootstrapping key file name")
		inputFileName    = fs.String("i", "", "Input file name (encrypted)")
		outputFileName   = fs.String("o", "", "Output file name (encrypted)")
		numGPU           = fs.Uint("g", 0, "Number of GPUs (Unspecify or set 0 for CPU mode)")
		whichCAHPCPU     = fs.String("cahp-cpu", defaultCAHPProc, "Which CAHP CPU you use, ruby or pearl")
		snapshotFileName = fs.String("snapshot", "", "Snapshot file name to write in")
		quiet            = fs.Bool("quiet", false, "Be quiet")
		iyokanArgs       arrayFlags
	)
	fs.Var(&iyokanArgs, "iyokan-args", "Raw arguments for Iyokan")
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}

	if *nClocks == 0 || *bkeyFileName == "" || *inputFileName == "" || *outputFileName == "" {
		return errors.New("Specify -c, -bkey, -i, and -o options properly")
	}

	blueprint, err := getPathOf(fmt.Sprintf("IYOKAN-BLUEPRINT-%s", strings.ToUpper(*whichCAHPCPU)))
	if err != nil {
		return err
	}

	args := []string{
		"-i", *inputFileName,
		"--blueprint", blueprint,
	}
	if *numGPU > 0 {
		args = append(args, "--enable-gpu", "--gpu_num", fmt.Sprint(*numGPU))
	}

	return runIyokanTFHE(*nClocks, *bkeyFileName, *outputFileName, *snapshotFileName, *quiet, args, iyokanArgs)
}

func doResume() error {
	// Parse command-line arguments.
	fs := flag.NewFlagSet("resume", flag.ExitOnError)
	var (
		nClocks          = fs.Uint("c", 0, "Number of clocks to run")
		bkeyFileName     = fs.String("bkey", "", "Bootstrapping key file name")
		inputFileName    = fs.String("i", "", "Snapshot file to resume from")
		outputFileName   = fs.String("o", "", "Output file name (encrypted)")
		snapshotFileName = fs.String("snapshot", "", "Snapshot file name to write in")
		quiet            = fs.Bool("quiet", false, "Be quiet")
		iyokanArgs       arrayFlags
	)
	fs.Var(&iyokanArgs, "iyokan-args", "Raw arguments for Iyokan")
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}

	if *nClocks == 0 || *bkeyFileName == "" || *inputFileName == "" || *outputFileName == "" {
		return errors.New("Specify -c, -bkey, -i, and -o options properly")
	}

	args := []string{
		"--resume", *inputFileName,
	}
	return runIyokanTFHE(*nClocks, *bkeyFileName, *outputFileName, *snapshotFileName, *quiet, args, iyokanArgs)
}

func runIyokanTFHE(nClocks uint, bkeyFileName string, outputFileName string, snapshotFileName string, quiet bool, otherArgs0 []string, otherArgs1 []string) error {
	var err error

	if snapshotFileName == "" {
		snapshotFileName = fmt.Sprintf(
			"kvsp_%s.snapshot", time.Now().Format("20060102150405"))
	}

	args := []string{
		"tfhe",
		"--bkey", bkeyFileName,
		"-o", outputFileName,
		"-c", fmt.Sprint(nClocks),
		"--snapshot", snapshotFileName,
	}
	if quiet {
		args = append(args, "--quiet")
	}
	args = append(args, otherArgs0...)
	args = append(args, otherArgs1...)
	if err = runIyokan(args, []string{}); err != nil {
		return err
	}

	if !quiet {
		fmt.Printf("\n")
		fmt.Printf("Snapshot was taken as file '%s'. You can resume the process like:\n", snapshotFileName)
		fmt.Printf("\t$ %s resume -c %d -i %s -o %s -bkey %s\n",
			os.Args[0], nClocks, snapshotFileName, outputFileName, bkeyFileName)
	}

	return nil
}

var kvspVersion = "unk"
var kvspRevision = "unk"
var iyokanRevision = "unk"
var iyokanL1Revision = "unk"
var cahpRubyRevision = "unk"
var cahpPearlRevision = "unk"
var cahpRtRevision = "unk"
var cahpSimRevision = "unk"
var llvmCahpRevision = "unk"
var yosysRevision = "unk"

func doVersion() error {
	fmt.Printf("KVSP v29+1KiB ROM/RAM\t(rev %s)\n", kvspRevision)
	fmt.Printf("- Iyokan\t(rev %s)\n", iyokanRevision)
	fmt.Printf("- Iyokan-L1\t(rev %s)\n", iyokanL1Revision)
	fmt.Printf("- cahp-ruby\t(rev %s)\n", cahpRubyRevision)
	fmt.Printf("- cahp-pearl\t(rev %s)\n", cahpPearlRevision)
	fmt.Printf("- cahp-rt\t(rev %s)\n", cahpRtRevision)
	fmt.Printf("- cahp-sim\t(rev %s)\n", cahpSimRevision)
	fmt.Printf("- llvm-cahp\t(rev %s)\n", llvmCahpRevision)
	fmt.Printf("- Yosys\t(rev %s)\n", yosysRevision)
	return nil
}

func main() {
	if envvarVerbose := os.Getenv("KVSP_VERBOSE"); envvarVerbose == "1" {
		flagVerbose = true
	}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: %s COMMAND [OPTIONS]... ARGS...

KVSP is the first virtual secure platform in the world, which makes your life better.

Commands:
	cc
	debug
	dec
	emu
	enc
	genkey
	genbkey
	plainpacket
	resume
	run
	version
`, os.Args[0])
		flag.PrintDefaults()
	}

	if len(os.Args) <= 1 {
		flag.Usage()
		os.Exit(1)
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
	case "genbkey":
		err = doGenbkey()
	case "plainpacket":
		err = doPlainpacket()
	case "resume":
		err = doResume()
	case "run":
		err = doRun()
	case "version":
		err = doVersion()
	default:
		flag.Usage()
		os.Exit(1)
	}

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
