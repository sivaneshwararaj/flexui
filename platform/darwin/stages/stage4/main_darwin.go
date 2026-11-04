//go:build darwin

// Command stage4 checks resizing and HiDPI: the page re-lays out to the
// window size and renders at the backing scale.
package main

import (
	"flag"
	"fmt"
	"image"

	"github.com/sivaneshwararaj/flexui"
	"github.com/sivaneshwararaj/flexui/platform/darwin"
)

const css = `body { background-color: #fff; padding: 12px; gap: 12px; font-size: 14px; }
.bar { background-color: #1e222a; color: #fff; padding: 6px 10px; border-radius: 6px; }
.grid { flex-direction: row; flex-wrap: wrap; gap: 10px; }
.grid div { width: 120px; height: 60px; background-color: #e8eefc; border: 1px solid #3478f6; border-radius: 6px; align-items: center; justify-content: center; }`

func main() {
	debug := flag.Bool("debug", false, "log every step to stderr")
	flag.Parse()
	darwin.Debug = *debug
	fmt.Println("EXPECT: a window \"flexui stage 4\" with a dark bar showing the current size and six boxes that wrap into rows. Resize the window: the boxes re-wrap and the size text updates. Move it to a display with a different scale: a RESULT line reports the new scale and text stays sharp.")
	size := "?"
	app := flexui.New(flexui.Options{Title: "flexui stage 4", Width: 460, Height: 320})
	app.SetViewFunc(func() string {
		return fmt.Sprintf(`<div class="bar">Window size: %s</div><div class="grid"><div>One</div><div>Two</div><div>Three</div><div>Four</div><div>Five</div><div>Six</div></div>`, size)
	}, css)
	var last float64
	var lastW, lastH int
	var w *darwin.Window
	w, err := darwin.Open(app.Config(), app)
	if err != nil {
		fmt.Println("RESULT: FAIL", err)
		return
	}
	w.OnFrame = func(img *image.RGBA, lw, lh int, scale float64) {
		if lw != lastW || lh != lastH || scale != last {
			lastW, lastH, last = lw, lh, scale
			size = fmt.Sprintf("%dx%d @%gx", lw, lh, scale)
			app.Invalidate()
			defer w.Redraw() // show the new size text
			fmt.Printf("RESULT: frame %dx%d points @%gx = %dx%d pixels\n", lw, lh, scale, img.Rect.Dx(), img.Rect.Dy())
		}
	}
	w.Loop()
}
