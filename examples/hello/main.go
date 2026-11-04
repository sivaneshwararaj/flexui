// Command hello is the smallest flexui program. It is written against the
// public API only, as an external project would be.
//
//	cd examples && go run ./hello out.png
package main

import (
	"fmt"
	"os"

	"github.com/sivaneshwararaj/flexui"
	"github.com/sivaneshwararaj/flexui/platform/headless"
)

const view = `<div class="root"><button onclick="Save">Save</button></div>`

const css = `.root { padding: 16px; }`

func main() {
	out := "hello.png"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	app := flexui.New(flexui.Options{Title: "Hello", Width: 320, Height: 200})
	app.SetView(view, css)
	app.Handle("Save", func(c *flexui.Ctx) { fmt.Println("saved") })
	if err := headless.Render(app, out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
