package cbfs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/flammit/fwtools/pkg/rom"
)

const (
	fmapSignature      = uint64(0x5f5f50414d465f5f) // "__FMAP__"
	fmapVerMajor       = 1
	fmapVerMinor       = 1
	fmapStrlen         = 32
	fmapAreaStatic     = 0x0001
	fmapAreaCompressed = 0x0002
	fmapAreaReadOnly   = 0x0004
)

type FmapHeader struct {
	Signature uint64
	VerMajor  uint8
	VerMinor  uint8
	Base      uint64
	Size      uint32
	Name      [fmapStrlen]byte
	NumAreas  uint16
}

func (h FmapHeader) String() string {
	return fmt.Sprintf("Base=0x%016x Size=0x%08x NumAreas=%v Name='%32s'",
		h.Base, h.Size, h.NumAreas, h.Name)
}

func (h FmapHeader) Valid() bool {
	return h.Signature == fmapSignature &&
		h.VerMajor == fmapVerMajor
	//	&& h.VerMinor == fmapVerMinor
}

type FmapArea struct {
	Offset uint32
	Size   uint32
	Name   [fmapStrlen]byte
	Flags  uint16
}

func (a FmapArea) String() string {
	return fmt.Sprintf("Offset=0x%08x Size=0x%08x Flags=0x%04x Name='%32s'",
		a.Offset, a.Size, a.Flags, a.Name)
}

func (a FmapArea) NameString() string {
	return strings.TrimRight(string(a.Name[:fmapStrlen]), "\u0000")
}

func DetectFlashMap(unknownRegion *rom.Region) []*rom.Region {
	bs := bytes.NewReader(unknownRegion.Raw)

	// check for signature
	// TODO: scan for signature instead of just at the beginning
	var header FmapHeader
	for off := uint32(0); off < unknownRegion.Size; off += 0x10 {
		bs.Seek(int64(off), io.SeekStart)
		binary.Read(bs, binary.LittleEndian, &header)
		if header.Valid() {
			break
		}
	}
	if !header.Valid() {
		return nil
	}
	log.Printf("FMAP Header: %v", header)

	areas := make([]FmapArea, header.NumAreas)
	binary.Read(bs, binary.LittleEndian, &areas)
	unknownRegion.Name = "fmap"
	unknownRegion.Type = "container"
	unknownRegion.Children = []*rom.Region{}

	// process the areas
	lastRegion := unknownRegion
	for i, area := range areas {
		log.Printf("FMAP Area %v: %v", i, area)
		if !unknownRegion.Contains(area.Offset, area.Size) {
			// skip regions as in samsung stumpy - like IFD/ME
			log.Printf("Skipping Area")
			continue
		}

		region := unknownRegion.Child(area.Offset, area.Size,
			"unknown", area.NameString())

		if area.NameString() == "FMAP" {
			// TODO: drop this region and have a custom struct handler
			region.Type = "raw"
		}

		parent := lastRegion
		for {
			if parent.Contains(region.Offset, region.Size) {
				break
			}
			parent = parent.Parent
		}
		region.Parent = parent
		parent.Type = "container"
		parent.Children = append(parent.Children, region)
		lastRegion = region
	}

	return []*rom.Region{unknownRegion}
}
