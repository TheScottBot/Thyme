package desktop

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"golang.org/x/sys/windows"

	"github.com/TheScottBot/Thyme/internal/eventlog"
	"github.com/TheScottBot/Thyme/internal/placement"
	"github.com/TheScottBot/Thyme/internal/product"
	"github.com/TheScottBot/Thyme/internal/render"
)

// tickMessage is posted by the timer goroutine. Only the GUI thread touches
// the window; the timer only asks it to.
const tickMessage = wmApp + 1

// win32Platform is the Windows half of the shell: it carries out what shell
// decides and turns window messages into shell's events. Every field is read
// and written on the GUI thread only.
type win32Platform struct {
	shell  *shell
	window uintptr
	bitmap *layeredBitmap

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
}

// activePlatform is reached by the window procedure, which Windows calls with
// no user data of ours; there is exactly one window per process.
var activePlatform *win32Platform

// Run is Thyme's whole life on Windows. It must be called on the main
// goroutine, which it locks to its OS thread for the life of the process,
// because a window belongs to the thread that created it.
func Run() int {
	runtime.LockOSThread()

	logger, closeLog := openLogger(localAppDataFolder)
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

	folder, folderError := localAppDataFolder()
	if folderError != nil {
		logger.Error("start failed", "reason", eventlog.ReasonWithoutPath(fmt.Errorf("finding the local app data folder: %w", folderError)))
		return 1
	}
	activePlatform = &win32Platform{shell: &shell{logger: logger}}
	activePlatform.shell.platform = activePlatform
	if startError := activePlatform.shell.start(folder); startError != nil {
		logger.Error("start failed", "reason", eventlog.ReasonWithoutPath(startError))
		return 1
	}
	loopError := pumpMessages()
	activePlatform.shell.stop()
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

func (platform *win32Platform) CreateWindow(position screenPoint) (int, error) {
	if _, timeZoneError := windows.GetTimeZoneInformation(&platform.startupTimeZone); timeZoneError != nil {
		platform.shell.logger.Warn("time zone not read", "reason", eventlog.ReasonWithoutPath(timeZoneError))
	}

	className := utf16OrPanic(product.Name + ".Widget")
	if registerError := registerWindowClass(className, windowProcedureCallback); registerError != nil {
		return 0, fmt.Errorf("registering the window class: %w", registerError)
	}
	window, createError := createLayeredToolWindow(className, utf16OrPanic(product.Name), position.X, position.Y)
	if createError != nil {
		return 0, fmt.Errorf("creating the window: %w", createError)
	}
	platform.window = window

	exePath, exePathError := os.Executable()
	if exePathError != nil {
		return 0, fmt.Errorf("finding the exe: %w", exePathError)
	}
	platform.exePath = exePath
	platform.taskbarCreatedMessage = registerWindowMessage("TaskbarCreated")
	return dpiForWindow(window), nil
}

func (platform *win32Platform) Show() {
	platform.pinToDesktop()
	showWithoutActivating(platform.window)
}

func (platform *win32Platform) StartLogFields() []any {
	return []any{"desktopViewFound", platform.defView != 0}
}

func (platform *win32Platform) PostTick() error {
	return postMessage(platform.window, tickMessage)
}

func (platform *win32Platform) TickHousekeeping() {
	platform.ensurePinned()
}

func (platform *win32Platform) Close() {
	destroyWindow(platform.window)
}

func (platform *win32Platform) Release() {
	if platform.trayIcon != 0 {
		destroyIcon(platform.trayIcon)
		platform.trayIcon = 0
	}
	if platform.bitmap != nil {
		platform.bitmap.release()
		platform.bitmap = nil
	}
}

func (platform *win32Platform) Present(frame render.Frame, moveTo *screenPoint) error {
	if platform.bitmap == nil || platform.bitmap.width != frame.Width || platform.bitmap.height != frame.Height {
		replacement, bitmapError := newLayeredBitmap(frame.Width, frame.Height)
		if bitmapError != nil {
			return fmt.Errorf("creating the bitmap: %w", bitmapError)
		}
		if platform.bitmap != nil {
			platform.bitmap.release()
		}
		platform.bitmap = replacement
	}
	copy(platform.bitmap.pixels, frame.Pixels)
	var position *win32Point
	if moveTo != nil {
		position = &win32Point{X: moveTo.X, Y: moveTo.Y}
	}
	return updateLayeredWindow(platform.window, platform.bitmap, position)
}

// InsetsToScreenUnits is the identity: Windows' screen units are pixels.
func (platform *win32Platform) InsetsToScreenUnits(pixels placement.Insets) placement.Insets {
	return pixels
}

func asPlacementRectangle(rectangle win32Rect) placement.Rectangle {
	return placement.Rectangle{Left: rectangle.Left, Top: rectangle.Top, Right: rectangle.Right, Bottom: rectangle.Bottom}
}

func (platform *win32Platform) WindowBounds() (placement.Rectangle, error) {
	windowBounds, rectangleError := windowRectangle(platform.window)
	if rectangleError != nil {
		return placement.Rectangle{}, rectangleError
	}
	return asPlacementRectangle(windowBounds), nil
}

func (platform *win32Platform) WorkAreas() ([]placement.Rectangle, error) {
	workAreaRects, enumerationError := monitorWorkAreas()
	if enumerationError != nil {
		return nil, enumerationError
	}
	workAreas := make([]placement.Rectangle, 0, len(workAreaRects))
	for _, workArea := range workAreaRects {
		workAreas = append(workAreas, asPlacementRectangle(workArea))
	}
	return workAreas, nil
}

func (platform *win32Platform) MoveWindow(position screenPoint) error {
	return moveWindow(platform.window, position.X, position.Y)
}

var windowProcedureCallback = windows.NewCallback(func(window uintptr, message uint32, wParam uintptr, lParam uintptr) uintptr {
	if activePlatform == nil || activePlatform.window != window {
		return defaultWindowProcedure(window, message, wParam, lParam)
	}
	return activePlatform.handleMessage(message, wParam, lParam)
})

func (platform *win32Platform) handleMessage(message uint32, wParam uintptr, lParam uintptr) uintptr {
	shell := platform.shell
	// TaskbarCreated is registered at run time, so it cannot be a case label.
	if message == platform.taskbarCreatedMessage && message != 0 {
		platform.handleTaskbarCreated()
		return 0
	}
	switch message {
	case tickMessage:
		shell.tick()
		return 0

	case wmContextMenu:
		// A right click on the widget. The point is in screen coordinates;
		// minus one in both means the keyboard asked, and the menu opens at
		// the panel's corner instead.
		x, y := signedLowWord(lParam), signedHighWord(lParam)
		if x == -1 && y == -1 {
			if windowBounds, rectangleError := windowRectangle(platform.window); rectangleError == nil {
				x, y = windowBounds.Left+shell.panelBounds.Left, windowBounds.Top+shell.panelBounds.Top
			}
		}
		shell.showMenu(OriginWidgetMenu, screenPoint{X: x, Y: y})
		return 0

	case trayCallbackMessage:
		platform.handleTrayEvent(wParam, lParam)
		return 0

	case wmNcHitTest:
		// Every pixel that receives the mouse is a caption, so Windows drags
		// the widget itself and Thyme needs no drag code.
		return htCaption

	case wmExitSizeMove:
		shell.moveFinished()
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
		// window should go; Thyme takes the position and sizes itself.
		shell.dpiChanged(int(wParam&0xFFFF), func() screenPoint {
			suggested := rectangleFromLParam(lParam)
			return screenPoint{X: suggested.Left, Y: suggested.Top}
		})
		return 0

	case wmTimeChange:
		shell.clockChanged()
		return 0

	case wmPowerBroadcast:
		if wParam == pbtApmResumeAutomatic || wParam == pbtApmResumeSuspend {
			shell.resumed()
		}
		return 1

	case wmDestroy:
		postQuitMessage()
		return 0
	}
	return defaultWindowProcedure(platform.window, message, wParam, lParam)
}

// NoticeTimeZoneChange logs, once, that Windows' time zone is no longer the
// one Go read for time.Local at start. Go never rereads it, so the local time
// stays in the old zone until Thyme restarts; the log makes that visible.
func (platform *win32Platform) NoticeTimeZoneChange() {
	if platform.timeZoneChangeLogged {
		return
	}
	var currentTimeZone windows.Timezoneinformation
	if _, timeZoneError := windows.GetTimeZoneInformation(&currentTimeZone); timeZoneError != nil {
		return
	}
	if currentTimeZone != platform.startupTimeZone {
		platform.timeZoneChangeLogged = true
		platform.shell.logger.Warn("time zone changed; the local time follows it after Thyme restarts")
	}
}
