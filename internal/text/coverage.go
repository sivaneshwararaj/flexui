package text

import (
	"reflect"
	"unsafe"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype/tables"
)

// Workaround for a bug in github.com/go-text/typesetting v0.3.5:
// harfbuzz's setDigest.addRange returns early for all three of its
// filters as soon as any one of them is saturated, so range-based
// coverage tables (format 2) produce false negatives in the lookup
// digests and some lookups are silently skipped (with Inter, most class
// kerning, e.g. "To" and "fo" are not kerned). C HarfBuzz (and so
// Chromium) checks each filter separately. Glyph-list coverages (format
// 1) take a code path without the bug, so after parsing we rewrite every
// format 2 coverage of GSUB and GPOS into the equivalent format 1 list.
// Most coverage fields are unexported, so they are written through
// reflect.NewAt (this file is the only use of unsafe in the module).
// TestKerning guards this; remove it once upstream is fixed.

func fixCoverages(f *font.Font) {
	walk(reflect.ValueOf(&f.GSUB).Elem())
	walk(reflect.ValueOf(&f.GPOS).Elem())
}

var coverageType = reflect.TypeOf((*tables.Coverage)(nil)).Elem()

// walk rewrites coverages inside v, which must be addressable.
func walk(v reflect.Value) {
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return
		}
		e := v.Elem()
		if c2, ok := e.Interface().(tables.Coverage2); ok && v.Type() == coverageType {
			if c1, ok := toList(c2); ok {
				v.Set(reflect.ValueOf(c1))
			}
			return
		}
		if e.Kind() == reflect.Pointer {
			walk(e)
			return
		}
		// Copy the dynamic value to make it addressable, then store it back.
		cp := reflect.New(e.Type()).Elem()
		cp.Set(e)
		walk(cp)
		v.Set(cp)
	case reflect.Pointer:
		if !v.IsNil() {
			walk(v.Elem())
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if !f.CanSet() {
				f = reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
			}
			walk(f)
		}
	case reflect.Slice, reflect.Array:
		if k := v.Type().Elem().Kind(); k == reflect.Uint8 || k == reflect.Uint16 || k == reflect.Int16 || k == reflect.Uint32 || k == reflect.Int32 {
			return
		}
		for i := 0; i < v.Len(); i++ {
			walk(v.Index(i))
		}
	}
}

// toList converts a range coverage into a glyph list with the same
// coverage indices; ok is false for malformed tables, which are kept.
func toList(c tables.Coverage2) (tables.Coverage1, bool) {
	var out tables.Coverage1
	for _, r := range c.Ranges {
		if int(r.StartCoverageIndex) != len(out.Glyphs) || r.EndGlyphID < r.StartGlyphID ||
			len(out.Glyphs) > 0 && r.StartGlyphID <= out.Glyphs[len(out.Glyphs)-1] {
			return out, false
		}
		for g := int(r.StartGlyphID); g <= int(r.EndGlyphID); g++ {
			out.Glyphs = append(out.Glyphs, tables.GlyphID(g))
		}
	}
	return out, true
}
