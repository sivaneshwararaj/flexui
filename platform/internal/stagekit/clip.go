package stagekit

import (
	"fmt"
	"html"
	"strings"

	"github.com/sivaneshwararaj/flexui"
	"github.com/sivaneshwararaj/flexui/internal/testtext"
	"github.com/sivaneshwararaj/flexui/platform"
)

// Expect8 is stage 8's expected result (cmd names the shortcut key).
func Expect8(cmd, editor string) string {
	return "one RESULT line per test string (Tamil, Hindi, Arabic, Hebrew, mixed, combining marks, emoji) saying \"byte-identical\", then \"RESULT: 13/13 byte-identical\". " +
		"Then a window: the clipboard now holds the Tamil sample \"" + testtext.Samples[0].Text + "\"; paste it into " + editor + " and it must read exactly that. " +
		"Copy any text in another app (e.g. Tamil from a web page) and press " + cmd + "+V in the field: a RESULT line prints its code points (U+...), which must match what you copied. " +
		"Text in the field is drawn with the bundled font packs (Tamil, Devanagari, Arabic, Hebrew, CJK); only emoji show as boxes."
}

// ClipRoundTrip writes every multilingual sample to the system
// clipboard, reads it back and compares the bytes. It returns the number
// of mismatches and leaves the first (Tamil) sample on the clipboard.
func ClipRoundTrip(c platform.Clipboard, logf func(string, ...any)) int {
	bad := 0
	ss := testtext.Samples
	for i := len(ss) - 1; i >= 0; i-- { // last write: the Tamil sample
		smp := ss[i]
		if err := c.WriteText(smp.Text); err != nil {
			logf("RESULT: %s write error: %v", smp.Name, err)
			bad++
			continue
		}
		got, err := c.ReadText()
		switch {
		case err != nil:
			logf("RESULT: %s read error: %v", smp.Name, err)
			bad++
		case got != smp.Text:
			logf("RESULT: %s MISMATCH\n  wrote %s\n  read  %s", smp.Name, CodePoints(smp.Text), CodePoints(got))
			bad++
		default:
			logf("RESULT: %s byte-identical (%d bytes, %d code points)", smp.Name, len(got), len([]rune(got)))
		}
	}
	logf("RESULT: %d/%d byte-identical", len(ss)-bad, len(ss))
	return bad
}

// CodePoints lists the code points of s ("U+0BA4 U+0BAE ...").
func CodePoints(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "U+%04X", r)
	}
	return b.String()
}

// App8 returns stage 8's app: a field whose pasted value is reported as
// code points (in the window, too, in ASCII so it reads without fonts).
func App8(title string, logf func(string, ...any)) *flexui.App {
	val := ""
	a := flexui.New(flexui.Options{Title: title, Width: 520, Height: 260})
	a.SetViewFunc(func() string {
		return fmt.Sprintf(`<div class="row">Paste <input id="f" placeholder="paste here" oninput="In"></div>
<div class="cp">%s</div>`, html.EscapeString(CodePoints(val)))
	}, `body { background-color: #f4f5f7; padding: 16px; gap: 10px; font-size: 14px; }
.row { flex-direction: row; gap: 8px; align-items: center; } input { width: 400px; }
.cp { font-size: 12px; color: #374151; }`)
	a.Handle("In", func(c *flexui.Ctx) {
		val = c.Value
		logf("RESULT: field holds %q = %s", val, CodePoints(val))
	})
	return a
}
