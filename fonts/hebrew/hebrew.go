// Package hebrew registers Noto Sans Hebrew (Hebrew script; right to left) as a fallback font for flexui.
// Import it for its side effect:
//
//	import _ "github.com/sivaneshwararaj/flexui/fonts/hebrew"
//
// Text that Inter (flexui's built-in font) has no glyphs for is then
// drawn with Noto Sans Hebrew. It adds about 35 KB to a binary (Regular and
// Bold, embedded). Fonts must be registered before the first layout; an
// import does that.
package hebrew

import (
	_ "embed"

	"github.com/sivaneshwararaj/flexui/internal/text"
)

//go:embed NotoSansHebrew-Regular.ttf
var regular []byte

//go:embed NotoSansHebrew-Bold.ttf
var bold []byte

func init() {
	text.Register(&text.Pack{Name: "NotoSansHebrew", Priority: 40, Regular: regular, Bold: bold})
}
