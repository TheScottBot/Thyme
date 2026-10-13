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

- WE4: the 100 percent golden image (`internal/render/testdata/golden/mockup_100.png`), viewed beside `docs/theme.png`, is accepted as the theme. Outstanding.
