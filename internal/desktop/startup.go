package desktop

import (
	"errors"
	"fmt"
)

// ActionOrigin names the surface a change came from. Two surfaces, the
// widget's menu and the tray's, can change the same things, so datum 0.14
// makes the origin a required field of every change that touches state
// outside the process, and the log says which surface acted.
type ActionOrigin string

const (
	OriginWidgetMenu ActionOrigin = "widget menu"
	OriginTrayMenu   ActionOrigin = "tray menu"
)

// StartupChange asks for Start with Windows to be on or off.
type StartupChange struct {
	Enable bool
	Origin ActionOrigin
}

// StartupStore is the two registry values Start with Windows depends on: the
// Run value Thyme writes, and the entry Task Manager writes when the user
// disables an app in its Startup tab.
type StartupStore interface {
	ReadRunValue() (string, bool, error)
	WriteRunValue(value string) error
	DeleteRunValue() error
	ReadApproval() ([]byte, bool, error)
	DeleteApproval() error
}

// RunValueFor quotes the exe path, because an unquoted path with a space in it
// is split at the space when Windows runs it at sign in.
func RunValueFor(exePath string) string {
	return `"` + exePath + `"`
}

// isApprovalDisabled reads Task Manager's undocumented entry. Observed on the
// author's machine (WE6 Plan): two apps Task Manager shows as Disabled have
// 0x01 as the first of twelve bytes, and apps with no entry show as enabled.
// An odd first byte is read as disabled; anything else, including an empty
// entry, as not disabled. The WE6 author gate checks both directions.
func isApprovalDisabled(approvalBytes []byte) bool {
	return len(approvalBytes) > 0 && approvalBytes[0]&1 == 1
}

// StartupEnabled reads the registry fresh. It is on only when the Run value
// launches this exe and Task Manager has not disabled it, so the menu's tick
// shows what will actually happen at the next sign in.
func StartupEnabled(store StartupStore, exePath string) (bool, error) {
	runValue, hasRunValue, runError := store.ReadRunValue()
	if runError != nil {
		return false, fmt.Errorf("reading the Run value: %w", runError)
	}
	if !hasRunValue || runValue != RunValueFor(exePath) {
		return false, nil
	}
	approvalBytes, hasApproval, approvalError := store.ReadApproval()
	if approvalError != nil {
		return false, fmt.Errorf("reading the Startup approval: %w", approvalError)
	}
	return !hasApproval || !isApprovalDisabled(approvalBytes), nil
}

// ChangeStartup turns Start with Windows on or off. Enabling also clears
// Thyme's own Task Manager disable (author's decision, WE6), which is the only
// way the tick can mean what it says; it never touches another app's entry.
// Every step is idempotent, and a failure part way is returned, never
// swallowed: the menu rereads the registry, so it shows what was left.
func ChangeStartup(store StartupStore, exePath string, change StartupChange) error {
	if change.Origin == "" {
		return errors.New("a startup change must say which surface asked for it")
	}
	if !change.Enable {
		if deleteError := store.DeleteRunValue(); deleteError != nil {
			return fmt.Errorf("deleting the Run value: %w", deleteError)
		}
		return nil
	}
	if writeError := store.WriteRunValue(RunValueFor(exePath)); writeError != nil {
		return fmt.Errorf("writing the Run value: %w", writeError)
	}
	_, hasApproval, readError := store.ReadApproval()
	if readError != nil {
		return fmt.Errorf("reading the Startup approval: %w", readError)
	}
	if !hasApproval {
		return nil
	}
	if deleteError := store.DeleteApproval(); deleteError != nil {
		return fmt.Errorf("clearing the Task Manager disable: %w", deleteError)
	}
	return nil
}
