package desktop

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheScottBot/Thyme/internal/eventlog"
	"github.com/TheScottBot/Thyme/internal/placement"
	"github.com/TheScottBot/Thyme/internal/render"
	"github.com/TheScottBot/Thyme/internal/settings"
)

// fakePlatform records what the shell asks for, so every decision in core.go
// is tested on every host without a window.
type fakePlatform struct {
	shell *shell

	dpi            int
	exeIsTemporary bool
	bounds         placement.Rectangle
	workAreas      []placement.Rectangle
	trayWorks      bool
	chosen         MenuCommand
	startupEnabled bool
	startupError   error

	trayAdds, trayRemoves, closes, levelCalls, presents int
	housekeepings, timeZoneNotices                      int
	lastLevel                                           bool
	moves                                               []screenPoint
	presentedMoves                                      []*screenPoint
	shownEntries                                        []MenuEntry
	startupChanges                                      []StartupChange
	accessibleTexts                                     []string
}

func (fake *fakePlatform) CreateWindow(position screenPoint) (int, error) {
	fake.bounds = placement.Rectangle{Left: position.X, Top: position.Y, Right: position.X + 100, Bottom: position.Y + 50}
	return fake.dpi, nil
}
func (fake *fakePlatform) Show()                {}
func (fake *fakePlatform) ExeIsTemporary() bool { return fake.exeIsTemporary }
func (fake *fakePlatform) Present(frame render.Frame, moveTo *screenPoint) error {
	fake.presents++
	fake.presentedMoves = append(fake.presentedMoves, moveTo)
	return nil
}
func (fake *fakePlatform) InsetsToScreenUnits(pixels placement.Insets) placement.Insets {
	return placement.Insets{}
}
func (fake *fakePlatform) WindowBounds() (placement.Rectangle, error) { return fake.bounds, nil }
func (fake *fakePlatform) WorkAreas() ([]placement.Rectangle, error)  { return fake.workAreas, nil }
func (fake *fakePlatform) MoveWindow(position screenPoint) error {
	fake.moves = append(fake.moves, position)
	return nil
}
func (fake *fakePlatform) ApplyLevel(alwaysOnTop bool) error {
	fake.levelCalls++
	fake.lastLevel = alwaysOnTop
	return nil
}
func (fake *fakePlatform) AddTrayIcon() bool {
	fake.trayAdds++
	return fake.trayWorks
}
func (fake *fakePlatform) RemoveTrayIcon() { fake.trayRemoves++ }
func (fake *fakePlatform) SetAccessibleText(text string) error {
	fake.accessibleTexts = append(fake.accessibleTexts, text)
	return nil
}
func (fake *fakePlatform) MenuLabels() MenuLabels { return macMenuLabels }
func (fake *fakePlatform) ShowMenu(entries []MenuEntry, anchor screenPoint) MenuCommand {
	fake.shownEntries = entries
	return fake.chosen
}
func (fake *fakePlatform) StartupEnabled() (bool, error) {
	return fake.startupEnabled, fake.startupError
}
func (fake *fakePlatform) ChangeStartup(change StartupChange) error {
	fake.startupChanges = append(fake.startupChanges, change)
	fake.startupEnabled = change.Enable
	return nil
}
func (fake *fakePlatform) PostTick() error       { return nil }
func (fake *fakePlatform) TickHousekeeping()     { fake.housekeepings++ }
func (fake *fakePlatform) NoticeTimeZoneChange() { fake.timeZoneNotices++ }
func (fake *fakePlatform) StartLogFields() []any { return []any{"platformField", true} }
func (fake *fakePlatform) Close()                { fake.closes++ }
func (fake *fakePlatform) Release()              {}

type shellFixture struct {
	shell    *shell
	fake     *fakePlatform
	log      *bytes.Buffer
	folder   string
	settings string
}

func newFakePlatform() *fakePlatform {
	return &fakePlatform{
		dpi:       96,
		trayWorks: true,
		workAreas: []placement.Rectangle{{Left: 0, Top: 0, Right: 4000, Bottom: 3000}},
	}
}

// startShell runs the real start against a fake, in a throwaway folder.
func startShell(t *testing.T, fake *fakePlatform, existingSettings string) shellFixture {
	t.Helper()
	folder := t.TempDir()
	settingsPath := filepath.Join(folder, settings.RelativeFilePath())
	if existingSettings != "" {
		if mkdirError := os.MkdirAll(filepath.Dir(settingsPath), 0o755); mkdirError != nil {
			t.Fatal(mkdirError)
		}
		if writeError := os.WriteFile(settingsPath, []byte(existingSettings), 0o644); writeError != nil {
			t.Fatal(writeError)
		}
	}
	logged := &bytes.Buffer{}
	started := &shell{platform: fake, logger: slog.New(slog.NewJSONHandler(logged, nil))}
	fake.shell = started
	if startError := started.start(folder); startError != nil {
		t.Fatalf("start failed: %v", startError)
	}
	t.Cleanup(started.stop)
	return shellFixture{shell: started, fake: fake, log: logged, folder: folder, settings: settingsPath}
}

func (fixture shellFixture) expectLogged(t *testing.T, fragment string) {
	t.Helper()
	if !strings.Contains(fixture.log.String(), fragment) {
		t.Errorf("expected the log to hold %q, got:\n%s", fragment, fixture.log.String())
	}
}

func (fixture shellFixture) savedSettings(t *testing.T) settings.Settings {
	t.Helper()
	loaded, report := settings.Load(fixture.settings)
	if len(report.Rejections) != 0 {
		t.Fatalf("saved settings were rejected: %+v", report.Rejections)
	}
	return loaded
}

func TestStart_DrawsMovesAndReportsThePlatformsFields(t *testing.T) {
	fixture := startShell(t, newFakePlatform(), "")

	first := fixture.fake.presentedMoves[0]
	defaults := settings.Defaults()
	if first == nil || first.X != defaults.Position.X || first.Y != defaults.Position.Y {
		t.Errorf("expected the first frame to move to the saved position, got %+v", first)
	}
	if fixture.fake.trayAdds != 1 || !fixture.shell.trayShown {
		t.Errorf("expected the tray icon added and shown, adds %d", fixture.fake.trayAdds)
	}
	if len(fixture.fake.accessibleTexts) != 1 {
		t.Errorf("expected the spoken time set once, got %v", fixture.fake.accessibleTexts)
	}
	fixture.expectLogged(t, `"platformField":true`)
}

func TestStart_ReportsATrayIconThatCouldNotBeAdded(t *testing.T) {
	fake := newFakePlatform()
	fake.trayWorks = false

	fixture := startShell(t, fake, "")

	if fixture.shell.trayShown {
		t.Error("expected the tray icon reported as not shown")
	}
}

func TestPerform_TogglesApplySaveAndNameTheirSurface(t *testing.T) {
	fixture := startShell(t, newFakePlatform(), "")
	defaults := settings.Defaults()

	fixture.shell.perform(CommandSeconds, OriginWidgetMenu)
	fixture.shell.perform(CommandAlwaysOnTop, OriginTrayMenu)
	fixture.shell.perform(CommandShowInTray, OriginTrayMenu)

	saved := fixture.savedSettings(t)
	if saved.ShowSeconds == defaults.ShowSeconds || saved.AlwaysOnTop == defaults.AlwaysOnTop || saved.ShowInTray == defaults.ShowInTray {
		t.Errorf("expected all three toggles saved, got %+v", saved)
	}
	if fixture.fake.lastLevel != saved.AlwaysOnTop {
		t.Errorf("expected the level applied, got %v", fixture.fake.lastLevel)
	}
	if fixture.fake.trayRemoves != 1 || fixture.shell.trayShown {
		t.Errorf("expected the tray icon removed, removes %d", fixture.fake.trayRemoves)
	}
	fixture.expectLogged(t, `"msg":"seconds toggled","showSeconds":false,"origin":"widget menu"`)
	fixture.expectLogged(t, `"origin":"tray menu"`)
}

func TestPerform_ScaleCommandRedrawsAndSaves(t *testing.T) {
	fixture := startShell(t, newFakePlatform(), "")
	presentsBefore := fixture.fake.presents

	fixture.shell.perform(commandScaleFirst+3, OriginWidgetMenu)

	if saved := fixture.savedSettings(t); saved.ScalePercent != 150 {
		t.Errorf("expected scale 150 saved, got %d", saved.ScalePercent)
	}
	if fixture.fake.presents != presentsBefore+1 {
		t.Errorf("expected one redraw, got %d", fixture.fake.presents-presentsBefore)
	}
}

func TestPerform_ExitSavesRemovesTheIconAndCloses(t *testing.T) {
	fixture := startShell(t, newFakePlatform(), "")

	fixture.shell.perform(CommandExit, OriginTrayMenu)

	if fixture.fake.closes != 1 || fixture.fake.trayRemoves != 1 {
		t.Errorf("expected one close and one removal, got %d and %d", fixture.fake.closes, fixture.fake.trayRemoves)
	}
	if _, statError := os.Stat(fixture.settings); statError != nil {
		t.Errorf("expected the settings saved on exit: %v", statError)
	}
}

// A file Thyme could not understand is the user's work, so no change made
// through the menu may ever save over it (WE3).
func TestPerform_NeverSavesOverAFileItCouldNotUnderstand(t *testing.T) {
	brokenFile := `{"version": 99}`
	fixture := startShell(t, newFakePlatform(), brokenFile)

	fixture.shell.perform(CommandSeconds, OriginWidgetMenu)
	fixture.shell.perform(CommandExit, OriginWidgetMenu)

	contents, readError := os.ReadFile(fixture.settings)
	if readError != nil {
		t.Fatal(readError)
	}
	if string(contents) != brokenFile {
		t.Errorf("expected the file untouched, got %q", contents)
	}
}

func TestPerform_StartAtLoginFollowsWhatTheSystemReports(t *testing.T) {
	fake := newFakePlatform()
	fake.startupEnabled = true
	fixture := startShell(t, fake, "")

	fixture.shell.perform(CommandStartAtLogin, OriginTrayMenu)

	if len(fake.startupChanges) != 1 || fake.startupChanges[0] != (StartupChange{Enable: false, Origin: OriginTrayMenu}) {
		t.Errorf("expected one change turning it off from the tray, got %+v", fake.startupChanges)
	}
}

func TestPerform_TemporaryExeCanTurnStartAtLoginOffButNotOn(t *testing.T) {
	fake := newFakePlatform()
	fake.exeIsTemporary = true
	fixture := startShell(t, fake, "")

	fixture.shell.perform(CommandStartAtLogin, OriginWidgetMenu)
	if len(fake.startupChanges) != 0 {
		t.Fatalf("expected turning on to be refused, got %+v", fake.startupChanges)
	}

	fake.startupEnabled = true
	fixture.shell.perform(CommandStartAtLogin, OriginWidgetMenu)
	if len(fake.startupChanges) != 1 || fake.startupChanges[0].Enable {
		t.Errorf("expected turning off to be allowed, got %+v", fake.startupChanges)
	}
}

func TestShowMenu_GreysStartAtLoginWhenItsStateCannotBeRead(t *testing.T) {
	fake := newFakePlatform()
	fake.startupError = errors.New("no answer")
	fixture := startShell(t, fake, "")

	fixture.shell.showMenu(OriginWidgetMenu, screenPoint{})

	entry := findEntry(t, fake.shownEntries, CommandStartAtLogin)
	if entry.Enabled || entry.Checked {
		t.Errorf("expected greyed and unchecked, got %+v", entry)
	}
}

func TestShowMenu_DismissedDoesNothingButReappliesTheLevel(t *testing.T) {
	fixture := startShell(t, newFakePlatform(), "")
	levelCallsBefore, before := fixture.fake.levelCalls, fixture.shell.current

	fixture.shell.showMenu(OriginWidgetMenu, screenPoint{})

	if fixture.fake.levelCalls != levelCallsBefore+1 {
		t.Errorf("expected the level reapplied once, got %d", fixture.fake.levelCalls-levelCallsBefore)
	}
	if fixture.shell.current.ShowSeconds != before.ShowSeconds {
		t.Error("expected a dismissed menu to change nothing")
	}
}

func TestShowMenu_PerformsTheChosenCommandFromThatSurface(t *testing.T) {
	fake := newFakePlatform()
	fake.chosen = CommandSeconds
	fixture := startShell(t, fake, "")

	fixture.shell.showMenu(OriginTrayMenu, screenPoint{})

	fixture.expectLogged(t, `"msg":"seconds toggled","showSeconds":false,"origin":"tray menu"`)
}

func TestMoveFinished_ClampsThenRemembersWhereTheWindowIs(t *testing.T) {
	fixture := startShell(t, newFakePlatform(), "")
	fixture.fake.bounds = placement.Rectangle{Left: 200, Top: 300, Right: 300, Bottom: 350}

	fixture.shell.moveFinished()

	if saved := fixture.savedSettings(t); saved.Position != (settings.Position{X: 200, Y: 300}) {
		t.Errorf("expected the position saved, got %+v", saved.Position)
	}
}

func TestClamp_PullsAWindowBackOnScreen(t *testing.T) {
	fixture := startShell(t, newFakePlatform(), "")
	fixture.fake.bounds = placement.Rectangle{Left: 3950, Top: 100, Right: 4050, Bottom: 150}
	fixture.fake.moves = nil

	fixture.shell.clampOntoAMonitor()

	if len(fixture.fake.moves) != 1 || fixture.fake.moves[0] != (screenPoint{X: 3900, Y: 100}) {
		t.Errorf("expected one move back on screen, got %+v", fixture.fake.moves)
	}
}

// The suggested position comes from a pointer in a window message, so it must
// not be read unless the DPI itself was believable.
func TestDpiChanged_RefusesADpiItCannotDrawAndKeepsTheLastGoodOne(t *testing.T) {
	fixture := startShell(t, newFakePlatform(), "")
	suggestionRead := false

	fixture.shell.dpiChanged(0, func() screenPoint {
		suggestionRead = true
		return screenPoint{}
	})

	if fixture.shell.dpi != 96 || suggestionRead {
		t.Errorf("expected dpi 96 kept and the suggestion unread, got %d, read %v", fixture.shell.dpi, suggestionRead)
	}
}

func TestDpiChanged_AcceptsAGoodDpiWithOrWithoutASuggestion(t *testing.T) {
	fixture := startShell(t, newFakePlatform(), "")

	fixture.shell.dpiChanged(192, nil)
	if fixture.shell.dpi != 192 || fixture.fake.presentedMoves[len(fixture.fake.presentedMoves)-1] != nil {
		t.Errorf("expected dpi 192 with no move, got %d", fixture.shell.dpi)
	}

	fixture.shell.dpiChanged(144, func() screenPoint { return screenPoint{X: 7, Y: 9} })
	last := fixture.fake.presentedMoves[len(fixture.fake.presentedMoves)-1]
	if fixture.shell.dpi != 144 || last == nil || *last != (screenPoint{X: 7, Y: 9}) {
		t.Errorf("expected dpi 144 moved to the suggestion, got %d, %+v", fixture.shell.dpi, last)
	}
}

func TestTick_RedrawsDoesThePlatformsWorkAndSetsTheSpokenTimeOncePerMinute(t *testing.T) {
	fake := newFakePlatform()
	fixture := startShell(t, fake, "")
	presentsBefore, textsBefore := fake.presents, len(fake.accessibleTexts)

	fixture.shell.tick()
	fixture.shell.tick()

	if fake.presents != presentsBefore+2 || fake.housekeepings != 2 {
		t.Errorf("expected two redraws and two housekeepings, got %d and %d", fake.presents-presentsBefore, fake.housekeepings)
	}
	if added := len(fake.accessibleTexts) - textsBefore; added > 1 {
		t.Errorf("expected the spoken time set at most once, got %d", added)
	}
}

func TestClockChanged_NoticesTheZoneAndRedraws(t *testing.T) {
	fake := newFakePlatform()
	fixture := startShell(t, fake, "")
	presentsBefore := fake.presents

	fixture.shell.clockChanged()

	if fake.timeZoneNotices != 1 || fake.presents != presentsBefore+1 {
		t.Errorf("expected one notice and one redraw, got %d and %d", fake.timeZoneNotices, fake.presents-presentsBefore)
	}
}

func TestResumed_Redraws(t *testing.T) {
	fake := newFakePlatform()
	fixture := startShell(t, fake, "")
	presentsBefore := fake.presents

	fixture.shell.resumed()

	if fake.presents != presentsBefore+1 {
		t.Errorf("expected one redraw, got %d", fake.presents-presentsBefore)
	}
}

// Logging must never stop the clock, so a log with nowhere to go is discarded.
func TestOpenLogger_DiscardsWhenThereIsNoFolder(t *testing.T) {
	logger, closeLog := openLogger(func() (string, error) { return "", errors.New("no folder") })
	defer closeLog()

	logger.Info("goes nowhere")
}

func TestOpenLogger_WritesBesideTheSettings(t *testing.T) {
	folder := t.TempDir()
	logger, closeLog := openLogger(func() (string, error) { return folder, nil })

	logger.Info("kept")
	closeLog()

	contents, readError := os.ReadFile(filepath.Join(folder, eventlog.RelativeFilePath()))
	if readError != nil || !strings.Contains(string(contents), "kept") {
		t.Errorf("expected the record in the log, got %q, %v", contents, readError)
	}
}

func TestStop_LeavesNoTrayIconBehind(t *testing.T) {
	fixture := startShell(t, newFakePlatform(), "")

	fixture.shell.stop()

	if fixture.fake.trayRemoves == 0 {
		t.Error("expected the tray icon removed on stop")
	}
}
