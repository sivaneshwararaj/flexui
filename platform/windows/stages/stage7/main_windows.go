//go:build windows

// Command stage7 runs a complete app through windows.Run.
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
	fmt.Println("EXPECT:", stagekit.Expect7)
	if err := windows.Run(stagekit.App7("flexui stage 7")); err != nil {
		fmt.Println("RESULT: FAIL", err)
		return
	}
	fmt.Println("RESULT: window closed, windows.Run returned")
}
