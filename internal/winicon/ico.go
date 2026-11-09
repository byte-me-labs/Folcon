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

// ParseICO validates an .ico file and returns the width of its widest frame.
// A width byte of 0 means 256.
func ParseICO(data []byte) (int, error) {
	if len(data) < icoHeaderSize {
		return 0, fmt.Errorf("图标文件过短")
	}
	if reserved := binary.LittleEndian.Uint16(data[0:2]); reserved != 0 {
		return 0, fmt.Errorf("不是有效的图标文件（reserved=%d）", reserved)
	}
	if typ := binary.LittleEndian.Uint16(data[2:4]); typ != 1 {
		return 0, fmt.Errorf("不是图标文件（type=%d）", typ)
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count <= 0 {
		return 0, fmt.Errorf("图标文件不包含图像")
	}
	if count > 1024 {
		return 0, fmt.Errorf("图标文件包含过多图像（%d）", count)
	}
	if len(data) < icoHeaderSize+count*icoEntrySize {
		return 0, fmt.Errorf("图标文件头不完整")
	}

	largest := 0
	for i := 0; i < count; i++ {
		off := icoHeaderSize + i*icoEntrySize
		width := int(data[off])
		if width == 0 {
			width = 256
		}
		size := int(binary.LittleEndian.Uint32(data[off+8 : off+12]))
		offset := int(binary.LittleEndian.Uint32(data[off+12 : off+16]))
		if size <= 0 || size > maxIconBytes || offset < 0 || offset+size > len(data) {
			return 0, fmt.Errorf("第 %d 个图标的偏移无效", i+1)
		}
		if width > largest {
			largest = width
		}
	}
	return largest, nil
}
