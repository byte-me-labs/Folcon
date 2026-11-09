// Command genicon renders the placeholder application icon.
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
)

const size = 1024

func main() {
	out := "resources/icon.png"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	// Folder tab.
	fillRoundRect(img, 108, 200, 474, 352, 44, color.NRGBA{R: 0xDF, G: 0x99, B: 0x2B, A: 0xFF})
	// Folder body.
	fillRoundRect(img, 108, 292, 916, 872, 54, color.NRGBA{R: 0xF5, G: 0xB9, B: 0x42, A: 0xFF})
	// A lighter lip, so the folder reads as one at small sizes.
	fillRoundRect(img, 150, 336, 874, 392, 26, color.NRGBA{R: 0xF9, G: 0xCE, B: 0x6B, A: 0xFF})

	// Badge shadow.
	fillRoundRect(img, 620, 568, 956, 904, 96, color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x38})
	// App badge.
	fillRoundRect(img, 604, 552, 940, 888, 96, color.NRGBA{R: 0x2F, G: 0x6B, B: 0xFF, A: 0xFF})

	// Application window: title bar, three dots and three content lines. A
	// window reads as "application" where the play triangle did not.
	window := color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	titleBar := color.NRGBA{R: 0xD9, G: 0xE4, B: 0xFF, A: 0xFF}
	fillRoundRect(img, 632, 592, 912, 848, 22, window)
	fillRoundRect(img, 632, 592, 912, 656, 22, titleBar)
	fillRoundRect(img, 632, 634, 912, 656, 0, titleBar)
	for _, x := range []float64{660, 690, 720} {
		fillRoundRect(img, x, 617, x+14, 631, 7, color.NRGBA{R: 0x4A, G: 0x6F, B: 0xE0, A: 0xFF})
	}
	line := color.NRGBA{R: 0xC3, G: 0xD2, B: 0xF2, A: 0xFF}
	fillRoundRect(img, 660, 680, 884, 708, 14, line)
	fillRoundRect(img, 660, 732, 828, 760, 14, line)
	fillRoundRect(img, 660, 784, 772, 812, 14, line)

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(out)
	if err != nil {
		log.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}

func fillRoundRect(img *image.NRGBA, x0, y0, x1, y1, r float64, c color.NRGBA) {
	for y := int(math.Floor(y0 - 2)); y <= int(math.Ceil(y1+2)); y++ {
		for x := int(math.Floor(x0 - 2)); x <= int(math.Ceil(x1+2)); x++ {
			cov := roundRectCoverage(float64(x)+0.5, float64(y)+0.5, x0, y0, x1, y1, r)
			blend(img, x, y, c, cov)
		}
	}
}

func roundRectCoverage(x, y, x0, y0, x1, y1, r float64) float64 {
	cx := math.Max(x0+r, math.Min(x, x1-r))
	cy := math.Max(y0+r, math.Min(y, y1-r))
	dx, dy := x-cx, y-cy
	dist := math.Hypot(dx, dy)
	return clamp01(r - dist + 0.5)
}

func blend(img *image.NRGBA, x, y int, c color.NRGBA, coverage float64) {
	if coverage <= 0 || x < 0 || y < 0 || x >= img.Rect.Dx() || y >= img.Rect.Dy() {
		return
	}
	a := float64(c.A) / 255 * coverage
	if a <= 0 {
		return
	}
	i := img.PixOffset(x, y)
	if a >= 1 {
		img.Pix[i+0] = c.R
		img.Pix[i+1] = c.G
		img.Pix[i+2] = c.B
		img.Pix[i+3] = 255
		return
	}
	dstA := float64(img.Pix[i+3]) / 255
	outA := a + dstA*(1-a)
	r := (float64(c.R)*a + float64(img.Pix[i+0])*dstA*(1-a)) / outA
	g := (float64(c.G)*a + float64(img.Pix[i+1])*dstA*(1-a)) / outA
	b := (float64(c.B)*a + float64(img.Pix[i+2])*dstA*(1-a)) / outA
	img.Pix[i+0] = uint8(math.Round(r))
	img.Pix[i+1] = uint8(math.Round(g))
	img.Pix[i+2] = uint8(math.Round(b))
	img.Pix[i+3] = uint8(math.Round(outA * 255))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
