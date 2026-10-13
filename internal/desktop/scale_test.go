package desktop

import (
	"testing"

	"github.com/TheScottBot/Thyme/internal/settings"
)

func TestBuildMenu_ScaleStepsAreEnabledAndTheCurrentOneTicked(t *testing.T) {
	current := settings.Defaults()
	current.ScalePercent = 150

	scaleEntries := BuildMenu(MenuState{Settings: current})[1].Children

	expectedLabels := []string{"75%", "100%", "125%", "150%", "200%"}
	if len(scaleEntries) != len(expectedLabels) {
		t.Fatalf("expected %d steps, got %d", len(expectedLabels), len(scaleEntries))
	}
	for stepIndex, entry := range scaleEntries {
		if entry.Label != expectedLabels[stepIndex] || !entry.Enabled {
			t.Errorf("step %d: expected enabled %q, got %+v", stepIndex, expectedLabels[stepIndex], entry)
		}
		if entry.Checked != (expectedLabels[stepIndex] == "150%") {
			t.Errorf("step %d: expected only 150%% ticked, got %+v", stepIndex, entry)
		}
	}
}

// Every scale entry's command maps back to its step, and nothing else maps to
// a step, so a menu command can never set a scale the settings would refuse.
func TestScalePercentForCommand_MapsEachStepAndNothingElse(t *testing.T) {
	scaleEntries := BuildMenu(MenuState{Settings: settings.Defaults()})[1].Children
	for stepIndex, entry := range scaleEntries {
		scalePercent, isScale := ScalePercentForCommand(entry.Command)
		if !isScale || scalePercent != settings.ScaleStepsPercent()[stepIndex] {
			t.Errorf("command %d: expected step %d, got %d (%v)", entry.Command, settings.ScaleStepsPercent()[stepIndex], scalePercent, isScale)
		}
	}
	for _, other := range []MenuCommand{0, CommandSeconds, CommandExit, commandScaleFirst - 1, commandScaleFirst + MenuCommand(len(scaleEntries))} {
		if _, isScale := ScalePercentForCommand(other); isScale {
			t.Errorf("command %d should not be a scale step", other)
		}
	}
}
