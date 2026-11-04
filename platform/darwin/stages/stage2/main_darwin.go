//go:build darwin

// Command stage2 opens a window and presents a test pattern drawn without
// flexui's renderer, to check the window, the presentation path, pixel
// orientation and channel order.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"

	"github.com/sivaneshwararaj/flexui/platform"
	"github.com/sivaneshwararaj/flexui/platform/darwin"
)

// pattern draws four quadrants: red top-left, green top-right, blue
// bottom-left, white bottom-right, with a 1-device-pixel black border.
type pattern struct{}

func (pattern) Frame(buf *image.RGBA, w, h int, scale float64) {
	pw, ph := buf.Rect.Dx(), buf.Rect.Dy()
	for y := 0; y < ph; y++ {
		for x := 0; x < pw; x++ {
			var c color.RGBA
			switch {
			case x == 0 || y == 0 || x == pw-1 || y == ph-1:
				c = color.RGBA{0, 0, 0, 255}
			case x < pw/2 && y < ph/2:
				c = color.RGBA{255, 0, 0, 255}
			case y < ph/2:
				c = color.RGBA{0, 255, 0, 255}
			case x < pw/2:
				c = color.RGBA{0, 0, 255, 255}
			default:
				c = color.RGBA{255, 255, 255, 255}
			}
			buf.SetRGBA(x, y, c)
		}
	}
}

func (pattern) Event(e platform.Event) {}

func main() {
	debug := flag.Bool("debug", false, "log every step to stderr")
	flag.Parse()
	darwin.Debug = *debug
	fmt.Println("EXPECT: a window titled \"flexui stage 2\" (480x320 points) with four quadrants: RED top-left, GREEN top-right, BLUE bottom-left, WHITE bottom-right, and a thin black border. Resizing redraws the pattern to the new size. Closing the window exits.")
	w, err := darwin.Open(platform.Config{Title: "flexui stage 2", Width: 480, Height: 320}, pattern{})
	if err != nil {
		fmt.Println("RESULT: FAIL", err)
		return
	}
	lw, lh, s := w.Size()
	fmt.Printf("RESULT: window open, %dx%d points, backing scale %g\n", lw, lh, s)
	w.Loop()
}
