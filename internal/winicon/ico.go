package winicon

import (
	"encoding/binary"
	"fmt"
)

const (
	icoHeaderSize = 6
	icoEntrySize  = 16
	maxIconBytes  = 32 << 20
)

// ParseICO splits an .ico file into frames. PNG frames are kept as-is; DIB
// frames are decoded on demand.
func ParseICO(data []byte) ([]Frame, error) {
	if len(data) < icoHeaderSize {
		return nil, fmt.Errorf("图标文件过短")
	}
	if reserved := binary.LittleEndian.Uint16(data[0:2]); reserved != 0 {
		return nil, fmt.Errorf("不是有效的图标文件（reserved=%d）", reserved)
	}
	if typ := binary.LittleEndian.Uint16(data[2:4]); typ != 1 {
		return nil, fmt.Errorf("不是图标文件（type=%d）", typ)
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count <= 0 {
		return nil, fmt.Errorf("图标文件不包含图像")
	}
	if count > 1024 {
		return nil, fmt.Errorf("图标文件包含过多图像（%d）", count)
	}
	if len(data) < icoHeaderSize+count*icoEntrySize {
		return nil, fmt.Errorf("图标文件头不完整")
	}

	frames := make([]Frame, 0, count)
	for i := 0; i < count; i++ {
		off := icoHeaderSize + i*icoEntrySize
		width := int(data[off])
		if width == 0 {
			width = 256
		}
		height := int(data[off+1])
		if height == 0 {
			height = 256
		}
		bpp := int(binary.LittleEndian.Uint16(data[off+6 : off+8]))
		size := int(binary.LittleEndian.Uint32(data[off+8 : off+12]))
		offset := int(binary.LittleEndian.Uint32(data[off+12 : off+16]))
		if size <= 0 || size > maxIconBytes || offset < 0 || offset+size > len(data) {
			return nil, fmt.Errorf("第 %d 个图标的偏移无效", i+1)
		}
		raw := data[offset : offset+size]
		frames = append(frames, Frame{
			Width:  width,
			Height: height,
			BPP:    bpp,
			Data:   raw,
			IsPNG:  hasPNGSignature(raw),
		})
	}
	return frames, nil
}
