package flexui_test

import (
	"image"
	"testing"

	"github.com/sivaneshwararaj/flexui"
)

func TestAppBasics(t *testing.T) {
	app := flexui.New(flexui.Options{Title: "T", Width: 20, Height: 10})
	app.SetView(`<div></div>`, ``)
	app.Handle("Save", func(*flexui.Ctx) {})
	if c := app.Config(); c.Width != 20 || c.Height != 10 || c.Title != "T" {
		t.Fatalf("config %+v", c)
	}
	buf := image.NewRGBA(image.Rect(0, 0, 20, 10))
	app.Frame(buf, 20, 10, 1)
	if buf.Pix[0] != 255 || buf.Pix[3] != 255 {
		t.Fatalf("background not painted: %v", buf.Pix[:4])
	}
}
