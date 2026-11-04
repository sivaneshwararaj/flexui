//go:build darwin

// Command stage6 checks text input and the system clipboard.
package main

import (
	"flag"
	"fmt"
	"html"

	"github.com/sivaneshwararaj/flexui"
	"github.com/sivaneshwararaj/flexui/platform/darwin"
)

const css = `body { background-color: #f4f5f7; padding: 16px; gap: 10px; font-size: 14px; }
.row { flex-direction: row; gap: 8px; align-items: center; }
input { width: 220px; }`

func main() {
	debug := flag.Bool("debug", false, "log every step to stderr")
	flag.Parse()
	darwin.Debug = *debug
	fmt.Println("EXPECT: two text fields. Click one and type: characters appear, the caret moves, and \"Echo\" mirrors the first field. Arrow keys, Option+Arrow (word), Cmd+Arrow (line), Shift to select, Backspace, Cmd+A/C/X/V all work; Cmd+V pastes text copied in another app, and Cmd+C in a field can be pasted elsewhere. Leaving a field prints a RESULT change line. Accented input via dead keys and IME composition is NOT expected to work yet.")
	name := ""
	app := flexui.New(flexui.Options{Title: "flexui stage 6", Width: 380, Height: 200})
	app.SetViewFunc(func() string {
		return fmt.Sprintf(`<div class="row">Name <input id="name" value="%s" placeholder="Type here" oninput="In" onchange="Ch"></div>
<div class="row">Other <input id="other" placeholder="Paste here" onchange="Ch"></div>
<div>Echo: %s</div>`, html.EscapeString(name), html.EscapeString(name))
	}, css)
	app.Handle("In", func(c *flexui.Ctx) { name = c.Value })
	app.Handle("Ch", func(c *flexui.Ctx) { fmt.Printf("RESULT: change %s = %q\n", c.ID, c.Value) })
	if err := darwin.Run(app); err != nil {
		fmt.Println("RESULT: FAIL", err)
	}
}
