package render

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// updateGoldenImages rewrites the golden images from the current renderer:
//
//	go test ./internal/render -run Golden -update
//
// A rewritten golden image is a visual change, accepted only by the author
// looking at it; the WE4 gate is the 100 percent mock-up image.
var updateGoldenImages = flag.Bool("update", false, "rewrite golden images")

// goldenChannelTolerance is how far a channel may differ from its golden
// image. Measured in WE4: glyph rasterisation differs between arm64 and amd64
// by at most 2 of 255 in a handful of bytes; shapes are byte identical.
const goldenChannelTolerance = 2

// mockUpText is what docs/theme.png shows.
func mockUpText() Text {
	return Text{
		Time:     "21:04:01",
		DateLine: "sun 2026-09-06",
		Zones: []ZoneLine{
			{Label: "UTC", Time: "20:04"},
			{Label: "LDN", Time: "21:04"},
			{Label: "BER", Time: "22:04"},
			{Label: "SAO", Time: "17:04"},
		},
	}
}

func textWith(showSeconds bool, zoneCount int) Text {
	text := mockUpText()
	if !showSeconds {
		text.Time = "21:04"
	}
	text.Zones = text.Zones[:zoneCount]
	return text
}

func newRendererOrFail(t testing.TB) *Renderer {
	t.Helper()
	renderer, rendererError := NewRenderer()
	if rendererError != nil {
		t.Fatalf("creating renderer: %v", rendererError)
	}
	return renderer
}

func renderOrFail(t testing.TB, renderer *Renderer, text Text, pixelScale float64) Frame {
	t.Helper()
	frame, renderError := renderer.Render(text, pixelScale)
	if renderError != nil {
		t.Fatalf("rendering: %v", renderError)
	}
	return frame
}

// frameToRGBA turns the BGRA output back into an image for comparison and
// for writing golden images, so the golden images check the bytes the shell
// will actually blit.
func frameToRGBA(frame Frame) *image.RGBA {
	picture := image.NewRGBA(image.Rect(0, 0, frame.Width, frame.Height))
	for pixelOffset := 0; pixelOffset < len(frame.Pixels); pixelOffset += 4 {
		picture.Pix[pixelOffset+0] = frame.Pixels[pixelOffset+2]
		picture.Pix[pixelOffset+1] = frame.Pixels[pixelOffset+1]
		picture.Pix[pixelOffset+2] = frame.Pixels[pixelOffset+0]
		picture.Pix[pixelOffset+3] = frame.Pixels[pixelOffset+3]
	}
	return picture
}

func alphaAt(frame Frame, x int, y int) uint8 {
	return frame.Pixels[(y*frame.Width+x)*4+3]
}

func goldenPath(goldenName string) string {
	return filepath.Join("testdata", "golden", goldenName+".png")
}

// compareWithGolden compares in premultiplied space, where the per channel
// tolerance means the same thing at every alpha.
func compareWithGolden(t *testing.T, goldenName string, frame Frame) {
	t.Helper()
	rendered := frameToRGBA(frame)
	if *updateGoldenImages {
		writePNGOrFail(t, goldenPath(goldenName), rendered)
	}

	goldenFile, openError := os.Open(goldenPath(goldenName))
	if openError != nil {
		t.Fatalf("opening golden image (run with -update to create it, then look at it): %v", openError)
	}
	defer goldenFile.Close()
	decoded, decodeError := png.Decode(goldenFile)
	if decodeError != nil {
		t.Fatalf("decoding golden image: %v", decodeError)
	}
	golden := image.NewRGBA(decoded.Bounds())
	draw.Draw(golden, golden.Bounds(), decoded, decoded.Bounds().Min, draw.Src)

	if golden.Bounds() != rendered.Bounds() {
		t.Fatalf("golden image is %v, rendered %v", golden.Bounds(), rendered.Bounds())
	}
	differingBytes, largestDifference := 0, 0
	for byteIndex := range rendered.Pix {
		difference := int(rendered.Pix[byteIndex]) - int(golden.Pix[byteIndex])
		if difference < 0 {
			difference = -difference
		}
		if difference > goldenChannelTolerance {
			differingBytes++
		}
		largestDifference = max(largestDifference, difference)
	}
	if differingBytes > 0 {
		failurePath := filepath.Join(t.TempDir(), goldenName+".actual.png")
		writePNGOrFail(t, failurePath, rendered)
		t.Errorf("%d bytes differ from the golden image by more than %d (largest %d); rendered image written to %s",
			differingBytes, goldenChannelTolerance, largestDifference, failurePath)
	}
}

func writePNGOrFail(t *testing.T, path string, picture image.Image) {
	t.Helper()
	if mkdirError := os.MkdirAll(filepath.Dir(path), 0o755); mkdirError != nil {
		t.Fatalf("creating folder for '%s': %v", path, mkdirError)
	}
	outputFile, createError := os.Create(path)
	if createError != nil {
		t.Fatalf("creating '%s': %v", path, createError)
	}
	encodeError := png.Encode(outputFile, picture)
	closeError := outputFile.Close()
	if encodeError != nil || closeError != nil {
		t.Fatalf("writing '%s': %v", path, fmt.Sprint(encodeError, closeError))
	}
}
