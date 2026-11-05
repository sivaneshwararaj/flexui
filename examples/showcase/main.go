// Showcase is a tour of what flexui can do, written against the public
// API only: an order form with multi-select, filters, quantities and
// live totals; a calculator with keyboard input; text in six languages
// (right to left included); and a gallery of layout and paint features.
//
//	go run ./showcase            # writes showcase-*.png (headless)
//	go run ./showcase -window    # native window (macOS, Windows)
package main

import (
	"flag"
	"fmt"
	"log"

	// Font packs for the Languages tab (each is optional).
	_ "github.com/sivaneshwararaj/flexui/fonts/arabic"
	_ "github.com/sivaneshwararaj/flexui/fonts/cjk"
	_ "github.com/sivaneshwararaj/flexui/fonts/devanagari"
	_ "github.com/sivaneshwararaj/flexui/fonts/hebrew"
	_ "github.com/sivaneshwararaj/flexui/fonts/tamil"
	"github.com/sivaneshwararaj/flexui/platform/headless"
)

func main() {
	window := flag.Bool("window", false, "open a native window (macOS, Windows)")
	flag.Parse()
	s := NewShowcase()
	if *window {
		if err := runWindow(s); err != nil {
			log.Fatal(err)
		}
		return
	}
	snapHook = func(step string, ses *headless.Session) {
		path := fmt.Sprintf("showcase-%s.png", step)
		if err := ses.Save(path); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote", path)
	}
	Script(s, headless.Start(s.App))
	if w := s.App.Warnings(); len(w) > 0 {
		log.Printf("warnings: %q", w)
	}
}
