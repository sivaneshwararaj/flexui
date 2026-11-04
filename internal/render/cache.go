package render

import (
	"github.com/sivaneshwararaj/flexui/internal/style"
	"github.com/sivaneshwararaj/flexui/internal/text"
)

// Cache keeps work between successive builds of a view: parsed style
// sheets and shaped text runs, keyed by their source and parameters.
// Entries not used by the latest build are dropped, so memory follows the
// current view. A Cache is not safe for concurrent use.
type Cache struct {
	sheets map[sheetKey]*sheetEntry
	texts  map[textKey]*textEntry
	gen    uint64
	Stats  Stats
}

// Stats counts the work done through a Cache (for tests and tuning).
type Stats struct {
	Styles      int // style passes (cascade + box tree)
	SheetParses int // style sheets parsed (cache misses)
	TextShapes  int // text runs prepared (cache misses)
	Layouts     int
	Paints      int
}

type sheetKey struct {
	src   string
	order int
}

type sheetEntry struct {
	sheet *style.Sheet
	warns []style.Warning
	gen   uint64
}

type textKey struct {
	raw string
	p   text.Params
}

type textEntry struct {
	t   *text.Text
	gen uint64
}

// NewCache returns an empty cache.
func NewCache() *Cache {
	return &Cache{sheets: map[sheetKey]*sheetEntry{}, texts: map[textKey]*textEntry{}}
}

func (c *Cache) sheet(src string, order int) (*style.Sheet, []style.Warning) {
	k := sheetKey{src, order}
	e, ok := c.sheets[k]
	if !ok {
		c.Stats.SheetParses++
		sh, w := style.ParseSheet(src, order)
		e = &sheetEntry{sheet: sh, warns: w}
		c.sheets[k] = e
	}
	e.gen = c.gen
	return e.sheet, e.warns
}

func (c *Cache) text(raw string, p text.Params) *text.Text {
	k := textKey{raw, p}
	e, ok := c.texts[k]
	if !ok {
		c.Stats.TextShapes++
		e = &textEntry{t: text.New(raw, p)}
		c.texts[k] = e
	}
	e.gen = c.gen
	return e.t
}

// sweep drops entries that the latest build did not use.
func (c *Cache) sweep() {
	for k, e := range c.sheets {
		if e.gen != c.gen {
			delete(c.sheets, k)
		}
	}
	for k, e := range c.texts {
		if e.gen != c.gen {
			delete(c.texts, k)
		}
	}
}
