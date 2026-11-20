package main

import (
	"encoding/xml"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/TheScottBot/Thyme/internal/product"
)

func TestInfoPlist_IsWellFormedAndNamesTheProduct(t *testing.T) {
	plist := infoPlist()

	decoder := xml.NewDecoder(strings.NewReader(plist))
	for {
		if _, tokenError := decoder.Token(); tokenError != nil {
			if tokenError.Error() != "EOF" {
				t.Fatalf("not well formed: %v", tokenError)
			}
			break
		}
	}
	for _, expected := range []string{"<string>" + product.Name + "</string>", "<string>" + product.Version + "</string>", "<key>LSUIElement</key>", bundleIdentifier, "<key>CFBundleIconFile</key>\n\t<string>" + iconName + "</string>"} {
		if !strings.Contains(plist, expected) {
			t.Errorf("expected the plist to hold %q", expected)
		}
	}
}

func writeFakeBinary(t *testing.T, folder string) string {
	t.Helper()
	binaryPath := filepath.Join(folder, "thyme")
	if writeError := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); writeError != nil {
		t.Fatal(writeError)
	}
	return binaryPath
}

func TestWriteBundle_LaysOutABundleAndReplacesAnOldOne(t *testing.T) {
	folder := t.TempDir()
	binaryPath := writeFakeBinary(t, folder)
	bundlePath := filepath.Join(folder, "Thyme.app")
	staleFile := filepath.Join(bundlePath, "stale")
	if mkdirError := os.MkdirAll(bundlePath, 0o755); mkdirError != nil {
		t.Fatal(mkdirError)
	}
	if writeError := os.WriteFile(staleFile, nil, 0o644); writeError != nil {
		t.Fatal(writeError)
	}

	if bundleError := writeBundle(binaryPath, bundlePath); bundleError != nil {
		t.Fatal(bundleError)
	}

	info, statError := os.Stat(filepath.Join(bundlePath, "Contents", "MacOS", product.Name))
	if statError != nil {
		t.Fatalf("expected the program inside: %v", statError)
	}
	// Windows reports no execute bit for any file.
	if runtime.GOOS != "windows" && info.Mode()&0o100 == 0 {
		t.Errorf("expected it to be executable, got %v", info.Mode())
	}
	if _, statError := os.Stat(staleFile); !os.IsNotExist(statError) {
		t.Errorf("expected the old bundle gone, got %v", statError)
	}
	if _, statError := os.Stat(filepath.Join(folder, "Thyme.building.app")); !os.IsNotExist(statError) {
		t.Errorf("expected the working folder gone, got %v", statError)
	}
}

// The tool deletes its destination, so it must refuse anything that does not
// look like a bundle rather than remove a folder given by mistake.
func TestWriteBundle_RefusesADestinationThatIsNotABundle(t *testing.T) {
	folder := t.TempDir()
	precious := filepath.Join(folder, "precious")
	if mkdirError := os.MkdirAll(precious, 0o755); mkdirError != nil {
		t.Fatal(mkdirError)
	}

	if bundleError := writeBundle(writeFakeBinary(t, folder), precious); bundleError == nil {
		t.Fatal("expected a refusal")
	}
	if _, statError := os.Stat(precious); statError != nil {
		t.Errorf("expected the folder kept: %v", statError)
	}
}

func TestWriteBundle_LeavesTheExistingBundleWhenTheBinaryIsMissing(t *testing.T) {
	folder := t.TempDir()
	bundlePath := filepath.Join(folder, "Thyme.app")
	if mkdirError := os.MkdirAll(bundlePath, 0o755); mkdirError != nil {
		t.Fatal(mkdirError)
	}

	if bundleError := writeBundle(filepath.Join(folder, "missing"), bundlePath); bundleError == nil {
		t.Fatal("expected a failure")
	}
	if _, statError := os.Stat(bundlePath); statError != nil {
		t.Errorf("expected the bundle kept: %v", statError)
	}
}

func TestWriteIconSet_WritesEverySizeIconutilExpects(t *testing.T) {
	folder := filepath.Join(t.TempDir(), "AppIcon.iconset")

	if iconError := writeIconSet(folder); iconError != nil {
		t.Fatal(iconError)
	}

	for _, file := range iconSetFiles {
		opened, openError := os.Open(filepath.Join(folder, file.name))
		if openError != nil {
			t.Fatalf("%s: %v", file.name, openError)
		}
		configuration, decodeError := png.DecodeConfig(opened)
		opened.Close()
		if decodeError != nil || configuration.Width != file.sizePixels || configuration.Height != file.sizePixels {
			t.Errorf("%s: expected %d pixels square, got %+v, %v", file.name, file.sizePixels, configuration, decodeError)
		}
	}
}

// Only macOS has iconutil, so the finished icns file is checked there.
func TestWriteBundle_HoldsTheIconOnMacOS(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("iconutil is macOS only")
	}
	folder := t.TempDir()
	bundlePath := filepath.Join(folder, "Thyme.app")

	if bundleError := writeBundle(writeFakeBinary(t, folder), bundlePath); bundleError != nil {
		t.Fatal(bundleError)
	}

	info, statError := os.Stat(filepath.Join(bundlePath, "Contents", "Resources", iconName+".icns"))
	if statError != nil || info.Size() == 0 {
		t.Errorf("expected a non empty icns file, got %v, %v", info, statError)
	}
	if _, statError := os.Stat(filepath.Join(bundlePath, iconName+".iconset")); !os.IsNotExist(statError) {
		t.Errorf("expected the icon set left out of the bundle, got %v", statError)
	}
}
