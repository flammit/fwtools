package uefi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"

	"github.com/flammit/fwtools/pkg/rom"
)

type FileHeader struct {
	GUID      [16]uint8 // 0x00
	HeaderSum uint8     // 0x10
	FileSum   uint8     // 0x11
	Type      uint8     // 0x12
	Attr      uint8     // 0x13
	Len24     [3]uint8  // 0x14
	State     uint8     // 0x17
	Len64     uint64    // 0x18 - optional extended field (v3?)
}

type SectionHeader struct {
	Len24 [3]uint8 // 0x00
	Type  uint8    // 0x03
}

var (
	fileGuidEmpty = "ffffffff-ffff-ffff-ffff-ffffffffffff"
	/*
		EFI_FIRMWARE_FILE_SYSTEM_GUID = "7A9354D9-0468-444A-81CE-0BF617D890DF"
		EFI_FIRMWARE_FILE_SYSTEM2_GUID = "8C8CE578-8A3D-4F1C-9935-896185C32DD3"
		EFI_FIRMWARE_FILE_SYSTEM3_GUID = "5473C07A-3DCB-4DCA-BD6F-1E9689E7349A"
		EFI_SYSTEM_NV_DATA_FV_GUID = "FFF12B8D-7696-4C8B-A985-2747075B4F50"
	*/
)

func detectEFIFiles(unknownRegion *rom.Region) []*rom.Region {
	bs := bytes.NewReader(unknownRegion.Raw)
	baseOffset := unknownRegion.Offset

	files := []*rom.Region{}
	offset, end := uint32(0), uint32(unknownRegion.Size)
	for {
		if offset >= end {
			break
		}
		bs.Seek(int64(offset), io.SeekStart)

		var fileHeader FileHeader
		binary.Read(bs, binary.LittleEndian, &fileHeader)

		size := rom.Size24(fileHeader.Len24)
		headerLen := uint32(0x18)
		if size == 0xffffff {
			size = uint32(fileHeader.Len64)
			headerLen = 0x20
		}
		if size >= end {
			break
		}

		guid := rom.GuidString(fileHeader.GUID)
		inc := uint32(rom.AlignUp(uint64(size), 8))
		log.Printf("  UEFI File %04d: guid=%v off=0x%08x len=0x%08x inc=0x%08x",
			len(files), guid, baseOffset+offset, size, inc)
		name := fmt.Sprintf("ffs_%04d", len(files))
		region := unknownRegion.Child(baseOffset+offset, inc, "container", name)

		headerRegion := region.Child(baseOffset+offset, headerLen,
			"raw", "header."+guid)
		region.Children = append(region.Children, headerRegion)

		dataRegion := region.Child(baseOffset+offset+headerLen, inc-headerLen,
			"unknown", "data."+guid)
		if guid == fileGuidEmpty {
			dataRegion.Type = "raw"
		} else {
			dataRegion = rom.DetectRegions(
				[]rom.Detector{detectEFISections},
				dataRegion,
			)
		}
		if !dataRegion.Empty() {
			region.Children = append(region.Children, dataRegion)
		}

		files = append(files, region)
		offset += inc
	}
	return files
}

func detectEFISections(unknownRegion *rom.Region) []*rom.Region {
	bs := bytes.NewReader(unknownRegion.Raw)
	baseOffset := unknownRegion.Offset

	sections := []*rom.Region{}
	var header SectionHeader
	var offset uint32
	for offset = uint32(0); offset < unknownRegion.Size; {
		bs.Seek(int64(offset), io.SeekStart)
		binary.Read(bs, binary.LittleEndian, &header)
		sectionLen := rom.Size24(header.Len24)
		dataOffset := uint32(0x4)
		if sectionLen == 0 {
			break
		}
		if sectionLen == 0xFFFFFF {
			var len64 uint64
			binary.Read(bs, binary.LittleEndian, &len64)
			dataOffset += 0x8
			sectionLen = uint32(len64)
		}

		// TODO: this needs to be zero padded! but we're just including this in the data
		sectionLen = uint32(rom.AlignUp(uint64(sectionLen), 4))
		if offset+sectionLen > unknownRegion.Size {
			log.Printf("    !!!Bad UEFI Section - using raw section: off=0x%08x len=0x%08x size=0x%08x",
				offset, sectionLen, unknownRegion.Size)
			return nil
		}

		if sectionLen == 0 {
			break
		}
		name := fmt.Sprintf("sec_%04d_%02x", len(sections), header.Type)
		log.Printf("    UEFI Section %04d: type=0x%02x 0x%08x 0x%08x",
			len(sections), header.Type, baseOffset+offset, sectionLen)

		region := unknownRegion.Child(baseOffset+offset, sectionLen, "raw", name)
		sections = append(sections, region)

		offset += sectionLen
	}

	// region can be ff padded at end
	offset = uint32(rom.AlignUp(uint64(offset), 8))
	if offset != unknownRegion.Size {
		return nil
	}

	return sections
}
