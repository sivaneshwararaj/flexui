//go:build windows

// Command stage6 checks text input and the Windows clipboard.
package main

import (
	"flag"
	"fmt"

	"github.com/sivaneshwararaj/flexui"
	_ "github.com/sivaneshwararaj/flexui/fonts/arabic"
	_ "github.com/sivaneshwararaj/flexui/fonts/cjk"
	_ "github.com/sivaneshwararaj/flexui/fonts/devanagari"
	_ "github.com/sivaneshwararaj/flexui/fonts/hebrew"
	_ "github.com/sivaneshwararaj/flexui/fonts/tamil"
	"github.com/sivaneshwararaj/flexui/platform"
	"github.com/sivaneshwararaj/flexui/platform/internal/stagekit"
	"github.com/sivaneshwararaj/flexui/platform/windows"
)

// loggedApp gives the app a clipboard that prints a RESULT line for every
// copy and paste, so clipboard failures are visible without -debug.
type loggedApp struct{ *flexui.App }

func (a loggedApp) SetClipboard(c platform.Clipboard) { a.App.SetClipboard(loggedClip{c}) }

type loggedClip struct{ c platform.Clipboard }

func (l loggedClip) ReadText() (string, error) {
	s, err := l.c.ReadText()
	fmt.Printf("RESULT: clipboard read %q, error %v\n", s, err)
	return s, err
}

func (l loggedClip) WriteText(s string) error {
	err := l.c.WriteText(s)
	fmt.Printf("RESULT: clipboard write %q, error %v\n", s, err)
	return err
}

func main() {
	debug := flag.Bool("debug", false, "log every step to stderr")
	flag.Parse()
	windows.Debug = *debug
	fmt.Println("EXPECT:", stagekit.Expect6(stagekit.Keys{Word: "Ctrl", Line: "Home/End", Cmd: "Ctrl"}))
	fmt.Println("EXPECT: every Ctrl+C/X prints \"RESULT: clipboard write ..., error <nil>\" and every Ctrl+V \"RESULT: clipboard read ...\".")
	app := stagekit.App6("flexui stage 6", func(f string, a ...any) { fmt.Printf(f+"\n", a...) })
	if err := windows.Run(loggedApp{app}); err != nil {
		fmt.Println("RESULT: FAIL", err)
	}
}
