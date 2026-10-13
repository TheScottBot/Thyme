# Thyme

A very small desktop clock for Windows 10 and 11, written in Go. One theme (neon
violet on a dark panel): local time with seconds, a date line and up to four
world clocks, on a borderless window you can drag anywhere.

Version 1.0.0. Runs on Windows 10 version 1703 or later and Windows 11, on
amd64 and arm64.

## Using it

Right click the widget, or click the tray icon, for the menu: Seconds, Scale,
Always on top, Start with Windows, Show in tray, Exit. From the keyboard,
Win+B reaches the tray; arrow to the Thyme icon and press Enter. Drag the
widget anywhere; it remembers where you leave it.

Start with Windows adds Thyme to your sign in programs for your account only.
Its tick shows what will really happen: if you disable Thyme in Task
Manager's Startup apps, the tick goes off, and ticking it again re-enables it
there too.

## Building

From the repository root, with Go 1.27.2 (go.mod pins it, and a newer local
toolchain downloads it automatically):

```bash
go test ./...
```

```bash
go run ./tools/repocheck
```

The Windows executable, built from any host:

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -ldflags "-H windowsgui" -o thyme.exe ./cmd/thyme
```

The same from PowerShell, where variables are set before the command rather
than in front of it (use `amd64` for an x64 machine):

```powershell
$env:CGO_ENABLED = "0"; $env:GOOS = "windows"; $env:GOARCH = "arm64"; go build -trimpath -ldflags "-H windowsgui" -o thyme.exe ./cmd/thyme
```

TESTING.md lists every suite and check, and where each runs.

### Release builds

A release exe is the build above with symbols stripped (`-s -w`), for each
architecture, then checked against the 6 MiB size budget:

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H windowsgui -s -w" -o thyme_amd64.exe ./cmd/thyme
```

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -ldflags "-H windowsgui -s -w" -o thyme_arm64.exe ./cmd/thyme
```

```bash
go run ./tools/sizecheck thyme_amd64.exe thyme_arm64.exe
```

There is no installer and the exe is not signed (specification WE8). Put it
anywhere you can write to and run it; Start with Windows remembers where.

## Settings

Thyme keeps its settings in `%LOCALAPPDATA%\Thyme\settings.json`, never beside
the exe. The file is JSON. A hand edit takes effect the next time Thyme starts;
a running Thyme does not watch the file.

```json
{
  "version": 1,
  "position": { "x": 120, "y": 120 },
  "scalePercent": 100,
  "alwaysOnTop": false,
  "showInTray": true,
  "showSeconds": true,
  "zones": [
    { "label": "UTC", "timeZone": "UTC" },
    { "label": "LDN", "timeZone": "Europe/London" },
    { "label": "BER", "timeZone": "Europe/Berlin" },
    { "label": "SAO", "timeZone": "America/Sao_Paulo" }
  ]
}
```

Those are also the defaults.

Thyme also writes a log, `%LOCALAPPDATA%\Thyme\thyme.log`, capped at 1 MiB
with one previous file kept as `thyme.log.1`. It records start up, rejected
settings by field and reason, and faults; never the settings file's contents
or any path beyond the folder name.

| Field | Meaning | Accepted values |
|---|---|---|
| `version` | the settings format | `1` only |
| `position` | top left corner, physical pixels of the virtual screen | whole numbers `x` and `y`, negative allowed; Thyme pulls the widget back on screen |
| `scalePercent` | size relative to the design size | `75`, `100`, `125`, `150` or `200` |
| `alwaysOnTop` | keep above other windows | `true` or `false` |
| `showInTray` | show the tray icon, the keyboard's way to the menu | `true` or `false` |
| `showSeconds` | show seconds; off wakes once a minute instead of once a second | `true` or `false` |
| `zones` | world clocks, zero to four | each a `label` of one to three printable characters and a `timeZone` IANA name such as `Europe/London` |

Every field is required and no other field is allowed. A field that is
missing, unknown, duplicated, of the wrong type or out of range is replaced by
its default, and the rejection is logged by field name and reason, never the
file's contents. A file over 4096 bytes, one that is not a single JSON object,
or one without `"version": 1` is ignored whole: the defaults are used and Thyme
does not save over it, so nothing it could not understand is destroyed. An
unknown time zone name is not a settings error: it is reported, naming the
label, when the zone is loaded.

## Known limitations

- A change of Windows' time zone while Thyme runs (for example a laptop with
  automatic time zone crossing a border) is not followed by the local time
  until Thyme restarts; the world clocks are unaffected, and the log records
  the change. Go reads the local zone once per process.
- English only (WD7).
- Task Manager's Startup state is kept in an undocumented registry value;
  Thyme's reading of it rests on observation (see
  docs/evaluation/ACTUAL_CONTRACT_EVALUATION.md, WE6 Plan) and may need
  revisiting if Windows changes it.
- The menus are Windows' standard menus. Thyme does not use the undocumented
  calls that make them dark (specification Part 1.2), so they may be light
  while Windows is set to dark.

## Layout

| Path | Holds |
|---|---|
| `internal/theme` | every visual constant, once |
| `internal/clockmodel` | zones, displayed strings, next wake |
| `internal/settings` | settings type, bounds, defaults, load, save |
| `internal/render` | one frame into a premultiplied BGRA buffer |
| `internal/placement` | work area clamping and scale arithmetic |
| `internal/desktop` | the Windows shell; the only package touching the OS |
| `cmd/thyme` | the entry point, wiring only |
| `internal/product` | the product name, once |
| `third_party/jetbrainsmono` | the vendored font, its licence and digests, and the package that embeds it |
| `internal/repocheck`, `tools/repocheck` | the import boundary check and the prose scan |

## Where it came from

Written from the project specification of 13 October 2026, which applies
ENGINEERING_DATUM.md version 2. The specification describes behaviour observed
in DeskTick. Thyme is a clean room reimplementation: the specification is the
only input, no DeskTick source was read, and nothing of DeskTick's code, name,
icons or images is used. Thyme is not affiliated with DeskTick.

## Licence

MIT, copyright TheScottBot. See LICENSE.

The embedded typeface is JetBrains Mono, copyright 2020 The JetBrains Mono
Project Authors, under the SIL Open Font Licence 1.1, whose text is in
`third_party/jetbrainsmono/OFL.txt`. PROVENANCE.md records where it came from.
