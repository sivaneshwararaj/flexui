// Package windows runs a flexui app in a native Windows window using
// golang.org/x/sys/windows (no cgo): a Win32 window class whose window
// procedure translates messages into flexui events, frames drawn with
// SetDIBitsToDevice, per-monitor DPI awareness, and the Windows clipboard.
//
// It was written and compile-checked on Linux (CGO_ENABLED=0 GOOS=windows
// for amd64 and arm64) but NOT run. See README.md for what is untested and
// the staged commands that verify it step by step on a Windows machine.
//
// Usage (from the main goroutine):
//
//	app := flexui.New(flexui.Options{Title: "Hello", Width: 800, Height: 600})
//	app.SetView(viewHTML, styleCSS)
//	if err := windows.Run(app); err != nil { log.Fatal(err) }
package windows
