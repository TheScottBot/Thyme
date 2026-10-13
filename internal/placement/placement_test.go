package placement

import (
	"fmt"
	"math"
	"testing"
)

// A 1920 by 1040 work area: a 1080 high monitor with a 40 pixel taskbar.
var primaryWorkArea = Rectangle{Left: 0, Top: 0, Right: 1920, Bottom: 1040}

func windowAt(left int32, top int32, width int32, height int32) Rectangle {
	return Rectangle{Left: left, Top: top, Right: left + width, Bottom: top + height}
}

func TestClampWindow_PullsBackAWindowHangingOffEachEdge(t *testing.T) {
	testCases := []struct {
		name     string
		window   Rectangle
		expected Rectangle
	}{
		{name: "off the left", window: windowAt(-150, 300, 400, 200), expected: windowAt(0, 300, 400, 200)},
		{name: "off the right", window: windowAt(1700, 300, 400, 200), expected: windowAt(1520, 300, 400, 200)},
		{name: "off the top", window: windowAt(600, -90, 400, 200), expected: windowAt(600, 0, 400, 200)},
		{name: "off the bottom, under the taskbar", window: windowAt(600, 900, 400, 200), expected: windowAt(600, 840, 400, 200)},
		{name: "off a corner", window: windowAt(1800, 1000, 400, 200), expected: windowAt(1520, 840, 400, 200)},
		{name: "already inside", window: windowAt(600, 300, 400, 200), expected: windowAt(600, 300, 400, 200)},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if clamped := ClampWindow(testCase.window, Insets{}, primaryWorkArea); clamped != testCase.expected {
				t.Errorf("expected %+v, got %+v", testCase.expected, clamped)
			}
		})
	}
}

// Bottom right is applied before top left, so when the window cannot fit,
// its top left corner, where the time is, stays on screen.
func TestClampWindow_LargerThanTheWorkAreaKeepsTopLeftOnScreen(t *testing.T) {
	clamped := ClampWindow(windowAt(500, 400, 2400, 1200), Insets{}, primaryWorkArea)

	if clamped.Left != 0 || clamped.Top != 0 {
		t.Errorf("expected the top left corner at the work area's, got %+v", clamped)
	}
	if clamped.Width() != 2400 || clamped.Height() != 1200 {
		t.Errorf("expected the size kept, got %+v", clamped)
	}
}

// The transparent glow margin may hang off screen; only the visible panel is
// kept inside, so the panel can sit flush against a screen edge.
func TestClampWindow_ClampsTheVisiblePartNotTheTransparentMargin(t *testing.T) {
	margin := Insets{Left: 27, Top: 27, Right: 27, Bottom: 27}

	clamped := ClampWindow(windowAt(-100, -100, 500, 230), margin, primaryWorkArea)

	if clamped != windowAt(-27, -27, 500, 230) {
		t.Errorf("expected the panel flush with the top left corner, got %+v", clamped)
	}
}

func TestNearestWorkArea_PrefersTheLargestOverlapAndElseTheNearest(t *testing.T) {
	secondaryWorkArea := Rectangle{Left: 1920, Top: -200, Right: 4480, Bottom: 1240}
	workAreas := []Rectangle{primaryWorkArea, secondaryWorkArea}
	testCases := []struct {
		name     string
		window   Rectangle
		expected Rectangle
	}{
		{name: "wholly on the secondary", window: windowAt(2500, 100, 400, 200), expected: secondaryWorkArea},
		{name: "straddling, mostly on the primary", window: windowAt(1700, 100, 400, 200), expected: primaryWorkArea},
		{name: "straddling, mostly on the secondary", window: windowAt(1850, 100, 400, 200), expected: secondaryWorkArea},
		{name: "on a monitor that no longer exists, left of everything", window: windowAt(-2600, 300, 400, 200), expected: primaryWorkArea},
		{name: "on a monitor that no longer exists, far right", window: windowAt(6000, 300, 400, 200), expected: secondaryWorkArea},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			nearest, found := NearestWorkArea(testCase.window, workAreas)
			if !found || nearest != testCase.expected {
				t.Errorf("expected %+v, got %+v (found %v)", testCase.expected, nearest, found)
			}
		})
	}
}

func TestNearestWorkArea_ReportsNoneWhenThereAreNoMonitors(t *testing.T) {
	if _, found := NearestWorkArea(windowAt(0, 0, 10, 10), nil); found {
		t.Error("expected no work area from an empty list")
	}
}

// The specification's case: the window's monitor was unplugged, so it is
// moved onto the nearest remaining monitor and clamped there.
func TestPlace_WindowOnAVanishedMonitorLandsOnTheNearestOne(t *testing.T) {
	placed, found := Place(windowAt(-1500, 500, 400, 200), Insets{}, []Rectangle{primaryWorkArea})

	if !found || placed != windowAt(0, 500, 400, 200) {
		t.Errorf("expected the window pulled onto the primary monitor, got %+v (found %v)", placed, found)
	}
}

func TestPixelScale_EveryStepAtEveryCommonDpi(t *testing.T) {
	expectedByDpi := map[int][]float64{
		96:  {0.75, 1, 1.25, 1.5, 2},
		120: {0.9375, 1.25, 1.5625, 1.875, 2.5},
		144: {1.125, 1.5, 1.875, 2.25, 3},
		192: {1.5, 2, 2.5, 3, 4},
	}
	for dpi, expectedScales := range expectedByDpi {
		for stepIndex, scalePercent := range []int{75, 100, 125, 150, 200} {
			t.Run(fmt.Sprintf("%d percent at %d DPI", scalePercent, dpi), func(t *testing.T) {
				if pixelScale := PixelScale(scalePercent, dpi); math.Abs(pixelScale-expectedScales[stepIndex]) > 1e-9 {
					t.Errorf("expected %v, got %v", expectedScales[stepIndex], pixelScale)
				}
			})
		}
	}
}
