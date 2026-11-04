// Demo is the flexui acceptance app: a contacts manager with a searchable
// list, an edit form and a confirmation dialog, written against the
// public API only. Run it headless to write PNGs of a scripted session,
// or in a window on macOS:
//
//	go run ./demo            # writes demo-*.png
//	go run ./demo -window    # macOS or Windows window
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/sivaneshwararaj/flexui/platform/headless"
)

func main() {
	window := flag.Bool("window", false, "open a native window (macOS, Windows)")
	flag.Parse()
	d := NewDemo()
	if *window {
		if err := runWindow(d); err != nil {
			log.Fatal(err)
		}
		return
	}
	snapHook = func(step string, s *headless.Session) {
		path := fmt.Sprintf("demo-%s.png", step)
		if err := s.Save(path); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote", path)
	}
	Script(d, headless.Start(d.App))
	if w := d.App.Warnings(); len(w) > 0 {
		log.Printf("warnings: %q", w)
	}
}
