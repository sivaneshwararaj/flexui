//go:build darwin

// Command stage5 checks pointer, wheel and keyboard input: hover,
// clicks, scrolling, Tab focus and Enter/Space activation.
package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/sivaneshwararaj/flexui"
	"github.com/sivaneshwararaj/flexui/platform/darwin"
)

const css = `body { background-color: #f4f5f7; padding: 12px; gap: 10px; font-size: 14px; }
.row { flex-direction: row; gap: 8px; align-items: center; }
.list { overflow: scroll; height: 160px; background-color: #fff; border: 1px solid #d0d4da; border-radius: 6px; }
.item { padding: 6px 10px; flex-shrink: 0; }
.item:hover { background-color: #e8eefc; }
.sel { background-color: #dbe6ff; font-weight: 700; }`

func main() {
	debug := flag.Bool("debug", false, "log every step to stderr")
	flag.Parse()
	darwin.Debug = *debug
	fmt.Println("EXPECT: a counter with - and + buttons and a list of 40 items. Hovering an item or button highlights it. Clicking + / - changes the count (a RESULT line per click). The list scrolls with the trackpad/wheel and by dragging its scrollbar thumb; clicking an item selects it. Tab moves a blue focus border between the buttons; Enter or Space activates the focused one.")
	count, sel := 0, ""
	app := flexui.New(flexui.Options{Title: "flexui stage 5", Width: 360, Height: 300})
	app.SetViewFunc(func() string {
		var b strings.Builder
		fmt.Fprintf(&b, `<div class="row"><button id="dec" onclick="Dec">−</button><div>Count: %d</div><button id="inc" onclick="Inc">+</button><div>Selected: %s</div></div><div class="list">`, count, sel)
		for i := 1; i <= 40; i++ {
			c := "item"
			if fmt.Sprint(i) == sel {
				c += " sel"
			}
			fmt.Fprintf(&b, `<div class="%s" onclick="Pick" data-i="%d">Item %d</div>`, c, i, i)
		}
		b.WriteString("</div>")
		return b.String()
	}, css)
	app.Handle("Inc", func(c *flexui.Ctx) { count++; fmt.Println("RESULT: Inc, count", count) })
	app.Handle("Dec", func(c *flexui.Ctx) { count--; fmt.Println("RESULT: Dec, count", count) })
	app.Handle("Pick", func(c *flexui.Ctx) { sel = c.Attr("data-i"); fmt.Println("RESULT: picked item", sel) })
	if err := darwin.Run(app); err != nil {
		fmt.Println("RESULT: FAIL", err)
	}
}
