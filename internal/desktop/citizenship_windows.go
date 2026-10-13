package desktop

import (
	"time"

	"golang.org/x/sys/windows"

	"github.com/TheScottBot/Thyme/internal/clockmodel"
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
func (shell *shell) pinToDesktop() {
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
		if !shell.defViewMissingLogged {
			shell.logger.Warn("desktop view not found; Show Desktop will hide the widget until it is", "retrying", "every tick")
			shell.defViewMissingLogged = true
		}
		shell.defView = 0
		return
	}
	setOwner(shell.window, defView)
	if shell.defViewMissingLogged {
		shell.logger.Info("desktop view found; the widget survives Show Desktop again")
		shell.defViewMissingLogged = false
	}
	shell.defView = defView
	shell.applyZOrder()
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
func (shell *shell) ensurePinned() {
	if shell.defView == 0 || !windows.IsWindow(windows.HWND(shell.defView)) {
		shell.pinToDesktop()
	}
}

// applyZOrder sets always on top, or sends the widget to the bottom. Off is
// HWND_BOTTOM and never HWND_NOTOPMOST, because HWND_NOTOPMOST raises a window
// above the others, the opposite of what off means.
func (shell *shell) applyZOrder() {
	insertAfter := hwndBottom
	if shell.current.AlwaysOnTop {
		insertAfter = hwndTopmost
	}
	if zOrderError := setZOrder(shell.window, insertAfter); zOrderError != nil {
		shell.logger.Warn("z order not applied", "reason", eventlog.ReasonWithoutPath(zOrderError))
	}
}

func (shell *shell) trayNotifyData() win32NotifyIconData {
	notifyData := win32NotifyIconData{
		Window:          shell.window,
		Identifier:      trayIconIdentifier,
		Flags:           nifMessage | nifIcon | nifTip | nifShowTip,
		CallbackMessage: trayCallbackMessage,
		Icon:            shell.trayIcon,
	}
	copy(notifyData.Tip[:len(notifyData.Tip)-1], windows.StringToUTF16(product.Name))
	return notifyData
}

// addTrayIcon adds the icon, drawing it from the theme at the system's small
// icon size. Version 4 makes Windows report keyboard selection as well as
// clicks, which is the keyboard's way to the menu (WD9).
func (shell *shell) addTrayIcon() {
	if shell.trayIcon == 0 {
		iconFrame, renderError := render.RenderTrayIcon(smallIconSize())
		if renderError != nil {
			shell.logger.Warn("tray icon not drawn", "reason", eventlog.ReasonWithoutPath(renderError))
			return
		}
		icon, iconError := createIcon(iconFrame.Pixels, iconFrame.Width, iconFrame.Height)
		if iconError != nil {
			shell.logger.Warn("tray icon not created", "reason", eventlog.ReasonWithoutPath(iconError))
			return
		}
		shell.trayIcon = icon
	}
	notifyData := shell.trayNotifyData()
	if addError := notifyIcon(nimAdd, &notifyData); addError != nil {
		shell.logger.Warn("tray icon not added", "reason", eventlog.ReasonWithoutPath(addError))
		return
	}
	notifyData.TimeoutOrVersion = notifyIconVersion4
	if versionError := notifyIcon(nimSetVersion, &notifyData); versionError != nil {
		shell.logger.Warn("tray icon version not set", "reason", eventlog.ReasonWithoutPath(versionError))
	}
	shell.trayShown = true
}

// removeTrayIcon deletes the icon without checking whether it was added, so
// an exit can never leave a dead icon behind.
func (shell *shell) removeTrayIcon() {
	notifyData := shell.trayNotifyData()
	notifyIcon(nimDelete, &notifyData)
	shell.trayShown = false
}

// updateWindowText sets the window's text to the spoken time once a minute,
// so a screen reader reading the window's name has the time to say (WD9).
func (shell *shell) updateWindowText(now time.Time) {
	spoken := product.Name + ": " + clockmodel.SpokenTime(now, time.Local)
	if spoken == shell.lastWindowText {
		return
	}
	if textError := setWindowText(shell.window, spoken); textError != nil {
		shell.logger.Warn("window text not set", "reason", eventlog.ReasonWithoutPath(textError))
		return
	}
	shell.lastWindowText = spoken
}

func (shell *shell) startWithWindowsState() (bool, bool) {
	enabled, readError := StartupEnabled(registryStartupStore{}, shell.exePath)
	if readError != nil {
		shell.logger.Warn("start with Windows state not read", "reason", eventlog.ReasonWithoutPath(readError))
		return false, true
	}
	return enabled, false
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

// showMenu builds the menu fresh, so the Start with Windows tick is read from
// the registry every time it opens. The foreground call lets a click
// elsewhere dismiss the menu; the posted WM_NULL is the documented way to let
// it close cleanly. Both can raise the widget, so the z order is reapplied.
func (shell *shell) showMenu(origin ActionOrigin, x int32, y int32) {
	startWithWindows, unknown := shell.startWithWindowsState()
	menu := buildNativeMenu(BuildMenu(MenuState{Settings: shell.current, StartWithWindows: startWithWindows, StartWithWindowsUnknown: unknown}))
	setForegroundWindow(shell.window)
	chosen := trackPopupMenu(menu, shell.window, x, y)
	postMessage(shell.window, wmNull)
	destroyMenu(menu)
	shell.applyZOrder()
	if chosen != 0 {
		shell.perform(MenuCommand(chosen), origin)
	}
}

// perform carries out a menu command. Each logs the command and the surface
// it came from (datum 0.14).
func (shell *shell) perform(command MenuCommand, origin ActionOrigin) {
	if scalePercent, isScale := ScalePercentForCommand(command); isScale {
		shell.changeScale(scalePercent, origin)
		return
	}
	switch command {
	case CommandSeconds:
		shell.current.ShowSeconds = !shell.current.ShowSeconds
		shell.logger.Info("seconds toggled", "showSeconds", shell.current.ShowSeconds, "origin", string(origin))
		shell.redraw(nil)
		shell.clampOntoAMonitor()
		shell.aimTimer()
		returnMemoryToWindows()
		shell.saveSettings(origin)

	case CommandAlwaysOnTop:
		shell.current.AlwaysOnTop = !shell.current.AlwaysOnTop
		shell.logger.Info("always on top toggled", "alwaysOnTop", shell.current.AlwaysOnTop, "origin", string(origin))
		shell.applyZOrder()
		shell.saveSettings(origin)

	case CommandStartWithWindows:
		enabled, _ := shell.startWithWindowsState()
		change := StartupChange{Enable: !enabled, Origin: origin}
		if changeError := ChangeStartup(registryStartupStore{}, shell.exePath, change); changeError != nil {
			shell.logger.Warn("start with Windows not changed", "enable", change.Enable, "origin", string(origin), "reason", eventlog.ReasonWithoutPath(changeError))
			return
		}
		shell.logger.Info("start with Windows changed", "enable", change.Enable, "origin", string(origin))

	case CommandShowInTray:
		shell.current.ShowInTray = !shell.current.ShowInTray
		shell.logger.Info("tray icon toggled", "showInTray", shell.current.ShowInTray, "origin", string(origin))
		if shell.current.ShowInTray {
			shell.addTrayIcon()
		} else {
			shell.removeTrayIcon()
		}
		shell.saveSettings(origin)

	case CommandExit:
		shell.logger.Info("exit chosen", "origin", string(origin))
		shell.saveSettings(origin)
		shell.removeTrayIcon()
		destroyWindow(shell.window)
	}
}

// handleTrayEvent opens the same menu for a left click, a right click or
// keyboard selection. With version 4 the event is in the low word of lParam
// and the anchor point is packed in wParam.
func (shell *shell) handleTrayEvent(wParam uintptr, lParam uintptr) {
	switch uint32(lParam & 0xFFFF) {
	case wmContextMenu, ninSelect, ninKeySelect:
		shell.showMenu(OriginTrayMenu, signedLowWord(wParam), signedHighWord(wParam))
	}
}

// handleTaskbarCreated runs after Explorer restarts: the tray icon and the
// desktop's view are both new, so the icon is added again and the widget
// re-pinned, or Show Desktop would silently stop sparing it.
func (shell *shell) handleTaskbarCreated() {
	shell.logger.Info("taskbar recreated; re-adding the tray icon and re-pinning")
	shell.pinToDesktop()
	if shell.current.ShowInTray {
		shell.addTrayIcon()
	}
}

// changeScale applies a Scale step (WE7). The renderer rebuilds its static
// layer for the new size; the window keeps its top left corner, because
// UpdateLayeredWindow without a position resizes from there; then it is
// clamped, since a larger widget may now hang off the screen, and saved.
func (shell *shell) changeScale(scalePercent int, origin ActionOrigin) {
	if scalePercent == shell.current.ScalePercent {
		return
	}
	shell.current.ScalePercent = scalePercent
	shell.logger.Info("scale changed", "scalePercent", scalePercent, "origin", string(origin))
	shell.redraw(nil)
	shell.clampOntoAMonitor()
	returnMemoryToWindows()
	shell.saveSettings(origin)
}
