// Package testtext holds multilingual strings shared by the clipboard
// and editing tests and the platform stage commands, so every layer is
// checked with the same bytes.
package testtext

// Sample is a named test string.
type Sample struct {
	Name string
	Text string
}

// Samples covers complex scripts (vowel signs, virama, conjuncts),
// right-to-left text, mixed scripts, combining marks, joiners and
// characters outside the Basic Multilingual Plane (UTF-16 surrogates).
var Samples = []Sample{
	{"tamil", "தமிழ் வணக்கம்"},
	{"tamil-sri", "ஸ்ரீ க்ஷேத்திரம் கௌ"},  // ஸ்ரீ, க்ஷ, two-part vowel sign ௌ
	{"tamil-decomposed", "க\u0bc6\u0bd7"}, // கௌ written as க + ெ + ௗ (not normalized)
	{"hindi", "नमस्ते दुनिया"},            // virama conjunct स्ते
	{"hindi-conjuncts", "क्षत्रिय स्त्री श्री"},
	{"hindi-joiners", "क्\u200dष क्\u200cष"}, // ZWJ, ZWNJ
	{"arabic", "مرحبا بالعالم"},
	{"arabic-marks", "بِسْمِ ٱللَّٰهِ"}, // harakat (combining)
	{"hebrew", "שָׁלוֹם עולם"},
	{"mixed", "Hello தமிழ் नमस्ते مرحبا 123 שלום!"},
	{"combining", "e\u0301 a\u0308\u0323 Z\u0324\u0354\u0367"}, // stacked combining marks
	{"astral", "👍🏽 👨\u200d👩\u200d👧 🇮🇳 𝔘𝔫𝔦 𠀀"},                  // skin tone, ZWJ family, flag, math letters, CJK Ext B
	{"astral-tamil", "நன்றி 🙏🏾 மிக்க நன்றி"},
}
