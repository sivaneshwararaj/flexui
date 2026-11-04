//go:build windows

// Command stage2 opens a window and presents a test pattern drawn without
// flexui's renderer (orientation, channel order, DPI sizing).
package main

import (
	"flag"
	"fmt"

	"github.com/sivaneshwararaj/flexui/platform"
	"github.com/sivaneshwararaj/flexui/platform/internal/stagekit"
	"github.com/sivaneshwararaj/flexui/platform/windows"
)

func main() {
	debug := flag.Bool("debug", false, "log every step to stderr")
	flag.Parse()
	windows.Debug = *debug
	fmt.Println("EXPECT:", stagekit.Expect2)
	w, err := windows.Open(platform.Config{Title: "flexui stage 2", Width: 480, Height: 320}, stagekit.Pattern{})
	if err != nil {
		fmt.Println("RESULT: FAIL", err)
		return
	}
	lw, lh, s := w.Size()
	fmt.Printf("RESULT: window open, %dx%d logical pixels, scale %g\n", lw, lh, s)
	w.Loop()
	fmt.Println("RESULT: window closed, message loop ended")
}
