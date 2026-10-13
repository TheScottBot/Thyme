package render

import (
	"math"
	"testing"
)

// The shell asks this before trusting a DPI from a window message, so a
// nonsense DPI keeps the last good one instead of freezing the widget.
func TestIsRenderableScale_AcceptsExactlyWhatRenderAccepts(t *testing.T) {
	renderer := newRendererOrFail(t)
	for _, pixelScale := range []float64{0, -1, math.NaN(), math.Inf(1), 0.001, 0.75, 1, 1.5, 2, MaximumPixelScale, MaximumPixelScale + 0.01, 52} {
		_, renderError := renderer.Render(mockUpText(), pixelScale)
		if IsRenderableScale(pixelScale) != (renderError == nil) {
			t.Errorf("scale %v: IsRenderableScale says %v, Render's error is %v", pixelScale, IsRenderableScale(pixelScale), renderError)
		}
	}
}
