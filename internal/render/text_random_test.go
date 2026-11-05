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

var (
	emitText   = flag.String("emit-text", "", "write the random text documents to this directory")
	emitInline = flag.String("emit-inline", "", "write the random inline documents to this directory")
	emitScript = flag.String("emit-script", "", "write the random multi-script documents to this directory")
	emitBidi   = flag.String("emit-bidi", "", "write the random bidi documents to this directory")
)

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

// writeRandomInline writes the inline documents (TestEmitText with
// -emit-inline).
func writeRandomScript(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for i := 0; i < randomScriptCount; i++ {
		name, src := randomScriptDoc(i)
		if err := os.WriteFile(filepath.Join(dir, name+".html"), []byte(src), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func writeRandomInline(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for i := 0; i < randomInlineCount; i++ {
		name, src := randomInlineDoc(i)
		if err := os.WriteFile(filepath.Join(dir, name+".html"), []byte(src), 0o644); err != nil {
			return err
		}
	}
	return nil
}

const randomInlineCount = 150

// randomInlineDoc is a random document whose paragraphs mix text with
// inline elements (spans with font size, weight, line height and color,
// strong, b, a, br), nested up to two levels.
func randomInlineDoc(i int) (name, src string) {
	r := rand.New(rand.NewPCG(uint64(i), 0x1a11e))
	var b strings.Builder
	var inline func(depth int)
	inline = func(depth int) {
		n := 1 + r.IntN(4)
		for k := 0; k < n; k++ {
			ws := make([]string, 1+r.IntN(5))
			for j := range ws {
				ws[j] = words[r.IntN(len(words))]
			}
			sp := []string{" ", "", " "}[r.IntN(3)]
			b.WriteString(strings.Join(ws, " ") + sp)
			switch x := r.IntN(10); {
			case depth > 0 && x < 4:
				var css []string
				if r.IntN(2) == 0 {
					css = append(css, fmt.Sprintf("font-size: %dpx", 9+r.IntN(16)))
				}
				if r.IntN(3) == 0 {
					css = append(css, "font-weight: 700")
				}
				if r.IntN(4) == 0 {
					css = append(css, []string{"line-height: 1.5", "line-height: 24px", "line-height: 1"}[r.IntN(3)])
				}
				tag := []string{"span", "span", "strong", "b", "a"}[r.IntN(5)]
				fmt.Fprintf(&b, `<%s style="%s">`, tag, strings.Join(css, "; "))
				inline(depth - 1)
				fmt.Fprintf(&b, "</%s>", tag)
				b.WriteString([]string{" ", "", " "}[r.IntN(3)])
			case x == 4:
				b.WriteString("<br>")
			}
		}
	}
	fmt.Fprintf(&b, "<!-- size: 340x420 -->\n<html><head><style>body { font-size: 14px; padding: 2px; gap: 4px; align-items: flex-start; }</style></head><body>")
	for p := 1 + r.IntN(3); p > 0; p-- {
		var css []string
		if r.IntN(2) == 0 {
			css = append(css, fmt.Sprintf("width: %dpx", 40+r.IntN(260)))
		}
		if r.IntN(3) == 0 {
			css = append(css, []string{"text-align: center", "text-align: right"}[r.IntN(2)])
		}
		if r.IntN(4) == 0 {
			css = append(css, []string{"line-height: 1.3", "line-height: 20px"}[r.IntN(2)])
		}
		if r.IntN(5) == 0 {
			css = append(css, fmt.Sprintf("font-size: %dpx", 10+r.IntN(10)))
		}
		if r.IntN(6) == 0 {
			css = append(css, "flex-direction: row")
		}
		fmt.Fprintf(&b, `<div style="%s">`, strings.Join(css, "; "))
		inline(2)
		b.WriteString("</div>")
	}
	b.WriteString("</body></html>\n")
	return fmt.Sprintf("inline_%04d", i), b.String()
}

const randomScriptCount = 150

var scriptWords = []string{
	"Hello", "world", "text", "(2024)", "flexui", "ok,", "the", "[note]",
	"தமிழ்", "வணக்கம்", "நன்றி", "ஸ்ரீ", "க்ஷேத்திரம்", "கௌரவம்", "மொழி", "(தமிழ்)", "\"இது\"", "பேசப்படுகிறது.",
	"नमस्ते", "दुनिया", "क्षत्रिय", "स्त्री", "श्री", "लक्ष्मी", "झरना", "भाषा।", "(हिन्दी)", "प्रश्न,",
	"123", "—", "«x»",
}

// randomScriptDoc is randomInlineDoc with words in Latin, Tamil and
// Devanagari (with brackets, quotes, digits and lang attributes), for the
// script itemization and font fallback (spec/SUBSET.md §7.3).
func randomScriptDoc(i int) (name, src string) {
	r := rand.New(rand.NewPCG(uint64(i), 0x5c41b7))
	var b strings.Builder
	var inline func(depth int)
	inline = func(depth int) {
		n := 1 + r.IntN(4)
		for k := 0; k < n; k++ {
			ws := make([]string, 1+r.IntN(5))
			for j := range ws {
				ws[j] = scriptWords[r.IntN(len(scriptWords))]
			}
			sp := []string{" ", "", " "}[r.IntN(3)]
			b.WriteString(strings.Join(ws, " ") + sp)
			switch x := r.IntN(10); {
			case depth > 0 && x < 4:
				var css []string
				if r.IntN(2) == 0 {
					css = append(css, fmt.Sprintf("font-size: %dpx", 9+r.IntN(16)))
				}
				if r.IntN(3) == 0 {
					css = append(css, "font-weight: 700")
				}
				if r.IntN(4) == 0 {
					css = append(css, []string{"line-height: 1.5", "line-height: 24px", "line-height: 1"}[r.IntN(3)])
				}
				tag := []string{"span", "span", "strong", "b", "a"}[r.IntN(5)]
				lang := ""
				if r.IntN(4) == 0 {
					lang = fmt.Sprintf(` lang="%s"`, []string{"hi", "mr", "ne", "ta", "en"}[r.IntN(5)])
				}
				fmt.Fprintf(&b, `<%s style="%s"%s>`, tag, strings.Join(css, "; "), lang)
				inline(depth - 1)
				fmt.Fprintf(&b, "</%s>", tag)
				b.WriteString([]string{" ", "", " "}[r.IntN(3)])
			case x == 4:
				b.WriteString("<br>")
			}
		}
	}
	fmt.Fprintf(&b, "<!-- size: 340x420 -->\n<html><head><style>body { font-size: 14px; padding: 2px; gap: 4px; align-items: flex-start; }</style></head><body>")
	for p := 1 + r.IntN(3); p > 0; p-- {
		var css []string
		if r.IntN(2) == 0 {
			css = append(css, fmt.Sprintf("width: %dpx", 40+r.IntN(260)))
		}
		if r.IntN(3) == 0 {
			css = append(css, []string{"text-align: center", "text-align: right"}[r.IntN(2)])
		}
		if r.IntN(4) == 0 {
			css = append(css, []string{"line-height: 1.3", "line-height: 20px"}[r.IntN(2)])
		}
		if r.IntN(5) == 0 {
			css = append(css, fmt.Sprintf("font-size: %dpx", 10+r.IntN(10)))
		}
		if r.IntN(6) == 0 {
			css = append(css, "flex-direction: row")
		}
		fmt.Fprintf(&b, `<div style="%s">`, strings.Join(css, "; "))
		inline(2)
		b.WriteString("</div>")
	}
	b.WriteString("</body></html>\n")
	return fmt.Sprintf("script_%04d", i), b.String()
}

const randomBidiCount = 150

var bidiWords = []string{
	"Hello", "world", "text", "flexui", "ok,", "the", "[note]", "(x)", "end.",
	"שלום", "עולם", "עברית", "טקסט", "(שלום)", "מילה,", "סוף.", "\"ציטוט\"",
	"مرحبا", "بالعالم", "العربية", "نص", "(كلمة)", "سلام،", "النهاية.",
	"123", "45.6%", "٧٨٩", "+1-555", "—", "a-b", "x/y", "!",
}

// randomBidiDoc makes a seeded random document of Latin, Hebrew and
// Arabic words (numbers, brackets, punctuation) in left-to-right and
// right-to-left paragraphs, with inline elements, some of them isolates
// (dir), for the bidi algorithm and line reordering (spec/SUBSET.md
// §7.6). Lines use line-height: normal so that Chromium's lines can be
// told apart by position.
func randomBidiDoc(i int) (name, src string) {
	r := rand.New(rand.NewPCG(uint64(i), 0xb1d1))
	var b strings.Builder
	var inline func(depth int)
	inline = func(depth int) {
		n := 1 + r.IntN(4)
		for k := 0; k < n; k++ {
			ws := make([]string, 1+r.IntN(5))
			for j := range ws {
				ws[j] = bidiWords[r.IntN(len(bidiWords))]
			}
			b.WriteString(strings.Join(ws, " ") + []string{" ", "", " "}[r.IntN(3)])
			switch x := r.IntN(10); {
			case depth > 0 && x < 4:
				var css []string
				if r.IntN(2) == 0 {
					css = append(css, fmt.Sprintf("font-size: %dpx", 10+r.IntN(12)))
				}
				if r.IntN(3) == 0 {
					css = append(css, "font-weight: 700")
				}
				dir := ""
				if r.IntN(3) == 0 {
					dir = fmt.Sprintf(` dir="%s"`, []string{"ltr", "rtl", "auto"}[r.IntN(3)])
				}
				tag := []string{"span", "b", "a"}[r.IntN(3)]
				fmt.Fprintf(&b, `<%s style="%s"%s>`, tag, strings.Join(css, "; "), dir)
				inline(depth - 1)
				fmt.Fprintf(&b, "</%s>", tag)
				b.WriteString([]string{" ", "", " "}[r.IntN(3)])
			case x == 4:
				b.WriteString("<br>")
			}
		}
	}
	fmt.Fprintf(&b, "<!-- size: 340x480 -->\n<html><head><style>body { font-size: 14px; padding: 2px; gap: 4px; align-items: flex-start; }</style></head><body>")
	for p := 1 + r.IntN(3); p > 0; p-- {
		var css []string
		if r.IntN(2) == 0 {
			css = append(css, fmt.Sprintf("width: %dpx", 60+r.IntN(240)))
		}
		if r.IntN(3) == 0 {
			css = append(css, "text-align: "+[]string{"center", "right", "left", "start", "end"}[r.IntN(5)])
		}
		dir := ""
		switch r.IntN(3) {
		case 0:
			dir = ` dir="rtl"`
		case 1:
			dir = ` dir="ltr"`
		}
		fmt.Fprintf(&b, `<div style="%s"%s>`, strings.Join(css, "; "), dir)
		inline(2)
		b.WriteString("</div>")
	}
	b.WriteString("</body></html>\n")
	return fmt.Sprintf("bidi_%04d", i), b.String()
}

func writeRandomBidi(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for i := 0; i < randomBidiCount; i++ {
		name, src := randomBidiDoc(i)
		if err := os.WriteFile(filepath.Join(dir, name+".html"), []byte(src), 0o644); err != nil {
			return err
		}
	}
	return nil
}
