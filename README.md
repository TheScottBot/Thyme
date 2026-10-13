# Thyme

A very small desktop clock for Windows 10 and 11, written in Go. One theme (neon
violet on a dark panel): local time with seconds, a date line and up to four
world clocks, on a borderless window you can drag anywhere.

Status: WE1. The skeleton and its checks exist; there is no clock yet.

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

TESTING.md lists every suite and check, and where each runs.

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
| `internal/repocheck`, `tools/repocheck` | the import boundary check and the prose scan |

## Where it came from

Written from the project specification of 13 October 2026, which applies
ENGINEERING_DATUM.md version 2. The specification describes behaviour observed
in DeskTick. Thyme is a clean room reimplementation: the specification is the
only input, no DeskTick source was read, and nothing of DeskTick's code, name,
icons or images is used. Thyme is not affiliated with DeskTick.

## Licence

MIT, copyright TheScottBot. See LICENSE.
