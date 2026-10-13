package jetbrainsmono

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

// The vendored files are kept verbatim (datum 0.13). This proves both the
// files on disk and the bytes compiled into the binary are the ones recorded
// in SHA256SUMS at the import.
func TestVendoredFiles_MatchRecordedDigests(t *testing.T) {
	sumsFile, openError := os.Open("SHA256SUMS")
	if openError != nil {
		t.Fatalf("opening SHA256SUMS: %v", openError)
	}
	defer sumsFile.Close()

	checkedCount := 0
	scanner := bufio.NewScanner(sumsFile)
	for scanner.Scan() {
		recordedDigest, fileName, found := strings.Cut(scanner.Text(), "  ")
		if !found {
			t.Fatalf("malformed SHA256SUMS line %q", scanner.Text())
		}
		fileContents, readError := os.ReadFile(fileName)
		if readError != nil {
			t.Fatalf("reading '%s': %v", fileName, readError)
		}
		actualDigest := sha256.Sum256(fileContents)
		if hex.EncodeToString(actualDigest[:]) != recordedDigest {
			t.Errorf("'%s' does not match its recorded digest", fileName)
		}
		if fileName == "JetBrainsMono-Regular.ttf" {
			embeddedDigest := sha256.Sum256(Regular)
			if hex.EncodeToString(embeddedDigest[:]) != recordedDigest {
				t.Error("the embedded font does not match its recorded digest")
			}
		}
		checkedCount++
	}
	if checkedCount != 2 {
		t.Errorf("expected two recorded files, checked %d", checkedCount)
	}
}
