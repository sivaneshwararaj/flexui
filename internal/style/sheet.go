package style

import (
	"fmt"
	"math/bits"
	"sort"
	"strings"
)

// Element is what selector matching needs from a document node.
type Element interface {
	Tag() string // lower-case
	ID() string
	HasClass(name string) bool
	ParentElement() Element // nil for the root
	State() State
}

// State holds dynamic pseudo-class flags of an element.
type State uint8

const (
	StateHover State = 1 << iota
	StateFocus
	StateActive
)

// Sheet is a parsed style sheet.
type Sheet struct {
	Rules []Rule
}

// Rule is one selector with its (validated) declarations. A source rule
// with a selector list becomes several Rules sharing an Order.
type Rule struct {
	Sel   Selector
	Decls []Declaration
	Order int // source position of the rule, across all sheets
	Spec  int // Sel.Specificity()
}

// Selector is a sequence of compound selectors joined by combinators.
type Selector struct {
	Parts []Compound
	Comb  []byte // Comb[i] joins Parts[i] and Parts[i+1]: ' ' or '>'
}

// HasPseudo reports whether any compound uses a pseudo-class.
func (s *Selector) HasPseudo() bool {
	for _, p := range s.Parts {
		if p.Pseudo != 0 {
			return true
		}
	}
	return false
}

// Specificity returns the selector's CSS specificity packed as
// a<<16 | b<<8 | c: a counts ids, b classes and pseudo-classes, c type
// selectors (`*` counts nothing). Each count saturates at 255.
func (s *Selector) Specificity() int {
	var a, b, c int
	for _, p := range s.Parts {
		if p.ID != "" {
			a++
		}
		b += len(p.Classes) + bits.OnesCount8(uint8(p.Pseudo))
		if p.Tag != "" {
			c++
		}
	}
	return min(a, 255)<<16 | min(b, 255)<<8 | min(c, 255)
}

// Compound is `tag#id.class:pseudo`; empty fields match anything.
type Compound struct {
	Tag     string // "" = any
	ID      string
	Classes []string
	Pseudo  State
}

// ParseSheet parses a style sheet. orderBase offsets rule orders so that
// several sheets can be cascaded together in sequence.
func ParseSheet(src string, orderBase int) (*Sheet, []Warning) {
	var sh Sheet
	var warns []Warning
	src = stripComments(src)
	order := orderBase
	for i := 0; i < len(src); {
		open := strings.IndexByte(src[i:], '{')
		if open < 0 {
			if strings.TrimSpace(src[i:]) != "" {
				warns = append(warns, Warning{Msg: fmt.Sprintf("trailing text without a rule body: %q", strings.TrimSpace(src[i:]))})
			}
			break
		}
		prelude := strings.TrimSpace(src[i : i+open])
		end := matchBrace(src, i+open)
		body := src[i+open+1 : end]
		i = end + 1
		if strings.HasPrefix(prelude, "@") {
			warns = append(warns, Warning{Msg: fmt.Sprintf("at-rule %q is not supported and was skipped", prelude)})
			continue
		}
		decls, w := ParseDeclarations(body)
		warns = append(warns, w...)
		// Validate once per rule, so warnings are not repeated per element.
		var valid []Declaration
		probe := Initial()
		for _, d := range decls {
			if err := probe.Apply(d, Context{}); err != nil {
				warns = append(warns, *err)
				continue
			}
			valid = append(valid, d)
		}
		for _, part := range splitTop(prelude, ',') {
			sel, err := ParseSelector(strings.TrimSpace(part))
			if err != nil {
				warns = append(warns, Warning{Msg: err.Error()})
				continue
			}
			sh.Rules = append(sh.Rules, Rule{Sel: sel, Decls: valid, Order: order, Spec: sel.Specificity()})
		}
		order++
	}
	return &sh, warns
}

// matchBrace returns the index of the '}' closing the '{' at open (or
// len(s)-1 if unbalanced), skipping quoted strings.
func matchBrace(s string, open int) int {
	depth := 0
	var quote byte
	for i := open; i < len(s); i++ {
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
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return len(s) - 1
}

// ParseSelector parses one complex selector of the subset: compound
// selectors (type, *, #id, .class, :hover, :focus, :active) joined by
// descendant (whitespace) or child (>) combinators.
func ParseSelector(s string) (Selector, error) {
	var sel Selector
	if s == "" {
		return sel, fmt.Errorf("empty selector")
	}
	i := 0
	expectCompound := true
	for i < len(s) {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f':
			i++
			if !expectCompound {
				// Descendant, unless a '>' follows.
				j := i
				for j < len(s) && strings.IndexByte(" \t\n\r\f", s[j]) >= 0 {
					j++
				}
				if j < len(s) && s[j] != '>' {
					sel.Comb = append(sel.Comb, ' ')
					expectCompound = true
				}
				i = j
			}
		case c == '>':
			if expectCompound {
				return sel, fmt.Errorf("selector %q: misplaced '>'", s)
			}
			sel.Comb = append(sel.Comb, '>')
			expectCompound = true
			i++
		case c == '+' || c == '~' || c == '[' || c == ',':
			return sel, fmt.Errorf("selector %q: %q is not supported", s, string(c))
		default:
			if !expectCompound {
				return sel, fmt.Errorf("selector %q: unexpected %q", s, string(c))
			}
			comp, n, err := parseCompound(s[i:])
			if err != nil {
				return sel, fmt.Errorf("selector %q: %v", s, err)
			}
			sel.Parts = append(sel.Parts, comp)
			i += n
			expectCompound = false
		}
	}
	if expectCompound {
		return sel, fmt.Errorf("selector %q is incomplete", s)
	}
	return sel, nil
}

func isIdent(c byte) bool {
	return c == '-' || c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}

func parseCompound(s string) (Compound, int, error) {
	var c Compound
	i := 0
	ident := func() string {
		j := i
		for j < len(s) && isIdent(s[j]) {
			j++
		}
		id := s[i:j]
		i = j
		return id
	}
	switch {
	case s[0] == '*':
		i = 1
	case isIdent(s[0]):
		c.Tag = strings.ToLower(ident())
	}
	for i < len(s) {
		switch s[i] {
		case '#':
			i++
			if c.ID = ident(); c.ID == "" {
				return c, i, fmt.Errorf("empty id")
			}
		case '.':
			i++
			cl := ident()
			if cl == "" {
				return c, i, fmt.Errorf("empty class")
			}
			c.Classes = append(c.Classes, cl)
		case ':':
			i++
			if i < len(s) && s[i] == ':' {
				return c, i, fmt.Errorf("pseudo-elements are not supported")
			}
			switch p := strings.ToLower(ident()); p {
			case "hover":
				c.Pseudo |= StateHover
			case "focus":
				c.Pseudo |= StateFocus
			case "active":
				c.Pseudo |= StateActive
			default:
				return c, i, fmt.Errorf("pseudo-class :%s is not supported", p)
			}
		default:
			if i == 0 {
				return c, i, fmt.Errorf("unexpected %q", string(s[i]))
			}
			return c, i, nil
		}
	}
	return c, i, nil
}

// Matches reports whether sel matches el.
func (sel *Selector) Matches(el Element) bool {
	return sel.matchAt(len(sel.Parts)-1, el)
}

func (sel *Selector) matchAt(i int, el Element) bool {
	if !sel.Parts[i].matches(el) {
		return false
	}
	if i == 0 {
		return true
	}
	if sel.Comb[i-1] == '>' {
		p := el.ParentElement()
		return p != nil && sel.matchAt(i-1, p)
	}
	for p := el.ParentElement(); p != nil; p = p.ParentElement() {
		if sel.matchAt(i-1, p) {
			return true
		}
	}
	return false
}

func (c *Compound) matches(el Element) bool {
	if c.Tag != "" && c.Tag != el.Tag() {
		return false
	}
	if c.ID != "" && c.ID != el.ID() {
		return false
	}
	for _, cl := range c.Classes {
		if !el.HasClass(cl) {
			return false
		}
	}
	return el.State()&c.Pseudo == c.Pseudo
}

// Cascade is a set of sheets applied together.
type Cascade struct {
	rules []Rule
}

// Pseudo returns the union of the pseudo-classes used anywhere in the
// cascade's selectors, so callers can skip restyling for state changes
// that no rule can observe.
func (c *Cascade) Pseudo() State {
	var u State
	for i := range c.rules {
		for _, p := range c.rules[i].Sel.Parts {
			u |= p.Pseudo
		}
	}
	return u
}

// NewCascade combines sheets, in order.
func NewCascade(sheets ...*Sheet) *Cascade {
	var c Cascade
	for _, s := range sheets {
		c.rules = append(c.rules, s.Rules...)
	}
	return &c
}

// Compute returns the computed style of el, whose parent has computed
// style parent (nil for the root). inline is the element's style=""
// attribute. Order (spec/SUBSET.md §3): inherited/initial values, then
// matching rules by ascending specificity, ties broken by source order,
// then inline declarations. Warnings come from inline declarations only
// (sheet warnings are reported once by ParseSheet).
func (c *Cascade) Compute(el Element, parent *Style, inline string) (Style, []Warning) {
	s := Inherit(parent)
	var matched []*Rule
	for i := range c.rules {
		if c.rules[i].Sel.Matches(el) {
			matched = append(matched, &c.rules[i])
		}
	}
	sort.SliceStable(matched, func(a, b int) bool {
		if matched[a].Spec != matched[b].Spec {
			return matched[a].Spec < matched[b].Spec
		}
		return matched[a].Order < matched[b].Order
	})
	for _, r := range matched {
		for _, d := range r.Decls {
			s.Apply(d, Context{})
		}
	}
	var warns []Warning
	if inline != "" {
		decls, w := ParseDeclarations(inline)
		warns = append(warns, w...)
		warns = append(warns, s.ApplyAll(decls, Context{})...)
	}
	s.Finish()
	return s, warns
}
