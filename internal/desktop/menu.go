package desktop

import (
	"strconv"

	"github.com/TheScottBot/Thyme/internal/settings"
)

// MenuEntryKind is what a menu line is.
type MenuEntryKind int

const (
	MenuItem MenuEntryKind = iota
	MenuSeparator
	MenuSubmenu
)

// MenuCommand identifies a chosen item. Zero is never used, because
// TrackPopupMenu returns zero when the menu is dismissed.
type MenuCommand int

const (
	CommandSeconds MenuCommand = iota + 1
	CommandAlwaysOnTop
	CommandStartAtLogin
	CommandShowInTray
	CommandExit

	// commandScaleFirst numbers the Scale submenu's steps in order.
	commandScaleFirst MenuCommand = 100
)

// MenuEntry is one line of the menu as data, so its items, ticks and enabled
// states are tested without a window.
type MenuEntry struct {
	Kind     MenuEntryKind
	Label    string
	Command  MenuCommand
	Checked  bool
	Enabled  bool
	Children []MenuEntry
}

// MenuState is everything the menu shows. Start at login comes from the
// operating system, read each time the menu opens; Unknown means that read
// failed.
type MenuState struct {
	Labels              MenuLabels
	Settings            settings.Settings
	StartAtLogin        bool
	StartAtLoginUnknown bool

	StartAtLoginFromTemporaryExe bool
}

// MenuLabels is the text of the menu's lines, which is all that differs
// between platforms: same items, same order, each platform's own words.
type MenuLabels struct {
	Seconds      string
	Scale        string
	AlwaysOnTop  string
	StartAtLogin string
	ShowInTray   string
	Exit         string
}

// The Windows labels carry an ampersand before their access key, so every item
// can be chosen from the keyboard once the menu is open (datum 0.16, WD9). The
// macOS labels carry none, because macOS menus are searched by typing instead.
// English only (WD7).
var (
	windowsMenuLabels = MenuLabels{
		Seconds:      "&Seconds",
		Scale:        "S&cale",
		AlwaysOnTop:  "Always on &top",
		StartAtLogin: "Start with &Windows",
		ShowInTray:   "Show in t&ray",
		Exit:         "E&xit",
	}
	macMenuLabels = MenuLabels{
		Seconds:      "Seconds",
		Scale:        "Scale",
		AlwaysOnTop:  "Always on Top",
		StartAtLogin: "Open at Login",
		ShowInTray:   "Show in Menu Bar",
		Exit:         "Quit Thyme",
	}
)

func toggle(label string, command MenuCommand, checked bool) MenuEntry {
	return MenuEntry{Kind: MenuItem, Label: label, Command: command, Checked: checked, Enabled: true}
}

// BuildMenu returns the menu in the specification's order (WE6), with the
// Scale steps of WD5 and the current one ticked (WE7).
func BuildMenu(state MenuState) []MenuEntry {
	scaleSteps := settings.ScaleStepsPercent()
	scaleEntries := make([]MenuEntry, 0, len(scaleSteps))
	for stepIndex, scalePercent := range scaleSteps {
		scaleEntries = append(scaleEntries, MenuEntry{
			Kind:    MenuItem,
			Label:   strconv.Itoa(scalePercent) + "%",
			Command: commandScaleFirst + MenuCommand(stepIndex),
			Checked: scalePercent == state.Settings.ScalePercent,
			Enabled: true,
		})
	}
	labels := state.Labels
	startAtLogin := toggle(labels.StartAtLogin, CommandStartAtLogin, state.StartAtLogin)
	if state.StartAtLoginUnknown {
		startAtLogin.Checked, startAtLogin.Enabled = false, false
	}
	// From a temporary exe (go run, or a zip opened in Explorer) the entry
	// would name a file that is soon deleted: a sign in entry pointing at
	// nothing, which anything could later fill. It can be turned off, so an
	// existing entry is never trapped, but not on.
	if state.StartAtLoginFromTemporaryExe && !startAtLogin.Checked {
		startAtLogin.Enabled = false
	}
	return []MenuEntry{
		toggle(labels.Seconds, CommandSeconds, state.Settings.ShowSeconds),
		{Kind: MenuSubmenu, Label: labels.Scale, Enabled: true, Children: scaleEntries},
		{Kind: MenuSeparator},
		toggle(labels.AlwaysOnTop, CommandAlwaysOnTop, state.Settings.AlwaysOnTop),
		startAtLogin,
		toggle(labels.ShowInTray, CommandShowInTray, state.Settings.ShowInTray),
		{Kind: MenuSeparator},
		{Kind: MenuItem, Label: labels.Exit, Command: CommandExit, Enabled: true},
	}
}

// ScalePercentForCommand maps a Scale submenu command back to its step. It is
// derived from the same list the menu is built from, so the two cannot
// disagree, and no other command can set a scale.
func ScalePercentForCommand(command MenuCommand) (int, bool) {
	stepIndex := int(command - commandScaleFirst)
	scaleSteps := settings.ScaleStepsPercent()
	if stepIndex < 0 || stepIndex >= len(scaleSteps) {
		return 0, false
	}
	return scaleSteps[stepIndex], true
}
