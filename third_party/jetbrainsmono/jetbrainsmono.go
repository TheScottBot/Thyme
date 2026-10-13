// Package jetbrainsmono embeds the theme's only typeface, JetBrains Mono
// Regular v2.304, under the SIL Open Font Licence 1.1 in OFL.txt beside it.
// The font is embedded rather than read from the system so the core renders
// the same picture on any host (WD3, WD4). The .ttf and OFL.txt are vendored
// verbatim and must never be edited; SHA256SUMS records their digests.
package jetbrainsmono

import _ "embed"

// Regular is the font file's bytes, parsed once by internal/render.
//
//go:embed JetBrainsMono-Regular.ttf
var Regular []byte
