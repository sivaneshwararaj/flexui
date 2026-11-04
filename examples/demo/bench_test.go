package main

import (
	"testing"

	"github.com/sivaneshwararaj/flexui"
	"github.com/sivaneshwararaj/flexui/platform/headless"
)

// Frame-time benchmarks of the demo at 2560x1600 device pixels (1280x800
// points at scale 2, e.g. a 13" MacBook Air), with 214 contacts. Each
// iteration is one input event plus one frame, as a platform's loop does
// it; the frame includes copying into the platform's buffer.
//
//	cd examples && go test ./demo -run xxx -bench Frame -benchtime 2s

func benchDemo(b *testing.B) (*Demo, *headless.Session) {
	d := newDemo(flexui.Options{Width: 1280, Height: 800, Scale: 2}, 200)
	s := headless.Start(d.App)
	if f := s.Frame(); f.Rect.Dx() != 2560 || f.Rect.Dy() != 1600 {
		b.Fatalf("frame %v", f.Rect)
	}
	return d, s
}

// Idle: an event that changes nothing (the pointer moves within the same
// element).
func BenchmarkFrameIdle(b *testing.B) {
	_, s := benchDemo(b)
	s.Move(700, 700)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Move(700+float64(i%2), 700)
	}
}

// Hover change: the pointer moves between two list items (each has a
// :hover background).
func BenchmarkFrameHover(b *testing.B) {
	d, s := benchDemo(b)
	_, ly, _, _, _ := d.App.Bounds("list")
	lx, _ := center(d.App, "list")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Move(lx, ly+1+float64(i%2)*49+24)
	}
}

// Typing: one character into the notes field per frame; the oninput
// handler updates the state and the view re-renders.
func BenchmarkFrameTyping(b *testing.B) {
	d, s := benchDemo(b)
	s.Click(center(d.App, "notes"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Type("x")
	}
}

// Scrolling: the list scrolls by 30px per frame (down, then up).
func BenchmarkFrameScroll(b *testing.B) {
	d, s := benchDemo(b)
	lx, ly := center(d.App, "list")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dy := 30.0
		if i/40%2 == 1 {
			dy = -30
		}
		s.Wheel(lx, ly, 0, dy)
	}
}
