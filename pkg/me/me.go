package me

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/flammit/fwtools/pkg/rom"
)

var (
	fptSignature         = uint32(0x54504624) // "$FPT"
	cpdSignature         = uint32(0x44504324) // "$CPD"
	cpdManifestSignature = uint32(0x324e4d24) // "$MN2"
)

type FptHeader struct {
	RomBypass      [16]uint8
	Marker         uint32
	NumEntries     uint32
	HeaderVersion  uint8
	EntryVersion   uint8
	HeaderLength   uint8
	HeaderChecksum uint8
	TicksToAdd     uint16
	TokensToAdd    uint16
	Reserved       uint32
	FlashLayout    uint32
	FitcMajor      uint16
	FitcMinor      uint16
	FitcHotfix     uint16
	FitcBuild      uint16
}

func (h FptHeader) Valid() bool {
	return h.Marker == fptSignature
}

type FptEntry struct {
	Name       [4]byte
	Reserved   uint32
	Offset     uint32
	Length     uint32
	Reserved1  uint32
	Reserved2  uint32
	Reserved3  uint32
	Attributes uint32
}

func (e FptEntry) String() string {
	return fmt.Sprintf("Name '%4s': Offset=0x%08x, Length=0x%08x, Attributes=0x%08x",
		e.Name, e.Offset, e.Length, e.Attributes)
}

func DetectME(unknownRegion *rom.Region) []*rom.Region {
	bs := bytes.NewReader(unknownRegion.Raw)
	baseOffset := unknownRegion.Offset

	var fptHeader FptHeader
	binary.Read(bs, binary.LittleEndian, &fptHeader)

	if !fptHeader.Valid() {
		return nil
	}

	log.Printf("ME FPT Header:\n%#v\n", fptHeader)
	fptEntries := make([]FptEntry, fptHeader.NumEntries)
	for n := uint32(0); n < fptHeader.NumEntries; n++ {
		binary.Read(bs, binary.LittleEndian, &fptEntries[n])
		log.Printf("ME FPT Entry: %v", fptEntries[n])
	}

	regions := []*rom.Region{}
	// $FPT
	// TODO: replace with typed FptHeader+FptEntry+??Footer??
	// TODO: need to figure out the footer at 0xd80[0x8]
	regions = append(regions, unknownRegion.Child(
		baseOffset, 0xe00, "raw", "FPT"))

	for _, fptEntry := range fptEntries {
		offset, len := fptEntry.Offset, fptEntry.Length
		fptName := strings.TrimRight(string(fptEntry.Name[:4]), "\000")
		// FTUP is NFTP + WCOD + LOCL
		// TODO: make them parents / children
		if (offset == 0 && len == 0) || fptName == "FTUP" || offset == 0xffffffff {
			continue
		}
		regions = append(regions, unknownRegion.Child(
			baseOffset+offset, len, "raw", fptName))
	}

	sort.Sort(rom.ByOffset(regions))
	return regions
}
