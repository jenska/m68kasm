package asm_test

import (
	"bytes"
	"testing"
)

// Expected bytes were hand-derived (IEEE-754 binary32/binary64 for
// single/double; the 68881/68882's own 96-bit extended format — sign+15
// -bit biased exponent, a 16-bit reserved zero word, then a 64-bit
// mantissa with an EXPLICIT leading integer bit — for extended, see
// encodeExtendedReal's own doc comment in encode.go), then confirmed
// against this assembler's own CLI output.
func TestAssembleFPUFloatImmediate(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"SingleOneAndHalf", "FMOVE.S #1.5,FP0\n", []byte{0xF2, 0x3C, 0x04, 0x00, 0x3F, 0xC0, 0x00, 0x00}},
		{"DoubleOneAndHalf", "FMOVE.D #1.5,FP0\n", []byte{0xF2, 0x3C, 0x14, 0x00, 0x3F, 0xF8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{"ExtendedOne", "FMOVE.X #1.0,FP0\n", []byte{
			0xF2, 0x3C, 0x08, 0x00, 0x3F, 0xFF, 0x00, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		}},
		{"ExtendedOneAndHalf", "FMOVE.X #1.5,FP0\n", []byte{
			0xF2, 0x3C, 0x08, 0x00, 0x3F, 0xFF, 0x00, 0x00, 0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		}},
		{"ExtendedNegativeOne", "FMOVE.X #-1.0,FP0\n", []byte{
			0xF2, 0x3C, 0x08, 0x00, 0xBF, 0xFF, 0x00, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		}},
		{"ExtendedZero", "FMOVE.X #0.0,FP0\n", []byte{
			0xF2, 0x3C, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		}},
		{"ExtendedIntegerLiteralAutoPromoted", "FADD.X #5,FP0\n", []byte{
			0xF2, 0x3C, 0x08, 0x22, 0x40, 0x01, 0x00, 0x00, 0xA0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assembleForTarget(t, tc.src, targetFPU)
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("unexpected output: got % X want % X", got, tc.want)
			}
		})
	}
}

func TestFPUFloatImmediateExponentSyntax(t *testing.T) {
	got := assembleForTarget(t, "FMOVE.X #1.5e2,FP0\n", targetFPU)
	if len(got) != 16 {
		t.Fatalf("got %d bytes, want 16", len(got))
	}
	// 150 = 0.5859375 * 2^8 -> biased extended exponent (8-1)+16383 = 16390 = 0x4006.
	if got[4] != 0x40 || got[5] != 0x06 {
		t.Fatalf("exponent word = %02X%02X, want 4006", got[4], got[5])
	}
}

// TestFloatImmediateRejectedForNonFPUInstruction guards
// FormDef.AllowFloatImm's whole reason for existing: a stray float
// literal typed against an ordinary integer instruction must be a hard
// error, never silently encoded as if the immediate were zero (which is
// what would happen if only Imm, and not ImmFloat, were consulted).
func TestFloatImmediateRejectedForNonFPUInstruction(t *testing.T) {
	mustAssembleErr(t, "MOVE.W #3.5,D0\n", targetFPU)
	mustAssembleErr(t, "ADD.L #1.25,D0\n", targetFPU)
}

// TestFloatLiteralRejectedInIntegerExpression guards parseExpr's own
// explicit float rejection (expr.go) — the safety net for every
// non-immediate use of an expression (displacements, .org, etc.), not
// just plain immediates.
func TestFloatLiteralRejectedInIntegerExpression(t *testing.T) {
	mustAssembleErr(t, ".org 1.5\n", targetFPU)
}
