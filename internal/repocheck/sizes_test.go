package repocheck

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFindOversizedFiles_FlagsAPlantedFileOneByteOverTheBudget(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"release/thyme_amd64.exe": strings.Repeat("x", 101),
		"release/thyme_arm64.exe": strings.Repeat("x", 100),
	})
	paths := []string{
		filepath.Join(fixtureRoot, "release", "thyme_amd64.exe"),
		filepath.Join(fixtureRoot, "release", "thyme_arm64.exe"),
	}

	oversized, findError := FindOversizedFiles(paths, 100)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(oversized) != 1 || !strings.HasSuffix(oversized[0].Path, "thyme_amd64.exe") || oversized[0].SizeBytes != 101 {
		t.Errorf("expected only the planted file, 101 bytes, got %+v", oversized)
	}
}

func TestFindOversizedFiles_RefusesAMissingFileRatherThanPassingIt(t *testing.T) {
	if _, findError := FindOversizedFiles([]string{filepath.Join(t.TempDir(), "missing.exe")}, 100); findError == nil {
		t.Error("expected a missing release file to be an error, since a check of nothing passes")
	}
}

func TestFindOversizedFiles_RefusesAnEmptyList(t *testing.T) {
	if _, findError := FindOversizedFiles(nil, 100); findError == nil {
		t.Error("expected an empty list to be an error, since a check of nothing passes")
	}
}
