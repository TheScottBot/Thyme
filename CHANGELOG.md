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

No contract digest applies: Thyme implements no shared contract.
