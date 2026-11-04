//go:build windows

// Command stage5 checks pointer, wheel and keyboard input.
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
	fmt.Println("EXPECT:", stagekit.Expect5)
	app := stagekit.App5("flexui stage 5", func(f string, a ...any) { fmt.Printf(f+"\n", a...) })
	if err := windows.Run(app); err != nil {
		fmt.Println("RESULT: FAIL", err)
	}
}
