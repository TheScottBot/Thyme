// Package eventlog keeps the log file. A GUI subsystem exe has no console, so
// this file is the only place a fault can surface (specification Part 5). It
// is capped in size and keeps one previous file, so a widget left running for
// months never fills a disk.
package eventlog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/TheScottBot/Thyme/internal/product"
)

// MaximumFileBytes caps the current log file. JSON lines of a few hundred
// bytes, written at start, on faults and on rare events, take years to reach
// it, while a fault that repeats every tick still cannot fill the disk.
const MaximumFileBytes = 1 << 20

const previousSuffix = ".1"

// RelativeFilePath is the log's path under the same per user folder as the
// settings file, so the user finds both in one place.
func RelativeFilePath() string {
	return filepath.Join(product.Name, strings.ToLower(product.Name)+".log")
}

// RotatingFile appends records to a file and, before a record would take it
// past the cap, moves it aside as the single previous file and starts afresh.
// It is safe for concurrent use, since log/slog may write from any goroutine.
type RotatingFile struct {
	mutex        sync.Mutex
	path         string
	maximumBytes int64
	currentFile  *os.File
	currentBytes int64
}

// OpenRotating opens or creates the log at path, creating its folder.
func OpenRotating(path string, maximumBytes int64) (*RotatingFile, error) {
	if maximumBytes <= 0 {
		return nil, fmt.Errorf("log cap must be positive, got %d", maximumBytes)
	}
	if mkdirError := os.MkdirAll(filepath.Dir(path), 0o755); mkdirError != nil {
		return nil, fmt.Errorf("creating the log folder: %w", mkdirError)
	}
	rotatingFile := &RotatingFile{path: path, maximumBytes: maximumBytes}
	if openError := rotatingFile.openCurrent(); openError != nil {
		return nil, openError
	}
	return rotatingFile, nil
}

func (rotatingFile *RotatingFile) openCurrent() error {
	currentFile, openError := os.OpenFile(rotatingFile.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if openError != nil {
		return fmt.Errorf("opening the log: %w", openError)
	}
	fileInformation, statError := currentFile.Stat()
	if statError != nil {
		currentFile.Close()
		return fmt.Errorf("reading the log's size: %w", statError)
	}
	rotatingFile.currentFile = currentFile
	rotatingFile.currentBytes = fileInformation.Size()
	return nil
}

func (rotatingFile *RotatingFile) rotate() error {
	closeError := rotatingFile.currentFile.Close()
	rotatingFile.currentFile = nil
	renameError := os.Rename(rotatingFile.path, rotatingFile.path+previousSuffix)
	if openError := rotatingFile.openCurrent(); openError != nil {
		return errors.Join(closeError, renameError, openError)
	}
	return errors.Join(closeError, renameError)
}

// Write appends one record whole. A record is never split across files.
func (rotatingFile *RotatingFile) Write(record []byte) (int, error) {
	rotatingFile.mutex.Lock()
	defer rotatingFile.mutex.Unlock()
	if rotatingFile.currentFile == nil {
		return 0, errors.New("the log is closed")
	}
	if rotatingFile.currentBytes > 0 && rotatingFile.currentBytes+int64(len(record)) > rotatingFile.maximumBytes {
		if rotateError := rotatingFile.rotate(); rotateError != nil && rotatingFile.currentFile == nil {
			return 0, fmt.Errorf("rotating the log: %w", rotateError)
		}
	}
	writtenBytes, writeError := rotatingFile.currentFile.Write(record)
	rotatingFile.currentBytes += int64(writtenBytes)
	return writtenBytes, writeError
}

// Close closes the current file. Further writes fail rather than reopen it.
func (rotatingFile *RotatingFile) Close() error {
	rotatingFile.mutex.Lock()
	defer rotatingFile.mutex.Unlock()
	if rotatingFile.currentFile == nil {
		return nil
	}
	closeError := rotatingFile.currentFile.Close()
	rotatingFile.currentFile = nil
	return closeError
}
