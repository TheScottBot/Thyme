package desktop

import (
	"testing"

	"github.com/TheScottBot/Thyme/internal/settings"
)

// The paths are Windows paths written as literals, and the function treats
// them as such on every host, so this runs on the Linux runner too.
func TestIsInsideFolder_RecognisesAnExeUnderTheTemporaryFolder(t *testing.T) {
	temporaryFolder := `C:\Users\someone\AppData\Local\Temp`
	testCases := []struct {
		name     string
		exePath  string
		folder   string
		isInside bool
	}{
		{name: "where go run builds", exePath: temporaryFolder + `\go-build123456\b001\exe\thyme.exe`, folder: temporaryFolder, isInside: true},
		{name: "a zip opened in Explorer", exePath: temporaryFolder + `\Temp1_thyme.zip\thyme.exe`, folder: temporaryFolder, isInside: true},
		{name: "different capitalisation", exePath: `c:\users\SOMEONE\appdata\local\temp\x\thyme.exe`, folder: temporaryFolder, isInside: true},
		{name: "forward slashes", exePath: `C:/Users/someone/AppData/Local/Temp/x/thyme.exe`, folder: temporaryFolder, isInside: true},
		{name: "the folder given with a trailing separator", exePath: temporaryFolder + `\x\thyme.exe`, folder: temporaryFolder + `\`, isInside: true},
		{name: "the recommended install folder", exePath: `C:\Users\someone\AppData\Local\Thyme\thyme.exe`, folder: temporaryFolder, isInside: false},
		{name: "a sibling whose name starts the same", exePath: `C:\Users\someone\AppData\Local\Temporary\thyme.exe`, folder: temporaryFolder, isInside: false},
		{name: "a path that climbs out of the folder", exePath: temporaryFolder + `\..\Thyme\thyme.exe`, folder: temporaryFolder, isInside: false},
		{name: "an empty folder is never everything", exePath: `C:\Thyme\thyme.exe`, folder: "", isInside: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if isInside := IsInsideFolder(testCase.exePath, testCase.folder); isInside != testCase.isInside {
				t.Errorf("expected %v for %s in %s", testCase.isInside, testCase.exePath, testCase.folder)
			}
		})
	}
}

// Start with Windows cannot be turned on from a temporary exe, but an entry
// that already exists can still be turned off, so the guard never traps one.
func TestBuildMenu_StartWithWindowsFromATemporaryExe(t *testing.T) {
	testCases := []struct {
		name            string
		alreadyEnabled  bool
		expectedEnabled bool
	}{
		{name: "off: greyed, so it cannot be turned on", alreadyEnabled: false, expectedEnabled: false},
		{name: "on: still available, so it can be turned off", alreadyEnabled: true, expectedEnabled: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			entries := BuildMenu(MenuState{Settings: settings.Defaults(), StartWithWindows: testCase.alreadyEnabled, StartWithWindowsFromTemporaryExe: true})

			entry := findEntry(t, entries, CommandStartWithWindows)
			if entry.Enabled != testCase.expectedEnabled || entry.Checked != testCase.alreadyEnabled {
				t.Errorf("expected enabled %v and checked %v, got %+v", testCase.expectedEnabled, testCase.alreadyEnabled, entry)
			}
		})
	}
}
