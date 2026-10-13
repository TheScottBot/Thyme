package placement

import "math"

// Rectangle is a window or a work area in physical pixels of the virtual
// screen, with Right and Bottom exclusive, as Win32's RECT is.
type Rectangle struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

func (rectangle Rectangle) Width() int32  { return rectangle.Right - rectangle.Left }
func (rectangle Rectangle) Height() int32 { return rectangle.Bottom - rectangle.Top }

func (rectangle Rectangle) offset(horizontal int32, vertical int32) Rectangle {
	return Rectangle{
		Left: rectangle.Left + horizontal, Top: rectangle.Top + vertical,
		Right: rectangle.Right + horizontal, Bottom: rectangle.Bottom + vertical,
	}
}

// Insets is how far the visible part of a window sits inside its edges. The
// widget's canvas has a transparent margin for the glow, which may hang off
// screen; only the panel inside it has to stay reachable.
type Insets struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

// ClampWindow moves window so its visible part lies inside workArea, keeping
// its size. Bottom right is clamped before top left, so a window too big for
// the work area keeps its top left corner, and the time, on screen. Nothing
// else can recover a widget that has been left off screen.
func ClampWindow(window Rectangle, visibleInset Insets, workArea Rectangle) Rectangle {
	visible := Rectangle{
		Left: window.Left + visibleInset.Left, Top: window.Top + visibleInset.Top,
		Right: window.Right - visibleInset.Right, Bottom: window.Bottom - visibleInset.Bottom,
	}
	horizontalShift, verticalShift := int32(0), int32(0)
	if visible.Right > workArea.Right {
		horizontalShift = workArea.Right - visible.Right
	}
	if visible.Bottom > workArea.Bottom {
		verticalShift = workArea.Bottom - visible.Bottom
	}
	if visible.Left+horizontalShift < workArea.Left {
		horizontalShift = workArea.Left - visible.Left
	}
	if visible.Top+verticalShift < workArea.Top {
		verticalShift = workArea.Top - visible.Top
	}
	return window.offset(horizontalShift, verticalShift)
}

func overlapArea(first Rectangle, second Rectangle) int64 {
	width := int64(min(first.Right, second.Right)) - int64(max(first.Left, second.Left))
	height := int64(min(first.Bottom, second.Bottom)) - int64(max(first.Top, second.Top))
	if width <= 0 || height <= 0 {
		return 0
	}
	return width * height
}

func gapDistance(first Rectangle, second Rectangle) float64 {
	horizontalGap := max(0, int64(second.Left)-int64(first.Right), int64(first.Left)-int64(second.Right))
	verticalGap := max(0, int64(second.Top)-int64(first.Bottom), int64(first.Top)-int64(second.Bottom))
	return math.Hypot(float64(horizontalGap), float64(verticalGap))
}

// NearestWorkArea chooses the work area a window belongs to, as Windows'
// MONITOR_DEFAULTTONEAREST does: the one it overlaps most, or, when it
// overlaps none, as after its monitor was unplugged, the closest.
func NearestWorkArea(window Rectangle, workAreas []Rectangle) (Rectangle, bool) {
	if len(workAreas) == 0 {
		return Rectangle{}, false
	}
	best := workAreas[0]
	bestOverlap, bestDistance := overlapArea(window, best), gapDistance(window, best)
	for _, candidate := range workAreas[1:] {
		candidateOverlap, candidateDistance := overlapArea(window, candidate), gapDistance(window, candidate)
		if candidateOverlap > bestOverlap || (bestOverlap == 0 && candidateOverlap == 0 && candidateDistance < bestDistance) {
			best, bestOverlap, bestDistance = candidate, candidateOverlap, candidateDistance
		}
	}
	return best, true
}

// Place puts a window on the work area it belongs to and clamps it there.
func Place(window Rectangle, visibleInset Insets, workAreas []Rectangle) (Rectangle, bool) {
	workArea, found := NearestWorkArea(window, workAreas)
	if !found {
		return window, false
	}
	return ClampWindow(window, visibleInset, workArea), true
}

// baselineDpi is the DPI at which one design unit is one pixel.
const baselineDpi = 96

// PixelScale is how many pixels one design unit takes for a scale step on a
// monitor of the given DPI.
func PixelScale(scalePercent int, dpi int) float64 {
	return float64(scalePercent) / 100 * float64(dpi) / baselineDpi
}
