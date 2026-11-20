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
	testCases := []struct {
		name     string
		labels   MenuLabels
		expected []string
	}{
		{name: "Windows", labels: windowsMenuLabels, expected: []string{"&Seconds", "S&cale", "-", "Always on &top", "Start with &Windows", "Show in t&ray", "-", "E&xit"}},
		{name: "macOS", labels: macMenuLabels, expected: []string{"Seconds", "Scale", "-", "Always on Top", "Open at Login", "Show in Menu Bar", "-", "Quit Thyme"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			summary := topLevelSummary(BuildMenu(MenuState{Labels: testCase.labels, Settings: settings.Defaults()}))
			if len(summary) != len(testCase.expected) {
				t.Fatalf("expected %v, got %v", testCase.expected, summary)
			}
			for entryIndex := range testCase.expected {
				if summary[entryIndex] != testCase.expected[entryIndex] {
					t.Errorf("entry %d: expected %q, got %q", entryIndex, testCase.expected[entryIndex], summary[entryIndex])
				}
			}
		})
	}
}

func TestBuildMenu_ChecksFollowTheSettingsAndTheRegistry(t *testing.T) {
	testCases := []struct {
		name  string
		state MenuState
	}{
		{name: "defaults, start with Windows off", state: MenuState{Settings: settings.Defaults(), StartAtLogin: false}},
		{name: "everything flipped", state: func() MenuState {
			flipped := settings.Defaults()
			flipped.ShowSeconds, flipped.AlwaysOnTop, flipped.ShowInTray = false, true, false
			return MenuState{Settings: flipped, StartAtLogin: true}
		}()},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			entries := BuildMenu(testCase.state)
			checks := map[MenuCommand]bool{
				CommandSeconds:      testCase.state.Settings.ShowSeconds,
				CommandAlwaysOnTop:  testCase.state.Settings.AlwaysOnTop,
				CommandStartAtLogin: testCase.state.StartAtLogin,
				CommandShowInTray:   testCase.state.Settings.ShowInTray,
				CommandExit:         false,
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
func TestBuildMenu_GreysStartAtLoginWhenItsStateIsUnknown(t *testing.T) {
	entries := BuildMenu(MenuState{Settings: settings.Defaults(), StartAtLogin: true, StartAtLoginUnknown: true})

	entry := findEntry(t, entries, CommandStartAtLogin)
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
