//go:build windows

// Command stage3 presents a static flexui page and checks that the
// presented frame is byte-identical to a headless render.
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
	fmt.Println("EXPECT:", stagekit.Expect3)
	app := stagekit.App3("flexui stage 3")
	w, err := windows.Open(app.Config(), app)
	if err != nil {
		fmt.Println("RESULT: FAIL", err)
		return
	}
	done := false
	check := func(img *image.RGBA, lw, lh int, scale float64) {
		if !done && img != nil {
			done = true
			fmt.Println(stagekit.Check3(img, lw, lh, scale))
		}
	}
	w.OnFrame = check
	check(w.LastFrame())
	w.Loop()
}
