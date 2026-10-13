package theme

import (
	"image/color"
	"math"
	"testing"
)

// relativeLuminance follows WCAG 2.x: each sRGB channel is linearised, then
// weighted by how bright the eye finds it.
func relativeLuminance(colour color.RGBA) float64 {
	linearise := func(channel uint8) float64 {
		normalised := float64(channel) / 255
		if normalised <= 0.04045 {
			return normalised / 12.92
		}
		return math.Pow((normalised+0.055)/1.055, 2.4)
	}
	return 0.2126*linearise(colour.R) + 0.7152*linearise(colour.G) + 0.0722*linearise(colour.B)
}

func contrastRatio(foreground color.RGBA, background color.RGBA) float64 {
	lighter, darker := relativeLuminance(foreground), relativeLuminance(background)
	if darker > lighter {
		lighter, darker = darker, lighter
	}
	return (lighter + 0.05) / (darker + 0.05)
}

// WCAG 2.x AA: 4.5 to 1 for normal text, 3 to 1 for large text, which is 18
// point (24 CSS pixels) and up for regular weight. Sizes here are design
// units, one unit being one pixel at 100 percent and 96 DPI, the smallest
// rendering, so the check holds at every scale step above it too.
const largeTextMinimumPixels = 24

func requiredContrast(fontSizeUnits float64) float64 {
	if fontSizeUnits >= largeTextMinimumPixels {
		return 3
	}
	return 4.5
}

func TestTextColours_MeetWcagAaAgainstThePanel(t *testing.T) {
	testCases := []struct {
		name          string
		colour        color.RGBA
		fontSizeUnits float64
	}{
		{name: "time", colour: TimeText, fontSizeUnits: TimeFontSize},
		{name: "date", colour: DateText, fontSizeUnits: DateFontSize},
		{name: "zone label", colour: ZoneLabelText, fontSizeUnits: ZoneFontSize},
		{name: "zone time", colour: ZoneTimeText, fontSizeUnits: ZoneFontSize},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ratio := contrastRatio(testCase.colour, PanelFill)
			if required := requiredContrast(testCase.fontSizeUnits); ratio < required {
				t.Errorf("contrast %.2f is below the %.1f AA requires at %.0f units", ratio, required, testCase.fontSizeUnits)
			}
		})
	}
}

// The scale steps go down to 75 percent, so text that is large at 100
// percent may be normal size there. The time is the only text sized as large;
// it must still pass the normal text ratio, so no scale step can fail AA.
func TestTextColours_MeetNormalTextRatioAtEveryScale(t *testing.T) {
	for _, colour := range []color.RGBA{TimeText, DateText, ZoneLabelText, ZoneTimeText} {
		if ratio := contrastRatio(colour, PanelFill); ratio < 4.5 {
			t.Errorf("colour %v has contrast %.2f, below 4.5", colour, ratio)
		}
	}
}

func TestContrastRatio_MatchesKnownReferenceValues(t *testing.T) {
	black, white := color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}
	if ratio := contrastRatio(white, black); math.Abs(ratio-21) > 0.001 {
		t.Errorf("expected white on black to be 21, got %.3f", ratio)
	}
	grey := color.RGBA{0x76, 0x76, 0x76, 255}
	if ratio := contrastRatio(grey, white); math.Abs(ratio-4.54) > 0.01 {
		t.Errorf("expected #767676 on white to be 4.54, got %.3f", ratio)
	}
}

func TestColours_AreOpaque(t *testing.T) {
	for _, colour := range []color.RGBA{PanelFill, TimeText, DateText, ZoneLabelText, ZoneTimeText, Outline, Divider, Glow} {
		if colour.A != 255 {
			t.Errorf("colour %v is not opaque; transparency is the renderer's job, not the palette's", colour)
		}
	}
}

func TestGlowReach_IsThreeBoxRadii(t *testing.T) {
	if GlowReach != GlowBoxPasses*GlowBoxRadius {
		t.Errorf("expected the reach to be exactly what the box passes spread, got %v", GlowReach)
	}
}
