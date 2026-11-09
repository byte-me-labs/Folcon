// Package winicon reads the icons out of Windows executables and .ico files.
package winicon

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"sort"
)

// Frame is one image inside an icon group.
type Frame struct {
	Width  int
	Height int
	BPP    int
	// Data is either a PNG stream or a DIB (BITMAPINFOHEADER followed by
	// the color and mask bits).
	Data  []byte
	IsPNG bool
}

// Group is one RT_GROUP_ICON resource, or the single group of an .ico file.
type Group struct {
	// Ordinal is the zero-based position in the file's icon resources. It is
	// the value Explorer expects after IconResource's comma.
	Ordinal  int
	ID       uint16
	Name     string
	IsString bool
	Frames   []Frame
	// Err records why this group's frames could not be read. Groups with an
	// error are skipped when choosing an icon.
	Err error
}

// Source is a file the user chose.
type Source struct {
	Path  string
	IsICO bool
	// Groups holds the icon resources of an executable, or one group for an
	// .ico file.
	Groups []Group
}

// Sizes lists the distinct widths in g, smallest first.
func (g *Group) Sizes() []int {
	seen := make(map[int]bool, len(g.Frames))
	out := make([]int, 0, len(g.Frames))
	for _, f := range g.Frames {
		if f.Width <= 0 || seen[f.Width] {
			continue
		}
		seen[f.Width] = true
		out = append(out, f.Width)
	}
	sort.Ints(out)
	return out
}

// Decode turns the frame into an image.
func (f Frame) Decode() (image.Image, error) {
	if len(f.Data) == 0 {
		return nil, errors.New("图标数据为空")
	}
	if f.IsPNG || hasPNGSignature(f.Data) {
		img, err := png.Decode(bytes.NewReader(f.Data))
		if err != nil {
			return nil, fmt.Errorf("解码 PNG 图标失败: %w", err)
		}
		return img, nil
	}
	return DecodeDIB(f.Data)
}

func hasPNGSignature(data []byte) bool {
	return len(data) >= 8 &&
		data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' &&
		data[4] == '\r' && data[5] == '\n' && data[6] == 0x1a && data[7] == '\n'
}
