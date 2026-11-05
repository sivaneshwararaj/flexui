package render

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	_ "image/png" // registers the PNG decoder
	"io/fs"
	"net/url"
	"strings"

	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/jpeg"
	"github.com/sivaneshwararaj/flexui/internal/paint"
)

// imageMeasure sizes an <img>: its natural size is its pixel size (one
// image pixel per logical pixel), and it keeps its aspect ratio.
type imageMeasure struct{ w, h geom.Unit }

func (m imageMeasure) Widths() (geom.Unit, geom.Unit) { return m.w, m.w }
func (m imageMeasure) Height(w geom.Unit) geom.Unit {
	if m.w == 0 {
		return m.h
	}
	return geom.Unit(geom.MulDiv(int64(w), int64(m.h), int64(m.w)))
}
func (m imageMeasure) Width(h geom.Unit) geom.Unit {
	if m.h == 0 {
		return m.w
	}
	return geom.Unit(geom.MulDiv(int64(h), int64(m.w), int64(m.h)))
}

type imageEntry struct {
	img *image.RGBA // premultiplied
	err error
	gen uint64
}

// image loads the src of an <img> box (spec/SUBSET.md §1.2): a data: URL
// or a path in the cache's Assets file system; PNG and JPEG.
func (d *Document) image(b *Box) {
	src := strings.TrimSpace(b.El.Attr("src"))
	if src == "" {
		d.warn("<img> without src")
		return
	}
	img, err := d.cache.loadImage(src)
	if err != nil {
		d.warn("<img src=%q>: %v", short(src), err)
		return
	}
	b.Image = img
	b.Lay.Measure = imageMeasure{geom.Px(img.Rect.Dx()), geom.Px(img.Rect.Dy())}
}

func short(s string) string {
	if len(s) > 40 {
		return s[:40] + "…"
	}
	return s
}

func (c *Cache) loadImage(src string) (*image.RGBA, error) {
	if e, ok := c.images[src]; ok {
		e.gen = c.gen
		return e.img, e.err
	}
	data, err := c.readImage(src)
	var img *image.RGBA
	if err == nil {
		if bytes.HasPrefix(data, []byte("\xff\xd8")) {
			// Our copy of Go 1.24's decoder: the standard library's
			// output changes between Go versions (internal/jpeg).
			img, err = jpeg.DecodeRGBA(bytes.NewReader(data))
		} else if m, _, derr := image.Decode(bytes.NewReader(data)); derr != nil {
			err = derr
		} else {
			img = image.NewRGBA(image.Rect(0, 0, m.Bounds().Dx(), m.Bounds().Dy()))
			draw.Draw(img, img.Rect, m, m.Bounds().Min, draw.Src)
		}
	}
	c.images[src] = &imageEntry{img, err, c.gen}
	return img, err
}

func (c *Cache) readImage(src string) ([]byte, error) {
	if strings.HasPrefix(src, "data:") {
		comma := strings.IndexByte(src, ',')
		if comma < 0 {
			return nil, fmt.Errorf("malformed data URL")
		}
		meta, payload := src[5:comma], src[comma+1:]
		if strings.HasSuffix(meta, ";base64") {
			return base64.StdEncoding.DecodeString(payload)
		}
		s, err := url.PathUnescape(payload)
		return []byte(s), err
	}
	if c.Assets == nil {
		return nil, fmt.Errorf("no asset file system (Options.Assets) to load it from")
	}
	if strings.Contains(src, "://") {
		return nil, fmt.Errorf("only data: URLs and asset paths are supported")
	}
	return fs.ReadFile(c.Assets, strings.TrimPrefix(src, "/"))
}

// paintImage draws an <img>'s content into its content box, clipped to
// the rounded padding box.
func (b *Box) paintImage(c paint.Canvas) {
	bx := b.Lay.Box
	r := geom.Rect{
		X: b.Abs.X + bx.Border.Left + bx.Padding.Left,
		Y: b.Abs.Y + bx.Border.Top + bx.Padding.Top,
		W: b.Abs.W - bx.Border.Left - bx.Border.Right - bx.Padding.Left - bx.Padding.Right,
		H: b.Abs.H - bx.Border.Top - bx.Border.Bottom - bx.Padding.Top - bx.Padding.Bottom,
	}
	rounded := b.rrect().Radii != [4]paint.Radius{}
	if rounded {
		c.PushClip(b.clipRRect())
	}
	c.DrawImage(b.Image, r)
	if rounded {
		c.PopClip()
	}
}
