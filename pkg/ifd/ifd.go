package ifd

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"

	"github.com/flammit/fwtools/pkg/rom"
)

var (
	ifdSignature = uint32(0x0ff0a55a)
)

type Header struct {
	FlMap0  uint32
	FlMap1  uint32
	FlMap2  uint32
	FlMap3  uint32
	_       [0xed8]uint8
	FlUmap0 uint32
}

// NR
func (h Header) NumRegions() uint32 {
	return (h.FlMap0 >> 24) & 7
}

// FRBA
func (h Header) FlashRegionBaseAddress() uint32 {
	return ((h.FlMap0 >> 16) & 0xff) << 4
}

// NC
func (h Header) NumComponents() uint32 {
	return ((h.FlMap0 >> 8) & 3) + 1
}

// FCBA
func (h Header) FlashComponentBaseAddress() uint32 {
	return (h.FlMap0 & 0xff) << 4
}

// ISL/PSL - ICH/PCH Strap Length
func (h Header) PchStrapLength() uint32 {
	return (h.FlMap1 >> 24) & 0xff
}

// FISBA/FPSBA - Flash ICH/PCH Base Address
func (h Header) FlashPchStrapBaseAddress() uint32 {
	return ((h.FlMap1 >> 16) & 0xff) << 4
}

// NM
func (h Header) NumMasters() uint32 {
	return (h.FlMap1 >> 8) & 3
}

// FMBA - Flash Master Base Address
func (h Header) FlashMasterBaseAddress() uint32 {
	return (h.FlMap1 & 0xff) << 4
}

// CPUSL
func (h Header) CpuStrapLength() uint32 {
	return (h.FlMap2 >> 8) & 0xff
}

// FCPUSBA - Flash CPU Strap Base Address
func (h Header) FlashCpuStrapBaseAddress() uint32 {
	return (h.FlMap2 & 0xff) << 4
}

// ICCRIBA - ICC register init base address
func (h Header) IccRegisterInitBaseAddress() uint32 {
	return ((h.FlMap2 >> 16) & 0xff) << 4
}

// VTL - VSCC Table Length
func (h Header) VsccTableLength() uint32 {
	return (h.FlUmap0 >> 8) & 0xff
}

// VTBA - VSCC Table Base Address
func (h Header) VsccTableBaseAddress() uint32 {
	return (h.FlUmap0 & 0xff) << 4
}

// 0xf00 - 0x1000 - OEM reserved

func (h Header) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "FLMAP0    : 0x%08x\n", h.FlMap0)
	fmt.Fprintf(&b, "  NR      : 0x%08x\n", h.NumRegions())
	fmt.Fprintf(&b, "  FRBA    : 0x%08x\n", h.FlashRegionBaseAddress())
	fmt.Fprintf(&b, "  NC      : 0x%08x\n", h.NumComponents())
	fmt.Fprintf(&b, "  FCBA    : 0x%08x\n", h.FlashComponentBaseAddress())
	fmt.Fprintf(&b, "FLMAP1    : 0x%08x\n", h.FlMap1)
	fmt.Fprintf(&b, "  PSL     : 0x%08x\n", h.PchStrapLength())
	fmt.Fprintf(&b, "  FPSBA   : 0x%08x\n", h.FlashPchStrapBaseAddress())
	fmt.Fprintf(&b, "  NM      : 0x%08x\n", h.NumMasters())
	fmt.Fprintf(&b, "  FMBA    : 0x%08x\n", h.FlashMasterBaseAddress())
	fmt.Fprintf(&b, "FLMAP2    : 0x%08x\n", h.FlMap2)
	fmt.Fprintf(&b, "  CPUSL   : 0x%08x\n", h.CpuStrapLength())
	fmt.Fprintf(&b, "  FCPUSBA : 0x%08x\n", h.FlashCpuStrapBaseAddress())
	fmt.Fprintf(&b, "  ICCRIBA : 0x%08x\n", h.IccRegisterInitBaseAddress())
	fmt.Fprintf(&b, "FLMAP3    : 0x%08x\n", h.FlMap3)
	fmt.Fprintf(&b, "FLUMAP0   : 0x%08x\n", h.FlUmap0)
	fmt.Fprintf(&b, "  VTL     : 0x%08x\n", h.VsccTableLength())
	fmt.Fprintf(&b, "  VTBA    : 0x%08x\n", h.VsccTableBaseAddress())
	return b.String()
}

var (
	RegionNames = []string{
		"ifd",
		"bios",
		"me",
		"gbe",
		"pd",
		"res5",
		"res6",
		"res7",
		"ec",
		"res9",
	}
)

type Regions struct {
	FlRegs [10]uint32
}

func (r Regions) Base(region int) uint32 {
	return r.FlRegs[region] & 0x7fff
}

func (r Regions) Limit(region int) uint32 {
	return (r.FlRegs[region] >> 16) & 0x7fff
}

func (r Regions) Start(region int) uint32 {
	return r.Base(region) * 0x1000
}

func (r Regions) End(region int) uint32 {
	return (r.Limit(region) + 1) * 0x1000
}

func (r Regions) String() string {
	var b strings.Builder
	for n := 0; n < 10; n++ {
		fmt.Fprintf(&b, "Region %v: start=0x%08x, end=0x%08x\n", n, r.Start(n), r.End(n))
	}
	return b.String()
}

type Descriptor struct {
	SigOffset uint32
	Header    *Header
	Regions   *Regions
}

func (d Descriptor) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Sig Offset: 0x%02x\n", d.SigOffset)
	fmt.Fprintf(&b, "Header:\n%v", d.Header)
	fmt.Fprintf(&b, "Regions:\n%v", d.Regions)
	return b.String()
}

func DetectIFD(unknownRegion *rom.Region) []*rom.Region {
	bs := bytes.NewReader(unknownRegion.Raw)

	// check 0x00 for signature
	var sig, sigOff uint32
	binary.Read(bs, binary.LittleEndian, &sig)
	if sig != ifdSignature {
		// check 0x10 for signature
		sigOff = 0x10
		bs.Seek(0x10, io.SeekStart)
		binary.Read(bs, binary.LittleEndian, &sig)
	}

	if sig != ifdSignature {
		return nil
	}
	var ifdHeader Header
	binary.Read(bs, binary.LittleEndian, &ifdHeader)

	bs.Seek(int64(ifdHeader.FlashRegionBaseAddress()), io.SeekStart)
	var ifdRegions Regions
	binary.Read(bs, binary.LittleEndian, &ifdRegions)

	desc := Descriptor{
		SigOffset: sigOff,
		Header:    &ifdHeader,
		Regions:   &ifdRegions,
	}
	log.Printf("\nIFD:\n%v", desc)

	unknownRegion.Type = "container"
	unknownRegion.Children = []*rom.Region{}
	nr := int(ifdHeader.NumRegions())
	for n, name := range RegionNames {
		if nr > 0 && n > nr {
			break
		}

		base := ifdRegions.Base(n)
		if base == 0x7fff {
			continue
		}

		start, end := ifdRegions.Start(n), ifdRegions.End(n)
		regionType := "unknown"
		if n == 0 {
			// for now use raw handler
			// TODO: transition to typed IFD struct w/ all non-FF data
			regionType = "raw"
		}

		ifdRegion := &rom.Region{
			Raw:    unknownRegion.Raw[start:end],
			Parent: unknownRegion,
			Type:   regionType,
			Name:   name,
			Offset: start,
			Size:   end - start,
		}
		log.Printf("IFD %v/%v: %v %v", n, name, ifdRegion.Type, ifdRegion.Offset)
		unknownRegion.Children = append(unknownRegion.Children, ifdRegion)
	}

	sort.Sort(rom.ByOffset(unknownRegion.Children))

	return []*rom.Region{unknownRegion}
}
