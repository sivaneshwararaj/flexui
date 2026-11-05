//go:build darwin

// Command stage8 checks that multilingual text survives the system
// clipboard byte for byte (UTF-16 with surrogate pairs on Windows, UTF-8
// NSString on macOS) and shows what another app's copy contains.
package main

import (
	"flag"
	"fmt"

	"github.com/sivaneshwararaj/flexui/platform/darwin"
	"github.com/sivaneshwararaj/flexui/platform/internal/stagekit"
)

func main() {
	debug := flag.Bool("debug", false, "log every step to stderr")
	flag.Parse()
	darwin.Debug = *debug
	fmt.Println("EXPECT:", stagekit.Expect8("Cmd", "TextEdit"))
	logf := func(f string, a ...any) { fmt.Printf(f+"\n", a...) }
	app := stagekit.App8("flexui stage 8", logf)
	w, err := darwin.Open(app.Config(), app)
	if err != nil {
		fmt.Println("RESULT: FAIL", err)
		return
	}
	clip := darwin.Clipboard{}
	stagekit.ClipRoundTrip(clip, logf)
	app.SetClipboard(clip)
	w.Loop()
}
