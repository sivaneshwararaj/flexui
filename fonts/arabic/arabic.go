// Package arabic registers Noto Sans Arabic (Arabic script: Arabic, Persian, Urdu, ...; right to left) as a fallback font for flexui.
// Import it for its side effect:
//
//	import _ "github.com/sivaneshwararaj/flexui/fonts/arabic"
//
// Text that Inter (flexui's built-in font) has no glyphs for is then
// drawn with Noto Sans Arabic. It adds about 280 KB to a binary (Regular and
// Bold, embedded). Fonts must be registered before the first layout; an
// import does that.
package arabic

import (
	_ "embed"

	"github.com/sivaneshwararaj/flexui/internal/text"
)

//go:embed NotoSansArabic-Regular.ttf
var regular []byte

//go:embed NotoSansArabic-Bold.ttf
var bold []byte

func init() {
	text.Register(&text.Pack{Name: "NotoSansArabic", Priority: 30, Regular: regular, Bold: bold})
}
