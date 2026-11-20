package render

import (
	"fmt"
	"image"
	"math"

	"golang.org/x/image/vector"

	"github.com/TheScottBot/Thyme/internal/theme"
)

const (
	// MinimumAppIconPixels and MaximumAppIconPixels bound the application
	// icon to the sizes a macOS icon set holds.
	MinimumAppIconPixels = 16
	MaximumAppIconPixels = 1024

	// appIconTileFraction is the tile's share of the canvas: 824 of 1024, the
	// macOS icon grid, so the clear margin matches other apps in the Dock and
	// in Finder.
	appIconTileFraction = 824.0 / 1024.0
)

// RenderAppIcon draws the application icon from the theme: the panel as a
// rounded tile with a heavy outline, and a clock face whose hands are the tray
// icon's, minute hand up and hour hand towards three o'clock. Its proportions
// follow the author's Punchline icon, so the two read as a family. It is
// drawn when the app bundle is built, so Thyme ships no icon file.
func RenderAppIcon(sizePixels int) (*image.RGBA, error) {
	if sizePixels < MinimumAppIconPixels || sizePixels > MaximumAppIconPixels {
		return nil, fmt.Errorf("app icon size %d is outside [%d, %d]", sizePixels, MinimumAppIconPixels, MaximumAppIconPixels)
	}
	size := float64(sizePixels)
	canvas := image.NewRGBA(image.Rect(0, 0, sizePixels, sizePixels))
	rasterizer := vector.NewRasterizer(sizePixels, sizePixels)

	tile := size * appIconTileFraction
	margin := (size - tile) / 2
	outlineWidth := math.Max(1, tile/16)
	panel := floatRect{
		Min: floatPoint{X: margin + outlineWidth/2, Y: margin + outlineWidth/2},
		Max: floatPoint{X: margin + tile - outlineWidth/2, Y: margin + tile - outlineWidth/2},
	}
	cornerRadius := tile * 0.22
	fillRoundedRectangle(rasterizer, canvas, panel, cornerRadius, 0, theme.PanelFill)
	rasterizer.Reset(sizePixels, sizePixels)
	addRoundedRectanglePath(rasterizer, panel, cornerRadius, outlineWidth/2, false)
	addRoundedRectanglePath(rasterizer, panel, cornerRadius, -outlineWidth/2, true)
	rasterizer.Draw(canvas, canvas.Bounds(), image.NewUniform(theme.Outline), image.Point{})

	// A square whose corner radius is half its side is a circle, so the face
	// is a ring made the same way as the outline.
	centre := size / 2
	strokeWidth := math.Max(1, tile*0.08)
	faceRadius := tile * 0.27
	face := floatRect{Min: floatPoint{X: centre - faceRadius, Y: centre - faceRadius}, Max: floatPoint{X: centre + faceRadius, Y: centre + faceRadius}}
	rasterizer.Reset(sizePixels, sizePixels)
	addRoundedRectanglePath(rasterizer, face, faceRadius, strokeWidth/2, false)
	addRoundedRectanglePath(rasterizer, face, faceRadius, -strokeWidth/2, true)
	rasterizer.Draw(canvas, canvas.Bounds(), image.NewUniform(theme.TimeText), image.Point{})

	// Each hand is a rectangle with fully rounded ends, starting just past the
	// centre so the two meet in a rounded corner.
	halfHand := strokeWidth / 2
	minuteHand := floatRect{Min: floatPoint{X: centre - halfHand, Y: centre - tile*0.17}, Max: floatPoint{X: centre + halfHand, Y: centre + halfHand}}
	hourHand := floatRect{Min: floatPoint{X: centre - halfHand, Y: centre - halfHand}, Max: floatPoint{X: centre + tile*0.12, Y: centre + halfHand}}
	fillRoundedRectangle(rasterizer, canvas, minuteHand, halfHand, 0, theme.TimeText)
	fillRoundedRectangle(rasterizer, canvas, hourHand, halfHand, 0, theme.TimeText)
	return canvas, nil
}
