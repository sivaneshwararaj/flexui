package windows

import (
	"errors"
	"unicode/utf16"
)

// Clipboard text conversion (CF_UNICODETEXT), kept free of Windows APIs
// so it is tested on every OS (utf16_test.go).

// encodeClipText returns s as UTF-16 with a terminating NUL. Characters
// outside the Basic Multilingual Plane become surrogate pairs. A NUL
// inside s is an error: Windows would end the text there.
func encodeClipText(s string) ([]uint16, error) {
	for i := 0; i < len(s); i++ {
		if s[i] == 0 {
			return nil, errors.New("windows: clipboard text contains NUL")
		}
	}
	return append(utf16.Encode([]rune(s)), 0), nil
}

// decodeClipText converts clipboard UTF-16 to UTF-8. It stops at the
// first NUL or at the end of the memory block (whichever comes first, so
// text without a terminator is not over-read); unpaired surrogates
// become U+FFFD.
func decodeClipText(u []uint16) string {
	for i, c := range u {
		if c == 0 {
			u = u[:i]
			break
		}
	}
	return string(utf16.Decode(u))
}
