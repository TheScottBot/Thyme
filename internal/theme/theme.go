package theme

import "image/color"

// Every size below is in design units: one unit is one pixel at 100 percent
// and 96 DPI. Every value was measured on docs/theme.png, which the author
// decided is a 200 percent render, so a unit is two mock-up pixels. The
// measurements and their uncertainties are in
// docs/evaluation/ACTUAL_CONTRACT_EVALUATION.md (WE4 Plan); every value is
// subject to the WE4 author gate.

// Colours. The palette is opaque; transparency comes from the renderer's
// coverage, never from here.
var (
	// PanelFill is the median of three empty panel regions (WU1).
	PanelFill = color.RGBA{R: 0x12, G: 0x13, B: 0x1c, A: 0xff}

	// TimeText is the brightest fifth of the large digits' ink (WU2). The
	// thick strokes reach the true fill; thin strokes never quite do.
	TimeText = color.RGBA{R: 0xf8, G: 0x94, B: 0xf8, A: 0xff}

	// ZoneTimeText measured within a few levels of the time, so it is the
	// same colour rather than a near duplicate.
	ZoneTimeText = TimeText

	// DateText measured a clear step dimmer than the time (WU2).
	DateText = color.RGBA{R: 0xe4, G: 0x8b, B: 0xe6, A: 0xff}

	// ZoneLabelText is the dimmest text, so the eye reads the times first.
	ZoneLabelText = color.RGBA{R: 0xd5, G: 0x83, B: 0xd7, A: 0xff}

	// Outline is the bright core of the panel edge (WU3).
	Outline = color.RGBA{R: 0xf5, G: 0x9e, B: 0xf3, A: 0xff}

	// Glow is the outline's hue, as the mock-up's glow is (WU3).
	Glow = Outline

	// Divider is the vertical rule between the time and the zones.
	Divider = color.RGBA{R: 0xd5, G: 0x99, B: 0xd7, A: 0xff}
)

// Font sizes, in units per em of JetBrains Mono Regular, fitted to the
// mock-up's digit heights and ink widths.
const (
	TimeFontSize = 49.0
	DateFontSize = 18.0
	ZoneFontSize = 17.0
)

// Panel geometry. The panel rectangle is the outline's centre line.
const (
	PanelCornerRadius = 6.5
	OutlineWidth      = 1.0

	// PaddingLeft runs from the panel edge to the time's and date's pen
	// origin; PaddingRight from the last zone time's advance to the edge.
	PaddingLeft  = 27.0
	PaddingRight = 27.0

	// PaddingVertical runs from the panel edge to the taller of the two
	// blocks; the shorter block is centred against it.
	PaddingVertical = 30.0

	// TimeBaselineToDateBaseline sets the date under the time.
	TimeBaselineToDateBaseline = 34.0

	TimeColumnToDivider = 25.0
	DividerWidth        = 1.0
	DividerToZones      = 26.0

	// DividerInset keeps the divider short of the panel's top and bottom.
	DividerInset = 20.0

	// ZoneLabelToZoneTime is the gap after the label's three advances.
	ZoneLabelToZoneTime = 24.0
	ZoneRowPitch        = 26.5
)

// The glow is the panel's shape blurred by GlowBoxPasses box blurs of
// GlowBoxRadius, which approximates a Gaussian of about 8 units: the fit to
// the mock-up's falloff (+35 red just outside the outline, fading to nothing
// about 20 units out). A box blur spreads exactly its radius per pass, so
// GlowReach is where the glow provably ends and every pixel beyond is fully
// transparent.
const (
	GlowBoxPasses = 3
	GlowBoxRadius = 8.0
	GlowReach     = GlowBoxPasses * GlowBoxRadius

	// GlowPeakAlpha scales the blurred shape: about half of it at the panel
	// edge, so roughly 0.2 alpha just outside, as measured.
	GlowPeakAlpha = 0.4
)

// Zone column capacity. The settings package bounds the zone list and the
// label length to the same numbers; the duplication is deliberate (datum 0.7),
// because settings may import nothing but the standard library, and these say
// what the theme has room for while settings says what a file may hold.
const (
	ZoneRowCapacity           = 4
	ZoneLabelColumnCharacters = 3
)
