package style

import "github.com/sivaneshwararaj/flexui/internal/geom"

// Factor is a flex-grow / flex-shrink factor in 1/1000 units (1000 == 1).
type Factor int64

// FactorScale is the Factor value that represents 1.
const FactorScale = 1000

// Style is the computed style of one element. Field order follows
// spec/SUBSET.md. Paint properties are added in M3.
type Style struct {
	Display  Display
	Position Position

	FlexDirection  FlexDirection
	FlexWrap       FlexWrap
	FlexGrow       Factor
	FlexShrink     Factor
	FlexBasis      Length
	JustifyContent Justify
	AlignItems     Align // never AlignAuto
	AlignSelf      Align
	AlignContent   AlignContent
	RowGap         Length
	ColumnGap      Length

	Width, Height       Length
	MinWidth, MinHeight Length // auto = automatic minimum size
	MaxWidth, MaxHeight Length // none = no maximum

	Margin  [4]Length    // top right bottom left; auto allowed
	Padding [4]Length    // no auto
	Border  [4]geom.Unit // border widths, px only
	Inset   [4]Length    // top right bottom left (for position offsets)

	BoxSizing BoxSizing
	Overflow  Overflow
	ZIndex    int32
	ZAuto     bool
}

// Initial returns the initial (default) style of every element.
func Initial() Style {
	return Style{
		FlexShrink: FactorScale,
		FlexBasis:  Auto,
		AlignItems: AlignStretch,
		AlignSelf:  AlignAuto,
		RowGap:     Px(0),
		ColumnGap:  Px(0),
		Width:      Auto, Height: Auto,
		MinWidth: Auto, MinHeight: Auto,
		MaxWidth: None, MaxHeight: None,
		Margin:  [4]Length{Px(0), Px(0), Px(0), Px(0)},
		Padding: [4]Length{Px(0), Px(0), Px(0), Px(0)},
		Inset:   [4]Length{Auto, Auto, Auto, Auto},
		ZAuto:   true,
	}
}
