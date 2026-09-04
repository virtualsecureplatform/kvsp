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
var evaluatorBackend = "tangor"

const defaultCPU = "ruby"
const ramBaseAddr = 0x10000

type cpuProfile struct {
	Name                    string
	RuntimeName             string
	BlueprintName           string
	InitializationOnlyReset bool
	ROMSize                 uint64
	RAMSize                 uint64
	PointerWidth            int
	StackAlign              int
	StackPointerOffset      uint64
	RegCount                int
	RegWidth                int
}

var cpuProfiles = map[string]cpuProfile{
	"ruby": {
		Name:               "ruby",
		RuntimeName:        "CAHP_RT",
		BlueprintName:      "IYOKAN-BLUEPRINT-RUBY",
		ROMSize:            4 * 1024,
		RAMSize:            1024,
		PointerWidth:       2,
		StackAlign:         2,
		StackPointerOffset: 1024 - 2,
		RegCount:           16,
		RegWidth:           16,
	},
	"pearl": {
		Name:               "pearl",
		RuntimeName:        "CAHP_RT",
		BlueprintName:      "IYOKAN-BLUEPRINT-PEARL",
		ROMSize:            4 * 1024,
		RAMSize:            1024,
		PointerWidth:       2,
		StackAlign:         2,
		StackPointerOffset: 1024 - 2,
		RegCount:           16,
		RegWidth:           16,
	},
	"alexandrite": {
		Name:               "alexandrite",
		RuntimeName:        "ALEXANDRITE_RT",
		BlueprintName:      "IYOKAN-BLUEPRINT-ALEXANDRITE",
		ROMSize:            4 * 1024,
		RAMSize:            1024,
		PointerWidth:       4,
		StackAlign:         4,
		StackPointerOffset: 8,
		RegCount:           32,
		RegWidth:           32,
	},
	"chrysoberyl": {
		Name:                    "chrysoberyl",
		RuntimeName:             "ALEXANDRITE_RT",
		BlueprintName:           "IYOKAN-BLUEPRINT-CHRYSOBERYL",
		InitializationOnlyReset: true,
		ROMSize:                 4 * 1024,
		RAMSize:                 1024,
		PointerWidth:            4,
		StackAlign:              4,
		StackPointerOffset:      8,
		RegCount:                32,
		RegWidth:                32,
	},
}

func addResetMode(args []string, profile cpuProfile) []string {
	if profile.InitializationOnlyReset {
		return append(args, "--skip-reset")
	}
	return args
}

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

func getCPUProfile(name string) (cpuProfile, error) {
	name = strings.ToLower(name)
	profile, ok := cpuProfiles[name]
	if !ok {
		return cpuProfile{}, fmt.Errorf("unknown CPU %q", name)
	}
	return profile, nil
}

func resolveCPU(cpuName, cahpCPUName string) (cpuProfile, error) {
	cpuName = strings.ToLower(cpuName)
	cahpCPUName = strings.ToLower(cahpCPUName)

	if cpuName != "" && cahpCPUName != "" && cpuName != cahpCPUName {
		return cpuProfile{}, errors.New("--cpu and --cahp-cpu specify different CPUs")
	}
	if cpuName == "" {
		cpuName = cahpCPUName
	}
	if cpuName == "" {
		cpuName = defaultCPU
	}

	if cahpCPUName != "" && cahpCPUName != "ruby" && cahpCPUName != "pearl" {
		return cpuProfile{}, errors.New("--cahp-cpu accepts only ruby or pearl")
	}

	return getCPUProfile(cpuName)
}

func addCPUFlags(fs *flag.FlagSet) (*string, *string) {
	cpuName := fs.String("cpu", "", "CPU target: ruby, pearl, alexandrite, or chrysoberyl")
	cahpCPUName := fs.String("cahp-cpu", "", "Compatibility alias for --cpu ruby|pearl")
	return cpuName, cahpCPUName
}

func addBackendFlag(fs *flag.FlagSet) *string {
	return fs.String("backend", "tangor", "Evaluator backend: tangor or iyokan")
}

func selectBackend(name string) error {
	switch strings.ToLower(name) {
	case "iyokan", "tangor":
		evaluatorBackend = strings.ToLower(name)
		return nil
	default:
		return fmt.Errorf("unknown evaluator backend %q (expected iyokan or tangor)", name)
	}
}

func stripCompilerCPUArgs(args []string) (cpuProfile, []string, error) {
	cpuName := ""
	cahpCPUName := ""
	out := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--cpu":
			if i+1 >= len(args) {
				return cpuProfile{}, nil, errors.New("--cpu requires a value")
			}
			i++
			cpuName = args[i]
		case strings.HasPrefix(arg, "--cpu="):
			cpuName = strings.TrimPrefix(arg, "--cpu=")
		case arg == "--cahp-cpu":
			if i+1 >= len(args) {
				return cpuProfile{}, nil, errors.New("--cahp-cpu requires a value")
			}
			i++
			cahpCPUName = args[i]
		case strings.HasPrefix(arg, "--cahp-cpu="):
			cahpCPUName = strings.TrimPrefix(arg, "--cahp-cpu=")
		default:
			out = append(out, arg)
		}
	}

	profile, err := resolveCPU(cpuName, cahpCPUName)
	return profile, out, err
}

func isCompileOnly(args []string) bool {
	for _, arg := range args {
		if arg == "-c" || arg == "-S" || arg == "-E" {
			return true
		}
	}
	return false
}

func writeLE(out []byte, val uint64) {
	for i := range out {
		out[i] = byte((val >> (8 * i)) & 0xff)
	}
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
		case "ALEXANDRITE_RT":
			path = "../share/kvsp/alexandrite-rt"
		case "CAHP_RT":
			path = "../share/kvsp/cahp-rt"
		case "CAHP_SIM":
			path = "cahp-sim"
		case "CLANG":
			path = "clang"
		case "IYOKAN":
			if evaluatorBackend == "tangor" {
				path = "tangor-iyokan"
			} else {
				path = "iyokan"
			}
		case "IYOKAN-BLUEPRINT-RUBY":
			path = "../share/kvsp/cahp-ruby.toml"
		case "IYOKAN-BLUEPRINT-PEARL":
			path = "../share/kvsp/cahp-pearl.toml"
		case "IYOKAN-BLUEPRINT-ALEXANDRITE":
			path = "../share/kvsp/alexandrite.toml"
		case "IYOKAN-BLUEPRINT-CHRYSOBERYL":
			path = "../share/kvsp/chrysoberyl.toml"
		case "IYOKAN-PACKET":
			if evaluatorBackend == "tangor" {
				path = "tangor-iyokan-packet"
			} else {
				path = "iyokan-packet"
			}
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
		// The AVX2 Tangor build uses suffixed binary names. Prefer the
		// release/AVX512 names above, then transparently use this portable build.
		if relative && evaluatorBackend == "tangor" {
			var fallback string
			switch name {
			case "IYOKAN":
				fallback = "tangor-iyokan-avx2"
			case "IYOKAN-PACKET":
				fallback = "tangor-iyokan-packet-avx2"
			}
			if fallback != "" {
				fallbackPath, err := prefixExecDir(fallback)
				if err != nil {
					return "", err
				}
				if fileExists(fallbackPath) {
					return fallbackPath, nil
				}
			}
		}
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
		if prog.ProgHeader.Type != elf.PT_LOAD {
			continue
		}
		addr := prog.ProgHeader.Vaddr
		size := prog.ProgHeader.Filesz
		if size == 0 {
			continue
		}

		var mem []byte
		if addr < ramBaseAddr { // ROM
			if addr+size > romSize {
				return nil, nil, errors.New("Invalid ROM size: too small")
			}
			mem = rom[addr : addr+size]
		} else { // RAM
			if addr-ramBaseAddr+size > ramSize {
				return nil, nil, errors.New("Invalid RAM size: too small")
			}
			mem = ram[addr-ramBaseAddr : addr-ramBaseAddr+size]
		}

		reader := prog.Open()
		if _, err := io.ReadFull(reader, mem); err != nil {
			return nil, nil, err
		}
	}

	return rom, ram, nil
}

func attachCommandLineOptions(ram []byte, cmdOptsSrc []string, profile cpuProfile) error {
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
	stackTop := ramSize
	if profile.StackPointerOffset+uint64(profile.PointerWidth) == uint64(ramSize) {
		stackTop = int(profile.StackPointerOffset)
	}
	index := stackTop

	// Set **argv to RAM
	for i := len(cmdOpts) - 1; i >= 0; i-- {
		opt := append([]byte(cmdOpts[i]), 0)
		for j := len(opt) - 1; j >= 0; j-- {
			index--
			if index < 0 {
				return errors.New("Invalid RAM size: command line arguments do not fit")
			}
			ram[index] = opt[j]
		}
		sargv = append(sargv, index)
	}
	// Align index
	index -= index % profile.StackAlign
	if index < 0 {
		return errors.New("Invalid RAM size: command line arguments do not fit")
	}
	// Set *argv to RAM
	for _, val := range sargv {
		index -= profile.PointerWidth
		if index < 0 {
			return errors.New("Invalid RAM size: command line arguments do not fit")
		}
		writeLE(ram[index:index+profile.PointerWidth], uint64(val))
	}
	// Save argc in RAM
	index -= profile.PointerWidth
	if index < 0 {
		return errors.New("Invalid RAM size: command line arguments do not fit")
	}
	writeLE(ram[index:index+profile.PointerWidth], uint64(argc))
	// Save initial stack pointer in RAM
	initSP := index
	if profile.StackPointerOffset+uint64(profile.PointerWidth) > uint64(len(ram)) {
		return errors.New("Invalid CPU profile: stack pointer slot is outside RAM")
	}
	spOffset := int(profile.StackPointerOffset)
	writeLE(ram[spOffset:spOffset+profile.PointerWidth], uint64(initSP))

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
	profile cpuProfile,
) error {
	if !fileExists(inputFileName) {
		return errors.New("File not found")
	}
	rom, ram, err := parseELF(inputFileName, profile.ROMSize, profile.RAMSize)
	if err != nil {
		return err
	}
	if err = attachCommandLineOptions(ram, cmdOpts, profile); err != nil {
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

func bytesToLE(bytes []int, bitWidth int) (int, error) {
	byteWidth := bitWidth / 8
	if bitWidth%8 != 0 || len(bytes) < byteWidth {
		return 0, errors.New("Invalid TOML for result packet")
	}
	val := 0
	for i := 0; i < byteWidth; i++ {
		val |= bytes[i] << (8 * i)
	}
	return val, nil
}

func (pkt *plainPacket) loadTOML(src string, profile cpuProfile) error {
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
			if len(entry.Bytes) < 1 {
				return errors.New("Invalid TOML for result packet")
			}
			if entry.Bytes[0] != 0 {
				pkt.Flags[entry.Name] = true
			} else {
				pkt.Flags[entry.Name] = false
			}
		} else if entry.Size == profile.RegWidth { // register
			val, err := bytesToLE(entry.Bytes, entry.Size)
			if err != nil {
				return err
			}
			pkt.Regs[entry.Name] = val
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
	for i := 0; i < profile.RegCount; i++ {
		name := fmt.Sprintf("reg_x%d", i)
		if _, ok := pkt.Regs[name]; !ok {
			return errors.New("Invalid TOML for result packet: '" + name + "' not found")
		}
	}

	return nil
}
func (pkt *plainPacket) print(w io.Writer, profile cpuProfile) error {
	fmt.Fprintf(w, "#cycle\t%d\n", pkt.NumCycles)
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "f0\t%t\n", pkt.Flags["finflag"])
	fmt.Fprintf(w, "\n")
	for i := 0; i < profile.RegCount; i++ {
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
	profile, userArgs, err := stripCompilerCPUArgs(os.Args[2:])
	if err != nil {
		return err
	}

	// Get the path of clang
	path, err := getPathOf("CLANG")
	if err != nil {
		return err
	}

	rtPath, err := getPathOf(profile.RuntimeName)
	if err != nil {
		return err
	}

	var args []string
	switch profile.Name {
	case "ruby", "pearl":
		args = []string{"-target", "cahp", "-mcpu=generic", "-Oz", "--sysroot", rtPath}
		args = append(args, userArgs...)
	case "alexandrite", "chrysoberyl":
		march := "rv32i"
		if profile.Name == "chrysoberyl" {
			march = "rv32ic_zba_zbb_zcb"
		}
		args = []string{
			"-target", "riscv32-unknown-elf",
			"-march=" + march,
			"-mabi=ilp32",
			"-Oz",
			"-ffreestanding",
			"-fno-builtin",
			"-fno-unwind-tables",
			"-fno-asynchronous-unwind-tables",
			"-isystem", rtPath,
		}
		if isCompileOnly(userArgs) {
			args = append(args, userArgs...)
		} else {
			args = append(args,
				"-fuse-ld=lld",
				"-nostdlib",
				filepath.Join(rtPath, "crt0.o"),
			)
			args = append(args, userArgs...)
			args = append(args,
				"-Wl,-T,"+filepath.Join(rtPath, "alexandrite.lds"),
				"-L", rtPath,
				"-lc",
			)
		}
	default:
		return errors.New("unreachable")
	}
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
		iyokanArgs arrayFlags
	)
	cpuName, cahpCPUName := addCPUFlags(fs)
	backend := addBackendFlag(fs)
	fs.Var(&iyokanArgs, "iyokan-args", "Raw arguments for Iyokan")
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if err := selectBackend(*backend); err != nil {
		return err
	}
	profile, err := resolveCPU(*cpuName, *cahpCPUName)
	if err != nil {
		return err
	}

	// Create tmp file for packing
	packedFile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer os.Remove(packedFile.Name())

	// Pack
	err = packELF(fs.Args()[0], packedFile.Name(), fs.Args()[1:], profile)
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
	blueprint, err := getPathOf(profile.BlueprintName)
	if err != nil {
		return err
	}
	args := []string{"plain", "-i", packedFile.Name(), "-o", resTmpFile.Name(), "--blueprint", blueprint}
	args = addResetMode(args, profile)
	err = runIyokan(args, iyokanArgs)
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
	if err := pkt.loadTOML(result, profile); err != nil {
		return err
	}
	pkt.print(os.Stdout, profile)

	return nil
}

func doDec() error {
	// Parse command-line arguments.
	fs := flag.NewFlagSet("dec", flag.ExitOnError)
	var (
		keyFileName   = fs.String("k", "", "Key file name")
		inputFileName = fs.String("i", "", "Input file name (encrypted)")
	)
	cpuName, cahpCPUName := addCPUFlags(fs)
	backend := addBackendFlag(fs)
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if err := selectBackend(*backend); err != nil {
		return err
	}
	profile, err := resolveCPU(*cpuName, *cahpCPUName)
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
	if err != nil {
		return err
	}

	// Unpack
	result, err := runIyokanPacket("packet2toml", "--in", packedFile.Name())
	if err != nil {
		return err
	}

	// Parse and print the result
	var pkt plainPacket
	if err := pkt.loadTOML(result, profile); err != nil {
		return err
	}
	pkt.print(os.Stdout, profile)

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
	cpuName, cahpCPUName := addCPUFlags(fs)
	backend := addBackendFlag(fs)
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if err := selectBackend(*backend); err != nil {
		return err
	}
	profile, err := resolveCPU(*cpuName, *cahpCPUName)
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
	err = packELF(*inputFileName, packedFile.Name(), fs.Args(), profile)
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
	backend := addBackendFlag(fs)
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if err := selectBackend(*backend); err != nil {
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
	backend := addBackendFlag(fs)
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if err := selectBackend(*backend); err != nil {
		return err
	}
	if *inputFileName == "" || *outputFileName == "" {
		return errors.New("Specify -i and -o options properly")
	}

	_, err = runIyokanPacket("genevalkey",
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
	cpuName, cahpCPUName := addCPUFlags(fs)
	backend := addBackendFlag(fs)
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if err := selectBackend(*backend); err != nil {
		return err
	}
	profile, err := resolveCPU(*cpuName, *cahpCPUName)
	if err != nil {
		return err
	}
	if *inputFileName == "" || *outputFileName == "" {
		return errors.New("Specify -i, and -o options properly")
	}

	return packELF(*inputFileName, *outputFileName, fs.Args(), profile)
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
		snapshotFileName = fs.String("snapshot", "", "Snapshot file name to write in")
		quiet            = fs.Bool("quiet", false, "Be quiet")
		iyokanArgs       arrayFlags
	)
	cpuName, cahpCPUName := addCPUFlags(fs)
	backend := addBackendFlag(fs)
	fs.Var(&iyokanArgs, "iyokan-args", "Raw arguments for Iyokan")
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if err := selectBackend(*backend); err != nil {
		return err
	}
	profile, err := resolveCPU(*cpuName, *cahpCPUName)
	if err != nil {
		return err
	}

	if *nClocks == 0 || *bkeyFileName == "" || *inputFileName == "" || *outputFileName == "" {
		return errors.New("Specify -c, -bkey, -i, and -o options properly")
	}

	blueprint, err := getPathOf(profile.BlueprintName)
	if err != nil {
		return err
	}

	args := []string{
		"-i", *inputFileName,
		"--blueprint", blueprint,
	}
	args = addResetMode(args, profile)
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
	backend := addBackendFlag(fs)
	fs.Var(&iyokanArgs, "iyokan-args", "Raw arguments for Iyokan")
	err := fs.Parse(os.Args[2:])
	if err != nil {
		return err
	}
	if err := selectBackend(*backend); err != nil {
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
		"--evalkey", bkeyFileName,
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
var tangorRevision = "unk"
var alexandriteRevision = "unk"
var chrysoberylRevision = "unk"
var alexandriteRtRevision = "unk"
var cahpRubyRevision = "unk"
var cahpPearlRevision = "unk"
var cahpRtRevision = "unk"
var cahpSimRevision = "unk"
var llvmCahpRevision = "unk"
var yosysRevision = "unk"

func doVersion() error {
	fmt.Printf("KVSP %s\t(rev %s)\n", kvspVersion, kvspRevision)
	fmt.Printf("- Iyokan\t(rev %s)\n", iyokanRevision)
	fmt.Printf("- Tangor\t(rev %s)\n", tangorRevision)
	fmt.Printf("- Alexandrite\t(rev %s)\n", alexandriteRevision)
	fmt.Printf("- Chrysoberyl\t(rev %s)\n", chrysoberylRevision)
	fmt.Printf("- alexandrite-rt\t(rev %s)\n", alexandriteRtRevision)
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
