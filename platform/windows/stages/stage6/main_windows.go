//go:build windows

// Command stage6 checks text input and the Windows clipboard.
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
	fmt.Println("EXPECT:", stagekit.Expect6(stagekit.Keys{Word: "Ctrl", Line: "Home/End", Cmd: "Ctrl"}))
	app := stagekit.App6("flexui stage 6", func(f string, a ...any) { fmt.Printf(f+"\n", a...) })
	if err := windows.Run(app); err != nil {
		fmt.Println("RESULT: FAIL", err)
	}
}
