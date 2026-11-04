// Package style holds computed styles and parses the CSS subset described in
// spec/SUBSET.md.
//
// At milestone M1 it contains the computed-value types and the declaration
// parser for layout properties. Selectors, the cascade and paint properties
// arrive in M3.
package style

import "github.com/sivaneshwararaj/flexui/geom"

// LengthKind says how a Length is interpreted.
type LengthKind uint8

const (
	// LengthAuto is the keyword `auto` (for min-*: the automatic minimum size).
	LengthAuto LengthKind = iota
	// LengthPx is an absolute length; V is in geom.Unit (1/64 px).
	LengthPx
	// LengthPercent is a percentage; V is in 1/1000 of a percent.
	LengthPercent
	// LengthNone is the keyword `none` (only valid for max-*).
	LengthNone
)

// PercentScale is the number of Length.V steps in 1%.
const PercentScale = 1000

// Length is a computed length value. rem values are converted to px at
// parse time, so layout only ever sees px, percent, auto and none.
type Length struct {
	Kind LengthKind
	V    int64
}

// Auto is the `auto` length.
var Auto = Length{Kind: LengthAuto}

// None is the `none` length.
var None = Length{Kind: LengthNone}

// Px returns an absolute length of v whole pixels.
func Px(v int) Length { return Length{Kind: LengthPx, V: int64(geom.Px(v))} }

// Units returns an absolute length of u.
func Units(u geom.Unit) Length { return Length{Kind: LengthPx, V: int64(u)} }

// Percent returns p whole percent.
func Percent(p int) Length { return Length{Kind: LengthPercent, V: int64(p) * PercentScale} }

// IsAuto reports whether l is `auto`.
func (l Length) IsAuto() bool { return l.Kind == LengthAuto }

// Resolve converts l to Units. Percentages resolve against base, which is
// only used when baseOK is true. auto, none, and percentages of an unknown
// base report ok=false.
func (l Length) Resolve(base geom.Unit, baseOK bool) (v geom.Unit, ok bool) {
	switch l.Kind {
	case LengthPx:
		return geom.Unit(l.V), true
	case LengthPercent:
		if !baseOK {
			return 0, false
		}
		return geom.Unit(geom.MulDiv(int64(base), l.V, 100*PercentScale)), true
	}
	return 0, false
}

// Display is the `display` property.
type Display uint8

const (
	DisplayFlex Display = iota
	DisplayNone
)

// Position is the `position` property.
type Position uint8

const (
	PositionRelative Position = iota
	PositionAbsolute
)

// FlexDirection is the `flex-direction` property.
type FlexDirection uint8

const (
	Column FlexDirection = iota // initial value in this subset
	ColumnReverse
	Row
	RowReverse
)

// IsRow reports whether the main axis is horizontal.
func (d FlexDirection) IsRow() bool { return d == Row || d == RowReverse }

// IsReverse reports whether main-start is at the physical end.
func (d FlexDirection) IsReverse() bool { return d == RowReverse || d == ColumnReverse }

// FlexWrap is the `flex-wrap` property.
type FlexWrap uint8

const (
	NoWrap FlexWrap = iota
	Wrap
	WrapReverse
)

// Justify is the `justify-content` property.
type Justify uint8

const (
	JustifyFlexStart Justify = iota
	JustifyFlexEnd
	JustifyCenter
	JustifySpaceBetween
	JustifySpaceAround
	JustifySpaceEvenly
)

// Align is used for `align-items` and `align-self`.
type Align uint8

const (
	AlignAuto Align = iota // only valid for align-self
	AlignStretch
	AlignFlexStart
	AlignFlexEnd
	AlignCenter
)

// AlignContent is the `align-content` property.
type AlignContent uint8

const (
	ContentStretch AlignContent = iota
	ContentFlexStart
	ContentFlexEnd
	ContentCenter
	ContentSpaceBetween
	ContentSpaceAround
	ContentSpaceEvenly
)

// BoxSizing is the `box-sizing` property.
type BoxSizing uint8

const (
	ContentBox BoxSizing = iota
	BorderBox
)

// Overflow is the `overflow` property.
type Overflow uint8

const (
	OverflowVisible Overflow = iota
	OverflowHidden
	OverflowScroll
)

// Side indexes per-side arrays.
const (
	Top = iota
	Right
	Bottom
	Left
)
