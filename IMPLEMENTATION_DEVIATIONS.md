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

## The settings file carries a version field

- Rule: the specification's WE3 lists six settings fields and no version; WD6
  calls the file six fields.
- Instead: a seventh member, `"version": 1`, is required. A missing or other
  version refuses the whole file: the defaults are used and the file is
  protected from being saved over.
- Why: datum 0.17 rule 7 forbids accepting input without a version check, and
  0.11 requires breaking changes to be versioned. Protecting the file means an
  older Thyme never destroys a newer one's settings.
- Author's decision: 13 October 2026, the WE3 version question, option
  "add version, no clobber".
- Covering test: `TestParse_VersionProblemsYieldDefaultsAndForbidOverwriting`.

## Overwrite protection covers every whole file refusal, not only version

- Rule: the author's version decision protects a file of an unknown version.
- Instead: `LoadReport.OverwriteAllowed` is false for every file that existed
  but was refused whole: unreadable, over the size bound, not one JSON object,
  or of a missing or other version.
- Why: each of these is a file Thyme did not understand, most likely a broken
  hand edit, and saving defaults over it would destroy the user's work with
  nothing to say so. A missing file is still written normally.
- Author's decision: proposed by the agent on 13 October 2026, awaiting the
  author. Reverting it is one line in `refusedWhole`.
- Covering tests: `TestParse_VersionProblemsYieldDefaultsAndForbidOverwriting`
  and `TestLoad_EveryWholeFileRefusalForbidsOverwriting`.

## Per field decoding in place of DisallowUnknownFields at the top level

- Rule: WD6 option (a) names `encoding/json` with `DisallowUnknownFields`.
- Instead: the top level object is read member by member with
  `json.Decoder.Token`, unknown members are rejected individually, and each
  known member is decoded on its own. Nested objects (position and each zone)
  are read the same way and must hold exactly their members.
- Why: WE3 requires a rejection to cost only that field, which a whole
  document `DisallowUnknownFields` decode cannot do: one unknown member fails
  everything. Reading members individually also catches duplicate members,
  which `encoding/json` resolves silently by letting the last one win, and a
  JSON `null`, which it silently treats as the zero value. Unknown fields are
  still rejected, as datum 0.11 requires.
- Author's decision: the requirement is the author's (WD6 and WE3); the
  mechanism is the agent's, recorded here for review.
- Covering tests: `TestParse_SpecificationRejectionsHaveDistinctReasons`,
  `TestParse_DuplicateTopLevelFieldFallsBack`, `TestParse_RejectsEachFieldLevelProblem`.

## A product package beside those of Part 2

- Rule: the specification's Part 2 table lists the packages.
- Instead: `internal/product` holds the WD2 name constant, standard library
  only.
- Why: WD2 requires the name once, as a constant, with every derived string
  built from it. Both `internal/settings` (the settings folder) and the shell
  (exe, Run value, mutex, tooltip) need it, and the pure packages may not
  import the shell, so neither existing home works.
- Author's decision: proposed by the agent on 13 October 2026, awaiting the
  author.
- Covering test: `TestRelativeFilePath_IsDerivedFromTheProductName`; the
  import boundary check covers the new package like every other.
