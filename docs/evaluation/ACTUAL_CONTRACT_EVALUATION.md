# Actual contract evaluation

The Plan stage record for Thyme, per ENGINEERING_DATUM.md version 2 (20 September
2026), Part 1, Stage one. Every claim below says whether it was read from source
or run for real. Phase records are history: later corrections are added as dated
notes, never by rewriting.

## WE1 Plan, 13 October 2026

Machine used for every experiment: Windows 11 Home 10.0.26200, arm64, Git for
Windows bash. Scratch work lived outside the repository and is not tracked.

### Decisions taken by the author

| Id | Decision | Recorded |
|---|---|---|
| WD1 | Clean room. The specification is the only input; the agent does not open DeskTick. Licence MIT, holder TheScottBot. | 13 October 2026 |
| WD2 | Name Thyme. Module path `github.com/TheScottBot/Thyme`, casing as the repository. | 13 October 2026 |
| WD10 | Windows 10 and 11, amd64 and arm64. | 13 October 2026 |

Consequence of WD1: no DeskTick file has been or will be opened. Every behaviour
the specification attributes to DeskTick is taken from the specification's own
wording, and where that wording is not enough the gap is raised as a question,
not filled from DeskTick. PROVENANCE.md is not required (nothing is imported or
vendored); README.md states the clean room origin instead.

### Go toolchain

- Source: `https://go.dev/dl/?mode=json`, read 13 October 2026. Latest stable is
  go1.27.2.
- The machine had go1.27.0 installed. `GOTOOLCHAIN=go1.27.2 go version` downloaded
  and reported `go version go1.27.2 windows/arm64`. Run for real.
- Conclusion accepted: pin go1.27.2 through the `go` directive in go.mod, so a
  newer local toolchain is not silently used to build a reproduction report.

### Modules

Read with `go list -m -versions` and `go list -m -json` through
`https://proxy.golang.org`, 13 October 2026, then the downloaded sources read
directly.

| Module | Version | Published | Declares go |
|---|---|---|---|
| golang.org/x/sys | v0.49.0 | 2026-10-06 | 1.26.0 |
| golang.org/x/image | v0.47.0 | 2026-10-09 | 1.26.0 |

The x/image subpackages actually needed (opentype, vector) are evaluated in WE4,
not here.

### Win32 reachability

Method: grep of `golang.org/x/sys@v0.49.0/windows/*.go` for exported functions,
validated first against names known to exist (`MessageBox`, `IsWindow`,
`GetForegroundWindow` all found), so an absence is a real absence. Then a probe
program built with `CGO_ENABLED=0` resolved every export with
`windows.NewLazySystemDLL(library).NewProc(name).Find()` on this machine. Run for
real: 45 of 45 exports found, exit status 0.

Exposed directly by x/sys/windows v0.49.0 (read from source):

| Need | x/sys/windows name |
|---|---|
| Enumerate top level windows for the WorkerW search | `EnumWindows` |
| Single instance mutex | `CreateMutex`, with `ERROR_ALREADY_EXISTS` |
| Exe path for the Run value | `GetModuleFileName` (or `os.Executable`, to be chosen in WE6) |
| `%LOCALAPPDATA%` | `KnownFolderPath(FOLDERID_LocalAppData, flags)` |
| Run key | package `golang.org/x/sys/windows/registry`, `HKEY_CURRENT_USER` |
| Window procedure and enumeration callbacks | `NewCallback` |
| Progman candidate | `GetShellWindow` |

Not exposed by x/sys/windows, loaded through a lazy system DLL procedure (export
resolved for real by the probe):

| DLL | Exports |
|---|---|
| user32.dll | `UpdateLayeredWindow`, `CreateWindowExW`, `RegisterClassExW`, `DefWindowProcW`, `GetMessageW`, `DispatchMessageW`, `TranslateMessage`, `PostMessageW`, `PostQuitMessage`, `DestroyWindow`, `MonitorFromRect`, `MonitorFromWindow`, `GetMonitorInfoW`, `FindWindowW`, `FindWindowExW`, `SetWindowLongPtrW`, `SetWindowPos`, `RegisterWindowMessageW`, `SetProcessDpiAwarenessContext`, `GetDpiForWindow`, `GetDpiForSystem`, `TrackPopupMenu`, `TrackPopupMenuEx`, `CreatePopupMenu`, `AppendMenuW`, `DestroyMenu`, `SetForegroundWindow`, `CreateIconIndirect`, `DestroyIcon`, `GetSystemMetricsForDpi`, `SetWindowTextW`, `MsgWaitForMultipleObjectsEx`, `GetDC`, `ReleaseDC`, `GetCursorPos`, `LoadCursorW`, `GetWindowRect`, `ShowWindow` |
| gdi32.dll | `CreateDIBSection`, `CreateBitmap`, `CreateCompatibleDC`, `SelectObject`, `DeleteObject`, `DeleteDC` |
| shell32.dll | `Shell_NotifyIconW` |

`CreateIconIndirect` is reachable, as WE6 asked WE1 to verify.

Uncertainty: the probe proves the exports exist on Windows 11 26200 arm64 only.
The minimum Windows 10 release for `SetProcessDpiAwarenessContext`,
`GetDpiForWindow` and `GetSystemMetricsForDpi` has not been read; WE5 reads the
Microsoft Learn pages before relying on them and records the oldest Windows 10
release supported.

### Win32 constants

None of the constants named in Part 1 is defined by x/sys/windows except
`HKEY_CURRENT_USER` and `FOLDERID_LocalAppData`. Their values were read from the
Windows SDK headers installed on this machine, version 10.0.26100.0
(`um/WinUser.h`, `um/wingdi.h`, `um/shellapi.h`, `shared/windef.h`,
`shared/winerror.h`), 13 October 2026:

| Constant | Value | Constant | Value |
|---|---|---|---|
| `WS_EX_LAYERED` | 0x00080000 | `WS_EX_TOOLWINDOW` | 0x00000080 |
| `WS_EX_NOACTIVATE` | 0x08000000 | `WS_POPUP` | 0x80000000 |
| `WM_NCHITTEST` | 0x0084 | `HTCAPTION` | 2 |
| `WM_EXITSIZEMOVE` | 0x0232 | `MONITOR_DEFAULTTONEAREST` | 0x00000002 |
| `WM_DISPLAYCHANGE` | 0x007E | `WM_SETTINGCHANGE` | `WM_WININICHANGE`, 0x001A |
| `SPI_SETWORKAREA` | 0x002F | `WM_TIMECHANGE` | 0x001E |
| `HWND_BOTTOM` | (HWND)1 | `HWND_NOTOPMOST` | (HWND)-2 |
| `HWND_TOPMOST` | (HWND)-1 | `WM_DPICHANGED` | 0x02E0 |
| `WM_POWERBROADCAST` | 0x0218 | `PBT_APMRESUMEAUTOMATIC` | 0x0012 |
| `PBT_APMRESUMESUSPEND` | 0x0007 | `ULW_ALPHA` | 0x00000002 |
| `AC_SRC_ALPHA` | 0x01 | `AC_SRC_OVER` | 0x00 |
| `DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2` | (DPI_AWARENESS_CONTEXT)-4 | `BI_RGB` | 0 |
| `DIB_RGB_COLORS` | 0 | `NIM_ADD` | 0x00000000 |
| `NIM_MODIFY` | 0x00000001 | `NIM_DELETE` | 0x00000002 |
| `NIM_SETVERSION` | 0x00000004 | `NOTIFYICON_VERSION_4` | 4 |
| `NIF_MESSAGE` | 0x00000001 | `NIF_ICON` | 0x00000002 |
| `NIF_TIP` | 0x00000004 | `NIF_SHOWTIP` | 0x00000080 |
| `NIN_SELECT` | `WM_USER` + 0 | `NIN_KEYSELECT` | `NIN_SELECT` or `NINF_KEY` (value of `NINF_KEY` not yet read) |
| `TPM_RIGHTBUTTON` | 0x0002 | `TPM_NONOTIFY` | 0x0080 |
| `TPM_RETURNCMD` | 0x0100 | `MF_STRING` | 0x00000000 |
| `MF_GRAYED` | 0x00000001 | `MF_CHECKED` | 0x00000008 |
| `MF_UNCHECKED` | 0x00000000 | `MF_POPUP` | 0x00000010 |
| `MF_SEPARATOR` | 0x00000800 | `WM_USER` | 0x0400 |
| `WM_APP` | 0x8000 | `GWLP_HWNDPARENT` | -8 |
| `SWP_NOSIZE` | 0x0001 | `SWP_NOMOVE` | 0x0002 |
| `SWP_NOZORDER` | 0x0004 | `SWP_NOACTIVATE` | 0x0010 |
| `SW_SHOWNOACTIVATE` | 4 | `SM_CXSMICON` | 49 |
| `SM_CYSMICON` | 50 | `USER_DEFAULT_SCREEN_DPI` | 96 |
| `WM_CONTEXTMENU` | 0x007B | `WM_LBUTTONUP` | 0x0202 |
| `WM_RBUTTONUP` | 0x0205 | `WM_DESTROY` | 0x0002 |
| `WM_CLOSE` | 0x0010 | `WM_QUIT` | 0x0012 |
| `WM_NULL` | 0x0000 | `ERROR_ALREADY_EXISTS` | 183 |
| `IDC_ARROW` | `MAKEINTRESOURCE(32512)` | | |

Struct layouts (`MSG`, `WNDCLASSEXW`, `MONITORINFO`, `BITMAPINFOHEADER`,
`BLENDFUNCTION`, `NOTIFYICONDATAW`, `ICONINFO`) are not yet read. They are read
from the same headers in the phase that first uses each one, with a size test per
struct on both architectures.

### Is MsgWaitForMultipleObjectsEx needed?

Conclusion, by design reasoning only: no. The tick is a Go timer whose goroutine
posts a private `WM_APP` based message with `PostMessageW`; the GUI thread blocks
in `GetMessageW`, which already wakes for posted messages. There is no kernel
object for the GUI thread to wait on besides its queue. This rests on posting
from a non GUI goroutine being sound, which is the documented purpose of
`PostMessageW` but has not been run here. WE5 confirms it with the TrackPopupMenu
experiment the specification already requires, and this conclusion is revisited
then.

### Console suppression

Assumption from WE1: `-H windowsgui` suppresses the console window. Read from
source: `go tool link` on go1.27.2 lists `-H type  set header type`. Run for real:
the probe built with `GOOS=windows`, `GOARCH` amd64 and arm64, `CGO_ENABLED=0`,
`-trimpath -ldflags "-H windowsgui -s -w"`; the PE optional header Subsystem field
of both exes reads 2, which is `IMAGE_SUBSYSTEM_WINDOWS_GUI`. Verified for both
architectures. Running the arm64 exe without a console flash is an author
observation still owed and is folded into the WE5 gate.

### Static analyser

Chosen: staticcheck, `honnef.co/go/tools/cmd/staticcheck@v0.8.2` (reports
`2026.2.2 (0.8.2)`), with `-checks all`. Run for real on go1.27.2: clean on the
probe, and exit status 1 with `U1000` on a planted unused function. It is a CI
tool that ships in nothing built, recorded here rather than as a dependency
under datum 0.13. `go vet` runs as well.

### Race detector

Read from source (go1.27.2 `src/internal/platform/supported.go`,
`RaceDetectorSupported`): on Windows only amd64 is supported. Run for real on this
machine: `go test -race` prints `-race is not supported on windows/arm64`.

Read from source (`src/cmd/go/internal/work/init.go`): outside macOS, `-race`
requires cgo. Test jobs therefore run with `CGO_ENABLED=1` and a C compiler;
release builds stay `CGO_ENABLED=0`, which only matters to the shell anyway since
the core is pure Go.

Read from the official runner image readmes (`actions/runner-images`,
13 October 2026): `windows-2025` is x64 and ships gcc 15.2.0; `ubuntu-24.04` is
x64; `windows-11-arm` is the arm64 Windows label.

Open question: see the batched questions below.

### Continuous integration inputs

Read from the GitHub API, 13 October 2026:

| Action | Tag | Commit |
|---|---|---|
| actions/checkout | v7.0.1 | 3d3c42e5aac5ba805825da76410c181273ba90b1 |
| actions/setup-go | v7.0.0 | b7ad1dad31e06c5925ef5d2fc7ad053ef454303e |

Runner labels pinned to OS versions rather than `latest`, since the readme says a
`latest` label migrates without notice: `ubuntu-24.04`, `windows-2025`.

### Reference implementations

Confirmed at the exact paths with the GitHub API, then cloned shallow into a
scratch directory, 13 October 2026:

| Repository | Commit |
|---|---|
| FtrOnOff/FtrIO | 7e214d5b3d62d31333031413b43f69d9c7e65a7d |
| FtrOnOff/ftrio-python | b95ec4e1feaf1b3c74a30a43edd5e03c90f7ad68 |
| FtrOnOff/ftrio-rust | e814d40758eb3524d64e9e40688bb8c59de65989 |

The author's local `FtrIO` checkout tracks `TheScottBot/FtrIO`, not the datum's
`FtrOnOff/FtrIO`, so it was not used.

Taken from them:

- Verbose role naming for types; acronyms written as one word (`Bgra`, `Dpi`,
  `Iana`), as the Rust CI enforces with `upper_case_acronyms`.
- Test names follow the C# `Method_ExpectedBehaviour` shape, which is also Go
  idiom: `TestNextWake_LandsAfterBoundaryByMargin`. Table cases carry a
  descriptive sentence as their subtest name.
- Test doubles take the role as prefix: `FakeClock`, `SpySettingsStore`.
- Error messages lower case with single quoted values, which is Go idiom too;
  the meaning of an error is kept and the shape follows Go (`ErrXxx` sentinels,
  `XxxError` types).
- Why first doc comments; a module header comment that says what the package
  owns and why.
- Deviation entries shaped as in their PORTING_NOTES: a heading naming the rule
  and the substitute, one paragraph on why, then consequences.
- CHANGELOG in Keep a Changelog form with an `[Unreleased]` section, grouped by
  Added, Changed, Fixed. The dash in their version headings is replaced with a
  comma, as the Rust one already does.
- Shared test fixtures in one place, temporary directories per test
  (`t.TempDir()`).
- The toolchain pinned and checked, as their minimum version job does.

Not taken: their actions are pinned by major tag; Thyme pins by commit with the
tag in a comment, because datum 0.2 asks for the exact version. None of the three
has a prose scan or an import boundary check, so both are new here.

### The theme mock-up

Received in the conversation on 13 October 2026 as an image, 1448 by 1086 pixels
by eye. It agrees by eye with WU5 (panel roughly from x 278 to 1172 and y 376 to
681, divider near 64 percent of the panel width). Its pixels have not been
sampled: the agent cannot read an image that only exists in the conversation. It
is needed as a tracked file before WE4 replaces WU1 to WU5.

### Decisions still needing the author

WE1 race detector on arm64 Windows (blocks WE1's CI workflow only):

1. Decision. Whether CI also runs the pure package tests on a `windows-11-arm`
   runner, where `-race` cannot run.
2. Why evidence does not settle it. The specification asks for `go test -race` on
   every package on the Windows and Linux runners, and names no architecture.
   amd64 runners satisfy it fully. The author's own machine is arm64, and the
   toolchain has no race detector there.
3. Options. (a) `ubuntu-24.04` and `windows-2025` only, both with `-race`; arm64
   covered by the cross build and the author gates. (b) As (a), plus
   `windows-11-arm` running the same tests without `-race`, recorded in
   IMPLEMENTATION_DEVIATIONS.md.
4. Trade offs. (a) needs no deviation, but an arm64 only fault (alignment,
   floating point contraction in the rasteriser changing golden images) is first
   seen at an author gate. (b) catches those in CI at the cost of one deviation
   entry and a third job.
5. Recommendation. (b).

Status: asked 13 October 2026. Author chose (b), 13 October 2026. Recorded in IMPLEMENTATION_DEVIATIONS.md.

### Prerequisites only the author can perform

- The local folder is not a git repository and the remote is empty. Datum 0.1
  means the author initialises or clones it.

## WE1 Execute, 13 October 2026

### Findings

- The tool that writes files for the agent turned the escape sequences for the
  em dash and en dash into the literal characters. The prose scan caught all
  four on its first run over the real tree, which was its first useful result.
  The escapes were restored and the scan is clean.
- `go run` of staticcheck with `GOOS=linux` set on a Windows host cross
  compiles staticcheck itself and fails in `runtime/cgo`. CI installs it for the
  host and passes `GOOS` only when running it. Seen for real; the corrected
  sequence is clean and still fails on a planted unused function (`U1000`).
- Assumption not yet verified: the `windows-2025` runner's gcc 15.2.0 is on the
  PATH for the cgo race build. The first CI run settles it.

### Planted violations in the real tree

Planted, run and removed, 13 October 2026, on this machine:
`internal/render/planted_violation.go` importing `golang.org/x/sys/windows`, and
`internal/clockmodel/planted_violation.go` holding an em dash, an en dash and a
double hyphen run in three comments. `go run ./tools/repocheck` reported all four
and exited 1. After removal it reported clean and exited 0. The same cases also
live on as fixture tests in `internal/repocheck`, so every CI run repeats the
proof.

### Reproduction report

Platform: Windows 11 Home 10.0.26200, arm64. Toolchain: go1.27.2 windows/arm64
(`GOTOOLCHAIN=go1.27.2`). staticcheck 2026.2.2 (0.8.2). Commands, from the
repository root, in order:

```bash
test -z "$(gofmt -l .)"
```

```bash
GOOS=windows go vet ./...
```

```bash
GOOS=linux go vet ./...
```

```bash
go install honnef.co/go/tools/cmd/staticcheck@v0.8.2
```

```bash
GOOS=windows "$(go env GOPATH)/bin/staticcheck" -checks all ./...
```

```bash
GOOS=linux "$(go env GOPATH)/bin/staticcheck" -checks all ./...
```

```bash
go run ./tools/repocheck
```

```bash
go test -count=1 ./...
```

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H windowsgui" ./...
```

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -ldflags "-H windowsgui" ./...
```

| Item | Result |
|---|---|
| Focused tests, `internal/repocheck` | pass, 28 tests and subtests |
| Full suite | pass; every other package has no tests yet |
| gofmt | clean |
| go vet, windows and linux | clean, 0 warnings |
| staticcheck `-checks all`, windows and linux | clean, 0 warnings |
| repocheck on the real tree | clean |
| Race detector | not run: unsupported on this machine's windows/arm64; runs in CI on the amd64 runners |
| Fuzzing | not applicable: no parser until WE3 |
| Cross builds, windows/amd64 and windows/arm64 | succeed |
| Tests skipped | none |
| Tests requiring the author | none |

Outstanding before WE1 is done: the author's reproduction on a clean machine,
and the CI workflow seen green on all three runners, which needs the repository
initialised and pushed by the author (datum 0.1). No author gate belongs to WE1.

## WE2 Plan, 13 October 2026

### Decision taken by the author

| Id | Decision | Recorded |
|---|---|---|
| WD8 | Seconds are a menu toggle, on by default. Off renders HH:MM and wakes once a minute. | 13 October 2026 |

### The mock-up as a file

The author placed the mock-up at `docs/theme.png` (1,550,257 bytes), the same
image received in the conversation. Its pixels are sampled in WE4.

### Zone loading, read from source

Go 1.27.2, `src/time/zoneinfo.go` and `src/time/zoneinfo_read.go`, read 13
October 2026. `LoadLocation` returns UTC for the empty name and for `UTC`, and
the host's zone for `Local`, before any lookup. Otherwise it tries `$ZONEINFO`,
then the platform sources (none on Windows: `zoneinfo_windows.go` declares
`platformZoneSources` empty), then the embedded tzdata if `time/tzdata` is
linked, then `runtime.GOROOT()/lib/time/zoneinfo.zip`. `runtime.GOROOT()` reads
the `GOROOT` environment variable first.

Consequences accepted:

- The empty name and `Local` are rejected by clockmodel before
  `LoadLocation` is called, because both would otherwise be the silent fallback
  the specification forbids. `UTC` is a real IANA name and stays.
- A nil `*time.Location` prints as `UTC`, so a caller that ignored a load error
  would display UTC silently. Load errors are returned, never swallowed.

The embedded tzdata in go1.27.2 is release 2026c (`lib/time/update.bash`,
`CODE=2026c`, `DATA=2026c`).

### Experiment: time/tzdata on Windows

Run for real, 13 October 2026, this machine, go1.27.2. Two throwaway programs
loading UTC, Europe/London, Europe/Berlin and America/Sao_Paulo, one importing
`time/tzdata` and one not, built `CGO_ENABLED=0 -trimpath -ldflags "-s -w"` for
windows/amd64 and windows/arm64, run with `GOROOT` and `ZONEINFO` unset so the
installed Go's zoneinfo.zip could not mask the result (`runtime.GOROOT()`
printed empty). The amd64 builds ran under the machine's x64 emulation.

| Build | UTC | London, Berlin, Sao Paulo | Exe size amd64 | Exe size arm64 |
|---|---|---|---|---|
| without `time/tzdata` | loads | `unknown time zone` | 1,731,584 | 1,635,328 |
| with `time/tzdata` | loads | load | 2,142,720 | 2,046,976 |

Embedding costs 411,136 bytes on amd64 and 411,648 bytes on arm64. Assumption
verified: zone loading on a machine without Go installed depends on
`time/tzdata`. A stock machine was approximated by unsetting the variables, not
by a second machine; this is recorded as the limit of the experiment.

The four mock-up zones map to UTC, Europe/London, Europe/Berlin and
America/Sao_Paulo; each name loads from the embedded 2026c data. That the labels
UTC, LDN, BER and SAO mean those zones is the specification's reading of the
mock-up, and the WE2 test of the mock-up instant checks the offsets agree with
the times it shows.

## WE2 Execute, 13 October 2026

### Findings

- The daylight saving cases were written out by hand before the code existed
  (London and Berlin on 29 March and 25 October 2026, Sao Paulo's last summer
  time start on 4 November 2018) and pass against the embedded 2026c data.
- A test claiming to prove minute boundaries follow a half hour zone's wall
  clock was written and then removed before implementation, because a whole
  minute offset cannot tell a wall clock implementation from a UTC aligned one.
  No real zone today has an offset with seconds in it, so the property is
  untestable with current data, and the implementation reads the wall clock
  anyway.
- The compiled clockmodel tests were also run with `GOROOT` and `ZONEINFO`
  unset and passed, so the package's own `time/tzdata` import is what loads the
  zones, not the Go installation.
- Owned by WE3, recorded for it: the four mock-up zones are currently written
  as a test fixture in `internal/clockmodel/fixture_test.go`. WE3's defaults
  become their single home, and the fixture should then read them from there.

### Reproduction report

Platform: Windows 11 Home 10.0.26200, arm64. Toolchain go1.27.2. staticcheck
2026.2.2 (0.8.2). Commands are those of the WE1 report, unchanged, plus:

```bash
go test -count=1 -v ./internal/clockmodel/
```

| Item | Result |
|---|---|
| Focused tests, `internal/clockmodel` | pass, 29 tests and subtests; each seen failing on an assertion against stubs first |
| Full suite | pass |
| gofmt, vet, staticcheck for windows and linux | clean, 0 warnings |
| repocheck | clean |
| Race detector | not run here (windows/arm64); runs in CI on amd64 |
| Cross builds, windows/amd64 and windows/arm64 | succeed |
| Tests skipped | none |
| Tests requiring the author | none; WE2 has no author gate |

Outstanding: the author's reproduction, and the suites seen green on the CI
runners, which have not yet run because the repository has not been pushed.

(13 October 2026, later the same day: the author pushed commit 77366e2. CI run
38454677843 passed on all three runners. On ubuntu-24.04 and windows-2025 the
race detector step ran and passed, so the windows-2025 gcc is on the PATH as
assumed; on windows-11-arm the plain test step ran, as designed.)

## WE3 Plan, 13 October 2026

### Decisions taken by the author

| Id | Decision | Recorded |
|---|---|---|
| WD6 | JSON through encoding/json, no dependency. | 13 October 2026 |
| WD5 | A Scale submenu with steps of 75, 100, 125, 150 and 200 percent. Asked now because the settings file stores the step. | 13 October 2026 |
| WD9 | Tray icon on by default, every action in the keyboard reachable tray menu, window text set to the spoken time. Asked now because the tray default is a settings default. | 13 October 2026 |
| WE3 version | A required `"version": 1`; a missing or other version refuses the file whole, uses the defaults and protects the file from being saved over. | 13 October 2026 |

### The settings contract

Provisional under datum 0.11 until the author freezes it. Shape: one JSON
object with exactly the members `version`, `position` (`x`, `y`),
`scalePercent`, `alwaysOnTop`, `showInTray`, `showSeconds`, `zones` (each
`label`, `timeZone`). Member names are camelCase, the common Go JSON idiom;
the reference implementations use PascalCase keys, which is the .NET
configuration idiom, and datum 0.5 puts the language's idiom first. Bounds:
4096 bytes, four zones, labels of one to three printable characters, time zone
names of one to 64 bytes, scale one of the WD5 steps, positions within 32 bit
signed integers.

### Rename on Windows, read from source

go1.27.2 `src/os/file_windows.go` and `src/internal/syscall/windows/syscall_windows.go`:
`os.Rename` calls `MoveFileExW` with `MOVEFILE_REPLACE_EXISTING` only (no
`MOVEFILE_WRITE_THROUGH`).

### Experiment: same folder rename over an existing file on NTFS

Run for real, 13 October 2026, this machine, volume C: reported as NTFS by
`Get-Volume`. For ten seconds a writer created a temporary file in the same
folder, wrote, synced and closed it, and renamed it over `settings.json`,
alternating a 23 byte and a 640 byte document; a reader goroutine read the
target continuously and compared it with both.

| Measure | Count |
|---|---|
| Successful renames | 2,450 |
| Failed renames (`Access is denied`) | 4,016 |
| Reads that returned one of the two complete documents | 100,365 |
| Reads that returned anything else (torn) | 0 |
| Reads that found no file | 0 |
| Reads that failed with another error | 7,769 |

Conclusions accepted: a reader never sees a partial or missing file, so the
rename is atomic enough for the purpose. The rename fails, rather than waits,
while another handle has the target open, and the reader's own open fails
while a rename is in flight. Save therefore returns the failure for the shell
to log and count, and removes its temporary file. The experiment ran without a
power cut, so durability across a crash rests on the rename design, not on a
measurement.

### Fuzzing platforms, read from source

go1.27.2 `src/internal/platform/supported.go`: fuzzing is supported on windows
and linux, and coverage instrumented on amd64 and arm64, so all three runners
run a guided fuzz.

## WE3 Execute, 13 October 2026

### Findings

- Two silent behaviours of `encoding/json` would have defeated the WE3 rules:
  a duplicated member is resolved by letting the last one win, and `null`
  decodes to the zero value. Both are now rejections, with tests.
- Overwrite protection was first implemented for every whole file refusal
  without a failing test for the non version cases. That breached datum 0.4;
  the test was then written, seen to fail with the protection disabled for
  those cases, and passes with it restored. The extension itself awaits the
  author (IMPLEMENTATION_DEVIATIONS.md).
- A local fuzz run of `FuzzParse` for 60 seconds executed 2,366,326 inputs
  with no failure and found 377 interesting inputs, none kept in the tree.

### Reproduction report

Platform: Windows 11 Home 10.0.26200, arm64. Toolchain go1.27.2. staticcheck
2026.2.2 (0.8.2). Commands are those of the WE1 report, plus:

```bash
go test -count=1 -v ./internal/settings/
```

```bash
go test -run='^$' -fuzz='^FuzzParse$' -fuzztime=60s ./internal/settings/
```

(13 October 2026, correction to the WE1 Plan record: it said acronyms are
written as one word, after the Rust reference. staticcheck ST1003, which is Go
idiom, requires its listed initialisms such as JSON, ID and URL in capitals,
and datum 0.5 puts the language's conventions first. Go initialisms are
therefore written in capitals; an acronym outside staticcheck's list is still
written as one word.)

| Item | Result |
|---|---|
| Focused tests, `internal/settings` | pass, all seen failing on assertions against stubs first |
| Full suite | pass |
| gofmt, vet, staticcheck for windows and linux | clean after renaming one parameter that ST1003 flagged |
| repocheck | clean |
| Fuzz, 60 s locally | 2,366,326 executions, no failure |
| Race detector | not run here (windows/arm64); runs in CI on amd64 |
| Cross builds | succeed |
| Tests skipped | none |
| Tests requiring the author | none; WE3 has no author gate |

Outstanding: the author's reproduction and the CI run of this change, and the
author's decision on the two agent proposals in IMPLEMENTATION_DEVIATIONS.md
(overwrite protection beyond version, and `internal/product`).

## WE4 Plan, 13 October 2026

(Earlier the same day: the author pushed 5715d22 and accepted both WE3
proposals. CI run 38456599877 passed on all three runners, including the first
30 s fuzz run of the settings parser on each.)

### Decisions taken by the author

| Id | Decision | Recorded |
|---|---|---|
| WD3 | Pure Go rendering: x/image opentype for text, x/image vector for shapes, a hand written box blur for the glow. | 13 October 2026 |
| WD4 | JetBrains Mono Regular only, embedded. The author approved downloading it. | 13 October 2026 |
| WU5 scale | The mock-up is a 200 percent render, so one design unit is two mock-up pixels. | 13 October 2026 |

### The font, vendored

Source: `https://github.com/JetBrains/JetBrainsMono`, latest release v2.304
(published 2023-01-14), tag commit cd5227bd1f61dff3bbd6c814ceaf7ffd95e947d9,
licence OFL-1.1 per the GitHub API, repository not archived. Downloaded 13
October 2026 from `raw.githubusercontent.com` at that commit. Git blob hashes
of both files equal the upstream blob hashes from the GitHub contents API.

| File | Bytes | Git blob | SHA-256 |
|---|---|---|---|
| `fonts/ttf/JetBrainsMono-Regular.ttf` | 273,900 | dff66cc50702c75abd025dcf49f62a4dcc2d72de | a0bf60ef0f83c5ed4d7a75d45838548b1f6873372dfac88f71804491898d138f |
| `OFL.txt` | 4,399 | 8bee4148c1d54dbf5dae6d6c117fc80414266abb | 30f0c136e3c88e422d0791acd97238870f9054a9729bc34cf2ff0d4ed8cac4ad |

Read from the file with x/image v0.47.0: full name `JetBrains Mono Regular`,
version `Version 2.304; ttfautohint (v1.8.4.7-5d5b)`, 1000 units per em, cap
height 730, x height 550, ascent 1020, descent 300, advance of every glyph 600
(monospaced), digit `2` ink from 76 to 527 horizontally and 740 high.

### Dependency records (datum 0.13)

golang.org/x/image v0.47.0:

- Behaviour: OpenType parsing and glyph rasterisation (`font/opentype`,
  `font/sfnt`), the `font.Drawer`, and an anti aliased vector rasteriser
  (`vector`).
- Why not the standard library: it has no font or path rasteriser.
- Maintenance: part of the Go project, v0.47.0 published 2026-10-09.
- Licence: BSD 3 clause, the Go licence.
- Size: measured in WE8 against WU6.
- Platforms: pure Go with optional amd64 assembly in `vector`; no cgo.
- Security: parses the embedded font only, which is fixed at build time and
  never user input.
- Tests: golden images and the allocation bound in `internal/render`.
- Replacement cost: confined to `internal/render`; another rasteriser would
  need new golden images, nothing else.

JetBrains Mono Regular v2.304:

- Behaviour: the theme's only typeface.
- Why: WD4. No system font is used, so the core renders identically on any
  host.
- Maintenance: last release January 2023, repository active (pushed January
  2025); a font needs no updates to keep working.
- Licence: SIL Open Font Licence 1.1, which permits embedding in software of
  any licence when the licence text travels with it; OFL.txt is kept beside the
  font and named in README.md.
- Size: 273,900 bytes in the exe.
- Security: parsed at start from the embedded bytes only.
- Replacement cost: one file and the golden images.

### Experiment: is rendering byte identical across architectures?

Read from source first: x/image v0.47.0 `vector/raster_floating.go` disables
fused multiply add explicitly so results agree across architectures, and
`vector/acc_amd64.go` uses SSE4.1 assembly when present, pure Go elsewhere.

Run for real, 13 October 2026: one program drew a rounded panel and the string
`21:04:01 sun 2026-09-06` (unhinted and fully hinted) at scales 0.75, 1, 1.25,
1.5 and 2, built for windows/arm64 (native) and windows/amd64 (run under the
machine's x64 emulation), and the pixel bytes were compared.

| Scale | Panel alone | Panel and text |
|---|---|---|
| 0.75 | identical | identical |
| 1.00 | identical | 10 bytes differ, at most 2 of 255 |
| 1.25 | identical | 3 bytes differ, at most 1 |
| 1.50 | identical | 3 bytes differ, at most 2 |
| 2.00 | identical | identical |

Conclusion accepted: shapes are byte identical; glyph rasterisation is not,
by at most 2 per channel in a handful of bytes. Golden images are therefore
compared with a per channel tolerance of 2, recorded in TESTING.md as the
specification allows. The amd64 side ran under emulation; the first CI run on
real amd64 hardware either confirms the tolerance or records a correction.
The root cause within glyph rasterisation was not pursued.

### The mock-up, sampled

`docs/theme.png`, 1448 by 1086, decoded as RGBA, sampled 13 October 2026 with a
throwaway program. Mock-up pixels, then design units at the decided 200
percent (one unit is two pixels).

| Value | Mock-up measurement | Design value |
|---|---|---|
| Panel fill (WU1) | medians #12131c, #12151d, #13141e | #12131c |
| Time digits (WU2) | brightest fifth of ink #fa95fa | #f894f8 |
| Zone times (WU2) | #f692f6 | as the time |
| Date line (WU2) | #e48be6 | #e48be6, a step dimmer than the time |
| Zone labels (WU2) | #d583d7 | #d583d7 |
| Outline (WU3) | bright core about 2 px, peaks #f7a0f3 to #feaefc | #f59ef3, 1 unit |
| Outer glow (WU3) | red excess over the backdrop +35 at 2 px, +16 at 14, +5 at 30, 0 by about 40 | reaches about 20 units |
| Divider | 2 px wide, #d599d7, x 851.5, from y 415 to 641 | 1 unit, #d599d7, inset 20 units top and bottom |
| Backdrop (WU4) | #29384c at the top to #1e2b3d at the bottom | wallpaper, not drawn |
| Panel outline box (WU5) | x 277 to 1171, y 375 to 680: 894 by 305 | 447 by 152.5 |
| Corner radius (WU5) | 12 to 14 px from both top corners | 6.5 |
| Time | digits 74 px high, ink 339 to 786 | JetBrains Mono at 49 units |
| Date | digits 26 px high | 18 units |
| Zone labels and times | digits 24 px high, rows every 53.3 px | 17 units, rows every 26.5 |
| Left padding to the time and date | pen origin about 55 px | 27 units |
| Right padding after the zone times | about 56 px | 27 units |
| Time column to divider | about 50 px | 25 units |
| Divider to labels | about 52 px | 26 units |
| Label to zone time | about 109 px between pen origins | label advance plus 24 units |
| Top and bottom padding to the zone block | 59 and 62 px | 30 units |

Font sizes were derived from JetBrains Mono's measured metrics: the time's 74
px digits at 0.74 em give an em near 100 px, and its ink width of 448 px over
seven advances of 0.6 em gives about 98; 49 units is the compromise. The
mock-up face is not certainly JetBrains Mono, so these sizes are fitted, not
read. The panel width these give with seconds and four zones is 27 + 235.2 +
25 + 1 + 26 + 30.6 + 24 + 51 + 27 = 446.8 units against the measured 447.

Uncertainties: the mock-up is a compressed image, so colours carry a few levels
of error, and the glow's falloff was fitted by eye to the red excess. The
glow looks stronger below the panel than to its left, partly because the
backdrop itself darkens downwards; a symmetric glow is drawn. Every value above
is subject to the WE4 author gate, the 100 percent golden image beside the
mock-up.

## WE4 Execute, 13 October 2026

### Findings

- The steady state tick allocates nothing: 0 allocations and 0 bytes per
  Render with a warmed cache, against a 412,824 byte frame at 100 percent. The
  bound is set at 0 allocations.
- The first layout read advances and cap heights from sized faces, which round
  both to 1/64 pixel, so the panel width drifted by up to 0.125 pixels between
  scale steps and three layout tests failed. Both are now read once in font
  units and scaled exactly.
- The divider is snapped to whole pixels so it stays one crisp line; the
  positions after it use the unsnapped value, so the snapping cannot make the
  layout disproportionate.
- The render tests built for windows/amd64 and run under emulation pass against
  golden images generated on arm64, so the tolerance of 2 holds across
  architectures on this machine. The CI runners repeat this on real amd64.
- For the author gate, compared side by side with the mock-up at 200 percent:
  geometry, glow, palette and spacing agree closely. The visible differences
  are that JetBrains Mono draws a dotted zero where the mock-up's zero is a
  plain oval (the mock-up was not set in JetBrains Mono; x/image does not apply
  OpenType stylistic alternates, so no undotted zero is reachable from code),
  the time is a little wider than the mock-up's, the date sits a little higher,
  and the mock-up has a faint inner glow that is not drawn.
- The glow fringe catches clicks wherever its alpha is above zero, out to about
  20 units from the panel, though the outer part is barely visible. Recorded by
  `TestRender_GlowFringeIsClickableWhereverItIsDrawn`; whether to trim the
  faint tail so it lets clicks through is a question for the WE5 gate, where
  dragging can be felt.

### Reproduction report

Platform: Windows 11 Home 10.0.26200, arm64. Toolchain go1.27.2. staticcheck
2026.2.2 (0.8.2). golang.org/x/image v0.47.0. Commands are those of the WE1
report, plus:

```bash
go test -count=1 -v ./internal/render/ ./internal/theme/ ./third_party/...
```

```bash
GOOS=windows GOARCH=amd64 go test -c -o render_amd64.test.exe ./internal/render
```

| Item | Result |
|---|---|
| Focused tests: render, theme, font | pass, 38 tests and subtests; each seen failing against stubs or missing golden images first |
| Full suite | pass |
| gofmt, vet, staticcheck for windows and linux | clean, 0 warnings |
| repocheck | clean, with the two vendored files excluded by exact path |
| Render tests as amd64 under emulation | pass against the arm64 golden images |
| Race detector | not run here (windows/arm64); runs in CI on amd64 |
| Cross builds | succeed |
| Tests skipped | none |
| Author gate | WE4 theme acceptance: outstanding |
