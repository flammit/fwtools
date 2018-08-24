package rom

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Region struct {
	Type     string
	Name     string
	Offset   uint32
	Size     uint32
	Align    uint32    // used during build if offset isn't specified
	Raw      []byte    `json:"-"`
	Parent   *Region   `json:"-"`
	Children []*Region `json:",omitempty"`
}

func (r Region) Write(layoutPath string) error {
	layoutBytes, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("layout: failed to marshal layout struct: err=%v", err)
	}
	summaryJson := filepath.Join(layoutPath, "summary.json")
	os.MkdirAll(filepath.Dir(summaryJson), os.ModePerm)
	if err := ioutil.WriteFile(summaryJson, layoutBytes, os.ModePerm); err != nil {
		return fmt.Errorf("layout: failed to write layout file '%v': err=%v", layoutPath, err)
	}

	return r.writeData(layoutPath)
}

func (r Region) writeData(layoutPath string) error {
	// write data -  only leaves
	if len(r.Children) > 0 {
		for _, child := range r.Children {
			if err := child.writeData(layoutPath); err != nil {
				return err
			}
		}
	}
	// TODO: handle with proper casting to region handlers
	if r.Type != "raw" {
		return nil
	}

	path := filepath.Join(layoutPath, r.Name+".raw")
	os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err := ioutil.WriteFile(path, r.Raw, os.ModePerm); err != nil {
		return fmt.Errorf("layout: failed to write region file '%v': err=%v", path, err)
	}

	return nil
}

func (r Region) Empty() bool {
	for _, b := range r.Raw {
		if b != emptyByte {
			return false
		}
	}
	return true
}

func (r Region) UnknownChild(offset, size uint32) *Region {
	return &Region{
		Raw:    r.Raw[offset-r.Offset : offset-r.Offset+size],
		Parent: &r,
		Type:   "unknown",
		Name:   filepath.Join(r.Name, fmt.Sprintf("unknown_%08x", offset)),
		Offset: offset,
		Size:   size,
	}
}

func (r Region) Contains(offset, size uint32) bool {
	return offset >= r.Offset && offset < (r.Offset+r.Size) &&
		(offset+size) <= (r.Offset+r.Size)
}

var emptyByte = byte(0xff)

type ByOffset []*Region

func (a ByOffset) Len() int           { return len(a) }
func (a ByOffset) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByOffset) Less(i, j int) bool { return a[i].Offset < a[j].Offset }
