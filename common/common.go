package common

import (
	"fmt"
	"io"
)

const (
	// pakcage version
	MajorVersion = 1
	MinorVersion = 1
	FixVersion   = 1
)

// make up version
func makeUpVersion(major, minor, fix uint8) uint32 {
	return uint32(major)<<24 | uint32(minor)<<16 | uint32(fix)<<8
}

// break down version
func breadDownVersion(version uint32) (uint8, uint8, uint8) {
	return uint8(version >> 24), uint8(version >> 16), uint8(version >> 8)
}

const (
	// package type definition
	Message     PackageType = 0x01
	Stream      PackageType = 0x02
	UserDefined PackageType = 0x04

	// flags
	FlagCompressed = 0x80

	// the len of magic sequence
	VersionSize      = 4
	PackageTypeSize  = 1
	PackageFlagsSize = 1
	PayloadLenSize   = 4
	HeaderSize       = VersionSize + PackageTypeSize + PackageFlagsSize + PayloadLenSize

	// filed offsets
	VersionOffset     = 0
	PackageTypeOffset = VersionSize
	FlagsOffset       = VersionSize + PackageTypeSize
	PayloadLenOffset  = VersionSize + PackageTypeSize + PackageFlagsSize
)

type PackageType uint8

type PackageHeader struct {
	// major_version: Version << 24
	// minor_version: Version << 16
	// fix_version: Version << 8
	// Version => {major_version} | {minor_version} | {fix_version}
	Version uint32

	// the package type
	// message package: 0x01
	// stream package: 0x02
	// user-defined package: 0x04
	PackageType PackageType

	// flags
	Flags uint8

	// payload length
	// the size of package payload
	PayloadLen uint32
}

// new package
func NewPackageHeader(packageType PackageType) *PackageHeader {
	return &PackageHeader{
		PackageType: packageType,
		Version:     makeUpVersion(MajorVersion, MinorVersion, FixVersion),
	}
}

// set version
func (h *PackageHeader) SetVersion(version uint32) *PackageHeader {
	h.Version = version
	return h
}

// get version
func (h *PackageHeader) GetVersion() uint32 {
	return h.Version
}

// set payload len
func (h *PackageHeader) SetPayloadLen(payloadLen uint32) *PackageHeader {
	h.PayloadLen = payloadLen
	return h
}

// get payload len of package
func (h *PackageHeader) GetPayloadLen() uint32 {
	return h.PayloadLen
}

// set package type
func (h *PackageHeader) SetPackageType(packageType PackageType) *PackageHeader {
	h.PackageType = packageType
	return h
}

// get package type
func (h *PackageHeader) GetPackageType() PackageType {
	return h.PackageType
}

// set package flags
func (h *PackageHeader) SetFlags(flags uint8) *PackageHeader {
	h.Flags = flags
	return h
}

// get package flags
func (h *PackageHeader) GetFlags() uint8 {
	return h.Flags
}


// do packing
func (h *PackageHeader) Pack(buffer *[]byte) {
	*buffer = append(*buffer,
		byte(h.Version>>24),
		byte(h.Version>>16),
		byte(h.Version>>8),
		byte(h.Version),
		byte(h.PackageType),
		byte(h.Flags),
		byte(h.PayloadLen>>24),
		byte(h.PayloadLen>>16),
		byte(h.PayloadLen>>8),
		byte(h.PayloadLen))
}

// do unpacking
func (h *PackageHeader) Unpack(header []byte) {
	h.Version = uint32(header[VersionOffset])<<24 |
		uint32(header[VersionOffset+1])<<16 |
		uint32(header[VersionOffset+2])<<8 |
		uint32(header[VersionOffset+3])
	h.PackageType = PackageType(header[PackageTypeOffset])
	h.Flags = uint8(header[FlagsOffset])
	h.PayloadLen = uint32(header[PayloadLenOffset])<<24 |
		uint32(header[PayloadLenOffset+1])<<16 |
		uint32(header[PayloadLenOffset+2])<<8 |
		uint32(header[PayloadLenOffset+3])
}


type Writer struct {
	Writer io.Writer
}

// new Writer instance
func NewWriter(w io.Writer) *Writer {
	return &Writer{Writer: w}
}

// Write message raw data
// steps:
// 1) packer the package header
// 2) write header
// 3) write message raw data
func (w *Writer) Write(data []byte) (int, error) {
	if w.Writer == nil {
		fmt.Println("bad io writer")
		return 0, fmt.Errorf("bad io writer")
	}

	// packing header
	header := NewPackageHeader(Message)
	header.SetPayloadLen(uint32(len(data)))
	var headerBuffer []byte
	header.Pack(&headerBuffer)

	// write header
	_, err := w.Writer.Write(headerBuffer)
	if err != nil {
		fmt.Println("failed to write header")
		return 0, err
	}

	// write payload
	_, err = w.Writer.Write(data)
	if err != nil {
		fmt.Println("failed to write payload")
		return 0, err
	}
	return len(data), nil
}

type Reader struct {
	Reader io.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{Reader: r}
}

// Read message raw data from reader
// steps:
// 1)read the package header
// 2)unpack the package header and get the payload length
// 3)read the payload
func (r *Reader) Read() ([]byte, error) {
	if r.Reader == nil {
		fmt.Println("bad io reader")
		return nil, fmt.Errorf("bad io reader")
	}

	headerBuffer := make([]byte, HeaderSize)
	_, err := io.ReadFull(r.Reader, headerBuffer)
	if err != nil {
		if err != io.EOF {
			fmt.Println("failed to read package header from buffer")
		}
		return nil, err
	}

	header := PackageHeader{}
	header.Unpack(headerBuffer)

	payloadBuffer := make([]byte, header.PayloadLen)
	_, err = io.ReadFull(r.Reader, payloadBuffer)
	if err != nil {
		if err != io.EOF {
			fmt.Println("failed to read payload from buffer")
		}
		return nil, err
	}

	return payloadBuffer, nil
}
