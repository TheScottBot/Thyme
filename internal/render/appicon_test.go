package render

import (
	"fmt"
	"image"
	"testing"

	"github.com/TheScottBot/Thyme/internal/theme"
)

func appIconFrameOrFail(t *testing.T, sizePixels int) Frame {
	t.Helper()
	icon, renderError := RenderAppIcon(sizePixels)
	if renderError != nil {
		t.Fatalf("rendering: %v", renderError)
	}
	pixels := make([]byte, len(icon.Pix))
	if convertError := ConvertRGBAToBGRA(pixels, icon); convertError != nil {
		t.Fatalf("converting: %v", convertError)
	}
	return Frame{Width: sizePixels, Height: sizePixels, Pixels: pixels, PanelBounds: icon.Bounds()}
}

// appIconChannelTolerance was measured on CI on 20 November 2026: the 256
// pixel icon, golden on arm64, differed on the amd64 runners by 3 of 255 in 4
// bytes. Its long curved edges round a little differently by architecture,
// most likely because Go may fuse multiply and add on arm64 but not on amd64;
// the clock face keeps goldenChannelTolerance, measured in WE4.
const appIconChannelTolerance = 3

func TestRenderAppIcon_GoldenAtIconSetSizes(t *testing.T) {
	for _, sizePixels := range []int{16, 32, 128, 256} {
		t.Run(fmt.Sprintf("%d pixels", sizePixels), func(t *testing.T) {
			compareWithGoldenWithin(t, fmt.Sprintf("appicon_%d", sizePixels), appIconFrameOrFail(t, sizePixels), appIconChannelTolerance)
		})
	}
}

// The tile sits on the macOS icon grid: the margin around it is clear, and
// inside the outline it is the panel's own colour.
func TestRenderAppIcon_HasAClearMarginAndThePanelInside(t *testing.T) {
	icon, renderError := RenderAppIcon(1024)
	if renderError != nil {
		t.Fatalf("rendering: %v", renderError)
	}
	if icon.Bounds() != image.Rect(0, 0, 1024, 1024) {
		t.Fatalf("expected 1024 square, got %v", icon.Bounds())
	}
	for _, point := range []image.Point{{0, 0}, {512, 50}, {50, 512}, {1023, 1023}, {512, 1000}} {
		if alpha := icon.RGBAAt(point.X, point.Y).A; alpha != 0 {
			t.Errorf("expected %v in the margin to be clear, got alpha %d", point, alpha)
		}
	}
	if inside := icon.RGBAAt(200, 512); inside != theme.PanelFill {
		t.Errorf("expected the panel colour between outline and face, got %v", inside)
	}
	if edge := icon.RGBAAt(512, 130); edge != theme.Outline {
		t.Errorf("expected the outline at the tile's top, got %v", edge)
	}
	if hands := icon.RGBAAt(512, 512); hands != theme.TimeText {
		t.Errorf("expected the hands to meet at the centre, got %v", hands)
	}
}

func TestRenderAppIcon_RefusesSizesOutsideTheBound(t *testing.T) {
	for _, sizePixels := range []int{0, MinimumAppIconPixels - 1, MaximumAppIconPixels + 1} {
		if _, renderError := RenderAppIcon(sizePixels); renderError == nil {
			t.Errorf("expected %d pixels to be refused", sizePixels)
		}
	}
}
