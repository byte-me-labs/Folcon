package winicon

import (
	"image/color"
	"testing"
)

func TestDecodeDIB32(t *testing.T) {
	img, err := DecodeDIB(twoByTwoDIB())
	if err != nil {
		t.Fatalf("DecodeDIB() error: %v", err)
	}
	if img.Bounds().Dx() != 2 || img.Bounds().Dy() != 2 {
		t.Fatalf("bounds = %v", img.Bounds())
	}

	// The top-left pixel is transparent because of the AND mask, despite its
	// alpha byte.
	if a := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA).A; a != 0 {
		t.Errorf("top-left alpha = %d, want 0", a)
	}
	// Bottom-left is red.
	if got := color.NRGBAModel.Convert(img.At(0, 1)).(color.NRGBA); got.R != 255 || got.G != 0 || got.B != 0 || got.A != 255 {
		t.Errorf("bottom-left = %+v, want opaque red", got)
	}
	// Bottom-right is green.
	if got := color.NRGBAModel.Convert(img.At(1, 1)).(color.NRGBA); got.R != 0 || got.G != 255 || got.B != 0 || got.A != 255 {
		t.Errorf("bottom-right = %+v, want opaque green", got)
	}
}

func TestDecodeDIB8Palette(t *testing.T) {
	const width, height = 2, 1
	hdr := make([]byte, 40)
	putU32(hdr[0:], 40)
	putU32(hdr[4:], width)
	putU32(hdr[8:], height*2)
	putU16(hdr[12:], 1)
	putU16(hdr[14:], 8)
	putU32(hdr[20:], 8)
	putU32(hdr[32:], 2) // two palette entries

	palette := []byte{
		0, 0, 0, 0, // black
		0, 0, 255, 0, // red (BGR)
	}
	stride := 4
	xor := make([]byte, stride*height)
	xor[0], xor[1] = 0, 1
	and := make([]byte, 4*height)

	data := append(hdr, palette...)
	data = append(data, xor...)
	data = append(data, and...)

	img, err := DecodeDIB(data)
	if err != nil {
		t.Fatalf("DecodeDIB() error: %v", err)
	}
	if got := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA); got.R != 0 || got.G != 0 || got.B != 0 || got.A != 255 {
		t.Errorf("palette index 0 = %+v, want opaque black", got)
	}
	if got := color.NRGBAModel.Convert(img.At(1, 0)).(color.NRGBA); got.R != 255 || got.G != 0 || got.B != 0 {
		t.Errorf("palette index 1 = %+v, want red", got)
	}
}

func TestDecodeDIBRejectsBadHeader(t *testing.T) {
	for _, data := range [][]byte{
		{},
		make([]byte, 40),
	} {
		if _, err := DecodeDIB(data); err == nil {
			t.Fatalf("DecodeDIB(% x) accepted bad data", data)
		}
	}
}

func putU32(b []byte, v uint32) {
	b[0], b[1], b[2], b[3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
}

func putU16(b []byte, v uint16) {
	b[0], b[1] = byte(v), byte(v>>8)
}
