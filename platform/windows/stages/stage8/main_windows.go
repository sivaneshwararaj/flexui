//go:build windows

// Command stage8 checks that multilingual text survives the system
// clipboard byte for byte (UTF-16 with surrogate pairs on Windows, UTF-8
// NSString on macOS) and shows what another app's copy contains.
package main

import (
	"flag"
	"fmt"

	_ "github.com/sivaneshwararaj/flexui/fonts/arabic"
	_ "github.com/sivaneshwararaj/flexui/fonts/cjk"
	_ "github.com/sivaneshwararaj/flexui/fonts/devanagari"
	_ "github.com/sivaneshwararaj/flexui/fonts/hebrew"
	_ "github.com/sivaneshwararaj/flexui/fonts/tamil"
	"github.com/sivaneshwararaj/flexui/platform/internal/stagekit"
	"github.com/sivaneshwararaj/flexui/platform/windows"
)

func main() {
	debug := flag.Bool("debug", false, "log every step to stderr")
	flag.Parse()
	windows.Debug = *debug
	fmt.Println("EXPECT:", stagekit.Expect8("Ctrl", "Notepad"))
	logf := func(f string, a ...any) { fmt.Printf(f+"\n", a...) }
	app := stagekit.App8("flexui stage 8", logf)
	w, err := windows.Open(app.Config(), app)
	if err != nil {
		fmt.Println("RESULT: FAIL", err)
		return
	}
	clip := w.Clipboard()
	stagekit.ClipRoundTrip(clip, logf)
	app.SetClipboard(clip)
	w.Loop()
}
