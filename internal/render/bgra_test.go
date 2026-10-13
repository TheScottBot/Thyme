package render

import (
	"image"
	"image/color"
	"slices"
	"testing"
)

func TestConvertRGBAToBGRA_SwapsRedAndBlueOfKnownPremultipliedColours(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 4, 1))
	source.SetRGBA(0, 0, color.RGBA{R: 0x12, G: 0x13, B: 0x1c, A: 0xff})
	source.SetRGBA(1, 0, color.RGBA{R: 0, G: 0, B: 0, A: 0})
	source.SetRGBA(2, 0, color.RGBA{R: 0x7c, G: 0x4a, B: 0x7c, A: 0x80})
	source.SetRGBA(3, 0, color.RGBA{R: 0x01, G: 0x00, B: 0x02, A: 0x03})
	destination := make([]byte, 16)

	if convertError := ConvertRGBAToBGRA(destination, source); convertError != nil {
		t.Fatalf("unexpected error: %v", convertError)
	}

	expected := []byte{
		0x1c, 0x13, 0x12, 0xff,
		0x00, 0x00, 0x00, 0x00,
		0x7c, 0x4a, 0x7c, 0x80,
		0x02, 0x00, 0x01, 0x03,
	}
	if !slices.Equal(destination, expected) {
		t.Errorf("expected % x, got % x", expected, destination)
	}
}

// A layered window consumes a tightly packed top down bitmap, so a source
// whose rows are wider than its bounds (a sub image) must be packed.
func TestConvertRGBAToBGRA_PacksRowsOfSubImage(t *testing.T) {
	whole := image.NewRGBA(image.Rect(0, 0, 3, 2))
	whole.SetRGBA(1, 0, color.RGBA{R: 1, G: 2, B: 3, A: 4})
	whole.SetRGBA(1, 1, color.RGBA{R: 5, G: 6, B: 7, A: 8})
	source := whole.SubImage(image.Rect(1, 0, 2, 2)).(*image.RGBA)
	destination := make([]byte, 8)

	if convertError := ConvertRGBAToBGRA(destination, source); convertError != nil {
		t.Fatalf("unexpected error: %v", convertError)
	}

	if expected := []byte{3, 2, 1, 4, 7, 6, 5, 8}; !slices.Equal(destination, expected) {
		t.Errorf("expected % x, got % x", expected, destination)
	}
}

func TestConvertRGBAToBGRA_RefusesWrongSizedDestination(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 2, 2))

	if convertError := ConvertRGBAToBGRA(make([]byte, 15), source); convertError == nil {
		t.Error("expected a destination one byte short to be refused rather than overrun")
	}
	if convertError := ConvertRGBAToBGRA(make([]byte, 17), source); convertError == nil {
		t.Error("expected a destination of the wrong size to be refused")
	}
}
