package edit

import (
	"fmt"
	"strings"
	"testing"
)

// clusters lists the text between consecutive grapheme boundaries.
func clusters(b *Buffer) string {
	var out []string
	bs := b.Boundaries()
	r := []rune(b.String())
	for i := 1; i < len(bs); i++ {
		out = append(out, string(r[bs[i-1]:bs[i]]))
	}
	return strings.Join(out, "|")
}

// TestIndicClusters records how the buffer splits Tamil and Devanagari
// (UAX #29 extended grapheme clusters, Unicode 15.1 as implemented by
// go-text/typesetting v0.3.5): Devanagari consonant+virama+consonant
// conjuncts are one cluster (rule GB9c, Indic_Conjunct_Break=Linker);
// Tamil's virama (pulli) is not a linker, so ஸ்ரீ is two clusters.
func TestIndicClusters(t *testing.T) {
	for _, c := range []struct{ s, want string }{
		{"தமிழ்", "த|மி|ழ்"},
		{"வணக்கம்", "வ|ண|க்|க|ம்"},
		{"ஸ்ரீ", "ஸ்|ரீ"},
		{"கௌ", "கௌ"},
		{"கௌ", "கௌ"},
		{"नमस्ते", "न|म|स्ते"},
		{"क्षत्रिय", "क्ष|त्रि|य"},
		{"स्त्री", "स्त्री"},
		{"क्‍ष", "क्‍ष"},
		{"क्‌ष", "क्‌|ष"},
		{"بِسْمِ", "بِ|سْ|مِ"},
		{"👨‍👩‍👧🇮🇳", "👨‍👩‍👧|🇮🇳"},
	} {
		if got := clusters(New(c.s)); got != c.want {
			t.Errorf("%s: %s, want %s", c.s, got, c.want)
		}
	}
}

// TestIndicEditing: caret movement, selection and deletion work on whole
// clusters; text is never split inside one.
func TestIndicEditing(t *testing.T) {
	b := New("नमस्ते")
	b.Move(false, Char, false)
	if b.Caret != 2 {
		t.Errorf("left from end: caret %d, want 2 (before स्ते)", b.Caret)
	}
	b.Move(false, Char, true)
	if b.Selected() != "म" {
		t.Errorf("shift+left selects %q", b.Selected())
	}
	b = New("नमस्ते")
	b.Delete(false, Char) // Backspace removes one code point (the vowel sign)
	if b.String() != "नमस्त" {
		t.Errorf("backspace: %q", b.String())
	}
	b = New("தமிழ் வணக்கம்")
	b.Delete(false, Char)
	if b.String() != "தமிழ் வணக்கம" {
		t.Errorf("backspace: %q", b.String())
	}
	b.SetCaret(2, false) // inside மி (between ம and ி): snapped forward
	if b.Caret != 3 {
		t.Errorf("SetCaret inside a cluster: %d", b.Caret)
	}
}

// TestIndicWords: vowel signs, viramas and joiners are part of words.
func TestIndicWords(t *testing.T) {
	for _, s := range []string{"தமிழ் வணக்கம்", "नमस्ते दुनिया", "क्‍ष क्‌ष", "بِسْمِ ٱللَّٰهِ"} {
		b := New(s)
		r := []rune(s)
		sp := strings.IndexRune(s, ' ')
		first := len([]rune(s[:sp]))
		if p := b.PrevWord(len(r)); p != first+1 {
			t.Errorf("%s: PrevWord from end %d, want %d", s, p, first+1)
		}
		if p := b.NextWord(0); p != first {
			t.Errorf("%s: NextWord from start %d, want %d", s, p, first)
		}
		b.Delete(false, Word)
		if want := string(r[:first+1]); b.String() != want {
			t.Errorf("%s: word backspace %q, want %q", s, b.String(), want)
		}
	}
}

// TestChromiumEditing checks Backspace, Delete and caret movement against
// Chromium 153, measured in an <input> with Playwright (typing the text,
// then Backspace three times; End and ArrowLeft; Home and Delete):
// Backspace removes one code point except in emoji sequences, Delete a
// whole cluster, arrows move by cluster. Positions are code points
// (Chromium reports UTF-16 units; the same outside the emoji).
func TestChromiumEditing(t *testing.T) {
	for _, c := range []struct {
		s, bs, left, del string
	}{
		{"नमस्ते", "नमस्त|नमस्|नमस", "6,2,1,0", "मस्ते"},
		{"क्ष", "क्|क|", "3,0", ""},
		{"தமிழ்", "தமிழ|தமி|தம", "5,3,1,0", "மிழ்"},
		{"கௌ", "க||", "2,0", ""},
		{"க\u0bc6\u0bd7", "க\u0bc6|க|", "3,0", ""},
		{"e\u0301", "e||", "2,0", ""},
		{"👨\u200d👩\u200d👧", "||", "5,0", ""},
		{"🇮🇳", "||", "2,0", ""},
		{"👍🏽", "||", "2,0", ""},
		{"بِسْمِ", "بِسْم|بِسْ|بِس", "6,4,2,0", "سْمِ"},
		{"שָׁלוֹם", "שָׁלוֹ|שָׁלו|שָׁל", "7,6,4,3,0", "לוֹם"},
		{"स्त्री", "स्त्र|स्त्|स्त", "6,0", ""},
		{"ஸ்ரீ", "ஸ்ர|ஸ்|ஸ", "4,2,0", "ரீ"},
	} {
		b := New(c.s)
		var bs []string
		for k := 0; k < 3; k++ {
			b.Delete(false, Char)
			bs = append(bs, b.String())
		}
		if got := strings.Join(bs, "|"); got != c.bs {
			t.Errorf("%q: backspace %q, Chromium %q", c.s, got, c.bs)
		}
		b = New(c.s)
		var pos []string
		for k := 0; k < 8; k++ {
			p := fmt.Sprint(b.Caret)
			if len(pos) > 0 && pos[len(pos)-1] == p {
				break
			}
			pos = append(pos, p)
			b.Move(false, Char, false)
		}
		if got := strings.Join(pos, ","); got != c.left {
			t.Errorf("%q: left positions %s, Chromium %s", c.s, got, c.left)
		}
		b.Move(false, Line, false)
		b.Delete(true, Char)
		if b.String() != c.del {
			t.Errorf("%q: delete %q, Chromium %q", c.s, b.String(), c.del)
		}
	}
}
