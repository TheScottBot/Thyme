package settings

import (
	"path/filepath"
	"slices"

	"github.com/TheScottBot/Thyme/internal/product"
)

// Position is the widget's top left corner in physical pixels of the virtual
// screen. It may be negative on a monitor left of or above the primary one;
// keeping it on a monitor is internal/placement's job, not this package's.
type Position struct {
	X int32
	Y int32
}

// ZoneSetting is one world clock as the user wrote it. Whether the time zone
// exists is decided by internal/clockmodel when it is loaded, because only the
// tz database can say; this package checks only its shape.
type ZoneSetting struct {
	Label    string
	TimeZone string
}

// Settings is everything Thyme persists.
type Settings struct {
	Position     Position
	ScalePercent int
	AlwaysOnTop  bool
	ShowInTray   bool
	ShowSeconds  bool
	Zones        []ZoneSetting
}

const (
	// MaximumFileBytes bounds the settings file before it is parsed. A full
	// file with four zones is well under 1 KB; the bound leaves room for hand
	// formatting while keeping a mistaken or hostile file from being read in
	// full.
	MaximumFileBytes = 4096

	// MaximumZoneCount is the number of world clocks the theme has room for.
	MaximumZoneCount = 4

	// Labels are counted in characters, not bytes, because the theme sets them
	// in a monospaced column three characters wide.
	MinimumLabelCharacters = 1
	MaximumLabelCharacters = 3

	// MaximumTimeZoneNameBytes is about twice the longest IANA name in use, so
	// it refuses nonsense without refusing any real zone.
	MaximumTimeZoneNameBytes = 64
)

// supportedVersion is the only settings version this build understands. A file
// of any other version is refused whole and protected from being saved over.
const supportedVersion = 1

const fileName = "settings.json"

// scaleStepsPercent are the Scale submenu's steps (WD5). Only these values are
// accepted, so the menu, the file and the arithmetic can never disagree.
var scaleStepsPercent = []int{75, 100, 125, 150, 200}

// ScaleStepsPercent returns a copy of the scale steps, smallest first.
func ScaleStepsPercent() []int {
	return slices.Clone(scaleStepsPercent)
}

// RelativeFilePath is the settings file's path under the per user writable
// folder the shell resolves. It is never beside the exe: a program under
// Program Files cannot write there, and the failure would be silent.
func RelativeFilePath() string {
	return filepath.Join(product.Name, fileName)
}

// Defaults are the settings used when there is no file, and for any field a
// file gets wrong. The zones are the mock-up's; the tray is on by default
// (WD9) because it is the keyboard's way in to a widget that never takes focus.
func Defaults() Settings {
	return Settings{
		Position:     Position{X: 120, Y: 120},
		ScalePercent: 100,
		AlwaysOnTop:  false,
		ShowInTray:   true,
		ShowSeconds:  true,
		Zones: []ZoneSetting{
			{Label: "UTC", TimeZone: "UTC"},
			{Label: "LDN", TimeZone: "Europe/London"},
			{Label: "BER", TimeZone: "Europe/Berlin"},
			{Label: "SAO", TimeZone: "America/Sao_Paulo"},
		},
	}
}
