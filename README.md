# Firmware Tools

Detects and extracts ROM regions for a variety of ROM protocols:

* IFD
* ME
* FIT
* UEFI
* FMAP
* CBFS

### Protocol TODO: 
* ME/MFS
* Boot Guard Structures

### Feature TODO:
* ~~Artifact Tree -> ROM file (unextract)~~

* Artifact Tree -> Protocol-Specific Config

Infer a basic build structure / system and allow modifications to
contents in each protocol, i.e. IFD should be represented as a
JSON struct not a RAW region.  Always fallback to RAW handling
for unkonwn protocol regions.

* Dependency Management Graph

Handle DAG of locations: bootblock -> CBFS header / FIT, 
FIT -> microcode, etc.

## Install / Run

```
go get github.com/flammit/fwtools/...
go install github.com/flammit/fwtools/...
fwcli extract firmware.bin output/
```

## Output

`summary.json` contains a hierarchy of ROM regions and the output
directory will contain directories containing each leaf region.

Non-empty regions of the ROM that do not belong to the supported
region formats are saved as "unknown_0xNNNNNNNN" to allow for
byte-for-byte reconstructions.

```json
{
  "Type": "container",
  "Name": "full",
  "Offset": 0,
  "Size": 16777216,
  "Children": [
    {
      "Type": "raw",
      "Name": "ifd",
      "Offset": 0,
      "Size": 4096
    },
    {
      "Type": "container",
      "Name": "me",
      "Offset": 4096,
      "Size": 2093056,
      "Children": [
        {
          "Type": "raw",
          "Name": "me/FPT",
          "Offset": 4096,
          "Size": 3584
        },
        {
          "Type": "raw",
          "Name": "me/FTPR",
          "Offset": 8192,
          "Size": 684032
        },
        {
          "Type": "raw",
          "Name": "me/MFS",
          "Offset": 692224,
          "Size": 409600
        }
      ]
    },
    {
      "Type": "container",
      "Name": "fmap",
      "Offset": 2097152,
      "Size": 14680064,
      "Children": [
        {
          "Type": "container",
          "Name": "fmap/BIOS",
          "Offset": 2097152,
          "Size": 14680064,
          "Children": [
            {
              "Type": "raw",
              "Name": "fmap/FMAP",
              "Offset": 2097152,
              "Size": 512
            },
            {
              "Type": "raw",
              "Name": "fmap/RW_MRC_CACHE",
              "Offset": 2162688,
              "Size": 65536
            },
            {
              "Type": "container",
              "Name": "fmap/COREBOOT",
              "Offset": 2228224,
              "Size": 14548992,
              "Children": [
                {
                  "Type": "raw",
                  "Name": "fmap/COREBOOT/cbfs master header/header",
                  "Offset": 2228224,
                  "Size": 56
                },
                {
                  "Type": "raw",
                  "Name": "fmap/COREBOOT/cbfs master header/data",
                  "Offset": 2228280,
                  "Size": 72
                },
                ...
              ]
            }
          ]
        }
      ]
    }
  ]
}

```
