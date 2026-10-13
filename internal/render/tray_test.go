package render

import (
	"fmt"
	"image"
	"testing"
)

// The tray icon is drawn at run time from the theme at the small icon metric,
// so there is no icon file. Golden images at the common small icon sizes.
func TestRenderTrayIcon_GoldenAtCommonSmallIconSizes(t *testing.T) {
	for _, sizePixels := range []int{16, 20, 24, 32} {
		t.Run(fmt.Sprintf("%d pixels", sizePixels), func(t *testing.T) {
			icon, renderError := RenderTrayIcon(sizePixels)
			if renderError != nil {
				t.Fatalf("rendering: %v", renderError)
			}
			compareWithGolden(t, fmt.Sprintf("tray_%d", sizePixels), icon)
		})
	}
}

func TestRenderTrayIcon_IsSquareWithAnOpaqueMiddleAndClearCorners(t *testing.T) {
	icon, renderError := RenderTrayIcon(24)
	if renderError != nil {
		t.Fatalf("rendering: %v", renderError)
	}

	if icon.Width != 24 || icon.Height != 24 || len(icon.Pixels) != 24*24*4 {
		t.Fatalf("expected a 24 pixel square, got %dx%d with %d bytes", icon.Width, icon.Height, len(icon.Pixels))
	}
	if alphaAt(icon, 12, 12) != 255 {
		t.Error("expected the middle to be opaque")
	}
	if alphaAt(icon, 0, 0) != 0 {
		t.Error("expected the rounded corner to be clear")
	}
	if icon.PanelBounds != image.Rect(0, 0, 24, 24) {
		t.Errorf("expected the whole icon to be its visible bounds, got %v", icon.PanelBounds)
	}
}

func TestRenderTrayIcon_RefusesSizesOutsideTheBound(t *testing.T) {
	for _, sizePixels := range []int{0, 7, MaximumTrayIconPixels + 1} {
		if _, renderError := RenderTrayIcon(sizePixels); renderError == nil {
			t.Errorf("expected %d pixels to be refused", sizePixels)
		}
	}
}
