//go:build windows

// Command stage1 checks that every Win32 function used resolves and
// prints the Windows version, without opening a window.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sivaneshwararaj/flexui/platform/windows"
)

func main() {
	debug := flag.Bool("debug", false, "log every step to stderr")
	flag.Parse()
	windows.Debug = *debug
	fmt.Println("EXPECT: two RESULT lines (all Win32 functions resolved, Windows version), then exit status 0. No window. Windows 10 1703 or later is needed.")
	if err := windows.Load(); err != nil {
		fmt.Println("RESULT: FAIL", err)
		os.Exit(1)
	}
	fmt.Println("RESULT: user32, gdi32 and kernel32 functions resolved")
	fmt.Println("RESULT:", windows.OSVersion())
}
