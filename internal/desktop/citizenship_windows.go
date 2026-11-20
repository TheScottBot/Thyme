package desktop

import (
	"os"

	"golang.org/x/sys/windows"

	"github.com/TheScottBot/Thyme/internal/eventlog"
	"github.com/TheScottBot/Thyme/internal/product"
	"github.com/TheScottBot/Thyme/internal/render"
)

// trayCallbackMessage carries the tray icon's mouse and keyboard events.
const trayCallbackMessage = wmApp + 2

// trayIconIdentifier is the uID of Thyme's one tray icon; the icon is found
// by window and uID, never by GUID, so a moved exe cannot orphan it.
const trayIconIdentifier = 1

// pinToDesktop makes the desktop's view the widget's owner, so Show Desktop
// (Win+D), which minimises everything above the desktop, spares it. The view
// lives under Progman, or under a WorkerW when a wallpaper slideshow is on.
// Its absence is logged once, and its recovery once, never every tick.
func (platform *win32Platform) pinToDesktop() {
	var defView uintptr
	if progman := findWindowByClass(0, "Progman"); progman != 0 {
		defView = findWindowByClass(progman, "SHELLDLL_DefView")
	}
	if defView == 0 {
		workerWindowDefViewFound = 0
		windows.EnumWindows(workerWindowSearchCallback, nil)
		defView = workerWindowDefViewFound
	}
	if defView == 0 {
		if !platform.defViewMissingLogged {
			platform.shell.logger.Warn("desktop view not found; Show Desktop will hide the widget until it is", "retrying", "every tick")
			platform.defViewMissingLogged = true
		}
		platform.defView = 0
		return
	}
	setOwner(platform.window, defView)
	if platform.defViewMissingLogged {
		platform.shell.logger.Info("desktop view found; the widget survives Show Desktop again")
		platform.defViewMissingLogged = false
	}
	platform.defView = defView
	platform.shell.applyLevel()
}

var workerWindowDefViewFound uintptr

// workerWindowSearchCallback looks under each top level WorkerW for the
// desktop's view. EnumWindows calls it synchronously on this thread.
var workerWindowSearchCallback = windows.NewCallback(func(window windows.HWND, userData uintptr) uintptr {
	if found := findWindowByClass(uintptr(window), "SHELLDLL_DefView"); found != 0 {
		workerWindowDefViewFound = found
		return 0
	}
	return 1
})

// ensurePinned re-pins on a tick if the view has gone, as it does when
// Explorer restarts before TaskbarCreated arrives.
func (platform *win32Platform) ensurePinned() {
	if platform.defView == 0 || !windows.IsWindow(windows.HWND(platform.defView)) {
		platform.pinToDesktop()
	}
}

// ApplyLevel sets always on top, or sends the widget to the bottom. Off is
// HWND_BOTTOM and never HWND_NOTOPMOST, because HWND_NOTOPMOST raises a window
// above the others, the opposite of what off means.
func (platform *win32Platform) ApplyLevel(alwaysOnTop bool) error {
	insertAfter := hwndBottom
	if alwaysOnTop {
		insertAfter = hwndTopmost
	}
	return setZOrder(platform.window, insertAfter)
}

func (platform *win32Platform) trayNotifyData() win32NotifyIconData {
	notifyData := win32NotifyIconData{
		Window:          platform.window,
		Identifier:      trayIconIdentifier,
		Flags:           nifMessage | nifIcon | nifTip | nifShowTip,
		CallbackMessage: trayCallbackMessage,
		Icon:            platform.trayIcon,
	}
	copy(notifyData.Tip[:len(notifyData.Tip)-1], windows.StringToUTF16(product.Name))
	return notifyData
}

// AddTrayIcon adds the icon, drawing it from the theme at the system's small
// icon size. Version 4 makes Windows report keyboard selection as well as
// clicks, which is the keyboard's way to the menu (WD9).
func (platform *win32Platform) AddTrayIcon() bool {
	logger := platform.shell.logger
	if platform.trayIcon == 0 {
		iconFrame, renderError := render.RenderTrayIcon(smallIconSize())
		if renderError != nil {
			logger.Warn("tray icon not drawn", "reason", eventlog.ReasonWithoutPath(renderError))
			return false
		}
		icon, iconError := createIcon(iconFrame.Pixels, iconFrame.Width, iconFrame.Height)
		if iconError != nil {
			logger.Warn("tray icon not created", "reason", eventlog.ReasonWithoutPath(iconError))
			return false
		}
		platform.trayIcon = icon
	}
	notifyData := platform.trayNotifyData()
	if addError := notifyIcon(nimAdd, &notifyData); addError != nil {
		logger.Warn("tray icon not added", "reason", eventlog.ReasonWithoutPath(addError))
		return false
	}
	notifyData.TimeoutOrVersion = notifyIconVersion4
	if versionError := notifyIcon(nimSetVersion, &notifyData); versionError != nil {
		logger.Warn("tray icon version not set", "reason", eventlog.ReasonWithoutPath(versionError))
	}
	return true
}

// RemoveTrayIcon deletes the icon without checking whether it was added.
func (platform *win32Platform) RemoveTrayIcon() {
	notifyData := platform.trayNotifyData()
	notifyIcon(nimDelete, &notifyData)
}

// SetAccessibleText sets the window's text, which is what a screen reader reads
// as the window's name.
func (platform *win32Platform) SetAccessibleText(text string) error {
	return setWindowText(platform.window, text)
}

func (platform *win32Platform) MenuLabels() MenuLabels {
	return windowsMenuLabels
}

func (platform *win32Platform) StartupEnabled() (bool, error) {
	return StartupEnabled(registryStartupStore{}, platform.exePath)
}

func (platform *win32Platform) ChangeStartup(change StartupChange) error {
	return ChangeStartup(registryStartupStore{}, platform.exePath, change)
}

func buildNativeMenu(entries []MenuEntry) uintptr {
	menu := createPopupMenu()
	for _, entry := range entries {
		switch entry.Kind {
		case MenuSeparator:
			appendMenuItem(menu, mfSeparator, 0, "")
		case MenuSubmenu:
			appendMenuItem(menu, mfPopup, buildNativeMenu(entry.Children), entry.Label)
		default:
			flags := uintptr(mfString)
			if entry.Checked {
				flags |= mfChecked
			}
			if !entry.Enabled {
				flags |= mfGrayed
			}
			appendMenuItem(menu, flags, uintptr(entry.Command), entry.Label)
		}
	}
	return menu
}

// ShowMenu shows the menu at a point. The foreground call lets a click
// elsewhere dismiss the menu; the posted WM_NULL is the documented way to let
// it close cleanly. Both can raise the widget, so the shell reapplies the
// level.
func (platform *win32Platform) ShowMenu(entries []MenuEntry, anchor screenPoint) MenuCommand {
	menu := buildNativeMenu(entries)
	setForegroundWindow(platform.window)
	chosen := trackPopupMenu(menu, platform.window, anchor.X, anchor.Y)
	postMessage(platform.window, wmNull)
	destroyMenu(menu)
	return MenuCommand(chosen)
}

// handleTrayEvent opens the same menu for a left click, a right click or
// keyboard selection. With version 4 the event is in the low word of lParam
// and the anchor point is packed in wParam.
func (platform *win32Platform) handleTrayEvent(wParam uintptr, lParam uintptr) {
	switch uint32(lParam & 0xFFFF) {
	case wmContextMenu, ninSelect, ninKeySelect:
		platform.shell.showMenu(OriginTrayMenu, screenPoint{X: signedLowWord(wParam), Y: signedHighWord(wParam)})
	}
}

// handleTaskbarCreated runs after Explorer restarts: the tray icon and the
// desktop's view are both new, so the icon is added again and the widget
// re-pinned, or Show Desktop would silently stop sparing it.
func (platform *win32Platform) handleTaskbarCreated() {
	platform.shell.logger.Info("taskbar recreated; re-adding the tray icon and re-pinning")
	platform.pinToDesktop()
	if platform.shell.current.ShowInTray {
		platform.shell.addTrayIcon()
	}
}

// longPathName expands 8.3 short names (PROGRA~1 and the like), so the
// temporary folder check compares the paths Windows would show. If Windows
// cannot expand a path, it is compared as given.
func longPathName(shortOrLongPath string) string {
	pathPointer, conversionError := windows.UTF16PtrFromString(shortOrLongPath)
	if conversionError != nil {
		return shortOrLongPath
	}
	expanded := make([]uint16, windows.MAX_LONG_PATH)
	length, expandError := windows.GetLongPathName(pathPointer, &expanded[0], uint32(len(expanded)))
	if expandError != nil || length == 0 || int(length) > len(expanded) {
		return shortOrLongPath
	}
	return windows.UTF16ToString(expanded[:length])
}

// ExeIsTemporary decides, once at start, whether this exe lives in the
// temporary folder, where Start with Windows must not point.
func (platform *win32Platform) ExeIsTemporary() bool {
	isTemporary := IsInsideFolder(longPathName(platform.exePath), longPathName(os.TempDir()))
	if isTemporary {
		platform.shell.logger.Warn("running from the temporary folder: Start with Windows can be turned off but not on; build into a folder of your own, as README.md describes")
	}
	return isTemporary
}
