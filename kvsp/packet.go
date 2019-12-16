package main

import (
	"encoding/binary"
	"errors"
	"io"
)

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

// KVSPPlainReqPacketHeader represents KVSP plain request packet header
type KVSPPlainReqPacketHeader struct {
	KVSP    [4]byte // VSP signature
	Version uint32  // Version
	ROMSize uint64  // Size of ROM
	RAMSize uint64  // Size of RAM
}

// KVSPPlainReqPacket represents KVSP plain request packet
type KVSPPlainReqPacket struct {
	KVSPPlainReqPacketHeader
	ROM []byte
	RAM []byte
}

// KVSPPlainResPacketHeader represents KVSP plain response packet header
type KVSPPlainResPacketHeader struct {
	KVSP      [4]byte // VSP signature
	Version   uint32  // Version
	Nflags    uint16  // Number of flags
	Nregs     uint16  // Number of registers
	FlagsSize uint64  // Size of flags
	RegsSize  uint64  // Size of registers
	RAMSize   uint64  // Size of RAM
}

// KVSPPlainResPacket represents KVSP plain response packet
type KVSPPlainResPacket struct {
	KVSPPlainResPacketHeader
	Flags []bool
	Regs  []uint16
	RAM   []uint8
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

// WriteTo will dump the content to writer
// FIXME: The first return value is dummy
func (packet *KVSPPlainReqPacket) WriteTo(writer io.Writer) (int64, error) {
	// Write packet's header
	err := binary.Write(writer, binary.LittleEndian, packet.KVSPPlainReqPacketHeader)
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

// ReadFrom will read KVSP plain response packet
// FIXME: The first return value is dummy
func (packet *KVSPPlainResPacket) ReadFrom(reader io.Reader) (int64, error) {
	// Read the header
	err := binary.Read(reader, binary.LittleEndian, &packet.KVSPPlainResPacketHeader)
	if err != nil {
		return 0, err
	}

	// Check if the header is correct
	header := &packet.KVSPPlainResPacketHeader
	if header.KVSP != [4]byte{'K', 'V', 'S', 'P'} || header.Version != 0 {
		return 0, errors.New("Invalid signature")
	}

	packet.Flags = make([]bool, header.Nflags)
	flagsBuf := make([]byte, header.Nflags)
	_, err = reader.Read(flagsBuf)
	if err != nil {
		return 0, err
	}
	for i := range flagsBuf {
		b := flagsBuf[i]
		if b != 0 {
			packet.Flags[i] = true
		} else {
			packet.Flags[i] = false
		}
	}

	packet.Regs = make([]uint16, header.Nregs)
	for i := range packet.Regs {
		err := binary.Read(reader, binary.LittleEndian, &packet.Regs[i])
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
