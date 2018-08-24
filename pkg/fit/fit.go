package fit

import (
	"bytes"
	"encoding/binary"
	"path/filepath"

	"github.com/flammit/fwtools/pkg/rom"
)

var (
	fitSignature = uint64(0x2020205f5449465f) // "_FIT_   "
)

type Header struct {
	Signature uint64
}

func (h Header) Valid() bool {
	return h.Signature == fitSignature
}

func DetectFIT(unknownRegion *rom.Region) []*rom.Region {
	bs := bytes.NewReader(unknownRegion.Raw)

	// scan for signature
	var header Header
	binary.Read(bs, binary.LittleEndian, &header)
	if !header.Valid() {
		return nil
	}

	return []*rom.Region{&rom.Region{
		Raw:    unknownRegion.Raw,
		Parent: unknownRegion.Parent,
		Type:   "raw", // TODO: replace
		Name:   filepath.Join(unknownRegion.Parent.Name, "fit"),
		Offset: unknownRegion.Offset,
		Size:   unknownRegion.Size,
	}}
}
