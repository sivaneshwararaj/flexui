//go:build darwin

#include "textflag.h"

// NSTextInputClient methods that return structs (see ime_darwin.go).
// AAPCS64: self X0, _cmd X1, arguments from X2; NSRange is returned in
// X0/X1, NSRect (four doubles) in F0-F3.

// - (NSRange)markedRange
TEXT ·markedRangeIMP(SB), NOSPLIT|NOFRAME, $0-0
	MOVD ·imeMarked+0(SB), R0
	MOVD ·imeMarked+8(SB), R1
	RET

// - (NSRange)selectedRange
TEXT ·selectedRangeIMP(SB), NOSPLIT|NOFRAME, $0-0
	MOVD ·imeSelected+0(SB), R0
	MOVD ·imeSelected+8(SB), R1
	RET

// - (NSRect)firstRectForCharacterRange:(NSRange)r actualRange:(NSRangePointer)a
// r in X2/X3, a in X4 (may be NULL: then it is not written).
TEXT ·firstRectIMP(SB), NOSPLIT|NOFRAME, $0-0
	CBZ  R4, noactual
	MOVD R2, 0(R4)
	MOVD R3, 8(R4)

noactual:
	FMOVD ·imeRect+0(SB), F0
	FMOVD ·imeRect+8(SB), F1
	FMOVD ·imeRect+16(SB), F2
	FMOVD ·imeRect+24(SB), F3
	RET

// func imeIMPs() (marked, selected, rect uintptr)
TEXT ·imeIMPs(SB), NOSPLIT, $0-24
	MOVD $·markedRangeIMP(SB), R0
	MOVD R0, marked+0(FP)
	MOVD $·selectedRangeIMP(SB), R0
	MOVD R0, selected+8(FP)
	MOVD $·firstRectIMP(SB), R0
	MOVD R0, rect+16(FP)
	RET
