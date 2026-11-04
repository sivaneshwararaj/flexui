//go:build windows

// Command stage4 checks resizing and DPI scaling.
package main

import (
	"flag"
	"fmt"
	"image"

	"github.com/sivaneshwararaj/flexui/platform/internal/stagekit"
	"github.com/sivaneshwararaj/flexui/platform/windows"
)

func main() {
	debug := flag.Bool("debug", false, "log every step to stderr")
	flag.Parse()
	windows.Debug = *debug
	fmt.Println("EXPECT:", stagekit.Expect4)
	size := "?"
	app := stagekit.App4("flexui stage 4", &size)
	var w *windows.Window
	var last float64
	var lastW, lastH int
	w, err := windows.Open(app.Config(), app)
	if err != nil {
		fmt.Println("RESULT: FAIL", err)
		return
	}
	w.OnFrame = func(img *image.RGBA, lw, lh int, scale float64) {
		if lw != lastW || lh != lastH || scale != last {
			lastW, lastH, last = lw, lh, scale
			size = fmt.Sprintf("%dx%d @%gx", lw, lh, scale)
			app.Invalidate()
			defer w.Redraw()
			fmt.Printf("RESULT: frame %dx%d logical @%gx = %dx%d pixels\n", lw, lh, scale, img.Rect.Dx(), img.Rect.Dy())
		}
	}
	w.Redraw()
	w.Loop()
}
