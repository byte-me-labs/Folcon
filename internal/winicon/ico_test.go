package winicon

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestParseICOWithPNG(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 10, G: 200, B: 30, A: 255})
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatal(err)
	}

	ico := makeICO(16, 16, 32, pngBuf.Bytes())
	frames, err := ParseICO(ico)
	if err != nil {
		t.Fatalf("ParseICO() error: %v", err)
	}
	if len(frames) != 1 {
		t.Fatalf("got %d frames, want 1", len(frames))
	}
	if !frames[0].IsPNG || frames[0].Width != 16 || frames[0].Height != 16 {
		t.Fatalf("frame = %+v", frames[0])
	}
	decoded, err := frames[0].Decode()
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}
	if got := decoded.Bounds().Dx(); got != 16 {
		t.Fatalf("decoded width = %d", got)
	}
}

func TestParseICORejectsBadInput(t *testing.T) {
	for _, data := range [][]byte{
		{},
		{0, 0, 2, 0, 1, 0},
		make([]byte, 22),
	} {
		if _, err := ParseICO(data); err == nil {
			t.Fatalf("ParseICO(% x) accepted bad data", data)
		}
	}
}

func TestParseICOBits(t *testing.T) {
	dib := twoByTwoDIB()
	ico := makeICO(2, 2, 32, dib)
	frames, err := ParseICO(ico)
	if err != nil {
		t.Fatal(err)
	}
	if frames[0].IsPNG {
		t.Fatal("DIB frame reported as PNG")
	}
	img, err := frames[0].Decode()
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}
	if got := img.Bounds().Dx(); got != 2 {
		t.Fatalf("width = %d", got)
	}
}

func TestGroupSizes(t *testing.T) {
	g := Group{Frames: []Frame{
		{Width: 16, Height: 16, BPP: 32},
		{Width: 48, Height: 48, BPP: 32},
		{Width: 16, Height: 16, BPP: 8},
	}}
	sizes := g.Sizes()
	if len(sizes) != 2 || sizes[0] != 16 || sizes[1] != 48 {
		t.Fatalf("Sizes() = %v", sizes)
	}
	if got := (&Group{}).Sizes(); len(got) != 0 {
		t.Fatalf("empty Sizes() = %v", got)
	}
}

func makeICO(width, height int, bpp uint16, payload []byte) []byte {
	out := make([]byte, icoHeaderSize+icoEntrySize+len(payload))
	binary.LittleEndian.PutUint16(out[2:4], 1)
	binary.LittleEndian.PutUint16(out[4:6], 1)
	out[6] = byte(width % 256)
	out[7] = byte(height % 256)
	binary.LittleEndian.PutUint16(out[10:12], 1)
	binary.LittleEndian.PutUint16(out[12:14], bpp)
	binary.LittleEndian.PutUint32(out[14:18], uint32(len(payload)))
	binary.LittleEndian.PutUint32(out[18:22], uint32(icoHeaderSize+icoEntrySize))
	copy(out[22:], payload)
	return out
}

// twoByTwoDIB builds a 2×2 icon DIB:
//
//	top-left  blue with alpha (also masked transparent)
//	top-right white with alpha 0
//	bottom-left red, bottom-right green
func twoByTwoDIB() []byte {
	const width, height = 2, 2
	stride := ((width*32 + 31) / 32) * 4
	hdr := make([]byte, 40)
	binary.LittleEndian.PutUint32(hdr[0:4], 40)
	binary.LittleEndian.PutUint32(hdr[4:8], width)
	binary.LittleEndian.PutUint32(hdr[8:12], height*2)
	binary.LittleEndian.PutUint16(hdr[12:14], 1)
	binary.LittleEndian.PutUint16(hdr[14:16], 32)
	binary.LittleEndian.PutUint32(hdr[20:24], uint32(stride*height))

	xor := make([]byte, stride*height)
	// Bottom row first (DIBs are bottom-up).
	setPixel(xor, stride, 0, 0, 0, 0, 255, 255) // red
	setPixel(xor, stride, 0, 1, 0, 255, 0, 255) // green
	setPixel(xor, stride, 1, 0, 255, 0, 0, 128) // blue
	setPixel(xor, stride, 1, 1, 255, 255, 255, 0)

	andStride := ((width + 31) / 32) * 4
	and := make([]byte, andStride*height)
	and[andStride+0] = 0x80 // top row, left pixel transparent

	out := append(hdr, xor...)
	out = append(out, and...)
	return out
}

func setPixel(buf []byte, stride, row, x int, b, g, r, a uint8) {
	off := row*stride + x*4
	buf[off+0], buf[off+1], buf[off+2], buf[off+3] = b, g, r, a
}
