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

	biosRegion := rom.DetectRegions(detectors, &rom.Region{
		Raw:    romBytes,
		Name:   "full",
		Type:   "unknown",
		Offset: 0,
		Size:   uint32(len(romBytes)),
	})

	err = biosRegion.Write(layoutPath)
	if err != nil {
		log.Panicf("extract: %v", err)
	}
}

func build(args []string) {
	log.Printf("build: starting")
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
