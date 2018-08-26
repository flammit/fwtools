package rom

import (
	"sort"
)

type Detector func(*Region) []*Region

func DetectRegions(detectors []Detector, unknownRegion *Region) *Region {
	if unknownRegion.Type == "unknown" {
		// log.Printf("Detect: %08x - %08x", unknownRegion.Offset, unknownRegion.Offset+unknownRegion.Size)
		regions := []*Region{}
		for _, detector := range detectors {
			regions = detector(unknownRegion)
			if len(regions) > 0 {
				break
			}
		}

		if len(regions) == 0 {
			// nothing detected, return raw region
			unknownRegion.Type = "raw"
			return unknownRegion
		}

		// check for gaps and recursively detect on unknown segments
		sort.Sort(ByOffset(regions))

		lastOffset := unknownRegion.Offset
		newRegions := []*Region{}
		for _, region := range regions {
			if region.Offset > lastOffset {
				gapRegion := unknownRegion.UnknownChild(lastOffset, region.Offset-lastOffset)
				if !gapRegion.Empty() {
					newRegions = append(newRegions, DetectRegions(detectors, gapRegion))
				}
			}

			newRegions = append(newRegions, DetectRegions(detectors, region))
			lastOffset = region.Offset + region.Size
		}
		if lastOffset < unknownRegion.Offset+unknownRegion.Size {
			gapRegion := unknownRegion.UnknownChild(lastOffset,
				(unknownRegion.Offset + unknownRegion.Size - lastOffset))
			if !gapRegion.Empty() {
				newRegions = append(newRegions, DetectRegions(detectors, gapRegion))
			}
		}

		if len(newRegions) == 1 {
			return newRegions[0]
		}
		unknownRegion.Type = "container"
		unknownRegion.Children = newRegions
		return unknownRegion
	}
	for i, child := range unknownRegion.Children {
		unknownRegion.Children[i] = DetectRegions(detectors, child)
	}
	return unknownRegion
}
