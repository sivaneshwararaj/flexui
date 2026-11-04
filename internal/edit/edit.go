// Package edit is the text-editing model of single-line inputs
// (spec/SUBSET.md §8.6): a text buffer with a caret and a selection
// anchor, edited and moved in grapheme clusters and words. It knows
// nothing about rendering or keys.
package edit

import (
	"unicode"

	"github.com/go-text/typesetting/segmenter"
)

// Buffer is the text of an input with its caret and selection. The
// selection is the range between Anchor and Caret (empty when equal).
// Positions are rune indices, always on grapheme cluster boundaries.
type Buffer struct {
	text   []rune
	Caret  int
	Anchor int
	bounds []int // grapheme cluster boundaries, including 0 and len
}

// New returns a buffer holding s with the caret at the end.
func New(s string) *Buffer {
	b := &Buffer{}
	b.Set(s)
	return b
}

// Set replaces the text and puts the caret at the end, deselecting.
func (b *Buffer) Set(s string) {
	b.text = []rune(sanitize(s))
	b.reindex()
	b.Caret, b.Anchor = len(b.text), len(b.text)
}

// sanitize keeps a single line: newlines and other control characters
// become nothing (tabs become spaces), as in an HTML text input.
func sanitize(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r == '\t':
			out = append(out, ' ')
		case r == '\n' || r == '\r' || unicode.IsControl(r):
		default:
			out = append(out, r)
		}
	}
	return string(out)
}

func (b *Buffer) reindex() {
	b.bounds = b.bounds[:0]
	b.bounds = append(b.bounds, 0)
	if len(b.text) > 0 {
		var seg segmenter.Segmenter
		seg.Init(b.text)
		it := seg.GraphemeIterator()
		for it.Next() {
			g := it.Grapheme()
			b.bounds = append(b.bounds, g.Offset+len(g.Text))
		}
	}
}

// String returns the text.
func (b *Buffer) String() string { return string(b.text) }

// Len returns the length in runes.
func (b *Buffer) Len() int { return len(b.text) }

// Selection returns the selected range [s, e) (s == e: none).
func (b *Buffer) Selection() (s, e int) {
	return min(b.Caret, b.Anchor), max(b.Caret, b.Anchor)
}

// Selected returns the selected text.
func (b *Buffer) Selected() string {
	s, e := b.Selection()
	return string(b.text[s:e])
}

// SelectAll selects the whole text.
func (b *Buffer) SelectAll() { b.Anchor, b.Caret = 0, len(b.text) }

// Insert replaces the selection with s (sanitized) and puts the caret
// after it. It reports whether the text changed.
func (b *Buffer) Insert(s string) bool {
	ins := []rune(sanitize(s))
	st, e := b.Selection()
	if len(ins) == 0 && st == e {
		return false
	}
	t := make([]rune, 0, len(b.text)-(e-st)+len(ins))
	t = append(append(append(t, b.text[:st]...), ins...), b.text[e:]...)
	b.text = t
	b.reindex()
	b.Caret = b.snap(st + len(ins))
	b.Anchor = b.Caret
	return true
}

// snap moves a position to the nearest grapheme boundary at or after it.
func (b *Buffer) snap(p int) int {
	for _, x := range b.bounds {
		if x >= p {
			return x
		}
	}
	return len(b.text)
}

// Unit is how far a movement or deletion goes.
type Unit int

const (
	Char Unit = iota // one grapheme cluster
	Word             // to the next word boundary (see NextWord, PrevWord)
	Line             // to the start or end of the text
)

// prevChar and nextChar return the neighboring grapheme boundaries.
func (b *Buffer) prevChar(p int) int {
	for i := len(b.bounds) - 1; i >= 0; i-- {
		if b.bounds[i] < p {
			return b.bounds[i]
		}
	}
	return 0
}

func (b *Buffer) nextChar(p int) int {
	for _, x := range b.bounds {
		if x > p {
			return x
		}
	}
	return len(b.text)
}

func isWord(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '\'' }

// PrevWord returns the start of the word before p: skip non-word
// characters backwards, then word characters (as browsers on all
// platforms do for Ctrl/Alt+Left).
func (b *Buffer) PrevWord(p int) int {
	for p > 0 && !isWord(b.text[p-1]) {
		p = b.prevChar(p)
	}
	for p > 0 && isWord(b.text[p-1]) {
		p = b.prevChar(p)
	}
	return p
}

// NextWord returns the end of the word after p: skip non-word characters,
// then word characters (macOS style; also used on other platforms).
func (b *Buffer) NextWord(p int) int {
	n := len(b.text)
	for p < n && !isWord(b.text[p]) {
		p = b.nextChar(p)
	}
	for p < n && isWord(b.text[p]) {
		p = b.nextChar(p)
	}
	return p
}

func (b *Buffer) target(from int, forward bool, u Unit) int {
	switch u {
	case Word:
		if forward {
			return b.NextWord(from)
		}
		return b.PrevWord(from)
	case Line:
		if forward {
			return len(b.text)
		}
		return 0
	}
	if forward {
		return b.nextChar(from)
	}
	return b.prevChar(from)
}

// Move moves the caret by one unit forward or backward. With extend the
// selection grows from the anchor; without, a selection collapses to its
// start or end (by a character move) or the caret moves from its end of
// the selection.
func (b *Buffer) Move(forward bool, u Unit, extend bool) {
	if extend {
		b.Caret = b.target(b.Caret, forward, u)
		return
	}
	s, e := b.Selection()
	if s != e && u == Char {
		if forward {
			b.Caret = e
		} else {
			b.Caret = s
		}
	} else {
		from := b.Caret
		if s != e {
			from = s
			if forward {
				from = e
			}
		}
		b.Caret = b.target(from, forward, u)
	}
	b.Anchor = b.Caret
}

// Delete deletes the selection, or else from the caret one unit backward
// (Backspace) or forward (Delete). It reports whether the text changed.
func (b *Buffer) Delete(forward bool, u Unit) bool {
	s, e := b.Selection()
	if s == e {
		t := b.target(b.Caret, forward, u)
		s, e = min(b.Caret, t), max(b.Caret, t)
		if s == e {
			return false
		}
	}
	b.text = append(b.text[:s:s], b.text[e:]...)
	b.reindex()
	b.Caret, b.Anchor = s, s
	return true
}

// SetCaret puts the caret at p (snapped to a grapheme boundary); with
// extend the anchor stays (pointer drag selection).
func (b *Buffer) SetCaret(p int, extend bool) {
	b.Caret = b.snap(max(0, min(p, len(b.text))))
	if !extend {
		b.Anchor = b.Caret
	}
}

// Boundaries returns the grapheme cluster boundaries (for hit testing).
func (b *Buffer) Boundaries() []int { return b.bounds }
