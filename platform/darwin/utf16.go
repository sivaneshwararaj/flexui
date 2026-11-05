package darwin

import "unicode/utf16"

// utf16ToRune converts an index in UTF-16 code units of s (NSString
// ranges) to a rune index; utf16Len is the length of s in UTF-16 units.
// Kept free of AppKit so it is tested on every OS.
func utf16ToRune(s string, u int) int {
	n := 0
	for i, r := range []rune(s) {
		if n >= u {
			return i
		}
		n += len(utf16.Encode([]rune{r}))
	}
	return len([]rune(s))
}

func utf16Len(s string) int { return len(utf16.Encode([]rune(s))) }
