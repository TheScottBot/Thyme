package eventlog

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Specification Part 5: no path beyond the folder name is logged. Go's file
// errors carry the whole path, which holds the user's name, so only the
// operating system's own reason may reach the log.
func TestReasonWithoutPath_DropsPathsFromFileErrors(t *testing.T) {
	privatePath := filepath.Join("C:", "Users", "someone.private", "AppData", "Local", "Thyme", "settings.json")
	testCases := []struct {
		name         string
		failure      error
		expectedText string
	}{
		{name: "a path error", failure: &fs.PathError{Op: "open", Path: privatePath, Err: fs.ErrPermission}, expectedText: "open: permission denied"},
		{name: "a link error", failure: &os.LinkError{Op: "rename", Old: privatePath + ".1.tmp", New: privatePath, Err: fs.ErrExist}, expectedText: "rename: file already exists"},
		{name: "a wrapped path error", failure: fmt.Errorf("replacing the settings file: %w", &os.LinkError{Op: "rename", Old: privatePath, New: privatePath, Err: fs.ErrPermission}), expectedText: "replacing the settings file: rename: permission denied"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reason := ReasonWithoutPath(testCase.failure)

			if strings.Contains(reason, "someone.private") || strings.Contains(reason, "settings.json") {
				t.Errorf("reason %q still carries the path", reason)
			}
			if reason != testCase.expectedText {
				t.Errorf("expected %q, got %q", testCase.expectedText, reason)
			}
		})
	}
}

func TestReasonWithoutPath_KeepsOtherErrorsWhole(t *testing.T) {
	if reason := ReasonWithoutPath(errors.New("pixel scale 9 is outside (0, 8]")); reason != "pixel scale 9 is outside (0, 8]" {
		t.Errorf("expected the message unchanged, got %q", reason)
	}
}
