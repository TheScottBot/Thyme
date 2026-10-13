package repocheck

import (
	"errors"
	"fmt"
	"os"
)

// OversizedFile names a release file over its size budget.
type OversizedFile struct {
	Path      string
	SizeBytes int64
}

// FindOversizedFiles checks each release file against the budget. A missing
// file or an empty list is an error, never a pass, because a size check that
// measured nothing would be trusted all the same.
func FindOversizedFiles(paths []string, maximumBytes int64) ([]OversizedFile, error) {
	if len(paths) == 0 {
		return nil, errors.New("no release files given to check")
	}
	var oversized []OversizedFile
	for _, path := range paths {
		fileInformation, statError := os.Stat(path)
		if statError != nil {
			return nil, fmt.Errorf("measuring '%s': %w", path, statError)
		}
		if fileInformation.Size() > maximumBytes {
			oversized = append(oversized, OversizedFile{Path: path, SizeBytes: fileInformation.Size()})
		}
	}
	return oversized, nil
}
