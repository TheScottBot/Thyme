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

## Testing deviations

- No race detector on windows/arm64. See IMPLEMENTATION_DEVIATIONS.md.

## Author gates

None in WE1.
