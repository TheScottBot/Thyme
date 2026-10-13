# Implementation deviations

Every departure from the project specification or from ENGINEERING_DATUM.md
version 2: the rule, what is done instead, why, the author's decision, and the
covering test.

## Tests on windows/arm64 run without the race detector

- Rule: the specification's WE1 asks for `go test -race` on every package that
  runs on the runner, and datum 0.10 asks for the language's memory analysis in
  CI.
- Instead: the `windows-11-arm` runner runs `go test -count=1 ./...` without
  `-race`. The `ubuntu-24.04` and `windows-2025` runners run the same suites
  with `-race`.
- Why: Go 1.27.2 has no race detector on windows/arm64 (read from
  `src/internal/platform/supported.go`, and seen for real: `go test -race`
  prints that it is not supported on windows/arm64). The arm64 runner is
  kept because the rasteriser's floating point results can differ by
  architecture, and the author's machine is arm64.
- Author's decision: 13 October 2026, option (b) of the WE1 question recorded in
  docs/evaluation/ACTUAL_CONTRACT_EVALUATION.md.
- Covering test: the same suites with `-race` on the two amd64 runners; the
  `race: false` matrix entry in `.github/workflows/ci.yml`.
