package render

import (
	"fmt"
	"image"
)

// ConvertRGBAToBGRA packs source into destination as top down, premultiplied
// BGRA, the layout a layered window's per pixel alpha blend consumes. Go's
// image.RGBA is already premultiplied, so only the channel order changes. The
// destination must be exactly width times height times four bytes; anything
// else is refused rather than overrun or left partly stale.
func ConvertRGBAToBGRA(destination []byte, source *image.RGBA) error {
	width, height := source.Rect.Dx(), source.Rect.Dy()
	if len(destination) != width*height*4 {
		return fmt.Errorf("destination holds %d bytes, a %dx%d frame needs %d", len(destination), width, height, width*height*4)
	}
	for row := 0; row < height; row++ {
		sourceRow := source.Pix[source.PixOffset(source.Rect.Min.X, source.Rect.Min.Y+row):][:width*4]
		destinationRow := destination[row*width*4:][:width*4]
		for pixelOffset := 0; pixelOffset < width*4; pixelOffset += 4 {
			destinationRow[pixelOffset+0] = sourceRow[pixelOffset+2]
			destinationRow[pixelOffset+1] = sourceRow[pixelOffset+1]
			destinationRow[pixelOffset+2] = sourceRow[pixelOffset+0]
			destinationRow[pixelOffset+3] = sourceRow[pixelOffset+3]
		}
	}
	return nil
}
