package render

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"

	"github.com/TheScottBot/Thyme/internal/theme"
	"github.com/TheScottBot/Thyme/third_party/jetbrainsmono"
)

const (
	// MaximumPixelScale bounds the canvas: 200 percent on a 384 DPI display
	// is a pixel scale of 8, beyond any real monitor, and it keeps a bad
	// scale from asking for gigabytes.
	MaximumPixelScale = 8.0

	// MaximumTextCharacters bounds each string for the same reason; the
	// longest the clock model produces is the 14 character date line.
	MaximumTextCharacters = 32

	// MaximumAllocationsPerTick is the measured steady state allocation count
	// of one Render, recorded in TESTING.md. None of them is frame sized; the
	// test proves that separately by bytes.
	MaximumAllocationsPerTick = 0
)

// staticKey is everything the static layer depends on. Comparing it costs
// no allocation, so every tick can check it.
type staticKey struct {
	pixelScale    float64
	timeRunes     int
	dateRunes     int
	zoneTimeRunes [theme.ZoneRowCapacity]int
	zoneCount     int
	labels        [theme.ZoneRowCapacity]string
}

func staticKeyFor(text Text, pixelScale float64) staticKey {
	key := staticKey{
		pixelScale: pixelScale,
		timeRunes:  utf8.RuneCountInString(text.Time),
		dateRunes:  utf8.RuneCountInString(text.DateLine),
		zoneCount:  len(text.Zones),
	}
	for zoneIndex, zone := range text.Zones {
		key.zoneTimeRunes[zoneIndex] = utf8.RuneCountInString(zone.Time)
		key.labels[zoneIndex] = zone.Label
	}
	return key
}

// Renderer draws frames, caching the static layer (glow, panel, outline,
// divider, labels) per layout and reusing every frame sized buffer, so a tick
// only copies the static layer, draws the changing text and converts.
type Renderer struct {
	parsedFont *opentype.Font

	// advancePerEm and capHeightPerEm are read once in font units, so layout
	// scales exactly; a face rounds both to 1/64 pixel at each size, which
	// would make the panel width drift between scale steps.
	advancePerEm   float64
	capHeightPerEm float64

	facesScale float64
	timeFace   font.Face
	dateFace   font.Face
	zoneFace   font.Face

	hasStaticLayer        bool
	currentKey            staticKey
	layout                frameLayout
	staticLayer           *image.RGBA
	workingFrame          *image.RGBA
	outputPixels          []byte
	staticLayerBuildCount int

	// shapeRasterizer is reused for every shape of every static layer build.
	shapeRasterizer *vector.Rasterizer

	timeColour     *image.Uniform
	dateColour     *image.Uniform
	zoneTimeColour *image.Uniform
}

// NewRenderer parses the embedded font once.
func NewRenderer() (*Renderer, error) {
	parsedFont, parseError := opentype.Parse(jetbrainsmono.Regular)
	if parseError != nil {
		return nil, fmt.Errorf("parsing the embedded font: %w", parseError)
	}
	advancePerEm, capHeightPerEm, metricsError := readFontProportions(parsedFont)
	if metricsError != nil {
		return nil, metricsError
	}
	return &Renderer{
		parsedFont:     parsedFont,
		advancePerEm:   advancePerEm,
		capHeightPerEm: capHeightPerEm,
		timeColour:     image.NewUniform(theme.TimeText),
		dateColour:     image.NewUniform(theme.DateText),
		zoneTimeColour: image.NewUniform(theme.ZoneTimeText),
	}, nil
}

func validateRequest(text Text, pixelScale float64) error {
	if math.IsNaN(pixelScale) || pixelScale <= 0 || pixelScale > MaximumPixelScale {
		return fmt.Errorf("pixel scale %v is outside (0, %v]", pixelScale, MaximumPixelScale)
	}
	if len(text.Zones) > theme.ZoneRowCapacity {
		return fmt.Errorf("%d zones, the theme has room for %d", len(text.Zones), theme.ZoneRowCapacity)
	}
	tooLong := func(value string) bool { return utf8.RuneCountInString(value) > MaximumTextCharacters }
	if tooLong(text.Time) || tooLong(text.DateLine) {
		return fmt.Errorf("text longer than %d characters", MaximumTextCharacters)
	}
	for _, zone := range text.Zones {
		if tooLong(zone.Label) || tooLong(zone.Time) {
			return fmt.Errorf("zone text longer than %d characters", MaximumTextCharacters)
		}
	}
	return nil
}

// Render returns the frame for text at pixelScale, which is the scale step
// times the monitor's DPI over 96. The returned pixels are valid until the
// next call.
func (renderer *Renderer) Render(text Text, pixelScale float64) (Frame, error) {
	if requestError := validateRequest(text, pixelScale); requestError != nil {
		return Frame{}, requestError
	}
	key := staticKeyFor(text, pixelScale)
	if !renderer.hasStaticLayer || key != renderer.currentKey {
		if buildError := renderer.rebuildStaticLayer(text, pixelScale, key); buildError != nil {
			return Frame{}, buildError
		}
	}

	copy(renderer.workingFrame.Pix, renderer.staticLayer.Pix)
	renderer.drawString(renderer.timeFace, renderer.timeColour, renderer.layout.timeOrigin, text.Time)
	renderer.drawString(renderer.dateFace, renderer.dateColour, renderer.layout.dateOrigin, text.DateLine)
	for zoneIndex, zone := range text.Zones {
		renderer.drawString(renderer.zoneFace, renderer.zoneTimeColour, renderer.layout.zoneTimeOrigins[zoneIndex], zone.Time)
	}
	if convertError := ConvertRGBAToBGRA(renderer.outputPixels, renderer.workingFrame); convertError != nil {
		return Frame{}, convertError
	}
	return Frame{
		Width:       renderer.layout.canvasWidth,
		Height:      renderer.layout.canvasHeight,
		Pixels:      renderer.outputPixels,
		PanelBounds: renderer.layout.visiblePanelBounds(),
	}, nil
}

func (renderer *Renderer) drawString(face font.Face, colour *image.Uniform, origin floatPoint, value string) {
	drawer := font.Drawer{
		Dst:  renderer.workingFrame,
		Src:  colour,
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.Int26_6(math.Round(origin.X * 64)), Y: fixed.Int26_6(math.Round(origin.Y * 64))},
	}
	drawer.DrawString(value)
}

// ensureFaces builds the three faces at a new pixel scale. Unhinted, so the
// glyphs scale smoothly between steps instead of snapping to a pixel grid
// that differs at every scale. A size in pixels is a size in points at 72 DPI.
func (renderer *Renderer) ensureFaces(pixelScale float64) error {
	if renderer.timeFace != nil && renderer.facesScale == pixelScale {
		return nil
	}
	newFace := func(designSize float64) (font.Face, error) {
		return opentype.NewFace(renderer.parsedFont, &opentype.FaceOptions{Size: designSize * pixelScale, DPI: 72, Hinting: font.HintingNone})
	}
	timeFace, timeError := newFace(theme.TimeFontSize)
	dateFace, dateError := newFace(theme.DateFontSize)
	zoneFace, zoneError := newFace(theme.ZoneFontSize)
	if timeError != nil || dateError != nil || zoneError != nil {
		return fmt.Errorf("creating font faces: %v %v %v", timeError, dateError, zoneError)
	}
	renderer.timeFace, renderer.dateFace, renderer.zoneFace = timeFace, dateFace, zoneFace
	renderer.facesScale = pixelScale
	return nil
}

// readFontProportions measures the font at one pixel per font unit, where
// fixed point rounding cannot lose anything. JetBrains Mono is monospaced, so
// every glyph advances as far as the digit zero.
func readFontProportions(parsedFont *opentype.Font) (float64, float64, error) {
	unitsPerEm := float64(parsedFont.UnitsPerEm())
	var sfntBuffer sfnt.Buffer
	glyphIndex, indexError := parsedFont.GlyphIndex(&sfntBuffer, '0')
	if indexError != nil {
		return 0, 0, fmt.Errorf("finding the digit zero: %w", indexError)
	}
	pixelsPerEm := fixed.I(int(unitsPerEm))
	advance, advanceError := parsedFont.GlyphAdvance(&sfntBuffer, glyphIndex, pixelsPerEm, font.HintingNone)
	if advanceError != nil {
		return 0, 0, fmt.Errorf("measuring the advance: %w", advanceError)
	}
	metrics, metricsError := parsedFont.Metrics(&sfntBuffer, pixelsPerEm, font.HintingNone)
	if metricsError != nil {
		return 0, 0, fmt.Errorf("reading the font metrics: %w", metricsError)
	}
	return float64(advance) / 64 / unitsPerEm, float64(metrics.CapHeight) / 64 / unitsPerEm, nil
}

func (renderer *Renderer) computeLayout(text Text, pixelScale float64) (frameLayout, error) {
	if requestError := validateRequest(text, pixelScale); requestError != nil {
		return frameLayout{}, requestError
	}
	if faceError := renderer.ensureFaces(pixelScale); faceError != nil {
		return frameLayout{}, faceError
	}
	measures := fontMeasures{
		timeAdvance:   renderer.advancePerEm * theme.TimeFontSize * pixelScale,
		dateAdvance:   renderer.advancePerEm * theme.DateFontSize * pixelScale,
		zoneAdvance:   renderer.advancePerEm * theme.ZoneFontSize * pixelScale,
		timeCapHeight: renderer.capHeightPerEm * theme.TimeFontSize * pixelScale,
		zoneCapHeight: renderer.capHeightPerEm * theme.ZoneFontSize * pixelScale,
	}
	return layOut(text, pixelScale, measures), nil
}

func (renderer *Renderer) rebuildStaticLayer(text Text, pixelScale float64, key staticKey) error {
	layout, layoutError := renderer.computeLayout(text, pixelScale)
	if layoutError != nil {
		return layoutError
	}
	bounds := image.Rect(0, 0, layout.canvasWidth, layout.canvasHeight)
	if renderer.staticLayer == nil || renderer.staticLayer.Rect != bounds {
		renderer.staticLayer = image.NewRGBA(bounds)
		renderer.workingFrame = image.NewRGBA(bounds)
		renderer.outputPixels = make([]byte, layout.canvasWidth*layout.canvasHeight*4)
	} else {
		clear(renderer.staticLayer.Pix)
	}

	if renderer.shapeRasterizer == nil {
		renderer.shapeRasterizer = vector.NewRasterizer(layout.canvasWidth, layout.canvasHeight)
	}
	drawGlow(renderer.shapeRasterizer, renderer.staticLayer, layout)
	fillRoundedRectangle(renderer.shapeRasterizer, renderer.staticLayer, layout.panel, layout.panelCornerRadius, 0, theme.PanelFill)
	drawOutline(renderer.shapeRasterizer, renderer.staticLayer, layout)
	if layout.hasDivider {
		fillRoundedRectangle(renderer.shapeRasterizer, renderer.staticLayer, layout.divider, 0, 0, theme.Divider)
	}
	labelColour := image.NewUniform(theme.ZoneLabelText)
	for zoneIndex, zone := range text.Zones {
		origin := layout.zoneLabelOrigins[zoneIndex]
		drawer := font.Drawer{
			Dst:  renderer.staticLayer,
			Src:  labelColour,
			Face: renderer.zoneFace,
			Dot:  fixed.Point26_6{X: fixed.Int26_6(math.Round(origin.X * 64)), Y: fixed.Int26_6(math.Round(origin.Y * 64))},
		}
		drawer.DrawString(zone.Label)
	}

	renderer.layout = layout
	renderer.currentKey = key
	renderer.hasStaticLayer = true
	renderer.staticLayerBuildCount++
	return nil
}

// addRoundedRectanglePath adds a closed rounded rectangle grown by outset
// pixels. reverse winds it the other way, which subtracts it from a shape
// wound forwards: that is how the outline becomes a ring.
func addRoundedRectanglePath(rasterizer *vector.Rasterizer, rectangle floatRect, radius float64, outset float64, reverse bool) {
	left, top := float32(rectangle.Min.X-outset), float32(rectangle.Min.Y-outset)
	right, bottom := float32(rectangle.Max.X+outset), float32(rectangle.Max.Y+outset)
	cornerRadius := float32(math.Max(radius+outset, 0))
	// The cubic Bezier control distance that best approximates a quarter
	// circle.
	control := cornerRadius * 0.5523
	type corner struct{ fromX, fromY, control1X, control1Y, control2X, control2Y, toX, toY, lineToX, lineToY float32 }
	forwardCorners := []corner{
		{right - cornerRadius, top, right - cornerRadius + control, top, right, top + cornerRadius - control, right, top + cornerRadius, right, bottom - cornerRadius},
		{right, bottom - cornerRadius, right, bottom - cornerRadius + control, right - cornerRadius + control, bottom, right - cornerRadius, bottom, left + cornerRadius, bottom},
		{left + cornerRadius, bottom, left + cornerRadius - control, bottom, left, bottom - cornerRadius + control, left, bottom - cornerRadius, left, top + cornerRadius},
		{left, top + cornerRadius, left, top + cornerRadius - control, left + cornerRadius - control, top, left + cornerRadius, top, right - cornerRadius, top},
	}
	if !reverse {
		rasterizer.MoveTo(left+cornerRadius, top)
		rasterizer.LineTo(right-cornerRadius, top)
		for _, nextCorner := range forwardCorners {
			rasterizer.CubeTo(nextCorner.control1X, nextCorner.control1Y, nextCorner.control2X, nextCorner.control2Y, nextCorner.toX, nextCorner.toY)
			rasterizer.LineTo(nextCorner.lineToX, nextCorner.lineToY)
		}
		rasterizer.ClosePath()
		return
	}
	rasterizer.MoveTo(right-cornerRadius, top)
	rasterizer.LineTo(left+cornerRadius, top)
	for cornerIndex := len(forwardCorners) - 1; cornerIndex >= 0; cornerIndex-- {
		previousCorner := forwardCorners[cornerIndex]
		rasterizer.CubeTo(previousCorner.control2X, previousCorner.control2Y, previousCorner.control1X, previousCorner.control1Y, previousCorner.fromX, previousCorner.fromY)
		if cornerIndex > 0 {
			rasterizer.LineTo(forwardCorners[cornerIndex-1].toX, forwardCorners[cornerIndex-1].toY)
		}
	}
	rasterizer.ClosePath()
}

// Each shape resets one shared rasterizer rather than making its own: a
// rasterizer holds a frame sized accumulator, and a fresh one per shape left
// several frames' worth of memory with the process after every rebuild
// (measured in WE8).
func fillRoundedRectangle(rasterizer *vector.Rasterizer, destination *image.RGBA, rectangle floatRect, radius float64, outset float64, colour color.RGBA) {
	bounds := destination.Bounds()
	rasterizer.Reset(bounds.Dx(), bounds.Dy())
	addRoundedRectanglePath(rasterizer, rectangle, radius, outset, false)
	rasterizer.Draw(destination, bounds, image.NewUniform(colour), image.Point{})
}

func drawOutline(rasterizer *vector.Rasterizer, destination *image.RGBA, layout frameLayout) {
	bounds := destination.Bounds()
	halfWidth := theme.OutlineWidth * layout.pixelScale / 2
	rasterizer.Reset(bounds.Dx(), bounds.Dy())
	addRoundedRectanglePath(rasterizer, layout.panel, layout.panelCornerRadius, halfWidth, false)
	addRoundedRectanglePath(rasterizer, layout.panel, layout.panelCornerRadius, -halfWidth, true)
	rasterizer.Draw(destination, bounds, image.NewUniform(theme.Outline), image.Point{})
}

// drawGlow blurs the panel's own shape and paints it in the glow colour
// beneath the panel. The radius is rounded down so the blur never spreads
// further than theme.GlowReach, which the transparency test relies on.
func drawGlow(rasterizer *vector.Rasterizer, destination *image.RGBA, layout frameLayout) {
	bounds := destination.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	shapeMask := image.NewAlpha(bounds)
	rasterizer.Reset(width, height)
	addRoundedRectanglePath(rasterizer, layout.panel, layout.panelCornerRadius, 0, false)
	rasterizer.Draw(shapeMask, bounds, image.Opaque, image.Point{})

	coverage := make([]float32, width*height)
	for pixelIndex, alpha := range shapeMask.Pix {
		coverage[pixelIndex] = float32(alpha) / 255
	}
	boxRadius := max(1, int(math.Floor(theme.GlowBoxRadius*layout.pixelScale)))
	scratch := make([]float32, max(width, height))
	for range theme.GlowBoxPasses {
		boxBlurRows(coverage, width, height, boxRadius, scratch)
		boxBlurColumns(coverage, width, height, boxRadius, scratch)
	}

	for pixelIndex, blurred := range coverage {
		glowAlpha := float64(blurred) * theme.GlowPeakAlpha
		if glowAlpha <= 0 {
			continue
		}
		pixelOffset := pixelIndex * 4
		destination.Pix[pixelOffset+0] = uint8(math.Round(float64(theme.Glow.R) * glowAlpha))
		destination.Pix[pixelOffset+1] = uint8(math.Round(float64(theme.Glow.G) * glowAlpha))
		destination.Pix[pixelOffset+2] = uint8(math.Round(float64(theme.Glow.B) * glowAlpha))
		destination.Pix[pixelOffset+3] = uint8(math.Round(255 * glowAlpha))
	}
}

// boxBlurRows replaces each value with the mean of the 2r+1 values around it
// in its row, treating everything beyond the edge as zero, with a running
// sum so the cost does not grow with the radius.
func boxBlurRows(values []float32, width int, height int, radius int, scratch []float32) {
	windowSize := float32(2*radius + 1)
	for row := 0; row < height; row++ {
		rowValues := values[row*width : (row+1)*width]
		copy(scratch[:width], rowValues)
		var runningSum float32
		for column := 0; column < min(radius, width); column++ {
			runningSum += scratch[column]
		}
		for column := 0; column < width; column++ {
			if entering := column + radius; entering < width {
				runningSum += scratch[entering]
			}
			if leaving := column - radius - 1; leaving >= 0 {
				runningSum -= scratch[leaving]
			}
			rowValues[column] = max(runningSum, 0) / windowSize
		}
	}
}

func boxBlurColumns(values []float32, width int, height int, radius int, scratch []float32) {
	windowSize := float32(2*radius + 1)
	for column := 0; column < width; column++ {
		for row := 0; row < height; row++ {
			scratch[row] = values[row*width+column]
		}
		var runningSum float32
		for row := 0; row < min(radius, height); row++ {
			runningSum += scratch[row]
		}
		for row := 0; row < height; row++ {
			if entering := row + radius; entering < height {
				runningSum += scratch[entering]
			}
			if leaving := row - radius - 1; leaving >= 0 {
				runningSum -= scratch[leaving]
			}
			values[row*width+column] = max(runningSum, 0) / windowSize
		}
	}
}
