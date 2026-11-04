package flexui_test

import (
	"image"
	"testing"

	"github.com/sivaneshwararaj/flexui"
)

func TestAppBasics(t *testing.T) {
	app := flexui.New(flexui.Options{Title: "T", Width: 20, Height: 10})
	app.SetView(`<div></div>`, `body { background-color: #000 }`)
	app.Handle("Save", func(*flexui.Ctx) {})
	if c := app.Config(); c.Width != 20 || c.Height != 10 || c.Title != "T" {
		t.Fatalf("config %+v", c)
	}
	buf := image.NewRGBA(image.Rect(0, 0, 20, 10))
	app.Frame(buf, 20, 10, 1)
	if buf.Pix[0] != 0 || buf.Pix[3] != 255 {
		t.Fatalf("view not rendered: %v", buf.Pix[:4])
	}
	img := flexui.RenderHTML(`<div style="width:5px;height:5px;background-color:red"></div>`, ``, 10, 10)
	if p := img.RGBAAt(2, 2); p.R != 255 || p.G != 0 || img.RGBAAt(7, 7).G != 255 {
		t.Fatalf("RenderHTML pixels %v %v", p, img.RGBAAt(7, 7))
	}
	if w := flexui.CheckHTML(`<blink>x</blink>`, `p { float: left }`); len(w) != 3 {
		t.Fatalf("warnings %q", w)
	}
}
