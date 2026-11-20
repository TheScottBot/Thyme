// Command macbundle wraps a built macOS binary in an app bundle with its icon
// and signs it ad hoc (WD11), because Open at Login registers a bundle, not a
// bare program. The icon is drawn by render.RenderAppIcon and turned into an
// icns file by iconutil, so Thyme ships no icon file.
//
// Usage: go run ./tools/macbundle <binary> <name.app>
package main

import (
	"errors"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/TheScottBot/Thyme/internal/product"
	"github.com/TheScottBot/Thyme/internal/render"
)

// bundleIdentifier names the app to the system, which keys the login item and
// its approval on it, so it must not change.
const bundleIdentifier = "com.thescottbot.thyme"

// minimumSystemVersion is WD11's floor, and what the build's
// MACOSX_DEPLOYMENT_TARGET in README.md must match.
const minimumSystemVersion = "13.0"

// iconName is the icon's file name in Resources, without its extension, as
// CFBundleIconFile names it.
const iconName = "AppIcon"

// iconSetFiles are the files iconutil expects: each point size at 1x and 2x.
var iconSetFiles = []struct {
	name       string
	sizePixels int
}{
	{"icon_16x16.png", 16}, {"icon_16x16@2x.png", 32},
	{"icon_32x32.png", 32}, {"icon_32x32@2x.png", 64},
	{"icon_128x128.png", 128}, {"icon_128x128@2x.png", 256},
	{"icon_256x256.png", 256}, {"icon_256x256@2x.png", 512},
	{"icon_512x512.png", 512}, {"icon_512x512@2x.png", 1024},
}

// writeIconSet draws every size of the icon into a folder for iconutil.
func writeIconSet(folder string) error {
	if mkdirError := os.MkdirAll(folder, 0o755); mkdirError != nil {
		return mkdirError
	}
	for _, file := range iconSetFiles {
		icon, renderError := render.RenderAppIcon(file.sizePixels)
		if renderError != nil {
			return renderError
		}
		output, createError := os.Create(filepath.Join(folder, file.name))
		if createError != nil {
			return createError
		}
		encodeError := png.Encode(output, icon)
		if closeError := output.Close(); encodeError == nil {
			encodeError = closeError
		}
		if encodeError != nil {
			return fmt.Errorf("writing %s: %w", file.name, encodeError)
		}
	}
	return nil
}

func infoPlist() string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key>
	<string>en</string>
	<key>CFBundleExecutable</key>
	<string>%[1]s</string>
	<key>CFBundleIconFile</key>
	<string>%[5]s</string>
	<key>CFBundleIdentifier</key>
	<string>%[2]s</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleName</key>
	<string>%[1]s</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>%[3]s</string>
	<key>CFBundleVersion</key>
	<string>%[3]s</string>
	<key>LSMinimumSystemVersion</key>
	<string>%[4]s</string>
	<key>LSUIElement</key>
	<true/>
	<key>NSHighResolutionCapable</key>
	<true/>
</dict>
</plist>
`, product.Name, bundleIdentifier, product.Version, minimumSystemVersion, iconName)
}

// writeBundle builds the bundle beside its destination and only then replaces
// what was there, so a failure leaves an existing Thyme.app as it was. Only a
// path ending in .app is ever replaced.
func writeBundle(binaryPath string, bundlePath string) error {
	if !strings.HasSuffix(bundlePath, ".app") {
		return errors.New("the bundle's name must end in .app")
	}
	binary, readError := os.ReadFile(binaryPath)
	if readError != nil {
		return fmt.Errorf("reading the binary: %w", readError)
	}
	workingPath := strings.TrimSuffix(bundlePath, ".app") + ".building.app"
	if removeError := os.RemoveAll(workingPath); removeError != nil {
		return removeError
	}
	defer os.RemoveAll(workingPath)

	contents := filepath.Join(workingPath, "Contents")
	executableFolder := filepath.Join(contents, "MacOS")
	if mkdirError := os.MkdirAll(executableFolder, 0o755); mkdirError != nil {
		return mkdirError
	}
	if writeError := os.WriteFile(filepath.Join(contents, "Info.plist"), []byte(infoPlist()), 0o644); writeError != nil {
		return writeError
	}
	if writeError := os.WriteFile(filepath.Join(executableFolder, product.Name), binary, 0o755); writeError != nil {
		return writeError
	}
	if runtime.GOOS == "darwin" {
		// The icon set is drawn inside the working bundle, so it goes when
		// that does, and only the icns file is kept.
		iconSetPath := filepath.Join(workingPath, iconName+".iconset")
		if iconError := writeIconSet(iconSetPath); iconError != nil {
			return fmt.Errorf("drawing the icon: %w", iconError)
		}
		resources := filepath.Join(contents, "Resources")
		if mkdirError := os.MkdirAll(resources, 0o755); mkdirError != nil {
			return mkdirError
		}
		if output, iconError := exec.Command("iconutil", "-c", "icns", "-o", filepath.Join(resources, iconName+".icns"), iconSetPath).CombinedOutput(); iconError != nil {
			return fmt.Errorf("making the icns file: %w: %s", iconError, output)
		}
		if removeError := os.RemoveAll(iconSetPath); removeError != nil {
			return removeError
		}
		// The dash names the ad hoc identity: no certificate is needed.
		if output, signError := exec.Command("codesign", "-s", "-", "-f", workingPath).CombinedOutput(); signError != nil {
			return fmt.Errorf("signing: %w: %s", signError, output)
		}
	}
	if removeError := os.RemoveAll(bundlePath); removeError != nil {
		return removeError
	}
	return os.Rename(workingPath, bundlePath)
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: macbundle <binary> <name.app>")
		os.Exit(2)
	}
	if bundleError := writeBundle(os.Args[1], os.Args[2]); bundleError != nil {
		fmt.Fprintf(os.Stderr, "macbundle: %v\n", bundleError)
		os.Exit(1)
	}
	fmt.Printf("macbundle: built %s\n", os.Args[2])
}
