// Package devanagari registers Noto Sans Devanagari (Hindi, Marathi, Nepali, Sanskrit and other languages in Devanagari) as a fallback font for flexui.
// Import it for its side effect:
//
//	import _ "github.com/sivaneshwararaj/flexui/fonts/devanagari"
//
// Text that Inter (flexui's built-in font) has no glyphs for is then
// drawn with Noto Sans Devanagari. It adds about 370 KB to a binary (Regular and
// Bold, embedded). Fonts must be registered before the first layout; an
// import does that.
package devanagari

import (
	_ "embed"

	"github.com/sivaneshwararaj/flexui/internal/text"
)

//go:embed NotoSansDevanagari-Regular.ttf
var regular []byte

//go:embed NotoSansDevanagari-Bold.ttf
var bold []byte

func init() {
	text.Register(&text.Pack{Name: "NotoSansDevanagari", Priority: 20, Regular: regular, Bold: bold})
}
