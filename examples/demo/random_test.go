package main

import (
	"math/rand/v2"
	"testing"

	"github.com/sivaneshwararaj/flexui"
	"github.com/sivaneshwararaj/flexui/platform"
	"github.com/sivaneshwararaj/flexui/platform/headless"
)

// TestRandomInteractions drives the demo with random pointer moves, wheel
// scrolls, clicks, drags, typing and keys. With FLEXUI_CHECK_DAMAGE on
// (TestMain), every frame repainted only in damaged rectangles and every
// incremental restyle is compared with a full rebuild and repaint, and a
// difference panics. It covers the incremental paths on arbitrary state
// sequences rather than the scripted ones.
func TestRandomInteractions(t *testing.T) {
	for seed := uint64(1); seed <= 3; seed++ {
		r := rand.New(rand.NewPCG(seed, 42))
		d := newDemo(flexui.Options{Width: 760, Height: 480, Scale: []float64{1, 2, 1.5}[seed-1]}, 40)
		s := headless.Start(d.App)
		for i := 0; i < 300; i++ {
			x, y := r.Float64()*760, r.Float64()*480
			switch k := r.IntN(10); {
			case k < 4:
				s.Move(x, y)
			case k < 6:
				s.Wheel(x, y, 0, float64(r.IntN(200)-100))
			case k < 8:
				s.Click(x, y)
			case k == 8:
				s.Drag(x, y, x+float64(r.IntN(60)-30), y+float64(r.IntN(60)-30), 3)
			default:
				switch r.IntN(4) {
				case 0:
					s.Type("ab")
				case 1:
					s.Key("Tab", 0)
				case 2:
					s.Key("Backspace", 0)
				default:
					s.Key("Escape", 0)
				}
			}
			if r.IntN(15) == 0 {
				s.Send(platform.Event{Kind: platform.EventPointerLeave})
			}
		}
	}
}
