package render

import "math"

// IsRenderableScale reports whether Render would accept a pixel scale. It is
// the one place the bound is checked: Render uses it, and the shell asks it
// before trusting a DPI that arrived in a window message, which any program
// running as the same user could have sent.
func IsRenderableScale(pixelScale float64) bool {
	return !math.IsNaN(pixelScale) && pixelScale > 0 && pixelScale <= MaximumPixelScale
}
