// Counter is a minimal stateful flexui app: two buttons change a number,
// and the view is re-rendered from state after every click. Run it
// headless to write PNGs, or in a window on macOS:
//
//	go run ./counter
//	go run ./counter -window
package main

import (
	"bytes"
	"flag"
	"html/template"
	"log"

	"github.com/sivaneshwararaj/flexui"
	"github.com/sivaneshwararaj/flexui/platform/headless"
)

// View and CSS are exported for the tests in the main module.

// CounterView is the view template.
const CounterView = `<div class="card">
  <div class="label">Count</div>
  <div id="value" class="value">{{.Count}}</div>
  <div class="row">
    <button id="dec" onclick="Dec">−</button>
    <button id="inc" onclick="Inc">+</button>
  </div>
  {{if ge .Count 3}}<div id="note" class="note">That is a lot of clicks.</div>{{end}}
</div>`

// CounterCSS styles it.
const CounterCSS = `
body { background-color: #f4f5f7; color: #1e222a; font-size: 14px; align-items: center; justify-content: center; }
.card { background-color: #fff; border: 1px solid #d0d4da; border-radius: 10px; padding: 16px 24px; gap: 8px; align-items: center; min-width: 160px; }
.label { color: #6b7280; font-size: 13px; }
.value { font-size: 40px; font-weight: 700; }
.row { flex-direction: row; gap: 8px; }
button { width: 44px; height: 36px; border-radius: 8px; background-color: #3478f6; color: #fff; font-size: 20px; font-weight: 700; align-items: center; justify-content: center; }
.note { color: #b42318; font-size: 12px; }
`

type state struct{ Count int }

func main() {
	st := &state{}
	tmpl := template.Must(template.New("v").Parse(CounterView))
	app := flexui.New(flexui.Options{Title: "Counter", Width: 280, Height: 220})
	app.SetViewFunc(func() string {
		var b bytes.Buffer
		if err := tmpl.Execute(&b, st); err != nil {
			log.Fatal(err)
		}
		return b.String()
	}, CounterCSS)
	app.Handle("Inc", func(*flexui.Ctx) { st.Count++ })
	app.Handle("Dec", func(*flexui.Ctx) { st.Count-- })

	window := flag.Bool("window", false, "open a native window (macOS)")
	flag.Parse()
	if *window {
		if err := runWindow(app); err != nil {
			log.Fatal(err)
		}
		return
	}

	s := headless.Start(app)
	if err := s.Save("counter-0.png"); err != nil {
		log.Fatal(err)
	}
	x, y, w, h, _ := app.Bounds("inc")
	for i := 0; i < 3; i++ {
		s.Click(x+w/2, y+h/2)
	}
	if err := s.Save("counter-3.png"); err != nil {
		log.Fatal(err)
	}
	log.Printf("count = %d; wrote counter-0.png and counter-3.png", st.Count)
}
