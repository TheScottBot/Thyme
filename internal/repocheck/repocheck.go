// Package repocheck holds the tree wide checks that continuous integration runs
// over the repository: the import boundary between the pure core and the
// Windows shell, and the prose and typography scan.
//
// They are Go rather than shell scripts so that one implementation runs
// identically on the Linux and Windows runners and on the author's machine, and
// so that each check can be proven against planted violations by ordinary tests.
package repocheck

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// isDirectoryTheGoToolIgnores mirrors the go command's own rule for which
// directories never hold packages, so the import check sees exactly the
// packages a build would.
func isDirectoryTheGoToolIgnores(directoryName string) bool {
	return directoryName == "testdata" ||
		strings.HasPrefix(directoryName, ".") ||
		strings.HasPrefix(directoryName, "_")
}

// relativeSlashPath reports paths in forward slash form on every host, so a
// violation reads the same in a Windows and a Linux log and matches the
// slash separated lists the checks are configured with.
func relativeSlashPath(scanRoot string, absolutePath string) (string, error) {
	relativePath, relativeError := filepath.Rel(scanRoot, absolutePath)
	if relativeError != nil {
		return "", relativeError
	}
	return filepath.ToSlash(relativePath), nil
}

// walkFiles visits every regular file below scanRoot with its slash separated
// relative path, letting skipDirectory prune whole subtrees.
func walkFiles(scanRoot string, skipDirectory func(relativePath string, directoryName string) bool, visitFile func(absolutePath string, relativePath string) error) error {
	return filepath.WalkDir(scanRoot, func(absolutePath string, entry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}
		if absolutePath == scanRoot {
			return nil
		}
		relativePath, relativeError := relativeSlashPath(scanRoot, absolutePath)
		if relativeError != nil {
			return relativeError
		}
		if entry.IsDir() {
			if skipDirectory(relativePath, entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		return visitFile(absolutePath, relativePath)
	})
}
