package render

import (
	"fmt"
	"testing"
)

// WE7: a golden image at every scale step at 96 DPI. The 100 and 200 percent
// steps are WE4's mock-up images, compared again rather than duplicated.
func TestRender_GoldenMockUpAtEveryScaleStep(t *testing.T) {
	goldenNameByStep := map[int]string{
		75:  "mockup_075",
		100: "mockup_100",
		125: "mockup_125",
		150: "mockup_150",
		200: "mockup_200",
	}
	for scalePercent, goldenName := range goldenNameByStep {
		t.Run(fmt.Sprintf("%d percent", scalePercent), func(t *testing.T) {
			compareWithGolden(t, goldenName, renderOrFail(t, newRendererOrFail(t), mockUpText(), float64(scalePercent)/100))
		})
	}
}
