package desktop

import (
	"path"
	"strings"
)

// IsInsideFolder reports whether a Windows path lies inside a folder. It
// treats both as Windows paths on every host, so it is tested on the Linux
// runner too: either slash is a separator, case is ignored as Windows ignores
// it, and ".." is resolved before comparing, so a path cannot climb out and
// still count as inside. A folder must match whole: Temp does not contain
// Temporary. An empty folder contains nothing.
//
// Short 8.3 names are not expanded here; the shell expands both paths with
// GetLongPathName before asking.
func IsInsideFolder(candidatePath string, folder string) bool {
	if strings.TrimSpace(folder) == "" {
		return false
	}
	normalise := func(windowsPath string) string {
		return strings.ToLower(path.Clean(strings.ReplaceAll(windowsPath, `\`, "/")))
	}
	normalisedCandidate, normalisedFolder := normalise(candidatePath), normalise(folder)
	return normalisedCandidate == normalisedFolder || strings.HasPrefix(normalisedCandidate, strings.TrimSuffix(normalisedFolder, "/")+"/")
}
