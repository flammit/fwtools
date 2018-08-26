package fit

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"

	"github.com/flammit/fwtools/pkg/rom"
)

const (
	fitSignature = uint64(0x2020205f5449465f) // "_FIT_   "
	fitVersion   = 0x0100
)

var (
	fitTypes = map[uint8]string{
		0x00: "header",
		0x01: "microcode",
		0x02: "startup_acm",
		0x07: "bios_startup_module",
		0x08: "tpm_policy",
		0x09: "bios_policy",
		0x0a: "txt_policy",
		0x0b: "key_manifest",
		0x0c: "boot_policy_manifest",
		0x10: "cse_secure_boot",
		0x2d: "txtsx_policy",
		0x2f: "jmp_debug_policy",
		0x7f: "skip",
	}
)

type Entry struct {
	Address  uint64
	Len24    [3]uint8 // length of component (units of 16 bytes) or # of entries inclusive of header
	Reserved uint8
	Version  uint16
	Type     uint8 // last bit is checksum
	Checksum uint8
}

func (e Entry) ValidHeader() bool {
	return e.Address == fitSignature &&
		e.Reserved == 0 &&
		e.Version == fitVersion
}

func DetectFIT(unknownRegion *rom.Region) []*rom.Region {
	bs := bytes.NewReader(unknownRegion.Raw)

	// scan for signature
	var header Entry
	var off uint32

	off = uint32(rom.AlignUp(uint64(unknownRegion.Offset), 0x10000)) -
		unknownRegion.Offset

	// just check for global offset alignment to 0x10000
	for ; off < unknownRegion.Size; off += 0x10000 {
		bs.Seek(int64(off), io.SeekStart)
		binary.Read(bs, binary.LittleEndian, &header)
		if header.ValidHeader() {
			break
		}
	}
	if !header.ValidHeader() {
		return nil
	}

	fitRegion := unknownRegion.Child(
		unknownRegion.Offset+off,
		unknownRegion.Size-off,
		"unknown",
		"fit",
	)
	fitRegion = rom.DetectRegions(
		[]rom.Detector{detectFITRegions},
		fitRegion,
	)
	return []*rom.Region{fitRegion}
}

func detectFITRegions(unknownRegion *rom.Region) []*rom.Region {
	bs := bytes.NewReader(unknownRegion.Raw)
	var header Entry
	binary.Read(bs, binary.LittleEndian, &header)
	if !header.ValidHeader() {
		return nil
	}

	numEntries := rom.Size24(header.Len24)
	headerSize := numEntries * 0x10
	headerRegion := unknownRegion.Child(unknownRegion.Offset, headerSize,
		"raw", fitTypes[header.Type])
	regions := []*rom.Region{headerRegion}
	log.Printf("FIT Header @ 0x%08x: Num Entries(inclusive)=%v",
		unknownRegion.Offset, numEntries)

	fullSize := unknownRegion.FullSize()
	for n := uint32(0); n < numEntries-1; n++ {
		var entry Entry
		binary.Read(bs, binary.LittleEndian, &entry)
		if entry.Reserved != 0 || entry.Type == 0x7f {
			continue
		}
		log.Printf("FIT Entry %d: %#v", n, entry)
		len := rom.Size24(entry.Len24) * 0x10
		romOff := fullSize + uint32(entry.Address)
		if !unknownRegion.Contains(romOff, 0) ||
			headerRegion.Contains(romOff, 0) {
			continue
		}

		if len == 0 {
			switch entry.Type {
			case 0x02:
				len = parseStartupAcmLen(unknownRegion, romOff)
			}
		}

		if len != 0 {
			regions = append(regions, unknownRegion.Child(
				romOff, len, "raw", fitTypes[entry.Type],
			))
		}
	}
	// TODO: add dependencies on external locations
	return regions
}

type StartupAcmHeader struct {
	ModuleType    uint16 // 0x00
	ModuleSubType uint16 // 0x02
	Misc          [0x14]uint8
	Size          uint32 // 0x18
}

func parseStartupAcmLen(unknownRegion *rom.Region, off uint32) uint32 {
	bs := bytes.NewReader(unknownRegion.Raw)
	var header StartupAcmHeader
	bs.Seek(int64(off-unknownRegion.Offset), io.SeekStart)
	binary.Read(bs, binary.LittleEndian, &header)
	if header.ModuleType == 0x0002 && header.ModuleSubType == 0x0001 {
		return header.Size * 4
	}
	return 0
}
