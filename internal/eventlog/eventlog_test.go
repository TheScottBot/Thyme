package eventlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readOrEmpty(t *testing.T, path string) string {
	t.Helper()
	contents, readError := os.ReadFile(path)
	if os.IsNotExist(readError) {
		return ""
	}
	if readError != nil {
		t.Fatalf("reading '%s': %v", path, readError)
	}
	return string(contents)
}

func openOrFail(t *testing.T, path string, maximumBytes int64) *RotatingFile {
	t.Helper()
	rotatingFile, openError := OpenRotating(path, maximumBytes)
	if openError != nil {
		t.Fatalf("opening: %v", openError)
	}
	t.Cleanup(func() { rotatingFile.Close() })
	return rotatingFile
}

func writeOrFail(t *testing.T, rotatingFile *RotatingFile, record string) {
	t.Helper()
	if _, writeError := rotatingFile.Write([]byte(record)); writeError != nil {
		t.Fatalf("writing: %v", writeError)
	}
}

func TestRotatingFile_KeepsRecordsUnderTheCapInOneFile(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "Thyme", "thyme.log")
	rotatingFile := openOrFail(t, logPath, 100)

	writeOrFail(t, rotatingFile, strings.Repeat("a", 40)+"\n")
	writeOrFail(t, rotatingFile, strings.Repeat("b", 40)+"\n")

	if current := readOrEmpty(t, logPath); len(current) != 82 {
		t.Errorf("expected both records in the current file, got %d bytes", len(current))
	}
	if previous := readOrEmpty(t, logPath+".1"); previous != "" {
		t.Errorf("expected no previous file yet, got %q", previous)
	}
}

func TestRotatingFile_MovesTheFullFileAsideBeforeCrossingTheCap(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "thyme.log")
	rotatingFile := openOrFail(t, logPath, 100)
	first, second := strings.Repeat("a", 60)+"\n", strings.Repeat("b", 60)+"\n"

	writeOrFail(t, rotatingFile, first)
	writeOrFail(t, rotatingFile, second)

	if previous := readOrEmpty(t, logPath+".1"); previous != first {
		t.Errorf("expected the first record moved aside, got %q", previous)
	}
	if current := readOrEmpty(t, logPath); current != second {
		t.Errorf("expected the current file to start afresh, got %q", current)
	}
}

func TestRotatingFile_KeepsOnlyOnePreviousFile(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "thyme.log")
	rotatingFile := openOrFail(t, logPath, 100)
	records := []string{strings.Repeat("a", 60) + "\n", strings.Repeat("b", 60) + "\n", strings.Repeat("c", 60) + "\n"}

	for _, record := range records {
		writeOrFail(t, rotatingFile, record)
	}

	if previous := readOrEmpty(t, logPath+".1"); previous != records[1] {
		t.Errorf("expected only the second record kept as previous, got %q", previous)
	}
	if current := readOrEmpty(t, logPath); current != records[2] {
		t.Errorf("expected the third record current, got %q", current)
	}
	directoryEntries, _ := os.ReadDir(filepath.Dir(logPath))
	if len(directoryEntries) != 2 {
		t.Errorf("expected exactly two files, got %v", directoryEntries)
	}
}

func TestRotatingFile_AppendsToAnExistingFileAndRotatesItWhenFull(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "thyme.log")
	earlier := strings.Repeat("e", 90) + "\n"
	if writeError := os.WriteFile(logPath, []byte(earlier), 0o644); writeError != nil {
		t.Fatalf("writing fixture: %v", writeError)
	}
	rotatingFile := openOrFail(t, logPath, 100)

	writeOrFail(t, rotatingFile, strings.Repeat("n", 20)+"\n")

	if previous := readOrEmpty(t, logPath+".1"); previous != earlier {
		t.Errorf("expected the earlier run's log moved aside, got %q", previous)
	}
}

// A single record larger than the cap is still written whole, after the
// rotation, because a cut record is worse than one oversized file.
func TestRotatingFile_WritesARecordLargerThanTheCapWhole(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "thyme.log")
	rotatingFile := openOrFail(t, logPath, 10)
	oversized := strings.Repeat("o", 30) + "\n"

	writeOrFail(t, rotatingFile, oversized)

	if current := readOrEmpty(t, logPath); current != oversized {
		t.Errorf("expected the oversized record whole, got %q", current)
	}
}

func TestOpenRotating_RefusesANonPositiveCap(t *testing.T) {
	if _, openError := OpenRotating(filepath.Join(t.TempDir(), "thyme.log"), 0); openError == nil {
		t.Error("expected a cap of zero to be refused")
	}
}

func TestRelativeFilePath_SitsBesideTheSettingsFile(t *testing.T) {
	if RelativeFilePath() != filepath.Join("Thyme", "thyme.log") {
		t.Errorf("expected Thyme/thyme.log, got '%s'", RelativeFilePath())
	}
}
