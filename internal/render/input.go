package render

import (
	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/text"
)

// inputMeasure sizes an <input> (spec/SUBSET.md §1.3): its content is one
// line of text in its font, and its intrinsic width is 160px (the width
// of the default style sheet), so a flex row does not shrink it below
// that (as in Chromium).
type inputMeasure struct{ h geom.Unit }

const inputWidth = 160 * geom.UnitsPerPx

func (m inputMeasure) Widths() (geom.Unit, geom.Unit) { return inputWidth, inputWidth }
func (m inputMeasure) Height(geom.Unit) geom.Unit     { return m.h }

func (d *Document) input(b *Box) {
	b.Lay.Measure = inputMeasure{text.LineHeightOf(textStyle(&b.Style))}
}
