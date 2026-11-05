//go:build windows

// Command stage9 checks IME composition (IMM32): marked text drawn in the
// field, commit, cancel, and the candidate window placed at the caret.
package main

import (
	"flag"
	"fmt"

	"github.com/sivaneshwararaj/flexui/platform/internal/stagekit"
	"github.com/sivaneshwararaj/flexui/platform/windows"
)

func main() {
	debug := flag.Bool("debug", false, "log every step to stderr")
	flag.Parse()
	windows.Debug = *debug
	fmt.Println("EXPECT:", stagekit.Expect9("Settings > Time & language > Language & region: add Japanese, then Win+Space to switch to Microsoft IME, Alt+` for hiragana"))
	logf := func(f string, a ...any) { fmt.Printf(f+"\n", a...) }
	app := stagekit.App9("flexui stage 9", logf)
	if err := windows.Run(stagekit.LogIME{App: app, Logf: logf}); err != nil {
		fmt.Println("RESULT: FAIL", err)
	}
}
