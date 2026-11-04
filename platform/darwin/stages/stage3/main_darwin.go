//go:build darwin

// Command stage3 presents a static flexui page and checks that the
// presented frame is byte-identical to the headless render of the same
// size and scale.
package main

import (
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/sivaneshwararaj/flexui"
	"github.com/sivaneshwararaj/flexui/platform/darwin"
)

const page = `<div class="card"><div class="title">Stage 3</div>
<p>This page is rendered by flexui and presented through a CALayer. Text should be sharp at your display's scale, with <strong>bold</strong> and <a>link</a> runs.</p>
<div class="row"><button>Cancel</button><button class="primary">OK</button></div></div>`

const css = `body { background-color: #f4f5f7; padding: 20px; font-size: 14px; color: #1e222a; }
.card { background-color: #fff; border: 1px solid #d0d4da; border-radius: 10px; padding: 16px; gap: 10px; box-shadow: 0 4px 12px rgba(0,0,0,0.12); }
.title { font-size: 20px; font-weight: 700; }
.row { flex-direction: row; gap: 8px; justify-content: flex-end; }
.primary { background-color: #3478f6; color: #fff; border-color: #2563eb; }`

func save(path string, img *image.RGBA) {
	var b bytes.Buffer
	png.Encode(&b, img)
	os.WriteFile(path, b.Bytes(), 0o644)
}

func main() {
	debug := flag.Bool("debug", false, "log every step to stderr")
	flag.Parse()
	darwin.Debug = *debug
	fmt.Println("EXPECT: a window \"flexui stage 3\" showing a white card (title \"Stage 3\", a paragraph, Cancel and OK buttons) on a light gray background, sharp on Retina. RESULT must say \"identical: true\". stage3-window.png and stage3-headless.png are written to the current directory.")
	app := flexui.New(flexui.Options{Title: "flexui stage 3", Width: 420, Height: 260})
	app.SetView(page, css)
	checked := false
	w, err := darwin.Open(app.Config(), app)
	if err != nil {
		fmt.Println("RESULT: FAIL", err)
		return
	}
	check := func(img *image.RGBA, lw, lh int, scale float64) {
		if checked {
			return
		}
		checked = true
		ref := flexui.New(flexui.Options{Width: lw, Height: lh})
		ref.SetView(page, css)
		buf := image.NewRGBA(img.Rect)
		ref.Frame(buf, lw, lh, scale)
		same := bytes.Equal(buf.Pix, img.Pix)
		save("stage3-window.png", img)
		save("stage3-headless.png", buf)
		fmt.Printf("RESULT: presented %dx%d @%gx, sha256 %x, identical: %v\n", lw, lh, scale, sha256.Sum256(img.Pix), same)
	}
	w.OnFrame = check
	check(w.LastFrame())
	w.Loop()
}
