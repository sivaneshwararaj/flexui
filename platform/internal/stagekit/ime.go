package stagekit

import (
	"github.com/sivaneshwararaj/flexui"
	"github.com/sivaneshwararaj/flexui/platform"
)

// Expect9 is stage 9's expected result (ime names how to switch to a
// Japanese input method on the platform).
func Expect9(ime string) string {
	return "a window with two fields. Switch to a Japanese input method (" + ime + "), click the first field and type \"kanji\": " +
		"the marked text appears inside the field, underlined, in Japanese glyphs (CJK font pack), " +
		"with RESULT lines \"compose ...\" and \"input ... composing=true\"; the candidate window opens next to the caret (not in a screen corner) after Space; " +
		"Enter commits: RESULT \"commit ...\" and the underline disappears. Escape during composition cancels: RESULT \"cancel\" and the marked text goes away. " +
		"Clicking the other field during composition commits the marked text as it is. The second field is right to left: marked text appears at its right side."
}

// LogIME wraps an app so that composition events print RESULT lines.
type LogIME struct {
	*flexui.App
	Logf func(string, ...any)
}

// Event implements platform.AppHandler.
func (l LogIME) Event(e platform.Event) {
	switch e.Kind {
	case platform.EventCompose:
		l.Logf("RESULT: compose %q caret %d (%s)", e.Text, e.SelStart, CodePoints(e.Text))
	case platform.EventComposeCommit:
		l.Logf("RESULT: commit %q", e.Text)
	case platform.EventComposeCancel:
		l.Logf("RESULT: cancel")
	}
	l.App.Event(e)
	if e.Kind == platform.EventCompose {
		if x, y, w, h, ok := l.App.CaretRect(); ok {
			l.Logf("RESULT: caret rect %.1f,%.1f %.1fx%.1f (logical px)", x, y, w, h)
		}
	}
}

// App9 returns stage 9's app.
func App9(title string, logf func(string, ...any)) *flexui.App {
	a := flexui.New(flexui.Options{Title: title, Width: 420, Height: 160})
	a.SetView(`<div class="row">Name <input id="ja" placeholder="type with an IME" oninput="In"></div>
<div class="row">RTL <input id="rtl" dir="rtl" placeholder="right to left" oninput="In"></div>`,
		`body { background-color: #f4f5f7; padding: 16px; gap: 10px; font-size: 16px; }
.row { flex-direction: row; gap: 8px; align-items: center; } input { width: 300px; }`)
	a.Handle("In", func(c *flexui.Ctx) {
		logf("RESULT: input %s = %q composing=%v", c.ID, c.Value, c.Composing)
	})
	return a
}
