package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/flammit/fwtools/pkg/cbfs"
	"github.com/flammit/fwtools/pkg/fit"
	"github.com/flammit/fwtools/pkg/ifd"
	"github.com/flammit/fwtools/pkg/me"
	"github.com/flammit/fwtools/pkg/rom"
	"github.com/flammit/fwtools/pkg/uefi"
)

var (
	detectors = []rom.Detector{
		ifd.DetectIFD,
		me.DetectME,
		cbfs.DetectFlashMap,
		cbfs.DetectVolume,
		uefi.DetectEFIVolume,
		fit.DetectFIT,
	}
)

func fatalUsage(message string) {
	log.Fatalf("%v: %v\nusage: %v [extract|build] ...",
		os.Args[0], message, os.Args[0])
}

func extract(args []string) {
	log.Printf("extract: starting")
	if len(args) != 2 {
		log.Fatalf("%v: extract usage: <rom_path> <layout_path>", os.Args[0])
	}
	romPath, layoutPath := args[0], args[1]
	romBytes, err := ioutil.ReadFile(romPath)
	if err != nil {
		log.Panicf("extract: failed to read rom path '%v': err=%v", romPath, err)
	}

	region := rom.DetectRegions(detectors, &rom.Region{
		Raw:    romBytes,
		Name:   "",
		Type:   "unknown",
		Offset: 0,
		Size:   uint32(len(romBytes)),
	})

	err = region.Save(layoutPath)
	if err != nil {
		log.Panicf("extract: %v", err)
	}

	// rebuild to check
	newRomBytes := make([]byte, len(romBytes))
	for n := 0; n < len(newRomBytes); n++ {
		newRomBytes[n] = 0xff
	}
	region.AddBytes(newRomBytes)
	for n := 0; n < len(newRomBytes); n++ {
		if newRomBytes[n] != romBytes[n] {
			log.Fatalf("rebuilt ROM doesn't match at 0x%08x: expected 0x%02x got 0x%02x",
				n, romBytes[n], newRomBytes[n])
		}
	}
}

func build(args []string) {
	log.Printf("build: starting")
	if len(args) != 2 {
		log.Fatalf("%v: extract usage: <layout_path> <rom_path>", os.Args[0])
	}
	layoutPath, romPath := args[0], args[1]

	region, err := rom.LoadRegion(layoutPath)
	if err != nil {
		log.Panicf("build: failed to load region: err=%v", err)
	}
	log.Printf("build: rom size is 0x%08x", region.Size)
	newRomBytes := make([]byte, region.Size)
	for n := 0; n < len(newRomBytes); n++ {
		newRomBytes[n] = 0xff
	}
	region.AddBytes(newRomBytes)
	err = ioutil.WriteFile(romPath, newRomBytes, os.ModePerm)
	if err != nil {
		log.Panicf("build: failed to write rom file: err=%v", err)
	}
}

func main() {
	if len(os.Args) < 2 {
		fatalUsage("missing command")
	}
	command := os.Args[1]
	switch command {
	case "extract":
		extract(os.Args[2:])
	case "build":
		build(os.Args[2:])
	default:
		fatalUsage("invalid command: " + command)
	}
}
