package rom

import "fmt"

func GuidString(guid [16]uint8) string {
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		guid[3], guid[2], guid[1], guid[0],
		guid[5], guid[4],
		guid[7], guid[6],
		guid[8], guid[9],
		guid[10],
		guid[11],
		guid[12],
		guid[13],
		guid[14],
		guid[15])
}

func AlignUp(off, align uint64) uint64 {
	return (align + off - 1) & (^(align - 1))
}
