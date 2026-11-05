package render

// The optional font packs take part in every render test, as in the
// Chromium oracle page (scripts/flexui-page.cjs lists the same packs).
import (
	_ "github.com/sivaneshwararaj/flexui/fonts/arabic"
	_ "github.com/sivaneshwararaj/flexui/fonts/cjk"
	_ "github.com/sivaneshwararaj/flexui/fonts/devanagari"
	_ "github.com/sivaneshwararaj/flexui/fonts/hebrew"
	_ "github.com/sivaneshwararaj/flexui/fonts/tamil"
)
