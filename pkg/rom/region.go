package rom

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Region struct {
	Type     string
	Name     string
	Offset   uint32
	Size     uint32
	Raw      []byte    `json:"-"`
	Parent   *Region   `json:"-"`
	Children []*Region `json:",omitempty"`
}

func (r Region) AddBytes(bs []byte) {
	if r.Type == "raw" {
		for n := 0; n < int(r.Size); n++ {
			bs[int(r.Offset)+n] = r.Raw[n]
		}
		return
	}
	for _, child := range r.Children {
		child.AddBytes(bs)
	}
}

func LoadRegion(layoutPath string) (*Region, error) {
	summaryJson := filepath.Join(layoutPath, "summary.json")
	layoutBytes, err := ioutil.ReadFile(summaryJson)
	if err != nil {
		return nil, err
	}
	var r Region
	json.Unmarshal(layoutBytes, &r)
	err = r.loadData(nil, layoutPath)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (r Region) Save(layoutPath string) error {
	layoutBytes, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("region: failed to marshal layout struct: err=%v", err)
	}
	summaryJson := filepath.Join(layoutPath, "summary.json")
	os.MkdirAll(filepath.Dir(summaryJson), os.ModePerm)
	if err := ioutil.WriteFile(summaryJson, layoutBytes, os.ModePerm); err != nil {
		return fmt.Errorf("region: failed to write layout file '%v': err=%v", layoutPath, err)
	}

	return r.saveData(layoutPath)
}

func (r *Region) loadData(parent *Region, layoutPath string) error {
	r.Parent = parent
	if len(r.Children) > 0 {
		for _, child := range r.Children {
			if err := child.loadData(r, layoutPath); err != nil {
				return err
			}
		}
		return nil
	}

	path := filepath.Join(layoutPath, r.Name+".raw")
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	r.Raw = raw
	return nil
}

func (r Region) saveData(layoutPath string) error {
	// write data - only leaves
	if len(r.Children) > 0 {
		for _, child := range r.Children {
			if err := child.saveData(layoutPath); err != nil {
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
		return fmt.Errorf("region: failed to write region file '%v': err=%v", path, err)
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

func (r Region) Child(offset, size uint32, regionType, name string) *Region {
	parent := r.KnownParent()
	return &Region{
		Raw:    r.Raw[offset-r.Offset : offset-r.Offset+size],
		Parent: r.KnownParent(),
		Type:   regionType,
		Name:   filepath.Join(parent.Name, name),
		Offset: offset,
		Size:   size,
	}
}

const (
	unknownPrefix = "unknown_"
)

func (r Region) UnknownChild(offset, size uint32) *Region {
	return r.Child(offset, size, "unknown", fmt.Sprintf("%s%08x", unknownPrefix, offset))
}

func (r Region) Contains(offset, size uint32) bool {
	return offset >= r.Offset && offset < (r.Offset+r.Size) &&
		(offset+size) <= (r.Offset+r.Size)
}

func (r Region) KnownParent() *Region {
	cur := &r
	for ; strings.HasPrefix(filepath.Base(cur.Name), unknownPrefix); cur = cur.Parent {
	}
	return cur
}

func (r Region) FullSize() uint32 {
	cur := &r
	for ; cur.Parent != nil; cur = cur.Parent {
	}
	return cur.Size
}

var emptyByte = byte(0xff)

type ByOffset []*Region

func (a ByOffset) Len() int           { return len(a) }
func (a ByOffset) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByOffset) Less(i, j int) bool { return a[i].Offset < a[j].Offset }
