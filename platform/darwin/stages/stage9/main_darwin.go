//go:build darwin

// Command stage9 checks IME composition (NSTextInputClient): marked text drawn in the
// field, commit, cancel, and the candidate window placed at the caret.
package main

import (
	"flag"
	"fmt"

	_ "github.com/sivaneshwararaj/flexui/fonts/arabic"
	_ "github.com/sivaneshwararaj/flexui/fonts/cjk"
	_ "github.com/sivaneshwararaj/flexui/fonts/devanagari"
	_ "github.com/sivaneshwararaj/flexui/fonts/hebrew"
	_ "github.com/sivaneshwararaj/flexui/fonts/tamil"
	"github.com/sivaneshwararaj/flexui/platform"
	"github.com/sivaneshwararaj/flexui/platform/darwin"
	"github.com/sivaneshwararaj/flexui/platform/internal/stagekit"
)

func main() {
	debug := flag.Bool("debug", false, "log every step to stderr")
	auto := flag.Bool("auto", false, "no keyboard needed: send the input-method messages to the view and check the results, then keep the window open")
	flag.Parse()
	darwin.Debug = *debug
	fmt.Println("EXPECT:", stagekit.Expect9("System Settings > Keyboard > Text Input > Edit > + Japanese - Romaji; switch with Ctrl+Space or the menu bar input menu"))
	logf := func(f string, a ...any) { fmt.Printf(f+"\n", a...) }
	app := stagekit.App9("flexui stage 9", logf)
	if !*auto {
		if err := darwin.Run(stagekit.LogIME{App: app, Logf: logf}); err != nil {
			fmt.Println("RESULT: FAIL", err)
		}
		return
	}
	h := stagekit.LogIME{App: app, Logf: logf}
	w, err := darwin.Open(app.Config(), h)
	if err != nil {
		fmt.Println("RESULT: FAIL", err)
		return
	}
	app.SetClipboard(darwin.Clipboard{})
	// Focus the first field as a click would.
	x, y, fw, fh, _ := app.Bounds("ja")
	h.Event(platform.Event{Kind: platform.EventPointerDown, X: x + fw/2, Y: y + fh/2})
	h.Event(platform.Event{Kind: platform.EventPointerUp, X: x + fw/2, Y: y + fh/2})
	if w.SelfTestIME(func() string { return app.Value("ja") }, logf) {
		fmt.Println("RESULT: self-test passed")
	} else {
		fmt.Println("RESULT: self-test FAILED")
	}
	w.Redraw()
	w.Loop()
}
