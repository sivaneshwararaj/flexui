// Package tamil registers Noto Sans Tamil (Tamil script) as a fallback font for flexui.
// Import it for its side effect:
//
//	import _ "github.com/sivaneshwararaj/flexui/fonts/tamil"
//
// Text that Inter (flexui's built-in font) has no glyphs for is then
// drawn with Noto Sans Tamil. It adds about 95 KB to a binary (Regular and
// Bold, embedded). Fonts must be registered before the first layout; an
// import does that.
package tamil

import (
	_ "embed"

	"github.com/sivaneshwararaj/flexui/internal/text"
)

//go:embed NotoSansTamil-Regular.ttf
var regular []byte

//go:embed NotoSansTamil-Bold.ttf
var bold []byte

func init() {
	text.Register(&text.Pack{Name: "NotoSansTamil", Priority: 10, Regular: regular, Bold: bold})
}
