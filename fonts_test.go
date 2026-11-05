package flexui

// The optional font packs take part in the root package's tests (Latin
// goldens are unaffected: Inter covers them).
import (
	_ "github.com/sivaneshwararaj/flexui/fonts/arabic"
	_ "github.com/sivaneshwararaj/flexui/fonts/cjk"
	_ "github.com/sivaneshwararaj/flexui/fonts/devanagari"
	_ "github.com/sivaneshwararaj/flexui/fonts/hebrew"
	_ "github.com/sivaneshwararaj/flexui/fonts/tamil"
)
