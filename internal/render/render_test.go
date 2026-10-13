package render

import (
	"fmt"
	"math"
	"runtime"
	"testing"

	"github.com/TheScottBot/Thyme/internal/theme"
)

func TestRender_GoldenMockUpAtOneHundredPercent(t *testing.T) {
	compareWithGolden(t, "mockup_100", renderOrFail(t, newRendererOrFail(t), mockUpText(), 1))
}

func TestRender_GoldenMockUpAtTwoHundredPercent(t *testing.T) {
	compareWithGolden(t, "mockup_200", renderOrFail(t, newRendererOrFail(t), mockUpText(), 2))
}

func TestRender_GoldenZoneCountsWithSecondsOnAndOff(t *testing.T) {
	for _, zoneCount := range []int{0, 1, 4} {
		for _, showSeconds := range []bool{true, false} {
			goldenName := fmt.Sprintf("zones%d_seconds_%s", zoneCount, map[bool]string{true: "on", false: "off"}[showSeconds])
			t.Run(goldenName, func(t *testing.T) {
				compareWithGolden(t, goldenName, renderOrFail(t, newRendererOrFail(t), textWith(showSeconds, zoneCount), 1))
			})
		}
	}
}

func TestRender_FrameIsTightlyPackedBGRA(t *testing.T) {
	frame := renderOrFail(t, newRendererOrFail(t), mockUpText(), 1)

	if len(frame.Pixels) != frame.Width*frame.Height*4 {
		t.Fatalf("expected %d bytes for %dx%d, got %d", frame.Width*frame.Height*4, frame.Width, frame.Height, len(frame.Pixels))
	}
	panel := computeLayoutForTest(t, mockUpText(), 1).panel
	centreOffset := (int(panel.Min.Y+20)*frame.Width + int(panel.Min.X+5)) * 4
	pixel := frame.Pixels[centreOffset : centreOffset+4]
	expected := []byte{theme.PanelFill.B, theme.PanelFill.G, theme.PanelFill.R, 0xff}
	if string(pixel) != string(expected) {
		t.Errorf("expected the empty panel in BGRA order % x, got % x", expected, pixel)
	}
}

// Fully transparent layered window pixels let clicks through, so every pixel
// of the panel must be opaque for the panel to be the drag target. Pixels on
// the anti aliased edge are excluded by testing one pixel inside it.
func TestRender_EveryPixelInsideThePanelIsOpaque(t *testing.T) {
	for _, pixelScale := range []float64{0.75, 1, 2} {
		for _, text := range []Text{mockUpText(), textWith(false, 0)} {
			frame := renderOrFail(t, newRendererOrFail(t), text, pixelScale)
			layout := computeLayoutForTest(t, text, pixelScale)
			for y := 0; y < frame.Height; y++ {
				for x := 0; x < frame.Width; x++ {
					if layout.isInsidePanelBy(float64(x)+0.5, float64(y)+0.5, 1) && alphaAt(frame, x, y) != 255 {
						t.Fatalf("scale %v: pixel (%d, %d) inside the panel has alpha %d", pixelScale, x, y, alphaAt(frame, x, y))
					}
				}
			}
		}
	}
}

func TestRender_EveryPixelBeyondTheGlowIsTransparent(t *testing.T) {
	for _, pixelScale := range []float64{0.75, 1, 2} {
		frame := renderOrFail(t, newRendererOrFail(t), mockUpText(), pixelScale)
		layout := computeLayoutForTest(t, mockUpText(), pixelScale)
		glowReachPixels := theme.GlowReach * pixelScale
		transparentRingSeen := false
		for y := 0; y < frame.Height; y++ {
			for x := 0; x < frame.Width; x++ {
				if layout.distanceOutsidePanel(float64(x)+0.5, float64(y)+0.5) > glowReachPixels+1 {
					transparentRingSeen = true
					if alphaAt(frame, x, y) != 0 {
						t.Fatalf("scale %v: pixel (%d, %d) beyond the glow has alpha %d", pixelScale, x, y, alphaAt(frame, x, y))
					}
				}
			}
		}
		if !transparentRingSeen {
			t.Errorf("scale %v: the canvas has no pixel beyond the glow, so the property was not tested", pixelScale)
		}
	}
}

// The WE4 test case the specification asks for: whether the glow fringe can
// be clicked. Any alpha above zero catches clicks on a layered window, so the
// glow is clickable as far as it is visible at all, and the canvas edge is
// not.
func TestRender_GlowFringeIsClickableWhereverItIsDrawn(t *testing.T) {
	frame := renderOrFail(t, newRendererOrFail(t), mockUpText(), 1)
	layout := computeLayoutForTest(t, mockUpText(), 1)
	middleRow := int(layout.panel.Min.Y + layout.panel.Dy()/2)

	justOutside := int(math.Floor(layout.panel.Min.X - 2))
	if alphaAt(frame, justOutside, middleRow) == 0 {
		t.Error("expected the glow just outside the panel to catch clicks")
	}
	if alphaAt(frame, 0, middleRow) != 0 {
		t.Error("expected the canvas edge to let clicks through")
	}
}

func TestRender_SecondsOffNarrowsThePanelToTheDateLine(t *testing.T) {
	withSeconds := computeLayoutForTest(t, textWith(true, 4), 1)
	withoutSeconds := computeLayoutForTest(t, textWith(false, 4), 1)

	if withoutSeconds.panel.Dx() >= withSeconds.panel.Dx() {
		t.Fatalf("expected seconds off to be narrower, got %v and %v", withoutSeconds.panel.Dx(), withSeconds.panel.Dx())
	}
	// Without seconds the time is shorter than the date line, so the date
	// line sets the column: 14 advances of the date instead of 8 of the time.
	expectedNarrowing := 8*0.6*theme.TimeFontSize - 14*0.6*theme.DateFontSize
	if narrowing := withSeconds.panel.Dx() - withoutSeconds.panel.Dx(); math.Abs(narrowing-expectedNarrowing) > 0.01 {
		t.Errorf("expected the panel to narrow by %.2f units, got %.2f", expectedNarrowing, narrowing)
	}
}

func TestRender_ZeroZonesDropTheDividerAndTheColumn(t *testing.T) {
	layout := computeLayoutForTest(t, textWith(true, 0), 1)

	if layout.hasDivider {
		t.Error("expected no divider without zones")
	}
	expectedWidth := theme.PaddingLeft + 8*0.6*theme.TimeFontSize + theme.PaddingRight
	if math.Abs(layout.panel.Dx()-expectedWidth) > 0.01 {
		t.Errorf("expected a panel %.2f units wide, got %.2f", expectedWidth, layout.panel.Dx())
	}
}

func TestRender_MockUpPanelMatchesTheMeasuredSize(t *testing.T) {
	layout := computeLayoutForTest(t, mockUpText(), 1)

	// Measured on docs/theme.png at 200 percent: 894 by 305 pixels.
	if math.Abs(layout.panel.Dx()-447) > 1 || math.Abs(layout.panel.Dy()-152.5) > 1.5 {
		t.Errorf("expected about 447 by 152.5 units, got %.2f by %.2f", layout.panel.Dx(), layout.panel.Dy())
	}
}

func TestRender_PanelScalesWithPixelScale(t *testing.T) {
	single := computeLayoutForTest(t, mockUpText(), 1)
	double := computeLayoutForTest(t, mockUpText(), 2)

	if math.Abs(double.panel.Dx()-2*single.panel.Dx()) > 0.01 || math.Abs(double.panel.Dy()-2*single.panel.Dy()) > 0.01 {
		t.Errorf("expected twice the panel, got %v then %v", single.panel, double.panel)
	}
}

func TestRender_RefusesScalesOutsideTheBound(t *testing.T) {
	renderer := newRendererOrFail(t)
	for _, pixelScale := range []float64{0, -1, math.NaN(), math.Inf(1), MaximumPixelScale + 0.01} {
		if _, renderError := renderer.Render(mockUpText(), pixelScale); renderError == nil {
			t.Errorf("expected scale %v to be refused", pixelScale)
		}
	}
}

func TestRender_StaticLayerIsBuiltOnceForChangingTimes(t *testing.T) {
	renderer := newRendererOrFail(t)
	renderOrFail(t, renderer, mockUpText(), 1)
	later := mockUpText()
	later.Time, later.Zones[0].Time = "21:04:02", "20:05"

	renderOrFail(t, renderer, later, 1)

	if renderer.staticLayerBuildCount != 1 {
		t.Errorf("expected one static layer build, got %d", renderer.staticLayerBuildCount)
	}
}

func TestRender_StaticLayerIsRebuiltWhenWhatItShowsChanges(t *testing.T) {
	testCases := []struct {
		name       string
		text       Text
		pixelScale float64
	}{
		{name: "another scale", text: mockUpText(), pixelScale: 1.25},
		{name: "a changed label", text: func() Text { text := mockUpText(); text.Zones[1].Label = "LON"; return text }(), pixelScale: 1},
		{name: "fewer zones", text: textWith(true, 1), pixelScale: 1},
		{name: "seconds off", text: textWith(false, 4), pixelScale: 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			renderer := newRendererOrFail(t)
			renderOrFail(t, renderer, mockUpText(), 1)

			renderOrFail(t, renderer, testCase.text, testCase.pixelScale)

			if renderer.staticLayerBuildCount != 2 {
				t.Errorf("expected a rebuild, got %d builds", renderer.staticLayerBuildCount)
			}
		})
	}
}

// Steady state ticks must not allocate frame sized buffers. Two measures: the
// count of allocations per frame against the bound recorded in TESTING.md, and
// the bytes allocated per frame against a small fraction of one frame, which
// is the property that matters.
func TestRender_SteadyStateTickAllocatesNoFrameSizedBuffer(t *testing.T) {
	renderer := newRendererOrFail(t)
	renderOrFail(t, renderer, mockUpText(), 1)
	tickTexts := []Text{mockUpText(), mockUpText()}
	tickTexts[1].Time = "21:04:02"
	tickIndex := 0
	renderTick := func() {
		tickIndex++
		if _, renderError := renderer.Render(tickTexts[tickIndex%2], 1); renderError != nil {
			t.Fatalf("rendering: %v", renderError)
		}
	}

	allocationsPerTick := testing.AllocsPerRun(50, renderTick)

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	const measuredTicks = 50
	for range measuredTicks {
		renderTick()
	}
	runtime.ReadMemStats(&after)
	bytesPerTick := (after.TotalAlloc - before.TotalAlloc) / measuredTicks
	frame := renderOrFail(t, renderer, mockUpText(), 1)
	frameBytes := uint64(len(frame.Pixels))

	if allocationsPerTick > MaximumAllocationsPerTick {
		t.Errorf("expected at most %d allocations per tick, got %.1f", MaximumAllocationsPerTick, allocationsPerTick)
	}
	if bytesPerTick > frameBytes/64 {
		t.Errorf("expected well under one frame (%d bytes) allocated per tick, got %d", frameBytes, bytesPerTick)
	}
	t.Logf("per tick: %.1f allocations, %d bytes; one frame is %d bytes", allocationsPerTick, bytesPerTick, frameBytes)
}

func computeLayoutForTest(t *testing.T, text Text, pixelScale float64) frameLayout {
	t.Helper()
	renderer := newRendererOrFail(t)
	layout, layoutError := renderer.computeLayout(text, pixelScale)
	if layoutError != nil {
		t.Fatalf("computing layout: %v", layoutError)
	}
	return layout
}
