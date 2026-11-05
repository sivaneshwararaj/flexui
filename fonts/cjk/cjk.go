// Package cjk registers Noto Sans SC (the Simplified Chinese subset of
// Noto Sans CJK, version 2.004) as a fallback font for flexui: Chinese
// characters, Japanese kana and CJK punctuation. Import it for its side
// effect:
//
//	import _ "github.com/sivaneshwararaj/flexui/fonts/cjk"
//
// It adds about 8.3 MB to a binary. Only the Regular face is bundled:
// bold CJK text uses it too (a Bold face would add another 8.5 MB).
// Japanese text gets the Chinese glyph forms of shared characters, and
// Korean Hangul is not covered (see fonts/README.md).
package cjk

import (
	_ "embed"

	"github.com/sivaneshwararaj/flexui/internal/text"
)

//go:embed NotoSansSC-Regular.otf
var regular []byte

func init() {
	text.Register(&text.Pack{Name: "NotoSansSC", Priority: 50, Regular: regular})
}
