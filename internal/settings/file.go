package settings

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Load reads the settings file at path. A missing file is the first run and
// yields the defaults with nothing to report. The read stops one byte past the
// bound, so an oversized file is refused before any of it is parsed.
func Load(path string) (Settings, LoadReport) {
	settingsFile, openError := os.Open(path)
	if errors.Is(openError, fs.ErrNotExist) {
		return Defaults(), LoadReport{Source: SourceDefaults, OverwriteAllowed: true}
	}
	if openError != nil {
		return refusedWhole("", ReasonFileUnreadable)
	}
	defer settingsFile.Close()

	fileContents, readError := io.ReadAll(io.LimitReader(settingsFile, MaximumFileBytes+1))
	if readError != nil {
		return refusedWhole("", ReasonFileUnreadable)
	}
	if len(fileContents) > MaximumFileBytes {
		return refusedWhole("", ReasonFileTooLarge)
	}
	return Parse(fileContents)
}

// Save writes settings to a temporary file in the same folder and renames it
// over path, so a crash mid write leaves the old file or the new one, never
// half of either. Measured on NTFS in WE3: no reader ever saw a torn or
// missing file, but the rename fails while another program holds the file
// open, so a failed save is returned for the caller to log, never ignored.
func Save(path string, settings Settings) (saveError error) {
	encoded, encodeError := Encode(settings)
	if encodeError != nil {
		return encodeError
	}

	settingsFolder := filepath.Dir(path)
	if mkdirError := os.MkdirAll(settingsFolder, 0o755); mkdirError != nil {
		return fmt.Errorf("creating the settings folder: %w", mkdirError)
	}
	temporaryFile, createError := os.CreateTemp(settingsFolder, filepath.Base(path)+".*.tmp")
	if createError != nil {
		return fmt.Errorf("creating a temporary settings file: %w", createError)
	}
	temporaryPath := temporaryFile.Name()
	defer func() {
		if saveError != nil {
			os.Remove(temporaryPath)
		}
	}()

	_, writeError := temporaryFile.Write(encoded)
	syncError := temporaryFile.Sync()
	closeError := temporaryFile.Close()
	if failure := errors.Join(writeError, syncError, closeError); failure != nil {
		return fmt.Errorf("writing the temporary settings file: %w", failure)
	}
	if renameError := os.Rename(temporaryPath, path); renameError != nil {
		return fmt.Errorf("replacing the settings file: %w", renameError)
	}
	return nil
}
