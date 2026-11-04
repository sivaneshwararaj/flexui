//go:build darwin

// Command stage1 checks that the frameworks load through purego and that
// the Objective-C runtime answers, without opening a window.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ebitengine/purego/objc"

	"github.com/sivaneshwararaj/flexui/platform/darwin"
)

func main() {
	debug := flag.Bool("debug", false, "log every step to stderr")
	flag.Parse()
	darwin.Debug = *debug
	fmt.Println("EXPECT: three RESULT lines (framework load, NSApplication class, macOS version), then exit status 0. No window.")
	if err := darwin.Load(); err != nil {
		fmt.Println("RESULT: FAIL load:", err)
		os.Exit(1)
	}
	fmt.Println("RESULT: frameworks loaded (AppKit, QuartzCore, CoreFoundation, CoreGraphics)")
	c := objc.GetClass("NSApplication")
	if c == 0 {
		fmt.Println("RESULT: FAIL NSApplication class not found")
		os.Exit(1)
	}
	fmt.Printf("RESULT: NSApplication class at %#x\n", c)
	fmt.Println("RESULT: macOS", darwin.OSVersion())
}
