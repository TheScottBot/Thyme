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
	CommandStartWithWindows
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

// MenuState is everything the menu shows. Start with Windows comes from the
// registry, read each time the menu opens; Unknown means that read failed.
type MenuState struct {
	Settings                settings.Settings
	StartWithWindows        bool
	StartWithWindowsUnknown bool
}

// The labels carry an ampersand before their access key, so every item can be
// chosen from the keyboard once the menu is open (datum 0.16, WD9). English
// only (WD7).
const (
	labelSeconds          = "&Seconds"
	labelScale            = "S&cale"
	labelAlwaysOnTop      = "Always on &top"
	labelStartWithWindows = "Start with &Windows"
	labelShowInTray       = "Show in t&ray"
	labelExit             = "E&xit"
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
	startWithWindows := toggle(labelStartWithWindows, CommandStartWithWindows, state.StartWithWindows)
	if state.StartWithWindowsUnknown {
		startWithWindows.Checked, startWithWindows.Enabled = false, false
	}
	return []MenuEntry{
		toggle(labelSeconds, CommandSeconds, state.Settings.ShowSeconds),
		{Kind: MenuSubmenu, Label: labelScale, Enabled: true, Children: scaleEntries},
		{Kind: MenuSeparator},
		toggle(labelAlwaysOnTop, CommandAlwaysOnTop, state.Settings.AlwaysOnTop),
		startWithWindows,
		toggle(labelShowInTray, CommandShowInTray, state.Settings.ShowInTray),
		{Kind: MenuSeparator},
		{Kind: MenuItem, Label: labelExit, Command: CommandExit, Enabled: true},
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
