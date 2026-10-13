# Changelog

The format follows Keep a Changelog. Per datum Stage three, each tag names any
contract digest built against, every change to shared code since the previous
tag, and which author gates were cleared at which commit.

## [Unreleased]

### Added

- **WE1 skeleton.** Module `github.com/TheScottBot/Thyme` on Go 1.27.2, the
  package layout of the specification's Part 2, and the required documents.
- **Import boundary check and prose scan** in `internal/repocheck`, run by
  `tools/repocheck`, each proven against planted violations.
- **Continuous integration** on ubuntu-24.04, windows-2025 and windows-11-arm:
  gofmt, vet and staticcheck for both operating systems, the tree wide checks,
  tests (with the race detector where the toolchain has one), and windows/amd64
  and windows/arm64 cross builds.
- **WE2 clock model.** Zones load from embedded tzdata and fail loudly, naming
  the label, for an unknown, empty or host dependent name. Snapshot gives the
  time (with or without seconds), the lower case date line and each zone's
  HH:MM. NextWake aims at the next whole second or minute plus a 20 ms margin.
- **WE3 settings.** JSON at `%LOCALAPPDATA%\Thyme\settings.json`, version 1, read
  bounded to 4096 bytes before parsing. Every rejected field falls back to its
  default and is counted by reason; files Thyme cannot understand are never
  saved over. Saves are written to a temporary file and renamed over. A fuzz
  target runs in CI.
- **`internal/product`** holds the name Thyme once.
- **WE4 rendering.** `internal/theme` holds every visual constant, measured from
  the mock-up at 200 percent, with a WCAG AA contrast test. `internal/render`
  draws the glow, panel, outline, divider and text into premultiplied BGRA with
  a cached static layer and no allocation per tick. JetBrains Mono Regular
  v2.304 is vendored and embedded (PROVENANCE.md).
- **WE5 window.** A layered, non activating tool window drawn with
  UpdateLayeredWindow, dragged anywhere, clamped onto the nearest monitor's
  work area, saved on each move, rescaled on DPI change, and ticked by a timer
  re aimed from the wall clock after every tick. Single instance by mutex. A
  capped JSON lines log beside the settings. Requires Windows 10 version 1703
  or later.
- **WE6 desktop citizenship.** Owned by the desktop's view so Show Desktop
  spares it, re pinned after Explorer restarts. A right click and tray menu:
  Seconds, Scale (greyed until WE7), Always on top, Start with Windows (read
  fresh from the registry, Task Manager's disable included), Show in tray,
  Exit. A tray icon drawn at run time from the theme, on by default and
  keyboard reachable. The window's text is the spoken time for screen readers.
  Every change outside the process logs which menu asked for it.
- **WE7 scale.** The Scale submenu's steps (75, 100, 125, 150 and 200 percent)
  act: the current one is ticked, and a change redraws at the new size from
  the top left corner, clamps onto the screen and is saved.

No contract digest applies: Thyme implements no shared contract.
