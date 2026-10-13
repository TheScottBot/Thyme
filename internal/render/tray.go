package render

import (
	"fmt"
	"image"
	"math"

	"golang.org/x/image/vector"

	"github.com/TheScottBot/Thyme/internal/theme"
)

const (
	// MinimumTrayIconPixels and MaximumTrayIconPixels bound the icon: below 8
	// pixels nothing of the design survives, and 256 is the largest icon
	// Windows uses anywhere.
	MinimumTrayIconPixels = 8
	MaximumTrayIconPixels = 256
)

// RenderTrayIcon draws the tray icon from the theme at the given size: the
// panel as a rounded square with its outline, and a clock face's two hands in
// the time's colour. It is drawn at run time at the system's small icon size,
// so Thyme ships no icon file and the icon is always crisp. Allocation here is
// fine; the icon is drawn only when the tray icon is added.
func RenderTrayIcon(sizePixels int) (Frame, error) {
	if sizePixels < MinimumTrayIconPixels || sizePixels > MaximumTrayIconPixels {
		return Frame{}, fmt.Errorf("tray icon size %d is outside [%d, %d]", sizePixels, MinimumTrayIconPixels, MaximumTrayIconPixels)
	}
	size := float64(sizePixels)
	canvas := image.NewRGBA(image.Rect(0, 0, sizePixels, sizePixels))
	outlineWidth := math.Max(1, math.Round(size/16))
	panel := floatRect{
		Min: floatPoint{X: outlineWidth / 2, Y: outlineWidth / 2},
		Max: floatPoint{X: size - outlineWidth/2, Y: size - outlineWidth/2},
	}
	cornerRadius := size * 0.22

	rasterizer := vector.NewRasterizer(sizePixels, sizePixels)
	fillRoundedRectangle(rasterizer, canvas, panel, cornerRadius, 0, theme.PanelFill)
	rasterizer.Reset(sizePixels, sizePixels)
	addRoundedRectanglePath(rasterizer, panel, cornerRadius, outlineWidth/2, false)
	addRoundedRectanglePath(rasterizer, panel, cornerRadius, -outlineWidth/2, true)
	rasterizer.Draw(canvas, canvas.Bounds(), image.NewUniform(theme.Outline), image.Point{})

	// The hands are snapped to whole pixels so they stay sharp at 16 pixels:
	// the minute hand straight up, the hour hand towards three o'clock.
	handWidth := math.Max(1, math.Round(size/10))
	centre := math.Round(size/2 - handWidth/2)
	minuteHand := floatRect{Min: floatPoint{X: centre, Y: math.Round(size * 0.22)}, Max: floatPoint{X: centre + handWidth, Y: centre + handWidth}}
	hourHand := floatRect{Min: floatPoint{X: centre, Y: centre}, Max: floatPoint{X: math.Round(size * 0.72), Y: centre + handWidth}}
	fillRoundedRectangle(rasterizer, canvas, minuteHand, 0, 0, theme.TimeText)
	fillRoundedRectangle(rasterizer, canvas, hourHand, 0, 0, theme.TimeText)

	pixels := make([]byte, sizePixels*sizePixels*4)
	if convertError := ConvertRGBAToBGRA(pixels, canvas); convertError != nil {
		return Frame{}, convertError
	}
	return Frame{Width: sizePixels, Height: sizePixels, Pixels: pixels, PanelBounds: canvas.Bounds()}, nil
}
