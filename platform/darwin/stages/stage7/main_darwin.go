//go:build darwin

// Command stage7 runs a complete app through darwin.Run, the public
// entry point.
package main

import (
	"flag"
	"fmt"

	"github.com/sivaneshwararaj/flexui"
	"github.com/sivaneshwararaj/flexui/platform/darwin"
)

func main() {
	debug := flag.Bool("debug", false, "log every step to stderr")
	flag.Parse()
	darwin.Debug = *debug
	fmt.Println("EXPECT: the counter app in a window titled \"flexui stage 7\"; + and - change the count; after 3 clicks a red note appears. Closing the window exits. For the full demo (list, form, modal), run: cd examples && go run ./demo -window")
	count := 0
	app := flexui.New(flexui.Options{Title: "flexui stage 7", Width: 280, Height: 220})
	app.SetViewFunc(func() string {
		note := ""
		if count >= 3 {
			note = `<div style="color: #b42318; font-size: 12px">That is a lot of clicks.</div>`
		}
		return fmt.Sprintf(`<div class="card"><div class="label">Count</div><div class="value">%d</div>
<div class="row"><button onclick="Dec">−</button><button onclick="Inc">+</button></div>%s</div>`, count, note)
	}, `body { background-color: #f4f5f7; align-items: center; justify-content: center; font-size: 14px; }
.card { background-color: #fff; border: 1px solid #d0d4da; border-radius: 10px; padding: 16px 24px; gap: 8px; align-items: center; }
.label { color: #6b7280; } .value { font-size: 40px; font-weight: 700; } .row { flex-direction: row; gap: 8px; }`)
	app.Handle("Inc", func(*flexui.Ctx) { count++ })
	app.Handle("Dec", func(*flexui.Ctx) { count-- })
	if err := darwin.Run(app); err != nil {
		fmt.Println("RESULT: FAIL", err)
	}
}
