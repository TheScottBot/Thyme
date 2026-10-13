package render

import (
	"math"
	"unicode/utf8"

	"github.com/TheScottBot/Thyme/internal/theme"
)

type floatPoint struct {
	X float64
	Y float64
}

type floatRect struct {
	Min floatPoint
	Max floatPoint
}

func (rectangle floatRect) Dx() float64 { return rectangle.Max.X - rectangle.Min.X }
func (rectangle floatRect) Dy() float64 { return rectangle.Max.Y - rectangle.Min.Y }

// frameLayout is where everything goes for one text shape at one pixel
// scale, in pixels of the canvas. Text positions are pen origins on the
// baseline.
type frameLayout struct {
	canvasWidth  int
	canvasHeight int
	pixelScale   float64

	panel             floatRect
	panelCornerRadius float64

	timeOrigin floatPoint
	dateOrigin floatPoint

	hasDivider bool
	divider    floatRect

	zoneLabelOrigins []floatPoint
	zoneTimeOrigins  []floatPoint
}

// fontMeasures are the per scale font numbers layout needs, read from the
// faces so layout follows the font rather than restating its metrics.
type fontMeasures struct {
	timeAdvance, dateAdvance, zoneAdvance float64
	timeCapHeight, zoneCapHeight          float64
}

// canvasMarginPixels leaves the glow's full reach plus a ring that is always
// fully transparent, so the window's edge never clips the glow and the
// transparent surround is provably there.
func canvasMarginPixels(pixelScale float64) float64 {
	return math.Ceil(theme.GlowReach*pixelScale) + 3
}

func layOut(text Text, pixelScale float64, measures fontMeasures) frameLayout {
	scaled := func(designUnits float64) float64 { return designUnits * pixelScale }
	margin := canvasMarginPixels(pixelScale)

	timeWidth := float64(utf8.RuneCountInString(text.Time)) * measures.timeAdvance
	dateWidth := float64(utf8.RuneCountInString(text.DateLine)) * measures.dateAdvance
	timeColumnWidth := math.Max(timeWidth, dateWidth)

	layout := frameLayout{pixelScale: pixelScale, panelCornerRadius: scaled(theme.PanelCornerRadius)}
	panelLeft, panelTop := margin, margin
	contentLeft := panelLeft + scaled(theme.PaddingLeft)
	panelRight := contentLeft + timeColumnWidth + scaled(theme.PaddingRight)

	leftBlockHeight := measures.timeCapHeight + scaled(theme.TimeBaselineToDateBaseline)
	zoneBlockHeight := 0.0
	if len(text.Zones) > 0 {
		zoneBlockHeight = measures.zoneCapHeight + float64(len(text.Zones)-1)*scaled(theme.ZoneRowPitch)
	}
	contentHeight := math.Max(leftBlockHeight, zoneBlockHeight)
	contentTop := panelTop + scaled(theme.PaddingVertical)
	panelBottom := contentTop + contentHeight + scaled(theme.PaddingVertical)

	timeBaseline := contentTop + (contentHeight-leftBlockHeight)/2 + measures.timeCapHeight
	layout.timeOrigin = floatPoint{X: contentLeft, Y: timeBaseline}
	layout.dateOrigin = floatPoint{X: contentLeft, Y: timeBaseline + scaled(theme.TimeBaselineToDateBaseline)}

	if len(text.Zones) > 0 {
		layout.hasDivider = true
		// The divider is snapped to whole pixels so a one unit rule stays one
		// crisp line rather than two half lit ones.
		// Only the drawn rule is snapped; positions after it use the exact
		// value so the layout stays proportional to the scale.
		exactDividerLeft := contentLeft + timeColumnWidth + scaled(theme.TimeColumnToDivider)
		dividerLeft := math.Round(exactDividerLeft)
		dividerRight := dividerLeft + math.Max(1, math.Round(scaled(theme.DividerWidth)))
		layout.divider = floatRect{
			Min: floatPoint{X: dividerLeft, Y: panelTop + scaled(theme.DividerInset)},
			Max: floatPoint{X: dividerRight, Y: panelBottom - scaled(theme.DividerInset)},
		}
		labelLeft := exactDividerLeft + scaled(theme.DividerWidth) + scaled(theme.DividerToZones)
		zoneTimeLeft := labelLeft + theme.ZoneLabelColumnCharacters*measures.zoneAdvance + scaled(theme.ZoneLabelToZoneTime)
		zoneTimeWidth := 0.0
		firstZoneBaseline := contentTop + (contentHeight-zoneBlockHeight)/2 + measures.zoneCapHeight
		for zoneIndex, zone := range text.Zones {
			baseline := firstZoneBaseline + float64(zoneIndex)*scaled(theme.ZoneRowPitch)
			layout.zoneLabelOrigins = append(layout.zoneLabelOrigins, floatPoint{X: labelLeft, Y: baseline})
			layout.zoneTimeOrigins = append(layout.zoneTimeOrigins, floatPoint{X: zoneTimeLeft, Y: baseline})
			zoneTimeWidth = math.Max(zoneTimeWidth, float64(utf8.RuneCountInString(zone.Time))*measures.zoneAdvance)
		}
		panelRight = zoneTimeLeft + zoneTimeWidth + scaled(theme.PaddingRight)
	}

	layout.panel = floatRect{Min: floatPoint{X: panelLeft, Y: panelTop}, Max: floatPoint{X: panelRight, Y: panelBottom}}
	layout.canvasWidth = int(math.Ceil(panelRight + margin))
	layout.canvasHeight = int(math.Ceil(panelBottom + margin))
	return layout
}

// signedDistanceToPanel is negative inside the rounded panel, positive
// outside, in pixels: the usual rounded rectangle distance function.
func (layout frameLayout) signedDistanceToPanel(x float64, y float64) float64 {
	halfWidth, halfHeight := layout.panel.Dx()/2, layout.panel.Dy()/2
	centreX, centreY := layout.panel.Min.X+halfWidth, layout.panel.Min.Y+halfHeight
	radius := layout.panelCornerRadius
	overX := math.Abs(x-centreX) - halfWidth + radius
	overY := math.Abs(y-centreY) - halfHeight + radius
	outside := math.Hypot(math.Max(overX, 0), math.Max(overY, 0))
	inside := math.Min(math.Max(overX, overY), 0)
	return outside + inside - radius
}

func (layout frameLayout) isInsidePanelBy(x float64, y float64, inset float64) bool {
	return layout.signedDistanceToPanel(x, y) <= -inset
}

func (layout frameLayout) distanceOutsidePanel(x float64, y float64) float64 {
	return math.Max(layout.signedDistanceToPanel(x, y), 0)
}
