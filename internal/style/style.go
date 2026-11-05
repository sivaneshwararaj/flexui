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

	// Paint properties (not inherited).
	BackgroundColor    Color
	BorderColor        Color
	BorderColorCurrent bool         // border-color: currentcolor (initial)
	BorderRadius       [4][2]Length // TL, TR, BR, BL; [0] horizontal, [1] vertical
	Opacity            int32        // 0..1000
	Shadow             BoxShadow

	// Text properties. Color, Font*, LineHeight, TextAlign and WhiteSpace
	// inherit; TextOverflow does not. Used from M4 on.
	Color        Color
	FontFamily   string // raw comma-separated list, case preserved
	Lang         string // content language: the inherited lang attribute (not a CSS property)
	Direction    Direction
	FontSize     geom.Unit
	FontWeight   int16 // 100..900
	LineHeight   LineHeight
	TextAlign    TextAlign
	WhiteSpace   WhiteSpace
	TextOverflow TextOverflow
}

// BoxShadow is a single outer box shadow.
type BoxShadow struct {
	Set                  bool
	DX, DY, Blur, Spread geom.Unit
	Color                Color
	CurrentColor         bool // no color given: use the element's color
}

// LineHeight is `normal`, a number (V in 1/1000) or a length (V in Units).
type LineHeight struct {
	Kind LineHeightKind
	V    int64
}

// LineHeightKind says how LineHeight.V is interpreted.
type LineHeightKind uint8

const (
	LineHeightNormal LineHeightKind = iota
	LineHeightNumber
	LineHeightLength
)

// TextAlign is the `text-align` property.
type TextAlign uint8

const (
	TextAlignLeft TextAlign = iota
	TextAlignRight
	TextAlignCenter
	TextAlignStart // initial: left in ltr, right in rtl
	TextAlignEnd
)

// Physical resolves start and end against a direction.
func (a TextAlign) Physical(d Direction) TextAlign {
	switch {
	case a == TextAlignStart && d == DirectionRTL, a == TextAlignEnd && d != DirectionRTL:
		return TextAlignRight
	case a == TextAlignStart, a == TextAlignEnd:
		return TextAlignLeft
	}
	return a
}

// Direction is the `direction` property (also set by the dir attribute).
type Direction uint8

const (
	DirectionLTR Direction = iota
	DirectionRTL
)

// WhiteSpace is the `white-space` property.
type WhiteSpace uint8

const (
	WhiteSpaceNormal WhiteSpace = iota
	WhiteSpaceNowrap
	WhiteSpacePre
	WhiteSpacePreWrap
)

// TextOverflow is the `text-overflow` property.
type TextOverflow uint8

const (
	TextOverflowClip TextOverflow = iota
	TextOverflowEllipsis
)

// Inherit returns the initial style of a child of an element with
// computed style parent: initial values, plus parent's inherited
// properties.
func Inherit(parent *Style) Style {
	s := Initial()
	if parent != nil {
		s.Color = parent.Color
		s.FontFamily = parent.FontFamily
		s.Lang = parent.Lang
		s.Direction = parent.Direction
		s.FontSize = parent.FontSize
		s.FontWeight = parent.FontWeight
		s.LineHeight = parent.LineHeight
		s.TextAlign = parent.TextAlign
		s.WhiteSpace = parent.WhiteSpace
	}
	return s
}

// Finish resolves values that depend on other properties (currentcolor).
// Call it after all declarations are applied.
func (s *Style) Finish() {
	if s.BorderColorCurrent {
		s.BorderColor = s.Color
	}
	if s.Shadow.Set && s.Shadow.CurrentColor {
		s.Shadow.Color = s.Color
	}
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

		BorderColorCurrent: true,
		BorderRadius:       [4][2]Length{{Px(0), Px(0)}, {Px(0), Px(0)}, {Px(0), Px(0)}, {Px(0), Px(0)}},
		Opacity:            1000,
		Color:              Color{0, 0, 0, 255},
		FontSize:           geom.Px(16),
		FontWeight:         400,
		TextAlign:          TextAlignStart,
	}
}

// SameLayout reports whether two computed styles can only differ in how
// the box is painted: they may differ in colors, background, border
// color, radii, opacity, shadow and z-index, but in nothing that affects
// layout or text shaping.
func SameLayout(a, b *Style) bool {
	x, y := *a, *b
	for _, s := range []*Style{&x, &y} {
		s.BackgroundColor, s.BorderColor, s.BorderColorCurrent = Color{}, Color{}, false
		s.BorderRadius = [4][2]Length{}
		s.Opacity, s.Shadow, s.Color = 0, BoxShadow{}, Color{}
		s.ZIndex, s.ZAuto = 0, false
	}
	return x == y
}
