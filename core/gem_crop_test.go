package core

import (
	"image"
	"image/color"
	"testing"
)

func TestCropGemPutsTheOpaquePixelsOnTheSameScale(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 20, 20))
	src.Set(2, 4, color.RGBA{R: 255, A: 255})
	src.Set(5, 7, color.RGBA{R: 255, A: 255})

	got := cropGem(src)
	b := got.Bounds()
	if b.Dx() != b.Dy() {
		t.Fatalf("crop should be square, got %v", b)
	}
	// longest side is 4 pixels (x 2..5), and that side fills 82% of the square
	if b.Dx() < 5 {
		t.Fatalf("square too small: %d", b.Dx())
	}
	opaque := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := got.At(x, y).RGBA(); a > 0 {
				opaque++
			}
		}
	}
	if opaque != 2 {
		t.Fatalf("kept %d opaque pixels, want 2", opaque)
	}
}
