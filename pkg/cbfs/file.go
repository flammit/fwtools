package cbfs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"

	"github.com/flammit/fwtools/pkg/rom"
)

const (
	// BE
	headerMagic    = uint32(0x4342524f) // "ORBC"
	headerVersion1 = uint32(0x31313131)
	headerVersion2 = uint32(0x32313131)

	fileMagic         = uint64(0x4c41524348495645) // "LARCHIVE"
	fileComponentNull = uint32(0xFFFFFFFF)
)

type VolumeHeader struct {
	Magic         uint32
	Version       uint32
	RomSize       uint32
	BootBlockSize uint32
	Align         uint32
	Offset        uint32
	Architecture  uint32
	Pad           uint32
}

func (h VolumeHeader) Valid() bool {
	return h.Magic == headerMagic &&
		(h.Version == headerVersion2 || h.Version == headerVersion1)
}

type FileHeader struct {
	Magic            uint64
	Len              uint32
	Type             uint32
	AttributesOffset uint32
	Offset           uint32
}

func (h FileHeader) Valid() bool {
	return h.Magic == fileMagic
}

func DetectVolume(unknownRegion *rom.Region) []*rom.Region {
	// check for file header and volume
	bs := bytes.NewReader(unknownRegion.Raw)
	baseOffset := unknownRegion.Offset

	// loop and generate all files
	var volume VolumeHeader
	volume.Align = 0x40
	files := []*rom.Region{}
	for off := uint32(0); off < uint32(unknownRegion.Size); {
		bs.Seek(int64(off), io.SeekStart)
		var file FileHeader
		binary.Read(bs, binary.BigEndian, &file)
		if !file.Valid() {
			return files
		}

		var nameLen uint32
		if file.AttributesOffset != 0 {
			nameLen = file.AttributesOffset - 24
		} else {
			nameLen = file.Offset - 24
		}
		nameBytes := make([]byte, nameLen)
		binary.Read(bs, binary.BigEndian, &nameBytes)
		nameEnd := bytes.IndexByte(nameBytes, 0)
		name := string(nameBytes[0:nameEnd])
		if file.Type == fileComponentNull {
			name = fmt.Sprintf("null_%08x", off)
		}
		log.Printf("File Header 0x%08x '%s': %#v",
			off, name, file)

		// name is zero padded for length
		// attributes is 16 bytes

		size := uint32(rom.AlignUp(uint64(file.Offset+file.Len),
			uint64(volume.Align)))

		// TODO: handle with non-raw structures
		fileRegion := unknownRegion.Child(baseOffset+off, size,
			"container", name)
		files = append(files, fileRegion)

		headerRegion := fileRegion.Child(baseOffset+off, file.Offset,
			"raw", "header")
		fileRegion.Children = append(fileRegion.Children, headerRegion)

		dataRegion := fileRegion.Child(
			baseOffset+off+file.Offset,
			size-file.Offset,
			"unknown",
			"data",
		)
		if !dataRegion.Empty() {
			fileRegion.Children = append(fileRegion.Children, dataRegion)
		}

		// TODO: handle alignment build parameters
		// when null files are seen
		off += size
	}

	return files
}
