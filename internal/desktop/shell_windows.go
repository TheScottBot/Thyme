package desktop

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"time"

	"golang.org/x/sys/windows"

	"github.com/TheScottBot/Thyme/internal/clockmodel"
	"github.com/TheScottBot/Thyme/internal/eventlog"
	"github.com/TheScottBot/Thyme/internal/placement"
	"github.com/TheScottBot/Thyme/internal/product"
	"github.com/TheScottBot/Thyme/internal/render"
	"github.com/TheScottBot/Thyme/internal/settings"
)

// tickMessage is posted by the timer goroutine. Only the GUI thread touches
// the window; the timer only asks it to.
const tickMessage = wmApp + 1

// shell is the one window's state. Every field is read and written on the
// GUI thread only, except the atomic counter the timer goroutine updates.
type shell struct {
	window   uintptr
	logger   *slog.Logger
	renderer *render.Renderer

	settingsPath     string
	current          settings.Settings
	overwriteAllowed bool
	zones            []clockmodel.Zone

	dpi         int
	bitmap      *layeredBitmap
	panelBounds placement.Insets
	tickTimer   *time.Timer

	// startupTimeZone is the zone Go read once for time.Local; a later change
	// is noticed and logged but only takes effect on restart (author's
	// decision after WE5; see README.md, known limitations).
	startupTimeZone      windows.Timezoneinformation
	timeZoneChangeLogged bool

	// Desktop citizenship (WE6).
	exePath               string
	taskbarCreatedMessage uint32
	defView               uintptr
	defViewMissingLogged  bool
	trayIcon              uintptr
	trayShown             bool
	exeIsTemporary        bool
	lastWindowText        string

	updateLayeredWindowFailures int
	renderFailures              int
	rejectedDpiChanges          int
	settingsSaveFailures        int
	tickPostFailures            atomic.Int64
}

// activeShell is reached by the window procedure, which Windows calls with
// no user data of ours; there is exactly one window per process.
var activeShell *shell

// Run is Thyme's whole life on Windows. It must be called on the main
// goroutine, which it locks to its OS thread for the life of the process,
// because a window belongs to the thread that created it.
func Run() int {
	runtime.LockOSThread()

	logger, closeLog := openLogger()
	defer closeLog()

	if awarenessError := setProcessPerMonitorAwareV2(); awarenessError != nil {
		logger.Warn("dpi awareness not set", "reason", eventlog.ReasonWithoutPath(awarenessError))
	}

	singleInstanceName := utf16OrPanic(`Local\` + product.Name + ".SingleInstance")
	mutex, mutexError := windows.CreateMutex(nil, false, singleInstanceName)
	if errors.Is(mutexError, windows.ERROR_ALREADY_EXISTS) {
		logger.Info("another instance is running; exiting")
		if mutex != 0 {
			windows.CloseHandle(mutex)
		}
		return 0
	}
	if mutexError != nil {
		logger.Error("single instance check failed", "reason", eventlog.ReasonWithoutPath(mutexError))
		return 1
	}
	defer windows.CloseHandle(mutex)

	activeShell = &shell{logger: logger}
	if startError := activeShell.start(); startError != nil {
		logger.Error("start failed", "reason", eventlog.ReasonWithoutPath(startError))
		return 1
	}
	loopError := pumpMessages()
	activeShell.stop()
	if loopError != nil {
		logger.Error("message loop failed", "reason", eventlog.ReasonWithoutPath(loopError))
		return 1
	}
	logger.Info("exited")
	return 0
}

func localAppDataFolder() (string, error) {
	return windows.KnownFolderPath(windows.FOLDERID_LocalAppData, 0)
}

// openLogger logs to the capped file beside the settings. If that cannot be
// opened there is nowhere else a GUI exe can report, so logging is discarded
// rather than stopping the clock.
func openLogger() (*slog.Logger, func()) {
	folder, folderError := localAppDataFolder()
	if folderError != nil {
		return slog.New(slog.NewJSONHandler(io.Discard, nil)), func() {}
	}
	logFile, openError := eventlog.OpenRotating(filepath.Join(folder, eventlog.RelativeFilePath()), eventlog.MaximumFileBytes)
	if openError != nil {
		return slog.New(slog.NewJSONHandler(io.Discard, nil)), func() {}
	}
	return slog.New(slog.NewJSONHandler(logFile, nil)), func() { logFile.Close() }
}

func (shell *shell) start() error {
	folder, folderError := localAppDataFolder()
	if folderError != nil {
		return fmt.Errorf("finding the local app data folder: %w", folderError)
	}
	shell.settingsPath = filepath.Join(folder, settings.RelativeFilePath())
	loaded, report := settings.Load(shell.settingsPath)
	shell.current, shell.overwriteAllowed = loaded, report.OverwriteAllowed
	for _, rejection := range report.Rejections {
		shell.logger.Warn("settings field rejected", "field", rejection.Field, "reason", string(rejection.Reason))
	}
	shell.loadZones()
	if _, timeZoneError := windows.GetTimeZoneInformation(&shell.startupTimeZone); timeZoneError != nil {
		shell.logger.Warn("time zone not read", "reason", eventlog.ReasonWithoutPath(timeZoneError))
	}

	renderer, rendererError := render.NewRenderer()
	if rendererError != nil {
		return rendererError
	}
	shell.renderer = renderer

	className := utf16OrPanic(product.Name + ".Widget")
	if registerError := registerWindowClass(className, windowProcedureCallback); registerError != nil {
		return fmt.Errorf("registering the window class: %w", registerError)
	}
	window, createError := createLayeredToolWindow(className, utf16OrPanic(product.Name), shell.current.Position.X, shell.current.Position.Y)
	if createError != nil {
		return fmt.Errorf("creating the window: %w", createError)
	}
	shell.window = window
	shell.dpi = dpiForWindow(window)

	exePath, exePathError := os.Executable()
	if exePathError != nil {
		return fmt.Errorf("finding the exe: %w", exePathError)
	}
	shell.exePath = exePath
	shell.noteTemporaryExe()
	shell.taskbarCreatedMessage = registerWindowMessage("TaskbarCreated")

	startPosition := win32Point{X: shell.current.Position.X, Y: shell.current.Position.Y}
	shell.redraw(&startPosition)
	shell.clampOntoAMonitor()
	shell.pinToDesktop()
	showWithoutActivating(window)
	shell.applyZOrder()
	if shell.current.ShowInTray {
		shell.addTrayIcon()
	}
	shell.updateWindowText(time.Now())
	returnMemoryToWindows()

	shell.logger.Info("started",
		"version", product.Version,
		"toolchain", runtime.Version(),
		"architecture", runtime.GOARCH,
		"dpi", shell.dpi,
		"settingsSource", string(report.Source),
		"settingsRejected", len(report.Rejections),
		"zonesShown", len(shell.zones),
		"desktopViewFound", shell.defView != 0,
		"trayRequested", shell.current.ShowInTray,
		"trayShown", shell.trayShown,
	)

	shell.tickTimer = time.AfterFunc(clockmodel.NextWake(time.Now(), shell.current.ShowSeconds), shell.postTick)
	return nil
}

// loadZones leaves out any zone that does not load and logs why by reason
// and position, never by the label or name the user wrote (datum 0.15).
func (shell *shell) loadZones() {
	shell.zones = shell.zones[:0]
	for zoneIndex, zoneSetting := range shell.current.Zones {
		zone, loadError := clockmodel.LoadZone(zoneSetting.Label, zoneSetting.TimeZone)
		var zoneLoadError *clockmodel.ZoneLoadError
		if errors.As(loadError, &zoneLoadError) {
			shell.logger.Warn("zone not loaded", "position", zoneIndex+1, "reason", string(zoneLoadError.Reason))
			continue
		}
		shell.zones = append(shell.zones, zone)
	}
}

func (shell *shell) stop() {
	if shell.tickTimer != nil {
		shell.tickTimer.Stop()
	}
	shell.removeTrayIcon()
	if shell.trayIcon != 0 {
		destroyIcon(shell.trayIcon)
		shell.trayIcon = 0
	}
	if shell.bitmap != nil {
		shell.bitmap.release()
		shell.bitmap = nil
	}
}

// postTick runs on the timer's goroutine. It never touches the window, only
// asks the GUI thread to tick.
func (shell *shell) postTick() {
	if postError := postMessage(shell.window, tickMessage); postError != nil {
		failureCount := shell.tickPostFailures.Add(1)
		if eventlog.IsWorthLogging(int(failureCount)) {
			shell.logger.Warn("tick could not be posted", "reason", eventlog.ReasonWithoutPath(postError), "count", failureCount)
		}
	}
}

// aimTimer re-aims from a fresh reading of the wall clock every time, never
// a fixed period, so drift cannot accumulate (specification 2.2).
func (shell *shell) aimTimer() {
	shell.tickTimer.Reset(clockmodel.NextWake(time.Now(), shell.current.ShowSeconds))
}

func (shell *shell) pixelScale() float64 {
	return placement.PixelScale(shell.current.ScalePercent, shell.dpi)
}

func renderTextFrom(snapshot clockmodel.Snapshot) render.Text {
	text := render.Text{Time: snapshot.Time, DateLine: snapshot.DateLine, Zones: make([]render.ZoneLine, len(snapshot.ZoneTimes))}
	for zoneIndex, zoneTime := range snapshot.ZoneTimes {
		text.Zones[zoneIndex] = render.ZoneLine{Label: zoneTime.Label, Time: zoneTime.Time}
	}
	return text
}

// redraw renders now and hands the frame to the window. moveTo nil keeps the
// window where it is.
func (shell *shell) redraw(moveTo *win32Point) {
	snapshot := clockmodel.TakeSnapshot(time.Now(), time.Local, shell.zones, shell.current.ShowSeconds)
	frame, renderError := shell.renderer.Render(renderTextFrom(snapshot), shell.pixelScale())
	if renderError != nil {
		shell.renderFailures++
		if eventlog.IsWorthLogging(shell.renderFailures) {
			shell.logger.Error("render failed", "reason", eventlog.ReasonWithoutPath(renderError), "count", shell.renderFailures)
		}
		return
	}
	if shell.bitmap == nil || shell.bitmap.width != frame.Width || shell.bitmap.height != frame.Height {
		replacement, bitmapError := newLayeredBitmap(frame.Width, frame.Height)
		if bitmapError != nil {
			shell.logger.Error("bitmap not created", "reason", eventlog.ReasonWithoutPath(bitmapError))
			return
		}
		if shell.bitmap != nil {
			shell.bitmap.release()
		}
		shell.bitmap = replacement
	}
	copy(shell.bitmap.pixels, frame.Pixels)
	shell.panelBounds = placement.Insets{
		Left: int32(frame.PanelBounds.Min.X), Top: int32(frame.PanelBounds.Min.Y),
		Right: int32(frame.Width - frame.PanelBounds.Max.X), Bottom: int32(frame.Height - frame.PanelBounds.Max.Y),
	}
	if updateError := updateLayeredWindow(shell.window, shell.bitmap, moveTo); updateError != nil {
		shell.updateLayeredWindowFailures++
		if eventlog.IsWorthLogging(shell.updateLayeredWindowFailures) {
			shell.logger.Warn("UpdateLayeredWindow failed", "reason", eventlog.ReasonWithoutPath(updateError), "count", shell.updateLayeredWindowFailures)
		}
	}
}

func asPlacementRectangle(rectangle win32Rect) placement.Rectangle {
	return placement.Rectangle{Left: rectangle.Left, Top: rectangle.Top, Right: rectangle.Right, Bottom: rectangle.Bottom}
}

// clampOntoAMonitor keeps the panel on the monitor it belongs to, or the
// nearest one if its monitor has gone. The decision is placement's; this only
// gathers the rectangles and applies the answer.
func (shell *shell) clampOntoAMonitor() {
	windowBounds, rectangleError := windowRectangle(shell.window)
	if rectangleError != nil {
		shell.logger.Warn("window rectangle unavailable", "reason", eventlog.ReasonWithoutPath(rectangleError))
		return
	}
	workAreaRects, enumerationError := monitorWorkAreas()
	if enumerationError != nil {
		shell.logger.Warn("monitors not enumerated", "reason", eventlog.ReasonWithoutPath(enumerationError))
		return
	}
	workAreas := make([]placement.Rectangle, 0, len(workAreaRects))
	for _, workArea := range workAreaRects {
		workAreas = append(workAreas, asPlacementRectangle(workArea))
	}
	current := asPlacementRectangle(windowBounds)
	placed, found := placement.Place(current, shell.panelBounds, workAreas)
	if !found || placed == current {
		return
	}
	if moveError := moveWindow(shell.window, placed.Left, placed.Top); moveError != nil {
		shell.logger.Warn("window not moved on screen", "reason", eventlog.ReasonWithoutPath(moveError))
	}
}

// rememberPosition saves where the user left the widget, once per move.
func (shell *shell) rememberPosition() {
	windowBounds, rectangleError := windowRectangle(shell.window)
	if rectangleError != nil {
		shell.logger.Warn("window rectangle unavailable", "reason", eventlog.ReasonWithoutPath(rectangleError))
		return
	}
	shell.current.Position = settings.Position{X: windowBounds.Left, Y: windowBounds.Top}
	shell.saveSettings("drag")
}

// saveSettings writes the settings file, a change outside the process, so it
// logs which surface asked (datum 0.14). A file Thyme could not understand is
// never saved over (WE3).
func (shell *shell) saveSettings(origin ActionOrigin) {
	if !shell.overwriteAllowed {
		shell.logger.Info("settings not saved: the file on disk was not understood and is kept", "origin", string(origin))
		return
	}
	if saveError := settings.Save(shell.settingsPath, shell.current); saveError != nil {
		shell.settingsSaveFailures++
		shell.logger.Warn("settings not saved", "origin", string(origin), "reason", eventlog.ReasonWithoutPath(saveError), "count", shell.settingsSaveFailures)
	}
}

var windowProcedureCallback = windows.NewCallback(func(window uintptr, message uint32, wParam uintptr, lParam uintptr) uintptr {
	if activeShell == nil || activeShell.window != window {
		return defaultWindowProcedure(window, message, wParam, lParam)
	}
	return activeShell.handleMessage(message, wParam, lParam)
})

func (shell *shell) handleMessage(message uint32, wParam uintptr, lParam uintptr) uintptr {
	// TaskbarCreated is registered at run time, so it cannot be a case label.
	if message == shell.taskbarCreatedMessage && message != 0 {
		shell.handleTaskbarCreated()
		return 0
	}
	switch message {
	case tickMessage:
		now := time.Now()
		shell.redraw(nil)
		shell.aimTimer()
		shell.ensurePinned()
		shell.updateWindowText(now)
		return 0

	case wmContextMenu:
		// A right click on the widget. The point is in screen coordinates;
		// minus one in both means the keyboard asked, and the menu opens at
		// the panel's corner instead.
		x, y := signedLowWord(lParam), signedHighWord(lParam)
		if x == -1 && y == -1 {
			if windowBounds, rectangleError := windowRectangle(shell.window); rectangleError == nil {
				x, y = windowBounds.Left+shell.panelBounds.Left, windowBounds.Top+shell.panelBounds.Top
			}
		}
		shell.showMenu(OriginWidgetMenu, x, y)
		return 0

	case trayCallbackMessage:
		shell.handleTrayEvent(wParam, lParam)
		return 0

	case wmNcHitTest:
		// Every pixel that receives the mouse is a caption, so Windows drags
		// the widget itself and Thyme needs no drag code.
		return htCaption

	case wmExitSizeMove:
		shell.clampOntoAMonitor()
		shell.rememberPosition()
		return 0

	case wmDisplayChange:
		shell.clampOntoAMonitor()
		return 0

	case wmSettingChange:
		if wParam == spiSetWorkArea {
			shell.clampOntoAMonitor()
		}
		return 0

	case wmDpiChanged:
		// The new DPI is in the low word of wParam. Windows suggests where the
		// window should go; Thyme takes the position and sizes itself. Any
		// program running as the same user can send this message, so a DPI
		// the renderer cannot draw at is refused and the last good one kept,
		// rather than leaving the widget frozen on failed renders.
		reportedDpi := int(wParam & 0xFFFF)
		if !render.IsRenderableScale(placement.PixelScale(shell.current.ScalePercent, reportedDpi)) {
			shell.rejectedDpiChanges++
			if eventlog.IsWorthLogging(shell.rejectedDpiChanges) {
				shell.logger.Warn("dpi change refused: the widget cannot be drawn at that scale", "dpi", reportedDpi, "count", shell.rejectedDpiChanges)
			}
			return 0
		}
		shell.dpi = reportedDpi
		suggested := rectangleFromLParam(lParam)
		shell.redraw(&win32Point{X: suggested.Left, Y: suggested.Top})
		returnMemoryToWindows()
		shell.logger.Info("dpi changed", "dpi", shell.dpi)
		return 0

	case wmTimeChange:
		// Go timers run on the monotonic clock and do not see a wall clock
		// jump, so the next wake is re-aimed from the new wall time.
		shell.noticeTimeZoneChange()
		shell.redraw(nil)
		shell.aimTimer()
		return 0

	case wmPowerBroadcast:
		if wParam == pbtApmResumeAutomatic || wParam == pbtApmResumeSuspend {
			shell.redraw(nil)
			shell.aimTimer()
		}
		return 1

	case wmDestroy:
		postQuitMessage()
		return 0
	}
	return defaultWindowProcedure(shell.window, message, wParam, lParam)
}

// noticeTimeZoneChange logs, once, that Windows' time zone is no longer the
// one Go read for time.Local at start. Go never rereads it, so the local time
// stays in the old zone until Thyme restarts; the log makes that visible.
func (shell *shell) noticeTimeZoneChange() {
	if shell.timeZoneChangeLogged {
		return
	}
	var currentTimeZone windows.Timezoneinformation
	if _, timeZoneError := windows.GetTimeZoneInformation(&currentTimeZone); timeZoneError != nil {
		return
	}
	if currentTimeZone != shell.startupTimeZone {
		shell.timeZoneChangeLogged = true
		shell.logger.Warn("time zone changed; the local time follows it after Thyme restarts")
	}
}

// returnMemoryToWindows is called after the rare events that rebuild the
// static layer (start, scale, seconds, DPI). A rebuild briefly needs several
// frames of scratch memory that Go would otherwise keep for the life of the
// process; a widget left running all day should hold only what it uses.
func returnMemoryToWindows() {
	debug.FreeOSMemory()
}
