package render

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
)

// Randomized text documents for TestTextChrome. They are generated from
// fixed seeds; `make crosscheck` writes them out (TestEmitText) and records
// Chromium's layout in testdata/text/random.json.

var emitText = flag.String("emit-text", "", "write the random text documents to this directory")

const randomTextCount = 300

var words = strings.Fields(`the quick brown fox jumps over lazy dog Pack my box with five
dozen liquor jugs AVAWAY To Wy flexui layout engine renders text wrapping
inside flex items supercalifragilistic a-b-c and/or 12:45 3.14 Hello, world!
x y z WWW iii Typography kerning`)

func randomTextDoc(i int) (name, src string) {
	r := rand.New(rand.NewPCG(uint64(i), 0x7e47))
	var b strings.Builder
	var sentence = func() string {
		n := 1 + r.IntN(9)
		ws := make([]string, n)
		for k := range ws {
			ws[k] = words[r.IntN(len(words))]
		}
		return strings.Join(ws, " ")
	}
	var node func(depth int)
	node = func(depth int) {
		var css []string
		if r.IntN(2) == 0 {
			css = append(css, "flex-direction: row")
		}
		if r.IntN(4) == 0 {
			css = append(css, "flex-wrap: wrap")
		}
		if r.IntN(3) == 0 {
			css = append(css, fmt.Sprintf("width: %dpx", 30+r.IntN(250)))
		}
		if r.IntN(5) == 0 {
			css = append(css, fmt.Sprintf("flex-grow: %d", r.IntN(3)))
		}
		if r.IntN(4) == 0 {
			css = append(css, []string{"align-items: flex-start", "align-items: center", "align-items: flex-end"}[r.IntN(3)])
		}
		if r.IntN(3) == 0 {
			css = append(css, fmt.Sprintf("padding: %dpx", r.IntN(6)))
		}
		if r.IntN(4) == 0 {
			css = append(css, fmt.Sprintf("gap: %dpx", r.IntN(8)))
		}
		if r.IntN(4) == 0 {
			css = append(css, []string{"text-align: left", "text-align: center", "text-align: right"}[r.IntN(3)])
		}
		if r.IntN(5) == 0 {
			css = append(css, fmt.Sprintf("font-size: %dpx", 9+r.IntN(18)))
		}
		if r.IntN(6) == 0 {
			css = append(css, "font-weight: 700")
		}
		if r.IntN(6) == 0 {
			css = append(css, []string{"line-height: 1.2", "line-height: 22px", "line-height: 1"}[r.IntN(3)])
		}
		if r.IntN(7) == 0 {
			css = append(css, []string{"white-space: nowrap", "white-space: pre-wrap"}[r.IntN(2)])
		}
		if r.IntN(8) == 0 {
			css = append(css, "white-space: nowrap; overflow: hidden; text-overflow: ellipsis")
		}
		fmt.Fprintf(&b, `<div style="%s">`, strings.Join(css, "; "))
		kids := 1 + r.IntN(3)
		for k := 0; k < kids; k++ {
			if depth == 0 || r.IntN(3) == 0 {
				b.WriteString(sentence())
				// Adjacent text after an element child is a new run.
				if k+1 < kids {
					b.WriteString(" ")
				}
			} else {
				node(depth - 1)
			}
		}
		b.WriteString("</div>")
	}
	w, h := 320, 400
	fmt.Fprintf(&b, "<!-- size: %dx%d -->\n<html><head><style>body { font-size: 14px; padding: 2px; align-items: flex-start; }</style></head><body>", w, h)
	node(1 + r.IntN(3))
	b.WriteString("</body></html>\n")
	return fmt.Sprintf("random_%04d", i), b.String()
}

// writeRandomText writes the documents (called by TestEmitText).
func writeRandomText(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for i := 0; i < randomTextCount; i++ {
		name, src := randomTextDoc(i)
		if err := os.WriteFile(filepath.Join(dir, name+".html"), []byte(src), 0o644); err != nil {
			return err
		}
	}
	return nil
}
