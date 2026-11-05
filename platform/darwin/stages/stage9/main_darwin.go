//go:build darwin

// Command stage9 checks IME composition (NSTextInputClient): marked text drawn in the
// field, commit, cancel, and the candidate window placed at the caret.
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
	fmt.Println("EXPECT:", stagekit.Expect9("System Settings > Keyboard > Text Input > Edit > + Japanese - Romaji; switch with Ctrl+Space or the menu bar input menu"))
	logf := func(f string, a ...any) { fmt.Printf(f+"\n", a...) }
	app := stagekit.App9("flexui stage 9", logf)
	if err := darwin.Run(stagekit.LogIME{App: app, Logf: logf}); err != nil {
		fmt.Println("RESULT: FAIL", err)
	}
}
