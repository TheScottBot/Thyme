# Hardware compatibility

What Thyme has been run on, by whom, and when. Only the author adds a row
saying a gate was cleared. The agent's rows are measurements and smoke runs,
never gates.

## Supported

Windows 10 version 1703 or later, and Windows 11, on amd64 and arm64 (WD10;
the 1703 floor is Per Monitor v2 DPI awareness, read from Microsoft Learn in
WE5).

## Runs

| Date | Who | Machine | Build | What | Result |
|---|---|---|---|---|---|
| 13 October 2026 | agent | Windows 11 Home 10.0.26200, arm64, 144 DPI | WE5 working tree, arm64 | smoke run: start, second instance exits, start up log | as expected; the screen was not observed |
| 13 October 2026 | agent | the same | WE6 working tree, arm64 | smoke run: desktop view found, tray icon added, registry untouched | as expected; the screen was not observed |
| 13 October 2026 | agent | the same | WE8 working tree, arm64, release flags | idle memory over 120 s, seconds on | working set 22.0 to 22.2 MB, private 20.5 to 20.7 MB; 109 ms of processor time over the 115 s after start |
| 13 October 2026 | author | not recorded | WE5 build | WE5 gate | cleared, in the author's words "WE5 gate passed" |
| 13 October 2026 | author | not recorded | WE6 build | WE6 gate | cleared, in the author's words "WE6 gate passed" |
| 13 October 2026 | author | not recorded | WE7 build | WE7 gate | cleared, in the author's words "WE7 gate passed" |
| 13 October 2026 | author, reported in the conversation | the author's Windows arm64 machine, not otherwise named | 1.0.0 release build, arm64 | `(Get-Process thyme).WorkingSet64 / 1MB` | 22.44140625 MB; whether ten minutes had passed and seconds were on was not stated |
| 14 October 2026 | author, reported in the conversation | the author's Windows amd64 machine, not otherwise named | release build, amd64 | ran "all day" in the author's words, then `(Get-Process thyme).WorkingSet64 / 1MB` and its total processor time | 22.65234375 MB working set; 118,453 ms of processor time over the day; the hours running, the build's commit and whether seconds were on were not stated |

## Measurements owed

| Measurement | Budget | Status |
|---|---|---|
| Working set after ten minutes idle, windows/arm64, release build | 30 MB (WU7, decided by the author in WE8) | reported by the author, 22.4 MB; within budget |
| Working set after ten minutes idle, windows/amd64, release build | 30 MB | reported by the author, 22.7 MB after running all day; within budget |

## Reference points

| Program | Working set | Private | Measured |
|---|---|---|---|
| A Go program that only sleeps, with `time/tzdata`, release flags, arm64 | 7.0 MB | 11.8 MB | by the agent, 13 October 2026 |
| Thyme before the WE8 memory changes, arm64 | 24.0 to 26.3 MB | 23.0 to 24.8 MB | by the agent, 13 October 2026, over 180 s |
