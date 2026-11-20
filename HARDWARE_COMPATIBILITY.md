# Hardware compatibility

What Thyme has been run on, by whom, and when. Only the author adds a row
saying a gate was cleared. The agent's rows are measurements and smoke runs,
never gates.

## Supported

Windows 10 version 1703 or later, and Windows 11, on amd64 and arm64 (WD10;
the 1703 floor is Per Monitor v2 DPI awareness, read from Microsoft Learn in
WE5).

macOS 13 or later on arm64 (WD11; the 13 floor is `SMAppService`, which Open at
Login uses). Built from source and ad hoc signed, so for the Macs it was built
on, not for distribution.

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
| 7 November 2026 | agent | Apple M6 MacBook Pro, macOS 27.2 (Darwin 27.2.0), 2x display | macos-port working tree, arm64, release flags, ad hoc bundle | smoke run: start, second instance exits, on screen at the saved position at 192 DPI, tray added, idle CPU | as expected, 0.1 percent CPU; physical footprint 33.3 MB three seconds after start, 30.6 MB after 106 s; window placement read from the window server across a 2x and a 1x display; the screen was not observed and no click was made |
| 7 November 2026 | author, reported in the conversation | the same Mac | macos-port working tree, arm64, release flags, ad hoc bundle in `/Applications` | looks right at 75 percent, drag between a 2x and a 1x display, both menus, Always on Top, Quit and relaunch, clicks on and past the glow, Open at Login accepted by macOS, and Thyme started by itself after a restart | as expected; the macOS gate is not cleared; Show Desktop, Spaces, display removal, VoiceOver, sleep and clock changes set aside as out of scope |

## Measurements owed

| Measurement | Budget | Status |
|---|---|---|
| Working set after ten minutes idle, windows/arm64, release build | 30 MB (WU7, decided by the author in WE8) | reported by the author, 22.4 MB; within budget |
| Working set after ten minutes idle, windows/amd64, release build | 30 MB | reported by the author, 22.7 MB after running all day; within budget |
| Physical footprint after ten minutes idle, darwin/arm64, release build | not set; the Windows figure measures something different, so the author sets this from the reading | owed |

## Reference points

| Program | Working set | Private | Measured |
|---|---|---|---|
| A Go program that only sleeps, with `time/tzdata`, release flags, arm64 | 7.0 MB | 11.8 MB | by the agent, 13 October 2026 |
| Thyme before the WE8 memory changes, arm64 | 24.0 to 26.3 MB | 23.0 to 24.8 MB | by the agent, 13 October 2026, over 180 s |
