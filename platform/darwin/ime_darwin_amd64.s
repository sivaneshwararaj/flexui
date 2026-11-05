//go:build darwin

#include "textflag.h"

// NSTextInputClient methods that return structs (see ime_darwin.go).
// System V: an NSRange (16 bytes, integer class) is returned in AX/DX;
// an NSRect (32 bytes) is returned in memory: the caller passes the
// result address in DI (then self in SI, _cmd in DX, the range in CX/R8,
// actualRange in R9), and the function returns it in AX.

// - (NSRange)markedRange
TEXT ·markedRangeIMP(SB), NOSPLIT, $0-0
	MOVQ ·imeMarked+0(SB), AX
	MOVQ ·imeMarked+8(SB), DX
	RET

// - (NSRange)selectedRange
TEXT ·selectedRangeIMP(SB), NOSPLIT, $0-0
	MOVQ ·imeSelected+0(SB), AX
	MOVQ ·imeSelected+8(SB), DX
	RET

// - (NSRect)firstRectForCharacterRange:(NSRange)r actualRange:(NSRangePointer)a
TEXT ·firstRectIMP(SB), NOSPLIT, $0-0
	TESTQ R9, R9
	JZ    noactual
	MOVQ  CX, 0(R9)
	MOVQ  R8, 8(R9)

noactual:
	MOVQ ·imeRect+0(SB), AX
	MOVQ AX, 0(DI)
	MOVQ ·imeRect+8(SB), AX
	MOVQ AX, 8(DI)
	MOVQ ·imeRect+16(SB), AX
	MOVQ AX, 16(DI)
	MOVQ ·imeRect+24(SB), AX
	MOVQ AX, 24(DI)
	MOVQ DI, AX
	RET

// func imeIMPs() (marked, selected, rect uintptr)
TEXT ·imeIMPs(SB), NOSPLIT, $0-24
	LEAQ ·markedRangeIMP(SB), AX
	MOVQ AX, marked+0(FP)
	LEAQ ·selectedRangeIMP(SB), AX
	MOVQ AX, selected+8(FP)
	LEAQ ·firstRectIMP(SB), AX
	MOVQ AX, rect+16(FP)
	RET
