package desktop

import (
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/TheScottBot/Thyme/internal/clockmodel"
	"github.com/TheScottBot/Thyme/internal/eventlog"
	"github.com/TheScottBot/Thyme/internal/placement"
	"github.com/TheScottBot/Thyme/internal/product"
	"github.com/TheScottBot/Thyme/internal/render"
	"github.com/TheScottBot/Thyme/internal/settings"
)

// screenPoint is a place on the desktop, measured from the top left in the
// platform's screen units: physical pixels on Windows, points on macOS.
type screenPoint struct {
	X int32
	Y int32
}

// platform is what the shell asks of the operating system. Every decision
// stays in shell, so it is tested with a fake on every host; an implementation
// only carries each request out. All of it runs on the GUI thread except
// PostTick, which the timer's goroutine calls.
type platform interface {
	// CreateWindow makes the hidden widget window and returns the DPI to draw
	// at. A failure stops Thyme.
	CreateWindow(position screenPoint) (int, error)
	// Show puts the window on screen without taking focus.
	Show()
	// ExeIsTemporary reports whether the exe is somewhere Start at login must
	// not point, because the path will not last.
	ExeIsTemporary() bool

	// Present shows a frame. moveTo nil keeps the window where it is. The
	// frame's pixels are valid only during the call.
	Present(frame render.Frame, moveTo *screenPoint) error
	// InsetsToScreenUnits converts insets in frame pixels to screen units.
	InsetsToScreenUnits(pixels placement.Insets) placement.Insets
	WindowBounds() (placement.Rectangle, error)
	WorkAreas() ([]placement.Rectangle, error)
	MoveWindow(position screenPoint) error
	ApplyLevel(alwaysOnTop bool) error

	// AddTrayIcon returns whether the icon is now shown, logging any failure.
	AddTrayIcon() bool
	RemoveTrayIcon()
	SetAccessibleText(text string) error
	MenuLabels() MenuLabels
	// ShowMenu blocks until the menu closes and returns the chosen command, or
	// zero if it was dismissed.
	ShowMenu(entries []MenuEntry, anchor screenPoint) MenuCommand

	StartupEnabled() (bool, error)
	ChangeStartup(change StartupChange) error

	// PostTick asks the GUI thread to call shell.tick.
	PostTick() error
	// TickHousekeeping is the platform's own work on each tick.
	TickHousekeeping()
	// NoticeTimeZoneChange logs, once, that local time is out of date.
	NoticeTimeZoneChange()
	// StartLogFields are extra key and value pairs for the start up log line.
	StartLogFields() []any
	// Close ends the message loop.
	Close()
	// Release frees what the platform holds, after the loop has ended.
	Release()
}

// shell is the one widget's state and every decision about it. Every field is
// read and written on the GUI thread only, except the atomic counter the timer
// goroutine updates.
type shell struct {
	platform platform
	logger   *slog.Logger
	renderer *render.Renderer

	settingsPath     string
	current          settings.Settings
	overwriteAllowed bool
	zones            []clockmodel.Zone

	dpi         int
	panelBounds placement.Insets
	tickTimer   *time.Timer

	trayShown      bool
	exeIsTemporary bool
	lastWindowText string

	presentFailures      int
	renderFailures       int
	rejectedDpiChanges   int
	settingsSaveFailures int
	tickPostFailures     atomic.Int64
}

// openLogger logs to the capped file beside the settings. If that cannot be
// opened there is nowhere else a GUI program can report, so logging is
// discarded rather than stopping the clock.
func openLogger(userDataFolder func() (string, error)) (*slog.Logger, func()) {
	folder, folderError := userDataFolder()
	if folderError != nil {
		return slog.New(slog.NewJSONHandler(io.Discard, nil)), func() {}
	}
	logFile, openError := eventlog.OpenRotating(filepath.Join(folder, eventlog.RelativeFilePath()), eventlog.MaximumFileBytes)
	if openError != nil {
		return slog.New(slog.NewJSONHandler(io.Discard, nil)), func() {}
	}
	return slog.New(slog.NewJSONHandler(logFile, nil)), func() { logFile.Close() }
}

// start loads the settings, makes the window and starts the clock. folder is
// the per user folder the settings live under.
func (shell *shell) start(folder string) error {
	shell.settingsPath = filepath.Join(folder, settings.RelativeFilePath())
	loaded, report := settings.Load(shell.settingsPath)
	shell.current, shell.overwriteAllowed = loaded, report.OverwriteAllowed
	for _, rejection := range report.Rejections {
		shell.logger.Warn("settings field rejected", "field", rejection.Field, "reason", string(rejection.Reason))
	}
	shell.loadZones()

	renderer, rendererError := render.NewRenderer()
	if rendererError != nil {
		return rendererError
	}
	shell.renderer = renderer

	startPosition := screenPoint{X: shell.current.Position.X, Y: shell.current.Position.Y}
	dpi, createError := shell.platform.CreateWindow(startPosition)
	if createError != nil {
		return createError
	}
	shell.dpi = dpi
	shell.exeIsTemporary = shell.platform.ExeIsTemporary()

	shell.redraw(&startPosition)
	shell.clampOntoAMonitor()
	shell.platform.Show()
	shell.applyLevel()
	if shell.current.ShowInTray {
		shell.addTrayIcon()
	}
	shell.updateWindowText(time.Now())
	returnMemoryToSystem()

	fields := []any{
		"version", product.Version,
		"toolchain", runtime.Version(),
		"architecture", runtime.GOARCH,
		"dpi", shell.dpi,
		"settingsSource", string(report.Source),
		"settingsRejected", len(report.Rejections),
		"zonesShown", len(shell.zones),
		"trayRequested", shell.current.ShowInTray,
		"trayShown", shell.trayShown,
	}
	shell.logger.Info("started", append(fields, shell.platform.StartLogFields()...)...)

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
	shell.platform.Release()
}

// postTick runs on the timer's goroutine. It never touches the window, only
// asks the GUI thread to tick.
func (shell *shell) postTick() {
	if postError := shell.platform.PostTick(); postError != nil {
		failureCount := shell.tickPostFailures.Add(1)
		if eventlog.IsWorthLogging(int(failureCount)) {
			shell.logger.Warn("tick could not be posted", "reason", eventlog.ReasonWithoutPath(postError), "count", failureCount)
		}
	}
}

// tick is the GUI thread's half of the timer.
func (shell *shell) tick() {
	now := time.Now()
	shell.redraw(nil)
	shell.aimTimer()
	shell.platform.TickHousekeeping()
	shell.updateWindowText(now)
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
func (shell *shell) redraw(moveTo *screenPoint) {
	snapshot := clockmodel.TakeSnapshot(time.Now(), time.Local, shell.zones, shell.current.ShowSeconds)
	frame, renderError := shell.renderer.Render(renderTextFrom(snapshot), shell.pixelScale())
	if renderError != nil {
		shell.renderFailures++
		if eventlog.IsWorthLogging(shell.renderFailures) {
			shell.logger.Error("render failed", "reason", eventlog.ReasonWithoutPath(renderError), "count", shell.renderFailures)
		}
		return
	}
	if presentError := shell.platform.Present(frame, moveTo); presentError != nil {
		shell.presentFailures++
		if eventlog.IsWorthLogging(shell.presentFailures) {
			shell.logger.Warn("frame not shown", "reason", eventlog.ReasonWithoutPath(presentError), "count", shell.presentFailures)
		}
		return
	}
	shell.panelBounds = shell.platform.InsetsToScreenUnits(placement.Insets{
		Left: int32(frame.PanelBounds.Min.X), Top: int32(frame.PanelBounds.Min.Y),
		Right: int32(frame.Width - frame.PanelBounds.Max.X), Bottom: int32(frame.Height - frame.PanelBounds.Max.Y),
	})
}

// clampOntoAMonitor keeps the panel on the monitor it belongs to, or the
// nearest one if its monitor has gone. The decision is placement's; this only
// gathers the rectangles and applies the answer.
func (shell *shell) clampOntoAMonitor() {
	current, rectangleError := shell.platform.WindowBounds()
	if rectangleError != nil {
		shell.logger.Warn("window rectangle unavailable", "reason", eventlog.ReasonWithoutPath(rectangleError))
		return
	}
	workAreas, enumerationError := shell.platform.WorkAreas()
	if enumerationError != nil {
		shell.logger.Warn("monitors not enumerated", "reason", eventlog.ReasonWithoutPath(enumerationError))
		return
	}
	placed, found := placement.Place(current, shell.panelBounds, workAreas)
	if !found || placed == current {
		return
	}
	if moveError := shell.platform.MoveWindow(screenPoint{X: placed.Left, Y: placed.Top}); moveError != nil {
		shell.logger.Warn("window not moved on screen", "reason", eventlog.ReasonWithoutPath(moveError))
	}
}

// rememberPosition saves where the user left the widget, once per move.
func (shell *shell) rememberPosition() {
	windowBounds, rectangleError := shell.platform.WindowBounds()
	if rectangleError != nil {
		shell.logger.Warn("window rectangle unavailable", "reason", eventlog.ReasonWithoutPath(rectangleError))
		return
	}
	shell.current.Position = settings.Position{X: windowBounds.Left, Y: windowBounds.Top}
	shell.saveSettings("drag")
}

// moveFinished runs when a drag ends.
func (shell *shell) moveFinished() {
	shell.clampOntoAMonitor()
	shell.rememberPosition()
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

// dpiChanged applies a new DPI. Any program running as the same user can send
// the Windows message this comes from, so a DPI the renderer cannot draw at is
// refused and the last good one kept, rather than leaving the widget frozen on
// failed renders. suggestedPosition is read only once the DPI is accepted, and
// is nil where the platform suggests none.
func (shell *shell) dpiChanged(reportedDpi int, suggestedPosition func() screenPoint) {
	if !render.IsRenderableScale(placement.PixelScale(shell.current.ScalePercent, reportedDpi)) {
		shell.rejectedDpiChanges++
		if eventlog.IsWorthLogging(shell.rejectedDpiChanges) {
			shell.logger.Warn("dpi change refused: the widget cannot be drawn at that scale", "dpi", reportedDpi, "count", shell.rejectedDpiChanges)
		}
		return
	}
	shell.dpi = reportedDpi
	var moveTo *screenPoint
	if suggestedPosition != nil {
		suggested := suggestedPosition()
		moveTo = &suggested
	}
	shell.redraw(moveTo)
	returnMemoryToSystem()
	shell.logger.Info("dpi changed", "dpi", shell.dpi)
}

// clockChanged runs when the wall clock or time zone is changed. Go timers run
// on the monotonic clock and do not see a wall clock jump, so the next wake is
// re-aimed from the new wall time.
func (shell *shell) clockChanged() {
	shell.platform.NoticeTimeZoneChange()
	shell.redraw(nil)
	shell.aimTimer()
}

// resumed runs after sleep.
func (shell *shell) resumed() {
	shell.redraw(nil)
	shell.aimTimer()
}

// applyLevel carries out the always on top setting.
func (shell *shell) applyLevel() {
	if levelError := shell.platform.ApplyLevel(shell.current.AlwaysOnTop); levelError != nil {
		shell.logger.Warn("z order not applied", "reason", eventlog.ReasonWithoutPath(levelError))
	}
}

func (shell *shell) addTrayIcon() {
	shell.trayShown = shell.platform.AddTrayIcon()
}

// removeTrayIcon never checks whether the icon was added, so an exit can never
// leave a dead icon behind.
func (shell *shell) removeTrayIcon() {
	shell.platform.RemoveTrayIcon()
	shell.trayShown = false
}

// updateWindowText sets the window's text to the spoken time once a minute,
// so a screen reader reading the window's name has the time to say (WD9).
func (shell *shell) updateWindowText(now time.Time) {
	spoken := product.Name + ": " + clockmodel.SpokenTime(now, time.Local)
	if spoken == shell.lastWindowText {
		return
	}
	if textError := shell.platform.SetAccessibleText(spoken); textError != nil {
		shell.logger.Warn("window text not set", "reason", eventlog.ReasonWithoutPath(textError))
		return
	}
	shell.lastWindowText = spoken
}

func (shell *shell) startAtLoginState() (bool, bool) {
	enabled, readError := shell.platform.StartupEnabled()
	if readError != nil {
		shell.logger.Warn("start at login state not read", "reason", eventlog.ReasonWithoutPath(readError))
		return false, true
	}
	return enabled, false
}

// showMenu builds the menu fresh, so the start at login tick is read from the
// operating system every time it opens. Opening a menu can raise the widget,
// so the level is reapplied afterwards.
func (shell *shell) showMenu(origin ActionOrigin, anchor screenPoint) {
	startAtLogin, unknown := shell.startAtLoginState()
	entries := BuildMenu(MenuState{
		Labels:                       shell.platform.MenuLabels(),
		Settings:                     shell.current,
		StartAtLogin:                 startAtLogin,
		StartAtLoginUnknown:          unknown,
		StartAtLoginFromTemporaryExe: shell.exeIsTemporary,
	})
	chosen := shell.platform.ShowMenu(entries, anchor)
	shell.applyLevel()
	if chosen != 0 {
		shell.perform(chosen, origin)
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
		returnMemoryToSystem()
		shell.saveSettings(origin)

	case CommandAlwaysOnTop:
		shell.current.AlwaysOnTop = !shell.current.AlwaysOnTop
		shell.logger.Info("always on top toggled", "alwaysOnTop", shell.current.AlwaysOnTop, "origin", string(origin))
		shell.applyLevel()
		shell.saveSettings(origin)

	case CommandStartAtLogin:
		enabled, _ := shell.startAtLoginState()
		change := StartupChange{Enable: !enabled, Origin: origin}
		if change.Enable && shell.exeIsTemporary {
			shell.logger.Warn("start at login not turned on: this program is not where it will stay", "origin", string(origin))
			return
		}
		if changeError := shell.platform.ChangeStartup(change); changeError != nil {
			shell.logger.Warn("start at login not changed", "enable", change.Enable, "origin", string(origin), "reason", eventlog.ReasonWithoutPath(changeError))
			return
		}
		shell.logger.Info("start at login changed", "enable", change.Enable, "origin", string(origin))

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
		shell.platform.Close()
	}
}

// changeScale applies a Scale step (WE7). The renderer rebuilds its static
// layer for the new size; the window keeps its top left corner; then it is
// clamped, since a larger widget may now hang off the screen, and saved.
func (shell *shell) changeScale(scalePercent int, origin ActionOrigin) {
	if scalePercent == shell.current.ScalePercent {
		return
	}
	shell.current.ScalePercent = scalePercent
	shell.logger.Info("scale changed", "scalePercent", scalePercent, "origin", string(origin))
	shell.redraw(nil)
	shell.clampOntoAMonitor()
	returnMemoryToSystem()
	shell.saveSettings(origin)
}

// returnMemoryToSystem is called after the rare events that rebuild the static
// layer (start, scale, seconds, DPI). A rebuild briefly needs several frames of
// scratch memory that Go would otherwise keep for the life of the process; a
// widget left running all day should hold only what it uses.
func returnMemoryToSystem() {
	debug.FreeOSMemory()
}
