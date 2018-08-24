package uefi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"path/filepath"

	"github.com/flammit/fwtools/pkg/rom"
)

var (
	volumeSignature = uint32(0x4856465F) // "_FVH"
	pageSize        = uint32(0x1000)
)

type VolumeHeader struct {
	Reserved       [16]uint8 // 0x00 - reserved
	GUID           [16]uint8 // 0x10
	Len            uint64    // 0x20
	Sig            uint32    // 0x28 - must be volumeSignature
	Attr           uint32    // 0x2c
	HeaderLen      uint16    // 0x30
	Checksum       uint16    // 0x32
	ExtHeaderOff   uint16    // 0x34
	Reserved1      uint8     // 0x36
	Revision       uint8     // 0x37
	NumBlocks      uint32    // 0x38
	BlockSize      uint32    // 0x3c
	TerminateBlock uint64    // 0x40 - must be 0x00000000
}

func (h VolumeHeader) Valid() bool {
	return h.Sig == volumeSignature && h.TerminateBlock == 0
}

func DetectEFIVolume(unknownRegion *rom.Region) []*rom.Region {
	bs := bytes.NewReader(unknownRegion.Raw)
	baseOffset := unknownRegion.Offset

	// scan for signature
	var header VolumeHeader
	volumes := []*rom.Region{}
	for offset := uint32(0); offset < unknownRegion.Size; {
		bs.Seek(int64(offset), io.SeekStart)
		binary.Read(bs, binary.LittleEndian, &header)
		if !header.Valid() {
			offset += pageSize
			continue
		}

		// good header
		log.Printf("UEFI Volume: offset=%08x len=%08x GUID=%v",
			baseOffset+offset, header.Len,
			rom.GuidString(header.GUID))

		// setup new region for the full volume
		name := fmt.Sprintf("fv_%08x", baseOffset+offset)
		size := uint32(header.Len)
		region := &rom.Region{
			Raw:      unknownRegion.Raw[offset : offset+size],
			Name:     filepath.Join(unknownRegion.Name, name),
			Type:     "container", // TODO: use FV handler
			Offset:   baseOffset + offset,
			Size:     uint32(size),
			Children: []*rom.Region{},
		}

		// generate headers and scan for files
		headerRegion := &rom.Region{
			Raw:    unknownRegion.Raw[offset : offset+uint32(header.HeaderLen)],
			Name:   filepath.Join(region.Name, "header"),
			Type:   "raw", // TODO: use FV handler
			Offset: baseOffset + offset,
			Size:   uint32(header.HeaderLen),
			Parent: region,
		}
		region.Children = append(region.Children, headerRegion)
		dataRegion := &rom.Region{
			Raw:    unknownRegion.Raw[offset+uint32(header.HeaderLen) : offset+uint32(size)],
			Name:   filepath.Join(region.Name, "data"),
			Type:   "unknown",
			Offset: baseOffset + offset + uint32(header.HeaderLen),
			Size:   uint32(size) - uint32(header.HeaderLen),
			Parent: region,
		}
		dataRegion = rom.DetectRegions(
			[]rom.Detector{detectEFIFiles},
			dataRegion,
		)
		region.Children = append(region.Children, dataRegion)

		volumes = append(volumes, region)
		offset += uint32(size)
	}

	return volumes
}
