package desktop

import (
	"testing"

	"github.com/TheScottBot/Thyme/internal/settings"
)

func topLevelSummary(entries []MenuEntry) []string {
	summary := make([]string, 0, len(entries))
	for _, entry := range entries {
		switch entry.Kind {
		case MenuSeparator:
			summary = append(summary, "-")
		default:
			summary = append(summary, entry.Label)
		}
	}
	return summary
}

func findEntry(t *testing.T, entries []MenuEntry, command MenuCommand) MenuEntry {
	t.Helper()
	for _, entry := range entries {
		if entry.Kind == MenuItem && entry.Command == command {
			return entry
		}
	}
	t.Fatalf("no entry for command %d", command)
	return MenuEntry{}
}

func TestBuildMenu_HasTheSpecifiedItemsInOrder(t *testing.T) {
	entries := BuildMenu(MenuState{Settings: settings.Defaults()})

	expected := []string{"&Seconds", "S&cale", "-", "Always on &top", "Start with &Windows", "Show in t&ray", "-", "E&xit"}
	summary := topLevelSummary(entries)
	if len(summary) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, summary)
	}
	for entryIndex := range expected {
		if summary[entryIndex] != expected[entryIndex] {
			t.Errorf("entry %d: expected %q, got %q", entryIndex, expected[entryIndex], summary[entryIndex])
		}
	}
}

func TestBuildMenu_ChecksFollowTheSettingsAndTheRegistry(t *testing.T) {
	testCases := []struct {
		name  string
		state MenuState
	}{
		{name: "defaults, start with Windows off", state: MenuState{Settings: settings.Defaults(), StartWithWindows: false}},
		{name: "everything flipped", state: func() MenuState {
			flipped := settings.Defaults()
			flipped.ShowSeconds, flipped.AlwaysOnTop, flipped.ShowInTray = false, true, false
			return MenuState{Settings: flipped, StartWithWindows: true}
		}()},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			entries := BuildMenu(testCase.state)
			checks := map[MenuCommand]bool{
				CommandSeconds:          testCase.state.Settings.ShowSeconds,
				CommandAlwaysOnTop:      testCase.state.Settings.AlwaysOnTop,
				CommandStartWithWindows: testCase.state.StartWithWindows,
				CommandShowInTray:       testCase.state.Settings.ShowInTray,
				CommandExit:             false,
			}
			for command, expectedChecked := range checks {
				entry := findEntry(t, entries, command)
				if entry.Checked != expectedChecked {
					t.Errorf("command %d: expected checked %v, got %v", command, expectedChecked, entry.Checked)
				}
				if !entry.Enabled {
					t.Errorf("command %d: expected enabled", command)
				}
			}
		})
	}
}

// A startup state that could not be read is shown greyed, so the menu never
// claims a state it does not know.
func TestBuildMenu_GreysStartWithWindowsWhenItsStateIsUnknown(t *testing.T) {
	entries := BuildMenu(MenuState{Settings: settings.Defaults(), StartWithWindows: true, StartWithWindowsUnknown: true})

	entry := findEntry(t, entries, CommandStartWithWindows)
	if entry.Enabled || entry.Checked {
		t.Errorf("expected greyed and unchecked, got %+v", entry)
	}
}

func TestBuildMenu_ScaleIsASubmenu(t *testing.T) {
	entries := BuildMenu(MenuState{Settings: settings.Defaults()})

	if entries[1].Kind != MenuSubmenu || len(entries[1].Children) == 0 {
		t.Errorf("expected Scale to be a submenu with entries, got %+v", entries[1])
	}
}
