package style

import (
	"fmt"
	"strings"

	"github.com/sivaneshwararaj/flexui/internal/geom"
)

// Declaration is one `property: value` pair.
type Declaration struct {
	Property string // lower-cased, trimmed
	Value    string // trimmed; keywords are matched case-insensitively
}

// Warning describes an unsupported or invalid piece of CSS. Unsupported
// input never silently changes behavior: it is ignored and reported.
type Warning struct {
	Property string
	Value    string
	Msg      string
}

func (w Warning) Error() string {
	if w.Property == "" {
		return w.Msg
	}
	return fmt.Sprintf("%s: %s: %s", w.Property, w.Value, w.Msg)
}

// ParseDeclarations splits a declaration block (the contents of a style=""
// attribute or of a rule's braces) into declarations. Comments are removed.
// `!important` is not supported: it is stripped and reported.
func ParseDeclarations(src string) ([]Declaration, []Warning) {
	var decls []Declaration
	var warns []Warning
	for _, part := range splitTop(stripComments(src), ';') {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		i := strings.IndexByte(part, ':')
		if i < 0 {
			warns = append(warns, Warning{Msg: fmt.Sprintf("malformed declaration %q", part)})
			continue
		}
		d := Declaration{
			Property: strings.ToLower(strings.TrimSpace(part[:i])),
			Value:    strings.TrimSpace(part[i+1:]),
		}
		if j := strings.Index(strings.ToLower(d.Value), "!important"); j >= 0 {
			d.Value = strings.TrimSpace(d.Value[:j])
			warns = append(warns, Warning{d.Property, d.Value, "!important is not supported and was ignored"})
		}
		decls = append(decls, d)
	}
	return decls, warns
}

func stripComments(s string) string {
	for {
		i := strings.Index(s, "/*")
		if i < 0 {
			return s
		}
		j := strings.Index(s[i+2:], "*/")
		if j < 0 {
			return s[:i]
		}
		s = s[:i] + " " + s[i+2+j+2:]
	}
}

// tokens splits a value at whitespace outside parentheses, so that
// `rgb(1, 2, 3)` stays one token.
func tokens(s string) []string {
	var out []string
	depth, start := 0, -1
	for i := 0; i <= len(s); i++ {
		var c byte = ' '
		if i < len(s) {
			c = s[i]
		}
		switch {
		case c == '(':
			depth++
		case c == ')' && depth > 0:
			depth--
		}
		space := c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
		if space && depth == 0 {
			if start >= 0 {
				out = append(out, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	return out
}

// splitTop splits s at sep, ignoring separators inside quotes or parens.
func splitTop(s string, sep byte) []string {
	var out []string
	depth := 0
	var quote byte
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '(':
			depth++
		case c == ')':
			if depth > 0 {
				depth--
			}
		case c == sep && depth == 0:
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// Context carries values needed to compute declarations.
type Context struct {
	// RootFontSize is the size of 1rem. Zero means 16px.
	RootFontSize geom.Unit
}

func (c Context) rem() geom.Unit {
	if c.RootFontSize == 0 {
		return geom.Px(16)
	}
	return c.RootFontSize
}

// ApplyAll applies declarations in order and returns all warnings.
func (s *Style) ApplyAll(decls []Declaration, ctx Context) []Warning {
	var warns []Warning
	for _, d := range decls {
		if err := s.Apply(d, ctx); err != nil {
			warns = append(warns, *err)
		}
	}
	return warns
}

// Apply applies one declaration. On error the style is left unchanged.
func (s *Style) Apply(d Declaration, ctx Context) *Warning {
	v := strings.ToLower(d.Value)
	bad := func(msg string) *Warning { return &Warning{d.Property, d.Value, msg} }
	invalid := bad("unsupported value")
	if v == "" {
		return bad("empty value")
	}
	if v == "inherit" || v == "initial" || v == "unset" || v == "revert" {
		return bad("CSS-wide keywords are not supported")
	}
	fields := tokens(v)

	// length parses one length, allowing the given keywords.
	length := func(tok string, allowAuto, allowNone, allowNeg bool) (Length, bool) {
		switch tok {
		case "auto":
			return Auto, allowAuto
		case "none":
			return None, allowNone
		}
		l, ok := parseLength(tok, ctx)
		if !ok || (!allowNeg && l.V < 0) {
			return Length{}, false
		}
		return l, true
	}
	one := func(dst *Length, allowAuto, allowNone, allowNeg bool) *Warning {
		if len(fields) != 1 {
			return invalid
		}
		l, ok := length(fields[0], allowAuto, allowNone, allowNeg)
		if !ok {
			return invalid
		}
		*dst = l
		return nil
	}
	sides := func(dst *[4]Length, allowAuto, allowNeg bool) *Warning {
		var ls []Length
		for _, f := range fields {
			l, ok := length(f, allowAuto, false, allowNeg)
			if !ok {
				return invalid
			}
			ls = append(ls, l)
		}
		out, ok := expandSides(ls)
		if !ok {
			return invalid
		}
		*dst = out
		return nil
	}
	side := func(dst *Length, allowAuto, allowNeg bool) *Warning {
		return one(dst, allowAuto, false, allowNeg)
	}
	keyword := func(table map[string]uint8, dst *uint8) *Warning {
		k, ok := table[v]
		if !ok {
			return invalid
		}
		*dst = k
		return nil
	}

	switch d.Property {
	case "display":
		// Elements are flex containers or inline boxes (§2, §7.1);
		// block/grid/... are rejected.
		switch v {
		case "flex":
			s.Display = DisplayFlex
		case "none":
			s.Display = DisplayNone
		case "inline":
			s.Display = DisplayInline
		default:
			return bad("only `flex`, `inline` and `none` are supported")
		}
	case "position":
		switch v {
		case "relative":
			s.Position = PositionRelative
		case "absolute":
			s.Position = PositionAbsolute
		default:
			return invalid
		}
	case "flex-direction":
		return keyword(map[string]uint8{"row": uint8(Row), "row-reverse": uint8(RowReverse),
			"column": uint8(Column), "column-reverse": uint8(ColumnReverse)}, (*uint8)(&s.FlexDirection))
	case "flex-wrap":
		return keyword(map[string]uint8{"nowrap": uint8(NoWrap), "wrap": uint8(Wrap),
			"wrap-reverse": uint8(WrapReverse)}, (*uint8)(&s.FlexWrap))
	case "flex-grow":
		f, ok := parseFactor(v)
		if !ok {
			return invalid
		}
		s.FlexGrow = f
	case "flex-shrink":
		f, ok := parseFactor(v)
		if !ok {
			return invalid
		}
		s.FlexShrink = f
	case "flex-basis":
		return one(&s.FlexBasis, true, false, false)
	case "flex":
		return s.applyFlex(fields, ctx, invalid)
	case "justify-content":
		return keyword(map[string]uint8{"flex-start": uint8(JustifyFlexStart), "flex-end": uint8(JustifyFlexEnd),
			"center": uint8(JustifyCenter), "space-between": uint8(JustifySpaceBetween),
			"space-around": uint8(JustifySpaceAround), "space-evenly": uint8(JustifySpaceEvenly)},
			(*uint8)(&s.JustifyContent))
	case "align-items":
		return keyword(alignTable, (*uint8)(&s.AlignItems))
	case "align-self":
		if v == "auto" {
			s.AlignSelf = AlignAuto
			return nil
		}
		return keyword(alignTable, (*uint8)(&s.AlignSelf))
	case "align-content":
		return keyword(map[string]uint8{"stretch": uint8(ContentStretch), "flex-start": uint8(ContentFlexStart),
			"flex-end": uint8(ContentFlexEnd), "center": uint8(ContentCenter),
			"space-between": uint8(ContentSpaceBetween), "space-around": uint8(ContentSpaceAround),
			"space-evenly": uint8(ContentSpaceEvenly)}, (*uint8)(&s.AlignContent))
	case "gap":
		if len(fields) < 1 || len(fields) > 2 {
			return invalid
		}
		r, ok1 := length(fields[0], false, false, false)
		c, ok2 := r, ok1
		if len(fields) == 2 {
			c, ok2 = length(fields[1], false, false, false)
		}
		if !ok1 || !ok2 {
			return invalid
		}
		s.RowGap, s.ColumnGap = r, c
	case "row-gap":
		return one(&s.RowGap, false, false, false)
	case "column-gap":
		return one(&s.ColumnGap, false, false, false)
	case "width":
		return one(&s.Width, true, false, false)
	case "height":
		return one(&s.Height, true, false, false)
	case "min-width":
		return one(&s.MinWidth, true, false, false)
	case "min-height":
		return one(&s.MinHeight, true, false, false)
	case "max-width":
		return one(&s.MaxWidth, false, true, false)
	case "max-height":
		return one(&s.MaxHeight, false, true, false)
	case "margin":
		return sides(&s.Margin, true, true)
	case "margin-top":
		return side(&s.Margin[Top], true, true)
	case "margin-right":
		return side(&s.Margin[Right], true, true)
	case "margin-bottom":
		return side(&s.Margin[Bottom], true, true)
	case "margin-left":
		return side(&s.Margin[Left], true, true)
	case "padding":
		return sides(&s.Padding, false, false)
	case "padding-top":
		return side(&s.Padding[Top], false, false)
	case "padding-right":
		return side(&s.Padding[Right], false, false)
	case "padding-bottom":
		return side(&s.Padding[Bottom], false, false)
	case "padding-left":
		return side(&s.Padding[Left], false, false)
	case "border-width":
		var ws []Length
		for _, f := range fields {
			u, ok := parseBorderWidth(f, ctx)
			if !ok {
				return invalid
			}
			ws = append(ws, Units(u))
		}
		out, ok := expandSides(ws)
		if !ok {
			return invalid
		}
		for i := range out {
			s.Border[i] = geom.Unit(out[i].V)
		}
	case "border-top-width", "border-right-width", "border-bottom-width", "border-left-width":
		if len(fields) != 1 {
			return invalid
		}
		u, ok := parseBorderWidth(fields[0], ctx)
		if !ok {
			return invalid
		}
		idx := map[string]int{"border-top-width": Top, "border-right-width": Right,
			"border-bottom-width": Bottom, "border-left-width": Left}[d.Property]
		s.Border[idx] = u
	case "box-sizing":
		return keyword(map[string]uint8{"content-box": uint8(ContentBox), "border-box": uint8(BorderBox)},
			(*uint8)(&s.BoxSizing))
	case "top":
		return side(&s.Inset[Top], true, true)
	case "right":
		return side(&s.Inset[Right], true, true)
	case "bottom":
		return side(&s.Inset[Bottom], true, true)
	case "left":
		return side(&s.Inset[Left], true, true)
	case "overflow":
		return keyword(map[string]uint8{"visible": uint8(OverflowVisible), "hidden": uint8(OverflowHidden),
			"scroll": uint8(OverflowScroll)}, (*uint8)(&s.Overflow))
	case "color":
		c, cur, ok := parseColor(v)
		if !ok {
			return invalid
		}
		if !cur { // `color: currentcolor` keeps the inherited color
			s.Color = c
		}
	case "background-color", "background":
		c, cur, ok := parseColor(v)
		if !ok || cur {
			if d.Property == "background" {
				return bad("only a single color is supported in `background`")
			}
			return invalid
		}
		s.BackgroundColor = c
	case "border-color":
		if len(fields) != 1 {
			return bad("only one border-color for all sides is supported (D15)")
		}
		c, cur, ok := parseColor(v)
		if !ok {
			return invalid
		}
		s.BorderColor, s.BorderColorCurrent = c, cur
	case "border-style":
		switch v {
		case "solid":
		case "none", "hidden":
			s.Border = [4]geom.Unit{}
		default:
			return bad("only `solid` and `none` borders are supported")
		}
	case "border":
		return s.applyBorder(fields, ctx, invalid)
	case "border-radius":
		var ls []Length
		for _, f := range fields {
			l, ok := length(f, false, false, false)
			if !ok {
				return bad("only 1-4 lengths are supported (no `/` elliptical syntax)")
			}
			ls = append(ls, l)
		}
		out, ok := expandSides(ls)
		if !ok {
			return invalid
		}
		// expandSides order (top,right,bottom,left) = (TL,TR,BR,BL) here.
		for i := range out {
			s.BorderRadius[i] = [2]Length{out[i], out[i]}
		}
	case "border-top-left-radius", "border-top-right-radius", "border-bottom-right-radius", "border-bottom-left-radius":
		if len(fields) < 1 || len(fields) > 2 {
			return invalid
		}
		h, ok1 := length(fields[0], false, false, false)
		vv, ok2 := h, ok1
		if len(fields) == 2 {
			vv, ok2 = length(fields[1], false, false, false)
		}
		if !ok1 || !ok2 {
			return invalid
		}
		idx := map[string]int{"border-top-left-radius": 0, "border-top-right-radius": 1,
			"border-bottom-right-radius": 2, "border-bottom-left-radius": 3}[d.Property]
		s.BorderRadius[idx] = [2]Length{h, vv}
	case "opacity":
		var o int64
		var ok bool
		if strings.HasSuffix(v, "%") {
			o, ok = parseFixed(v[:len(v)-1], 10)
		} else {
			o, ok = parseFixed(v, 1000)
		}
		if !ok || len(fields) != 1 {
			return invalid
		}
		s.Opacity = int32(min(max(o, 0), 1000))
	case "box-shadow":
		if len(splitTop(v, ',')) > 1 {
			return bad("only one shadow is supported (D16)")
		}
		return s.applyShadow(fields, ctx, bad, invalid)
	case "font-family":
		s.FontFamily = d.Value
	case "font-size":
		var l Length
		if w := one(&l, false, false, false); w != nil {
			return w
		}
		if l.Kind != LengthPx {
			return bad("only px and rem font sizes are supported")
		}
		s.FontSize = geom.Unit(l.V)
	case "font-weight":
		switch v {
		case "normal":
			s.FontWeight = 400
		case "bold":
			s.FontWeight = 700
		default:
			n, ok := parseFixed(v, 1)
			if !ok || !isInteger(v) || n < 1 || n > 1000 {
				return invalid
			}
			s.FontWeight = int16(n)
		}
	case "line-height":
		if v == "normal" {
			s.LineHeight = LineHeight{}
			return nil
		}
		if n, ok := parseFixed(v, 1000); ok && n >= 0 {
			s.LineHeight = LineHeight{LineHeightNumber, n}
			return nil
		}
		l, ok := parseLength(v, ctx)
		if !ok || l.Kind != LengthPx || l.V < 0 {
			return invalid
		}
		s.LineHeight = LineHeight{LineHeightLength, l.V}
	case "text-align":
		return keyword(map[string]uint8{"left": uint8(TextAlignLeft), "right": uint8(TextAlignRight),
			"center": uint8(TextAlignCenter)}, (*uint8)(&s.TextAlign))
	case "white-space":
		return keyword(map[string]uint8{"normal": uint8(WhiteSpaceNormal), "nowrap": uint8(WhiteSpaceNowrap),
			"pre": uint8(WhiteSpacePre), "pre-wrap": uint8(WhiteSpacePreWrap)}, (*uint8)(&s.WhiteSpace))
	case "text-overflow":
		return keyword(map[string]uint8{"clip": uint8(TextOverflowClip), "ellipsis": uint8(TextOverflowEllipsis)},
			(*uint8)(&s.TextOverflow))
	case "z-index":
		if v == "auto" {
			s.ZAuto, s.ZIndex = true, 0
			return nil
		}
		n, ok := parseFixed(v, 1)
		if !ok || !isInteger(v) || n > 1<<31-1 || n < -1<<31 {
			return invalid
		}
		s.ZAuto, s.ZIndex = false, int32(n)
	default:
		return bad("unsupported property")
	}
	return nil
}

var alignTable = map[string]uint8{
	"stretch": uint8(AlignStretch), "flex-start": uint8(AlignFlexStart),
	"flex-end": uint8(AlignFlexEnd), "center": uint8(AlignCenter),
}

// applyFlex handles the `flex` shorthand:
//
//	none          -> 0 0 auto
//	auto          -> 1 1 auto
//	<g>           -> g 1 0
//	<g> <s>       -> g s 0
//	<g> <basis>   -> g 1 basis
//	<basis>       -> 1 1 basis
//	<g> <s> <basis>
func (s *Style) applyFlex(fields []string, ctx Context, invalid *Warning) *Warning {
	g, sh, b := Factor(FactorScale), Factor(FactorScale), Px(0)
	switch {
	case len(fields) == 1 && fields[0] == "none":
		g, sh, b = 0, 0, Auto
	case len(fields) == 1 && fields[0] == "auto":
		b = Auto
	default:
		var nums []Factor
		var basis *Length
		for _, f := range fields {
			if n, ok := parseFactor(f); ok && basis == nil && len(nums) < 2 {
				nums = append(nums, n)
				continue
			}
			if basis != nil {
				return invalid
			}
			var l Length
			if f == "auto" {
				l = Auto
			} else {
				var ok bool
				l, ok = parseLength(f, ctx)
				if !ok || l.V < 0 {
					return invalid
				}
			}
			basis = &l
		}
		if len(fields) > 3 || (len(nums) == 0 && basis == nil) {
			return invalid
		}
		if len(nums) > 0 {
			g = nums[0]
		}
		if len(nums) > 1 {
			sh = nums[1]
		}
		if basis != nil {
			b = *basis
		}
	}
	s.FlexGrow, s.FlexShrink, s.FlexBasis = g, sh, b
	return nil
}

// applyBorder handles `border: [<width>] [solid|none] [<color>]`.
func (s *Style) applyBorder(fields []string, ctx Context, invalid *Warning) *Warning {
	w, c, cur := geom.Px(3), Color{}, true // CSS initial: medium, currentcolor
	none := false
	seen := map[string]bool{}
	for _, f := range fields {
		switch {
		case f == "solid" && !seen["style"]:
			seen["style"] = true
		case (f == "none" || f == "hidden") && !seen["style"]:
			seen["style"], none = true, true
		default:
			if u, ok := parseBorderWidth(f, ctx); ok && !seen["width"] {
				seen["width"], w = true, u
				continue
			}
			if cc, cu, ok := parseColor(f); ok && !seen["color"] {
				seen["color"], c, cur = true, cc, cu
				continue
			}
			return invalid
		}
	}
	if none || !seen["style"] {
		// A border without a style draws nothing in CSS.
		w = 0
	}
	s.Border = [4]geom.Unit{w, w, w, w}
	s.BorderColor, s.BorderColorCurrent = c, cur
	return nil
}

// applyShadow handles `box-shadow: none | <x> <y> [<blur> [<spread>]] [<color>]`
// (the color may also come first).
func (s *Style) applyShadow(fields []string, ctx Context, bad func(string) *Warning, invalid *Warning) *Warning {
	if len(fields) == 1 && fields[0] == "none" {
		s.Shadow = BoxShadow{}
		return nil
	}
	sh := BoxShadow{Set: true, CurrentColor: true}
	var ls []geom.Unit
	for i, f := range fields {
		if f == "inset" {
			return bad("inset shadows are not supported (D16)")
		}
		if l, ok := parseLength(f, ctx); ok && l.Kind == LengthPx {
			ls = append(ls, geom.Unit(l.V))
			continue
		}
		if c, cur, ok := parseColor(f); ok && sh.CurrentColor && (i == 0 || i == len(fields)-1) {
			sh.Color, sh.CurrentColor = c, cur
			continue
		}
		return invalid
	}
	if len(ls) < 2 || len(ls) > 4 || (len(ls) > 2 && ls[2] < 0) {
		return invalid
	}
	sh.DX, sh.DY = ls[0], ls[1]
	if len(ls) > 2 {
		sh.Blur = ls[2]
	}
	if len(ls) > 3 {
		sh.Spread = ls[3]
	}
	s.Shadow = sh
	return nil
}

func expandSides(ls []Length) ([4]Length, bool) {
	switch len(ls) {
	case 1:
		return [4]Length{ls[0], ls[0], ls[0], ls[0]}, true
	case 2:
		return [4]Length{ls[0], ls[1], ls[0], ls[1]}, true
	case 3:
		return [4]Length{ls[0], ls[1], ls[2], ls[1]}, true
	case 4:
		return [4]Length{ls[0], ls[1], ls[2], ls[3]}, true
	}
	return [4]Length{}, false
}

func parseBorderWidth(tok string, ctx Context) (geom.Unit, bool) {
	switch tok {
	case "thin":
		return geom.Px(1), true
	case "medium":
		return geom.Px(3), true
	case "thick":
		return geom.Px(5), true
	}
	l, ok := parseLength(tok, ctx)
	if !ok || l.Kind != LengthPx || l.V < 0 {
		return 0, false
	}
	return snapBorder(geom.Unit(l.V)), true
}

// snapBorder applies the computed-value rule for border widths, as
// browsers do: a width of at least 1px is floored to whole pixels, a
// width between 0 and 1px becomes 1px. Layout therefore sees the same
// widths that are painted, at every device scale.
func snapBorder(w geom.Unit) geom.Unit {
	switch {
	case w <= 0:
		return 0
	case w < geom.Px(1):
		return geom.Px(1)
	}
	return w / geom.UnitsPerPx * geom.UnitsPerPx
}

// parseLength parses `<number>px`, `<number>%`, `<number>rem` or a bare `0`.
func parseLength(tok string, ctx Context) (Length, bool) {
	switch {
	case strings.HasSuffix(tok, "px"):
		u, ok := parseUnits(tok[:len(tok)-2])
		return Length{LengthPx, int64(u)}, ok
	case strings.HasSuffix(tok, "rem"):
		micro, ok := parseFixed(tok[:len(tok)-3], 1_000_000)
		if !ok {
			return Length{}, false
		}
		return Length{LengthPx, roundDiv(micro*int64(ctx.rem()), 1_000_000)}, true
	case strings.HasSuffix(tok, "%"):
		p, ok := parseFixed(tok[:len(tok)-1], PercentScale)
		return Length{LengthPercent, p}, ok
	}
	if n, ok := parseFixed(tok, 1_000_000); ok && n == 0 {
		return Px(0), true
	}
	return Length{}, false
}

// parseUnits parses a decimal pixel count into Units, rounding to the
// nearest 1/64 px (ties away from zero).
func parseUnits(s string) (geom.Unit, bool) {
	micro, ok := parseFixed(s, 1_000_000)
	if !ok {
		return 0, false
	}
	return geom.Unit(roundDiv(micro*geom.UnitsPerPx, 1_000_000)), true
}

func parseFactor(s string) (Factor, bool) {
	n, ok := parseFixed(s, FactorScale)
	if !ok || n < 0 {
		return 0, false
	}
	return Factor(n), true
}

func roundDiv(a, b int64) int64 {
	if a < 0 {
		return -((-a + b/2) / b)
	}
	return (a + b/2) / b
}

func isInteger(s string) bool { return !strings.ContainsAny(s, ".eE") }

// parseFixed parses a plain decimal number (optional sign, digits, optional
// fraction; no exponent) into an integer scaled by scale, truncating digits
// beyond the scale's precision. It uses no floating point.
func parseFixed(s string, scale int64) (int64, bool) {
	if s == "" {
		return 0, false
	}
	neg := false
	switch s[0] {
	case '-':
		neg = true
		s = s[1:]
	case '+':
		s = s[1:]
	}
	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i+1:]
	}
	if intPart == "" && frac == "" {
		return 0, false
	}
	var n int64
	for _, c := range intPart {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int64(c-'0')
		if n > 1<<40 {
			return 0, false
		}
	}
	n *= scale
	div := scale
	for _, c := range frac {
		if c < '0' || c > '9' {
			return 0, false
		}
		if div >= 10 {
			div /= 10
			n += int64(c-'0') * div
		}
	}
	if neg {
		n = -n
	}
	return n, true
}
