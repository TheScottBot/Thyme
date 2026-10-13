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
