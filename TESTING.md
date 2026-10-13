# Testing

## Suites and checks

All commands run from the repository root.

| Suite or check | Command | Runs on |
|---|---|---|
| Unit tests, all pure packages | `go test -race -count=1 ./...` | CI: ubuntu-24.04, windows-2025; locally on amd64 |
| Unit tests without the race detector | `go test -count=1 ./...` | CI: windows-11-arm; locally on arm64 Windows |
| Formatting | `gofmt -l .` (must print nothing) | CI: every runner |
| Vet, both operating systems | `GOOS=windows go vet ./...` and `GOOS=linux go vet ./...` | CI: every runner |
| Static analysis | `go install honnef.co/go/tools/cmd/staticcheck@v0.8.2`, then `GOOS=windows "$(go env GOPATH)/bin/staticcheck" -checks all ./...` and the same with `GOOS=linux` | CI: every runner |
| Import boundary and prose scan | `go run ./tools/repocheck` | CI: every runner |
| Settings parser fuzzing | `go test -run='^$' -fuzz='^FuzzParse$' -fuzztime=30s ./internal/settings` | CI: every runner, 30 s per run |
| Known vulnerabilities | `go install golang.org/x/vuln/cmd/govulncheck@v1.9.0`, then `GOOS=windows "$(go env GOPATH)/bin/govulncheck" ./...` and the same with `GOOS=linux` | CI: every runner |
| Release builds and size budget | the release build commands in README.md, then `go run ./tools/sizecheck` on both exes (6 MiB each) | CI: every runner |
| Cross builds | `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H windowsgui" ./...`, and the same with `GOARCH=arm64` | CI: every runner |

The race detector needs cgo on Linux and Windows, so the race runs set
`CGO_ENABLED=1` and need a C compiler. Shipped builds are always
`CGO_ENABLED=0`.

## The tree wide checks

`internal/repocheck` holds both checks; `tools/repocheck` runs them over the
tree and exits non zero on any hit.

- Import boundary: no package outside `internal/desktop` and `cmd/thyme` may
  import `syscall`, `unsafe` or anything under `golang.org/x/sys`. Every Go
  file is parsed, test files and files behind build constraints included.
- Prose scan: em dash, en dash, and runs of two or more hyphens, in every text
  file and in the comments and string literals of Go files. Exempt: Markdown
  table separator rows, command line flag tokens, and the explicit
  `vendoredPaths` list in `tools/repocheck/main.go` (empty today).

Each check is proven against planted violations by tests built in temporary
directories, so the proof reruns on every runner on every change. The
specification's planted cases (an x/sys import in `internal/render`; an em
dash, an en dash and a double hyphen run in a Go comment) were also planted in
the real tree on 13 October 2026 and seen to fail before being removed; the
record is in docs/evaluation/ACTUAL_CONTRACT_EVALUATION.md.

## Suites by package

| Package | What its tests pin down |
|---|---|
| `internal/repocheck` | both tree wide checks against planted violations |
| `internal/clockmodel` | the mock-up instant, daylight saving transitions in every mock-up zone, the date line, next wake at .000, .999 and .980 seconds in both modes, and zone load failures by reason |
| `internal/settings` | every field round trips; the size bound refuses one byte over before parsing and accepts exactly the bound; every rejection reason, each costing only its field; whole file refusals and overwrite protection; rejections never carry file contents; atomic save, temporary file removed on failure; the fuzz target `FuzzParse` |
| `internal/theme` | WCAG 2.x AA contrast of every text colour against the panel, computed from the constants; the normal text ratio at every scale; an opaque palette; the glow reach equals what the box passes spread |
| `internal/render` | golden images (the mock-up at 100 and 200 percent; zero, one and four zones with seconds on and off); every pixel inside the panel opaque; every pixel beyond the glow transparent; the glow fringe clickable; BGRA conversion; layout widths; static layer caching; the steady state allocation bound |
| `third_party/jetbrainsmono` | the vendored files and the embedded font match SHA256SUMS |
| `internal/placement` | each edge, a corner, larger than the work area (top left wins), the visible panel clamped rather than the transparent margin, the largest overlap or else the nearest monitor, a vanished monitor, pixel scale for every step at 96, 120, 144 and 192 DPI |
| `internal/eventlog` | the cap, one previous file kept, appending to and rotating an earlier run's log, an oversized record kept whole, paths stripped from logged errors |
| `internal/desktop` | Win32 struct sizes against the SDK (Windows runners only); the menu model's items, order, ticks and greyed states; Start with Windows against a fake registry: the quoted Run value, Task Manager's disable, enabling clearing it, idempotence, partial failures and the required origin; the window and tray behaviour is the WE5 and WE6 author gates |
| `internal/clockmodel`, spoken time | the window's text for screen readers, in the local zone |
| `internal/render`, scale steps | golden images of the mock-up at 75, 100, 125, 150 and 200 percent |
| `internal/desktop`, scale | every step enabled with the current one ticked; each Scale command maps to its step and no other command does |
| `internal/render`, tray icon | golden images at 16, 20, 24 and 32 pixels; opaque middle, clear corners; size bounds |

## Golden images

`internal/render/testdata/golden` holds one PNG per case. They are compared
in premultiplied RGBA with a tolerance of 2 per channel: measured in WE4, glyph
rasterisation differs between arm64 and amd64 by at most 2 of 255 in a handful
of bytes, while shapes are byte identical. A failing comparison writes the
rendered image to the test's temporary folder and names it.

To rewrite them after an intended visual change:

```bash
go test ./internal/render -run Golden -update
```

A rewritten golden image is accepted only by the author looking at it.

## Allocation bound

`TestRender_SteadyStateTickAllocatesNoFrameSizedBuffer` renders alternating
times into a warmed renderer. Bound: `MaximumAllocationsPerTick` = 0
allocations per tick, and under one sixty fourth of a frame in bytes. Measured
13 October 2026 on windows/arm64: 0 allocations and 0 bytes per tick against
a 412,824 byte frame at 100 percent.

## Testing deviations

- No race detector on windows/arm64. See IMPLEMENTATION_DEVIATIONS.md.

## Author gates

- WE4: the 100 percent golden image (`internal/render/testdata/golden/mockup_100.png`), viewed beside `docs/theme.png`, is accepted as the theme. Cleared by the author, 13 October 2026, in their words: "theme accepted". Recorded from the conversation; nothing else was written down. The dotted zero, the slightly wider time, the slightly higher date and the missing inner glow were shown to the author before acceptance.
- WE5, on Windows 11 arm64 and on one amd64 machine, with the exe built as in
  README.md. Cleared by the author, 13 October 2026, in their words: "WE5 gate
  passed". Recorded from the conversation; which machines were used and what
  was observed for each item was not written down.
  1. The widget appears with a transparent surround: only the panel and its
     glow are visible.
  2. It drags from anywhere on the panel, including across monitors.
  3. Ended and started again, it reopens where it was left.
  4. With its monitor unplugged, it is pulled back onto a remaining one.
  5. Moved between monitors of different scaling, it redraws crisply at the
     new size.
  6. Over two minutes of watching, the seconds never visibly skip or repeat.
  7. Idle processor use in Task Manager is zero with seconds off (set
     `"showSeconds": false` in the settings file and restart) and negligible
     with them on.
- WE6. Cleared by the author, 13 October 2026, in their words: "WE6 gate
  passed". Recorded from the conversation; which items were run (item 8 is
  optional), on which machines, and what Narrator read were not written down.
  1. Win+D leaves the widget visible.
  2. Restart Explorer from Task Manager (Details, explorer.exe, End task, then
     File, Run new task, explorer): the widget stays visible, the tray icon
     comes back, and Win+D still spares the widget afterwards.
  3. Always on top on keeps the widget above other windows; off sends it to
     the bottom and never raises it above the app that launched it.
  4. Start with Windows ticked survives a sign out and in. Unticking Thyme in
     Task Manager's Startup apps is shown unticked in the menu; ticking it in
     the menu again shows it Enabled in Task Manager.
  5. The tray menu works from the keyboard alone: Win+B, arrows to the Thyme
     icon, Enter (or Shift+F10) opens the menu, arrows and Enter choose.
  6. Exit removes the tray icon and the widget.
  7. Narrator, navigated to the widget, reads its text: "Thyme:" and the time
     and date. If it does not, the gap is recorded as a known limitation.
  8. Optional, if the author uses a wallpaper slideshow: Win+D still spares
     the widget with the slideshow on.
- WE7. Cleared by the author, 13 October 2026, in their words: "WE7 gate
  passed". Recorded from the conversation; the displays used and what was
  seen at each step were not written down. On a 100 percent display and on a 200 percent display,
  choose each Scale step in turn (75, 100, 125, 150, 200): each looks right,
  the widget keeps its top left corner, a step that would run off the screen
  is pulled back on, and the step chosen is ticked and remembered after Exit
  and restart.
- WE8. Outstanding. With the release build for each architecture (README.md),
  start Thyme, leave it idle for ten minutes with seconds on, then read its
  working set with `(Get-Process thyme).WorkingSet64 / 1MB` in PowerShell
  (Task Manager's default Memory column is the private working set, a
  different figure). Add the two results to HARDWARE_COMPATIBILITY.md against the 30 MB budget.
