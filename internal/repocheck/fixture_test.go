package repocheck

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFixtureTree builds a throwaway tree under a per test temporary directory
// from relative slash separated paths, so every check is proven against
// planted files without planting anything in the real repository.
func writeFixtureTree(t *testing.T, contentsByRelativePath map[string]string) string {
	t.Helper()
	fixtureRoot := t.TempDir()
	for relativePath, fileContents := range contentsByRelativePath {
		absolutePath := filepath.Join(fixtureRoot, filepath.FromSlash(relativePath))
		if mkdirError := os.MkdirAll(filepath.Dir(absolutePath), 0o755); mkdirError != nil {
			t.Fatalf("creating fixture directory for '%s': %v", relativePath, mkdirError)
		}
		if writeError := os.WriteFile(absolutePath, []byte(fileContents), 0o644); writeError != nil {
			t.Fatalf("writing fixture file '%s': %v", relativePath, writeError)
		}
	}
	return fixtureRoot
}
