// Package darwin runs a flexui app in a native macOS window, using
// purego (no cgo): AppKit for the window and events, a layer-backed view
// whose contents are the app's rendered frames, the general pasteboard as
// the clipboard.
//
// It was written and compile-checked on Linux (CGO_ENABLED=0 GOOS=darwin
// for amd64 and arm64) but NOT run there. See README.md for what is
// untested and the staged commands that verify it step by step on a Mac.
//
// Usage (from the main goroutine):
//
//	app := flexui.New(flexui.Options{Title: "Hello", Width: 800, Height: 600})
//	app.SetView(viewHTML, styleCSS)
//	if err := darwin.Run(app); err != nil { log.Fatal(err) }
package darwin
