package winicon

import (
	"encoding/binary"
	"fmt"
	"image"
	"math/bits"
)

const (
	biRGB       = 0
	biBitfields = 3

	maxIconDimension = 4096
)

// DecodeDIB decodes the DIB stored inside an .ico frame.
//
// An icon DIB is a BITMAPINFOHEADER whose height is twice the real height:
// the color bitmap is followed by a 1-bit AND mask that marks transparent
// pixels.
func DecodeDIB(data []byte) (image.Image, error) {
	if len(data) < 40 {
		return nil, fmt.Errorf("DIB 数据过短")
	}
	hdrSize := int(binary.LittleEndian.Uint32(data[0:4]))
	if hdrSize < 40 || hdrSize > len(data) {
		return nil, fmt.Errorf("DIB 头长度无效（%d）", hdrSize)
	}
	width := int(int32(binary.LittleEndian.Uint32(data[4:8])))
	rawHeight := int(int32(binary.LittleEndian.Uint32(data[8:12])))
	planes := binary.LittleEndian.Uint16(data[12:14])
	bitCount := int(binary.LittleEndian.Uint16(data[14:16]))
	compression := binary.LittleEndian.Uint32(data[16:20])
	clrUsed := int(binary.LittleEndian.Uint32(data[32:36]))

	if width <= 0 || width > maxIconDimension {
		return nil, fmt.Errorf("图标宽度无效（%d）", width)
	}
	if rawHeight == 0 {
		return nil, fmt.Errorf("图标高度无效（0）")
	}
	topDown := rawHeight < 0
	height := rawHeight
	if topDown {
		height = -height
	}
	height /= 2
	if height <= 0 || height > maxIconDimension {
		return nil, fmt.Errorf("图标高度无效（%d）", height)
	}
	if planes != 1 {
		return nil, fmt.Errorf("图标颜色平面数无效（%d）", planes)
	}
	switch bitCount {
	case 1, 4, 8, 16, 24, 32:
	default:
		return nil, fmt.Errorf("不支持的图标色深（%d 位）", bitCount)
	}

	var (
		masksPresent bool
		rMask        uint32
		gMask        uint32
		bMask        uint32
		aMask        uint32
	)
	paletteOffset := hdrSize
	switch compression {
	case biRGB:
	case biBitfields:
		if bitCount != 16 && bitCount != 32 {
			return nil, fmt.Errorf("位域压缩不支持 %d 位色", bitCount)
		}
		masksPresent = true
		if hdrSize >= 52 {
			rMask = binary.LittleEndian.Uint32(data[40:44])
			gMask = binary.LittleEndian.Uint32(data[44:48])
			bMask = binary.LittleEndian.Uint32(data[48:52])
			if hdrSize >= 56 {
				aMask = binary.LittleEndian.Uint32(data[52:56])
			}
		} else {
			if hdrSize+12 > len(data) {
				return nil, fmt.Errorf("DIB 颜色掩码不完整")
			}
			rMask = binary.LittleEndian.Uint32(data[hdrSize : hdrSize+4])
			gMask = binary.LittleEndian.Uint32(data[hdrSize+4 : hdrSize+8])
			bMask = binary.LittleEndian.Uint32(data[hdrSize+8 : hdrSize+12])
			paletteOffset += 12
		}
		if rMask == 0 || gMask == 0 || bMask == 0 {
			rMask, gMask, bMask = 0x7c00, 0x03e0, 0x001f
		}
	default:
		return nil, fmt.Errorf("不支持的图标压缩方式（%d）", compression)
	}

	var palette []byte
	if bitCount <= 8 {
		n := clrUsed
		if n <= 0 || n > 1<<bitCount {
			n = 1 << bitCount
		}
		if paletteOffset+n*4 <= len(data) {
			palette = data[paletteOffset : paletteOffset+n*4]
			paletteOffset += n * 4
		} else {
			palette = defaultPalette(n)
		}
	}

	xorStride := ((width*bitCount + 31) / 32) * 4
	xorSize := xorStride * height
	xorOffset := paletteOffset
	if xorOffset < 0 || xorSize < 0 || xorOffset+xorSize > len(data) {
		return nil, fmt.Errorf("图标像素数据不完整")
	}

	andStride := ((width + 31) / 32) * 4
	andOffset := xorOffset + xorSize
	hasMask := andOffset+andStride*height <= len(data)

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	anyAlpha := false

	for y := 0; y < height; y++ {
		srcRow := y
		if !topDown {
			srcRow = height - 1 - y
		}
		row := xorOffset + srcRow*xorStride
		maskRow := andOffset + srcRow*andStride
		for x := 0; x < width; x++ {
			var r, g, b, a uint8 = 0, 0, 0, 255
			switch bitCount {
			case 32:
				p := row + x*4
				b, g, r, a = data[p], data[p+1], data[p+2], data[p+3]
				if a != 0 {
					anyAlpha = true
				}
				if masksPresent {
					v := binary.LittleEndian.Uint32(data[p : p+4])
					r = maskChannel(v, rMask)
					g = maskChannel(v, gMask)
					b = maskChannel(v, bMask)
					if aMask != 0 {
						a = maskChannel(v, aMask)
						if a != 0 {
							anyAlpha = true
						}
					}
				}
			case 24:
				p := row + x*3
				b, g, r, a = data[p], data[p+1], data[p+2], 255
			case 16:
				p := row + x*2
				v := uint32(data[p]) | uint32(data[p+1])<<8
				if masksPresent {
					r = maskChannel(v, rMask)
					g = maskChannel(v, gMask)
					b = maskChannel(v, bMask)
					if aMask != 0 {
						a = maskChannel(v, aMask)
						if a != 0 {
							anyAlpha = true
						}
					}
				} else {
					r = scale5(uint8((v >> 10) & 0x1f))
					g = scale5(uint8((v >> 5) & 0x1f))
					b = scale5(uint8(v & 0x1f))
				}
			case 8:
				idx := int(data[row+x])
				r, g, b = paletteColor(palette, idx)
			case 4:
				v := data[row+x/2]
				idx := int(v >> 4)
				if x%2 == 1 {
					idx = int(v & 0x0f)
				}
				r, g, b = paletteColor(palette, idx)
			case 1:
				v := data[row+x/8]
				idx := int((v >> (7 - uint(x%8))) & 1)
				r, g, b = paletteColor(palette, idx)
			}
			// A set AND-mask bit means transparent. For 32-bit icons with a
			// real alpha channel the mask is normally all zero, so this is a
			// no-op there.
			if hasMask && maskBit(data[maskRow:maskRow+andStride], x) {
				a = 0
			}
			o := y*img.Stride + x*4
			img.Pix[o+0] = r
			img.Pix[o+1] = g
			img.Pix[o+2] = b
			img.Pix[o+3] = a
		}
	}

	// Legacy 32-bit icons often leave the alpha channel at zero and rely on
	// the AND mask instead. Without a mask they are meant to be opaque.
	if bitCount == 32 && !anyAlpha {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				o := y*img.Stride + x*4
				if hasMask {
					if maskBit(data[andOffset+y*andStride:andOffset+(y+1)*andStride], x) {
						img.Pix[o+3] = 0
					} else {
						img.Pix[o+3] = 255
					}
				} else {
					img.Pix[o+3] = 255
				}
			}
		}
	}
	return img, nil
}

func maskBit(row []byte, x int) bool {
	return row[x/8]&(0x80>>uint(x%8)) != 0
}

func paletteColor(palette []byte, idx int) (uint8, uint8, uint8) {
	off := idx * 4
	if off < 0 || off+3 >= len(palette) {
		return 0, 0, 0
	}
	// RGBQUAD is blue, green, red, reserved.
	return palette[off+2], palette[off+1], palette[off]
}

func defaultPalette(n int) []byte {
	out := make([]byte, n*4)
	for i := 0; i < n; i++ {
		v := uint8(0)
		if n > 1 {
			v = uint8(i * 255 / (n - 1))
		}
		out[i*4+0] = v
		out[i*4+1] = v
		out[i*4+2] = v
	}
	return out
}

func scale5(v uint8) uint8 {
	return uint8(uint16(v) * 255 / 31)
}

func maskChannel(v, mask uint32) uint8 {
	if mask == 0 {
		return 0
	}
	shift := bits.TrailingZeros32(mask)
	max := mask >> shift
	if max == 0 {
		return 0
	}
	val := (v & mask) >> shift
	return uint8(uint64(val) * 255 / uint64(max))
}
