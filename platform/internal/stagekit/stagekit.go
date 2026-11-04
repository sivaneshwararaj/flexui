// Package stagekit holds the platform-neutral content of the staged
// verification commands of the window backends (platform/windows/stages;
// the macOS stages predate it and carry their own copy): test pattern,
// pages, apps and the expected results.
package stagekit

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"

	"github.com/sivaneshwararaj/flexui"
	"github.com/sivaneshwararaj/flexui/platform"
)

// Keys names the platform's modifier keys in expectation texts.
type Keys struct {
	Word, Line, Cmd string // e.g. "Ctrl", "Home/End", "Ctrl" on Windows
}

// Pattern draws four quadrants (red top-left, green top-right, blue
// bottom-left, white bottom-right) with a 1-device-pixel black border,
// without flexui's renderer.
type Pattern struct{}

// Frame implements platform.AppHandler.
func (Pattern) Frame(buf *image.RGBA, w, h int, scale float64) {
	pw, ph := buf.Rect.Dx(), buf.Rect.Dy()
	for y := 0; y < ph; y++ {
		for x := 0; x < pw; x++ {
			var c color.RGBA
			switch {
			case x == 0 || y == 0 || x == pw-1 || y == ph-1:
				c = color.RGBA{0, 0, 0, 255}
			case x < pw/2 && y < ph/2:
				c = color.RGBA{255, 0, 0, 255}
			case y < ph/2:
				c = color.RGBA{0, 255, 0, 255}
			case x < pw/2:
				c = color.RGBA{0, 0, 255, 255}
			default:
				c = color.RGBA{255, 255, 255, 255}
			}
			buf.SetRGBA(x, y, c)
		}
	}
}

// Event implements platform.AppHandler.
func (Pattern) Event(platform.Event) {}

// Expect2 is stage 2's expected result.
const Expect2 = "a window titled \"flexui stage 2\" (480x320 logical pixels) with four quadrants: RED top-left, GREEN top-right, BLUE bottom-left, WHITE bottom-right, and a thin black border. Resizing redraws the pattern to the new size. Closing the window ends the program."

// Page3 and CSS3 are stage 3's page.
const Page3 = `<div class="card"><div class="title">Stage 3</div>
<p>This page is rendered by flexui and presented by the window backend. Text should be sharp at your display's scale, with <strong>bold</strong> and <a>link</a> runs.</p>
<div class="row"><button>Cancel</button><button class="primary">OK</button></div></div>`

const CSS3 = `body { background-color: #f4f5f7; padding: 20px; font-size: 14px; color: #1e222a; }
.card { background-color: #fff; border: 1px solid #d0d4da; border-radius: 10px; padding: 16px; gap: 10px; box-shadow: 0 4px 12px rgba(0,0,0,0.12); }
.title { font-size: 20px; font-weight: 700; }
.row { flex-direction: row; gap: 8px; justify-content: flex-end; }
.primary { background-color: #3478f6; color: #fff; border-color: #2563eb; }`

// Expect3 is stage 3's expected result.
const Expect3 = "a window \"flexui stage 3\" showing a white card (title \"Stage 3\", a paragraph, Cancel and OK buttons) on a light gray background, sharp at your display scale. RESULT must say \"identical: true\". stage3-window.png and stage3-headless.png are written to the current directory."

// App3 returns stage 3's app.
func App3(title string) *flexui.App {
	a := flexui.New(flexui.Options{Title: title, Width: 420, Height: 260})
	a.SetView(Page3, CSS3)
	return a
}

// Check3 compares a presented frame with a headless render of the same
// size and scale, writes both PNGs and returns the RESULT line.
func Check3(img *image.RGBA, lw, lh int, scale float64) string {
	ref := flexui.New(flexui.Options{Width: lw, Height: lh})
	ref.SetView(Page3, CSS3)
	buf := image.NewRGBA(img.Rect)
	ref.Frame(buf, lw, lh, scale)
	save("stage3-window.png", img)
	save("stage3-headless.png", buf)
	return fmt.Sprintf("RESULT: presented %dx%d @%gx, sha256 %x, identical: %v", lw, lh, scale,
		sha256.Sum256(img.Pix), bytes.Equal(buf.Pix, img.Pix))
}

func save(path string, img *image.RGBA) {
	var b bytes.Buffer
	png.Encode(&b, img)
	os.WriteFile(path, b.Bytes(), 0o644)
}

// Expect4 is stage 4's expected result.
const Expect4 = "a window \"flexui stage 4\" with a dark bar showing the current size and six boxes that wrap into rows. Resize the window: the boxes re-wrap and the size text updates. Move it to a monitor with another scaling (or change the scaling in Settings > Display): a RESULT line reports the new scale and text stays sharp."

// App4 returns stage 4's app; *size is shown in its bar.
func App4(title string, size *string) *flexui.App {
	a := flexui.New(flexui.Options{Title: title, Width: 460, Height: 320})
	a.SetViewFunc(func() string {
		return `<div class="bar">Window size: ` + html.EscapeString(*size) + `</div><div class="grid"><div>One</div><div>Two</div><div>Three</div><div>Four</div><div>Five</div><div>Six</div></div>`
	}, `body { background-color: #fff; padding: 12px; gap: 12px; font-size: 14px; }
.bar { background-color: #1e222a; color: #fff; padding: 6px 10px; border-radius: 6px; }
.grid { flex-direction: row; flex-wrap: wrap; gap: 10px; }
.grid div { width: 120px; height: 60px; background-color: #e8eefc; border: 1px solid #3478f6; border-radius: 6px; align-items: center; justify-content: center; }`)
	return a
}

// Expect5 is stage 5's expected result.
const Expect5 = "a counter with - and + buttons and a list of 40 items. Hovering an item or button highlights it. Clicking + / - changes the count (a RESULT line per click). The list scrolls with the mouse wheel (and touchpad) and by dragging its scrollbar thumb; clicking an item selects it. Tab moves a blue focus border between the buttons; Enter or Space activates the focused one."

// App5 returns stage 5's app; results are printed with logf.
func App5(title string, logf func(string, ...any)) *flexui.App {
	count, sel := 0, ""
	a := flexui.New(flexui.Options{Title: title, Width: 360, Height: 300})
	a.SetViewFunc(func() string {
		var b strings.Builder
		fmt.Fprintf(&b, `<div class="row"><button id="dec" onclick="Dec">−</button><div>Count: %d</div><button id="inc" onclick="Inc">+</button><div>Selected: %s</div></div><div class="list">`, count, sel)
		for i := 1; i <= 40; i++ {
			c := "item"
			if fmt.Sprint(i) == sel {
				c += " sel"
			}
			fmt.Fprintf(&b, `<div class="%s" onclick="Pick" data-i="%d">Item %d</div>`, c, i, i)
		}
		b.WriteString("</div>")
		return b.String()
	}, `body { background-color: #f4f5f7; padding: 12px; gap: 10px; font-size: 14px; }
.row { flex-direction: row; gap: 8px; align-items: center; }
.list { overflow: scroll; height: 160px; background-color: #fff; border: 1px solid #d0d4da; border-radius: 6px; }
.item { padding: 6px 10px; flex-shrink: 0; }
.item:hover { background-color: #e8eefc; }
.sel { background-color: #dbe6ff; font-weight: 700; }`)
	a.Handle("Inc", func(*flexui.Ctx) { count++; logf("RESULT: Inc, count %d", count) })
	a.Handle("Dec", func(*flexui.Ctx) { count--; logf("RESULT: Dec, count %d", count) })
	a.Handle("Pick", func(c *flexui.Ctx) { sel = c.Attr("data-i"); logf("RESULT: picked item %s", sel) })
	return a
}

// Expect6 returns stage 6's expected result for the platform's keys.
func Expect6(k Keys) string {
	return "two text fields. Click one and type: characters appear, the caret moves, and \"Echo\" mirrors the first field. Arrow keys, " +
		k.Word + "+Arrow (word), " + k.Line + " (line start/end), Shift to select, Backspace, " + k.Cmd + "+A/C/X/V all work; " +
		k.Cmd + "+V pastes text copied in another app, and " + k.Cmd + "+C in a field can be pasted elsewhere. Characters outside the BMP (emoji) insert as one character. Leaving a field prints a RESULT change line. IME composition (e.g. Japanese input) is NOT expected to work yet."
}

// App6 returns stage 6's app.
func App6(title string, logf func(string, ...any)) *flexui.App {
	name := ""
	a := flexui.New(flexui.Options{Title: title, Width: 380, Height: 200})
	a.SetViewFunc(func() string {
		return fmt.Sprintf(`<div class="row">Name <input id="name" value="%s" placeholder="Type here" oninput="In" onchange="Ch"></div>
<div class="row">Other <input id="other" placeholder="Paste here" onchange="Ch"></div>
<div>Echo: %s</div>`, html.EscapeString(name), html.EscapeString(name))
	}, `body { background-color: #f4f5f7; padding: 16px; gap: 10px; font-size: 14px; }
.row { flex-direction: row; gap: 8px; align-items: center; }
input { width: 220px; }`)
	a.Handle("In", func(c *flexui.Ctx) { name = c.Value })
	a.Handle("Ch", func(c *flexui.Ctx) { logf("RESULT: change %s = %q", c.ID, c.Value) })
	return a
}

// Expect7 is stage 7's expected result.
const Expect7 = "the counter app in a window titled \"flexui stage 7\"; + and - change the count; after 3 clicks a red note appears. Closing the window ends the program normally (exit status 0). For the full demo (list, form, modal): cd examples && go run ./demo -window"

// App7 returns stage 7's app.
func App7(title string) *flexui.App {
	count := 0
	a := flexui.New(flexui.Options{Title: title, Width: 280, Height: 220})
	a.SetViewFunc(func() string {
		note := ""
		if count >= 3 {
			note = `<div style="color: #b42318; font-size: 12px">That is a lot of clicks.</div>`
		}
		return fmt.Sprintf(`<div class="card"><div class="label">Count</div><div class="value">%d</div>
<div class="row"><button onclick="Dec">−</button><button onclick="Inc">+</button></div>%s</div>`, count, note)
	}, `body { background-color: #f4f5f7; align-items: center; justify-content: center; font-size: 14px; }
.card { background-color: #fff; border: 1px solid #d0d4da; border-radius: 10px; padding: 16px 24px; gap: 8px; align-items: center; }
.label { color: #6b7280; } .value { font-size: 40px; font-weight: 700; } .row { flex-direction: row; gap: 8px; }`)
	a.Handle("Inc", func(*flexui.Ctx) { count++ })
	a.Handle("Dec", func(*flexui.Ctx) { count-- })
	return a
}
