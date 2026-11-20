//go:build darwin && cgo

package desktop

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework QuartzCore -framework ServiceManagement
#include <stdlib.h>
#include "appkit_darwin.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/TheScottBot/Thyme/internal/eventlog"
	"github.com/TheScottBot/Thyme/internal/placement"
	"github.com/TheScottBot/Thyme/internal/product"
	"github.com/TheScottBot/Thyme/internal/render"
)

const (
	// statusItemPoints is the icon's size in the menu bar, and statusItemPixels
	// what is drawn for it, so it is crisp on a Retina screen.
	statusItemPoints = 18
	statusItemPixels = 36

	// maximumScreens bounds the work area list; more than this are not seen
	// by the clamp, which then treats the widget as being on one of the first.
	maximumScreens = 32

	// designDpi is the DPI at which one screen point is one design unit, so a
	// widget is the same size in points on any Mac.
	designDpi = 96
)

// init locks the main goroutine to the process's first thread, which is the
// only one AppKit may run on. Run is too late to do this.
func init() {
	runtime.LockOSThread()
}

// macPlatform is the macOS half of the shell. Every field is read and written
// on the main thread only.
type macPlatform struct {
	shell *shell

	startupTimeZone      string
	timeZoneChangeLogged bool
}

// activePlatform is reached by the callbacks from Objective-C; there is
// exactly one window per process.
var activePlatform *macPlatform

// Run is Thyme's whole life on macOS. It must be called on the main goroutine.
func Run() int {
	logger, closeLog := openLogger(os.UserConfigDir)
	defer closeLog()

	folder, folderError := os.UserConfigDir()
	if folderError != nil {
		logger.Error("start failed", "reason", eventlog.ReasonWithoutPath(fmt.Errorf("finding the user data folder: %w", folderError)))
		return 1
	}
	release, alreadyRunning, lockError := holdSingleInstanceLock(folder)
	if lockError != nil {
		logger.Error("single instance check failed", "reason", eventlog.ReasonWithoutPath(lockError))
		return 1
	}
	if alreadyRunning {
		logger.Info("another instance is running; exiting")
		return 0
	}
	defer release()

	C.thymeInitApplication()
	activePlatform = &macPlatform{shell: &shell{logger: logger}}
	activePlatform.shell.platform = activePlatform
	if startError := activePlatform.shell.start(folder); startError != nil {
		logger.Error("start failed", "reason", eventlog.ReasonWithoutPath(startError))
		return 1
	}
	C.thymeRunApplication()
	activePlatform.shell.stop()
	logger.Info("exited")
	return 0
}

// holdSingleInstanceLock takes an exclusive lock on a file beside the settings.
// The kernel drops it when the process ends, so a crash leaves nothing stale.
func holdSingleInstanceLock(folder string) (func(), bool, error) {
	productFolder := filepath.Join(folder, product.Name)
	if mkdirError := os.MkdirAll(productFolder, 0o755); mkdirError != nil {
		return nil, false, fmt.Errorf("creating the settings folder: %w", mkdirError)
	}
	lockFile, openError := os.OpenFile(filepath.Join(productFolder, "thyme.lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if openError != nil {
		return nil, false, fmt.Errorf("opening the lock file: %w", openError)
	}
	if lockError := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); lockError != nil {
		lockFile.Close()
		if errors.Is(lockError, syscall.EWOULDBLOCK) {
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("locking the lock file: %w", lockError)
	}
	return func() { lockFile.Close() }, false, nil
}

func dpiFromBackingScale(scale C.double) int {
	return int(math.Round(float64(scale) * designDpi))
}

func (platform *macPlatform) backingScale() float64 {
	return float64(platform.shell.dpi) / designDpi
}

func (platform *macPlatform) CreateWindow(position screenPoint) (int, error) {
	platform.startupTimeZone = systemTimeZoneName()
	return dpiFromBackingScale(C.thymeCreateWindow(C.int(position.X), C.int(position.Y))), nil
}

func (platform *macPlatform) Show() {
	C.thymeShowWindow()
}

func (platform *macPlatform) StartLogFields() []any { return nil }

func (platform *macPlatform) PostTick() error {
	C.thymePostTick()
	return nil
}

// TickHousekeeping has nothing to do: unlike Windows, nothing Thyme depends on
// can vanish between ticks.
func (platform *macPlatform) TickHousekeeping() {}

func (platform *macPlatform) Close() {
	C.thymeStopApplication()
}

func (platform *macPlatform) Release() {}

func (platform *macPlatform) Present(frame render.Frame, moveTo *screenPoint) error {
	var hasMove, moveX, moveY C.int
	if moveTo != nil {
		hasMove, moveX, moveY = 1, C.int(moveTo.X), C.int(moveTo.Y)
	}
	result := C.thymePresent((*C.uint8_t)(unsafe.Pointer(&frame.Pixels[0])), C.int(frame.Width), C.int(frame.Height), C.double(platform.backingScale()), hasMove, moveX, moveY)
	if result != 0 {
		return errors.New("the frame could not be made into an image")
	}
	return nil
}

func (platform *macPlatform) InsetsToScreenUnits(pixels placement.Insets) placement.Insets {
	toPoints := func(value int32) int32 { return int32(math.Round(float64(value) / platform.backingScale())) }
	return placement.Insets{Left: toPoints(pixels.Left), Top: toPoints(pixels.Top), Right: toPoints(pixels.Right), Bottom: toPoints(pixels.Bottom)}
}

func asPlacementRectangle(rectangle C.ThymeRect) placement.Rectangle {
	return placement.Rectangle{Left: int32(rectangle.left), Top: int32(rectangle.top), Right: int32(rectangle.right), Bottom: int32(rectangle.bottom)}
}

func (platform *macPlatform) WindowBounds() (placement.Rectangle, error) {
	return asPlacementRectangle(C.thymeWindowBounds()), nil
}

func (platform *macPlatform) WorkAreas() ([]placement.Rectangle, error) {
	var areas [maximumScreens]C.ThymeRect
	count := int(C.thymeWorkAreas(&areas[0], maximumScreens))
	workAreas := make([]placement.Rectangle, 0, count)
	for _, area := range areas[:count] {
		workAreas = append(workAreas, asPlacementRectangle(area))
	}
	return workAreas, nil
}

func (platform *macPlatform) MoveWindow(position screenPoint) error {
	C.thymeMoveWindow(C.int(position.X), C.int(position.Y))
	return nil
}

func (platform *macPlatform) ApplyLevel(alwaysOnTop bool) error {
	C.thymeApplyLevel(cBool(alwaysOnTop))
	return nil
}

func cBool(value bool) C.int {
	if value {
		return 1
	}
	return 0
}

func (platform *macPlatform) AddTrayIcon() bool {
	logger := platform.shell.logger
	iconFrame, renderError := render.RenderTrayIcon(statusItemPixels)
	if renderError != nil {
		logger.Warn("tray icon not drawn", "reason", eventlog.ReasonWithoutPath(renderError))
		return false
	}
	result := C.thymeAddStatusItem((*C.uint8_t)(unsafe.Pointer(&iconFrame.Pixels[0])), C.int(iconFrame.Width), C.int(iconFrame.Height), statusItemPoints)
	if result != 0 {
		logger.Warn("tray icon not created")
		return false
	}
	return true
}

func (platform *macPlatform) RemoveTrayIcon() {
	C.thymeRemoveStatusItem()
}

func (platform *macPlatform) SetAccessibleText(text string) error {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	C.thymeSetAccessibleText(cText)
	return nil
}

func (platform *macPlatform) MenuLabels() MenuLabels {
	return macMenuLabels
}

func buildNativeMenu(title string, entries []MenuEntry) unsafe.Pointer {
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	menu := C.thymeMenuCreate(cTitle)
	for _, entry := range entries {
		label := C.CString(entry.Label)
		switch entry.Kind {
		case MenuSeparator:
			C.thymeMenuAddSeparator(menu)
		case MenuSubmenu:
			C.thymeMenuAddSubmenu(menu, buildNativeMenu(entry.Label, entry.Children), label)
		default:
			C.thymeMenuAddItem(menu, label, C.int(entry.Command), cBool(entry.Checked), cBool(entry.Enabled))
		}
		C.free(unsafe.Pointer(label))
	}
	return menu
}

func (platform *macPlatform) ShowMenu(entries []MenuEntry, anchor screenPoint) MenuCommand {
	return MenuCommand(C.thymeMenuShow(buildNativeMenu(product.Name, entries), C.int(anchor.X), C.int(anchor.Y)))
}

func systemTimeZoneName() string {
	name := C.thymeSystemTimeZoneName()
	defer C.free(unsafe.Pointer(name))
	return C.GoString(name)
}

// NoticeTimeZoneChange logs, once, that the system's time zone is no longer
// the one Go read for time.Local at start. Go never rereads it, so the local
// time stays in the old zone until Thyme restarts; the log makes that visible.
func (platform *macPlatform) NoticeTimeZoneChange() {
	if platform.timeZoneChangeLogged || systemTimeZoneName() == platform.startupTimeZone {
		return
	}
	platform.timeZoneChangeLogged = true
	platform.shell.logger.Warn("time zone changed; the local time follows it after Thyme restarts")
}

//export thymeGoTick
func thymeGoTick() {
	activePlatform.shell.tick()
}

//export thymeGoContextMenu
func thymeGoContextMenu(x C.int, y C.int) {
	activePlatform.shell.showMenu(OriginWidgetMenu, screenPoint{X: int32(x), Y: int32(y)})
}

//export thymeGoStatusItemClicked
func thymeGoStatusItemClicked(x C.int, y C.int) {
	activePlatform.shell.showMenu(OriginTrayMenu, screenPoint{X: int32(x), Y: int32(y)})
}

//export thymeGoMoveFinished
func thymeGoMoveFinished() {
	activePlatform.shell.moveFinished()
}

//export thymeGoScreensChanged
func thymeGoScreensChanged() {
	activePlatform.shell.clampOntoAMonitor()
}

//export thymeGoBackingChanged
func thymeGoBackingChanged(scale C.double) {
	if dpi := dpiFromBackingScale(scale); dpi != activePlatform.shell.dpi {
		// A new scale can change the widget's size in points by rounding, and
		// the shell clamped it with the old size.
		activePlatform.shell.dpiChanged(dpi, nil)
		activePlatform.shell.clampOntoAMonitor()
	}
}

//export thymeGoClockChanged
func thymeGoClockChanged() {
	activePlatform.shell.clockChanged()
}

//export thymeGoResumed
func thymeGoResumed() {
	activePlatform.shell.resumed()
}
