# Provenance

Where every file not written for Thyme came from. Written once at the import;
after that the repository's own history is the provenance.

## JetBrains Mono Regular

| | |
|---|---|
| Repository | https://github.com/JetBrains/JetBrainsMono |
| Release | v2.304 |
| Commit | cd5227bd1f61dff3bbd6c814ceaf7ffd95e947d9 |
| Imported | 13 October 2026, by the agent with the author's approval |
| Licence | SIL Open Font Licence 1.1, `third_party/jetbrainsmono/OFL.txt` |

| Tree path | Upstream path | Git blob | SHA-256 |
|---|---|---|---|
| `third_party/jetbrainsmono/JetBrainsMono-Regular.ttf` | `fonts/ttf/JetBrainsMono-Regular.ttf` | dff66cc50702c75abd025dcf49f62a4dcc2d72de | a0bf60ef0f83c5ed4d7a75d45838548b1f6873372dfac88f71804491898d138f |
| `third_party/jetbrainsmono/OFL.txt` | `OFL.txt` | 8bee4148c1d54dbf5dae6d6c117fc80414266abb | 30f0c136e3c88e422d0791acd97238870f9054a9729bc34cf2ff0d4ed8cac4ad |

Both files are LF only and binary or ASCII, so the stored and upstream forms
are the same bytes. `third_party/jetbrainsmono/SHA256SUMS` holds the digests,
and `TestVendoredFiles_MatchRecordedDigests` checks the files and the embedded
font against it on every run. The files are excluded from the prose scan by
their exact paths in `tools/repocheck/main.go`.

Nothing from DeskTick is in this repository (WD1, clean room).
