package settings

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// everyFieldChanged differs from Defaults in every field, so a round trip that
// silently dropped a field back to its default would be caught.
func everyFieldChanged() Settings {
	return Settings{
		Position:     Position{X: -1920, Y: 1080},
		ScalePercent: 150,
		AlwaysOnTop:  true,
		ShowInTray:   false,
		ShowSeconds:  false,
		Zones: []ZoneSetting{
			{Label: "NYC", TimeZone: "America/New_York"},
			{Label: "T", TimeZone: "Asia/Tokyo"},
		},
	}
}

func writeSettingsFile(t *testing.T, fileContents string) string {
	t.Helper()
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	if writeError := os.WriteFile(settingsPath, []byte(fileContents), 0o644); writeError != nil {
		t.Fatalf("writing fixture settings file: %v", writeError)
	}
	return settingsPath
}

func hasRejection(report LoadReport, field string, reason RejectionReason) bool {
	return slices.Contains(report.Rejections, Rejection{Field: field, Reason: reason})
}

// validDocumentWith is a complete, valid version 1 document with one top level
// member replaced, so each rejection test changes exactly one thing.
func validDocumentWith(t *testing.T, fieldName string, replacementMemberText string) string {
	t.Helper()
	members := map[string]string{
		"version":      "1",
		"position":     `{"x": 10, "y": 20}`,
		"scalePercent": "125",
		"alwaysOnTop":  "true",
		"showInTray":   "true",
		"showSeconds":  "false",
		"zones":        `[{"label": "LDN", "timeZone": "Europe/London"}]`,
	}
	if _, known := members[fieldName]; !known {
		t.Fatalf("fixture error: '%s' is not a field of the valid document", fieldName)
	}
	members[fieldName] = replacementMemberText
	return assembleDocument(members)
}

func assembleDocument(members map[string]string) string {
	document := "{"
	firstMember := true
	for _, fieldName := range slices.Sorted(maps.Keys(members)) {
		if !firstMember {
			document += ","
		}
		document += `"` + fieldName + `":` + members[fieldName]
		firstMember = false
	}
	return document + "}"
}
