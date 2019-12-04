package main

import (
	"debug/elf"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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

	switch name {
	case "clang":
		path = "../llvm-cahp/bin/clang"
	case "tfheutil":
		path = "../tfheutil/tfheutil"
	case "cahp-sim":
		path = "../cahp-sim/cahp-sim"
	case "cahp-rt":
		path = "../cahp-rt"
	default:
		return "", errors.New("Invalid name")
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
	tfheutilPath, err := getPathOf("tfheutil")
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
	tfheutilPath, err := getPathOf("tfheutil")
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
	tfheutilPath, err := getPathOf("tfheutil")
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
	tfheutilPath, err := getPathOf("tfheutil")
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
	tfheutilPath, err := getPathOf("tfheutil")
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

// KVSPReqPacketHeader represents KVSP request packet header
type KVSPReqPacketHeader struct {
	KVSP         [4]byte // VSP signature
	Version      uint32  // Version
	CloudKeySize uint64  // Size of cloud key
	ROMSize      uint64  // Size of ROM
	RAMSize      uint64  // Size of RAM
}

// KVSPReqPacket represents KVSP request packet
type KVSPReqPacket struct {
	KVSPReqPacketHeader
	CloudKey []byte
	ROM      []byte
	RAM      []byte
}

// WriteTo will dump the content to writer
// FIXME: The first return value is dummy
func (packet *KVSPReqPacket) WriteTo(writer io.Writer) (int64, error) {
	// Write packet's header
	err := binary.Write(writer, binary.LittleEndian, packet.KVSPReqPacketHeader)
	if err != nil {
		return 0, err
	}

	// Write the cloud key
	_, err = writer.Write(packet.CloudKey)
	if err != nil {
		return 0, err
	}

	// Write the encrypted ROM image
	_, err = writer.Write(packet.ROM)
	if err != nil {
		return 0, err
	}

	// Write the encrypted RAM image
	_, err = writer.Write(packet.RAM)
	if err != nil {
		return 0, err
	}

	return 0, nil
}

// ReadFrom will read KVSP request packet
// FIXME: The first return value is dummy
func (packet *KVSPReqPacket) ReadFrom(reader io.Reader) (int64, error) {
	// Read the header
	err := binary.Read(reader, binary.LittleEndian, &packet.KVSPReqPacketHeader)
	if err != nil {
		return 0, err
	}

	// Check if the header is correct
	header := &packet.KVSPReqPacketHeader
	if header.KVSP != [4]byte{'K', 'V', 'S', 'P'} || header.Version != 0 {
		return 0, errors.New("Invalid signature")
	}

	packet.CloudKey = make([]byte, packet.KVSPReqPacketHeader.CloudKeySize)
	_, err = reader.Read(packet.CloudKey)
	if err != nil {
		return 0, nil
	}

	packet.ROM = make([]byte, packet.KVSPReqPacketHeader.ROMSize)
	_, err = reader.Read(packet.ROM)
	if err != nil {
		return 0, nil
	}

	packet.RAM = make([]byte, packet.KVSPReqPacketHeader.RAMSize)
	_, err = reader.Read(packet.RAM)
	if err != nil {
		return 0, nil
	}

	return 0, nil
}

// KVSPResPacketHeader represents KVSP response packet header
type KVSPResPacketHeader struct {
	KVSP      [4]byte // VSP signature
	Version   uint32  // Version
	Nflags    uint16  // Number of flags
	Nregs     uint16  // Number of registers
	FlagsSize uint64  // Size of flags
	RegsSize  uint64  // Size of registers
	RAMSize   uint64  // Size of RAM
}

// KVSPResPacket represents KVSP response packet
type KVSPResPacket struct {
	KVSPResPacketHeader
	Flags [][]byte
	Regs  [][]byte
	RAM   []byte
}

// NewKVSPResPacket is ctor of KVSPReqPacket
func NewKVSPResPacket(flags [][]byte, regs [][]byte, ram []byte) *KVSPResPacket {
	return &KVSPResPacket{
		KVSPResPacketHeader{
			[4]byte{'K', 'V', 'S', 'P'},
			0,
			uint16(len(flags)),
			uint16(len(regs)),
			uint64(len(flags) * len(flags[0])), // FIXME: Assume all slices inside have the same length
			uint64(len(regs) * len(regs[0])),   // FIXME: Assume all slices inside have the same length
			uint64(len(ram)),
		},
		flags,
		regs,
		ram,
	}
}

// WriteTo will dump the content to writer
// FIXME: The first return value is dummy
func (packet *KVSPResPacket) WriteTo(writer io.Writer) (int64, error) {
	// Write packet's header
	err := binary.Write(writer, binary.LittleEndian, packet.KVSPResPacketHeader)
	if err != nil {
		return 0, err
	}

	// Write the encrypted flags
	for _, flag := range packet.Flags {
		_, err := writer.Write(flag)
		if err != nil {
			return 0, err
		}
	}

	// Write the encrypted registers
	for _, reg := range packet.Regs {
		_, err := writer.Write(reg)
		if err != nil {
			return 0, err
		}
	}

	// Write the encrypted RAM image
	_, err = writer.Write(packet.RAM)
	if err != nil {
		return 0, err
	}

	return 0, nil
}

// ReadFrom will read KVSP response packet
// FIXME: The first return value is dummy
func (packet *KVSPResPacket) ReadFrom(reader io.Reader) (int64, error) {
	// Read the header
	err := binary.Read(reader, binary.LittleEndian, &packet.KVSPResPacketHeader)
	if err != nil {
		return 0, err
	}

	// Check if the header is correct
	header := &packet.KVSPResPacketHeader
	if header.KVSP != [4]byte{'K', 'V', 'S', 'P'} || header.Version != 0 {
		return 0, errors.New("Invalid signature")
	}

	packet.Flags = make([][]byte, header.Nflags)
	for i := range packet.Flags {
		packet.Flags[i] = make([]byte, header.FlagsSize/uint64(header.Nflags))
		_, err = reader.Read(packet.Flags[i])
		if err != nil {
			return 0, err
		}
	}

	packet.Regs = make([][]byte, header.Nregs)
	for i := range packet.Regs {
		packet.Regs[i] = make([]byte, header.RegsSize/uint64(header.Nregs))
		_, err = reader.Read(packet.Regs[i])
		if err != nil {
			return 0, err
		}
	}

	packet.RAM = make([]byte, header.RAMSize)
	_, err = reader.Read(packet.RAM)
	if err != nil {
		return 0, err
	}

	return 0, nil
}

func doCC() error {
	// Get the path of clang
	path, err := getPathOf("clang")
	if err != nil {
		fatalExit(err)
	}

	// Get the path of cahp-rt
	cahpRtPath, err := getPathOf("cahp-rt")
	if err != nil {
		fatalExit(err)
	}

	// Run
	return execCmd(path, append(os.Args[2:], "-target", "cahp", "--sysroot", cahpRtPath))
}

func doEmu() error {
	// Get the path of cahp-sim
	path, err := getPathOf("cahp-sim")
	if err != nil {
		fatalExit(err)
	}

	// Run
	return execCmd(path, os.Args[2:])
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
	} else {
		// Print the result
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

	// Parse the input as ELF and get ROM and RAM images
	input, err := elf.Open(*inputFileName)
	if err != nil {
		return err
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
			return err
		}
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
	path, err := getPathOf("tfheutil")
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

	// Run the encrypted program as is.
	// FIXME: That's just a stub
	reader, err := os.Open(*inputFileName)
	if err != nil {
		return err
	}
	inputPacket := KVSPReqPacket{}
	_, err = inputPacket.ReadFrom(reader)
	if err != nil {
		return err
	}

	flags := make([][]byte, 1)
	flags[0] = inputPacket.RAM[0 : len(inputPacket.RAM)/512/8]
	regs := make([][]byte, 16)
	for i := 0; i < 16; i++ {
		regs[i] = inputPacket.RAM[0 : len(inputPacket.RAM)/512*2]
	}

	outputPacket := KVSPResPacket{
		KVSPResPacketHeader{
			[4]byte{'K', 'V', 'S', 'P'},
			0,
			uint16(len(flags)),
			uint16(len(regs)),
			uint64(len(flags) * len(flags[0])), // FIXME: Assume all slices inside have the same length
			uint64(len(regs) * len(regs[0])),   // FIXME: Assume all slices inside have the same length
			uint64(len(inputPacket.RAM)),
		},
		flags,
		regs,
		inputPacket.RAM,
	}

	writer, err := os.Create(*outputFileName)
	if err != nil {
		return err
	}
	outputPacket.WriteTo(writer)

	return nil
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
	case "emu":
		err = doEmu()
	default:
		printUsageAndExit()
	}

	if err != nil {
		fatalExit(err)
	}
}
