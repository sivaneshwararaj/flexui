package edit

import "testing"

func state(b *Buffer) string {
	s := []rune(b.String())
	out := ""
	for i := 0; i <= len(s); i++ {
		if i == b.Anchor && b.Anchor != b.Caret {
			out += "["
		}
		if i == b.Caret {
			out += "|"
		}
		if i < len(s) {
			out += string(s[i])
		}
	}
	return out
}

func TestEditing(t *testing.T) {
	b := New("hello world")
	check := func(want string) {
		t.Helper()
		if got := state(b); got != want {
			t.Errorf("got %q want %q", got, want)
		}
	}
	check("hello world|")
	b.Move(false, Word, false)
	check("hello |world")
	b.Move(false, Word, false)
	check("|hello world")
	b.Move(true, Word, false)
	check("hello| world")
	b.Move(true, Char, true)
	b.Move(true, Word, true)
	check("hello[ world|")
	b.Insert("!")
	check("hello!|")
	b.Delete(false, Char)
	check("hello|")
	b.Move(false, Line, false)
	b.Delete(true, Char)
	check("|ello")
	b.Insert("J")
	check("J|ello")
	b.SelectAll()
	check("[Jello|")
	b.Move(false, Char, false)
	check("|Jello")
	b.Insert("a\nb\tc") // newlines dropped, tabs become spaces
	check("ab c|Jello")
	b.Delete(false, Word)
	check("ab |Jello")
	b.Move(true, Line, true)
	if b.Selected() != "Jello" {
		t.Errorf("selected %q", b.Selected())
	}
	if b.Delete(true, Char); state(b) != "ab |" {
		t.Errorf("delete selection: %q", state(b))
	}
	if b.Delete(true, Char) {
		t.Error("delete at end changed text")
	}
}

// Grapheme clusters move and delete as one character.
func TestGraphemes(t *testing.T) {
	b := New("aéx") // a, e + combining acute, x
	b.Move(false, Char, false)
	b.Move(false, Char, false)
	if b.Caret != 1 {
		t.Fatalf("caret %d after two moves left, want 1", b.Caret)
	}
	b.Delete(true, Char)
	if b.String() != "ax" {
		t.Errorf("delete removed part of a cluster: %q", b.String())
	}
	b.SetCaret(2, false)
	if b.Caret != 2 {
		t.Errorf("SetCaret %d", b.Caret)
	}
	c := New("é")
	c.SetCaret(1, false) // inside the cluster: snaps to its end
	if c.Caret != 2 {
		t.Errorf("snap %d", c.Caret)
	}
}
