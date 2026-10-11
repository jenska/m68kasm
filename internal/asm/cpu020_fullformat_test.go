package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// TestFullFormatSyntax covers the 68020 full extension word forms: memory
// indirect without a base displacement or base register, suppressed
// registers (ZAn, ZDn, ZPC), and indexed or displaced operands whose base
// displacement does not fit the brief format.
func TestFullFormatSyntax(t *testing.T) {
	for _, tc := range []struct {
		src  string
		want []byte
	}{
		{"LEA ([A0]),A1", []byte{0x43, 0xF0, 0x01, 0x51}},
		{"LEA ([A0,D1.L*4]),A1", []byte{0x43, 0xF0, 0x1D, 0x11}}, // preindexed
		{"LEA ([A0],D1.L*4),A1", []byte{0x43, 0xF0, 0x1D, 0x15}}, // postindexed
		{"MOVE.W ([PC]),D0", []byte{0x30, 0x3B, 0x01, 0x51}},
		{"LEA ([$1234.L]),A0", []byte{0x41, 0xF0, 0x01, 0xF1, 0x00, 0x00, 0x12, 0x34}},
		{"LEA ([$10.W,ZA0],ZD0,$20.L),A2", []byte{0x45, 0xF0, 0x01, 0xE3, 0x00, 0x10, 0x00, 0x00, 0x00, 0x20}},
		{"MOVE.L ([8,A0,D1.W*2],-4),D3", []byte{0x26, 0x30, 0x13, 0x33, 0, 0, 0, 8, 0xFF, 0xFF, 0xFF, 0xFC}},
		{"LEA ($12345678,A0,D2.L*4),A1", []byte{0x43, 0xF0, 0x2D, 0x30, 0x12, 0x34, 0x56, 0x78}},
		{"LEA ($1000.W,ZA0,D1.W),A3", []byte{0x47, 0xF0, 0x11, 0xA0, 0x10, 0x00}},
		{"LEA (ZA0,D1.W*8),A3", []byte{0x47, 0xF0, 0x17, 0x90}},
		{"MOVE.L (100000,A0),D0", []byte{0x20, 0x30, 0x01, 0x70, 0x00, 0x01, 0x86, 0xA0}},
		{"MOVE.L (8.L,A0,D1.W),D0", []byte{0x20, 0x30, 0x11, 0x30, 0, 0, 0, 8}},
	} {
		if got := assembleForTarget(t, "\t"+tc.src+"\n", target68020); !bytes.Equal(got, tc.want) {
			t.Errorf("%s: got % X want % X", tc.src, got, tc.want)
		}
	}
}

func TestFullFormatRequires68020(t *testing.T) {
	for _, src := range []string{"LEA ($1000.W,ZA0,D1.W),A3\n", "MOVE.L (100000,A0),D0\n", "LEA (ZA0,D1.W),A3\n"} {
		mustAssembleErr(t, src, instructions.Target68000)
	}
}

// TestFMoveControlRegister covers FMOVE.L with a single FPU control
// register, the same encoding as FMOVEM.L with a one-register list.
func TestFMoveControlRegister(t *testing.T) {
	for _, tc := range []struct {
		src  string
		want []byte
	}{
		{"FMOVE.L FPSR,D4", []byte{0xF2, 0x04, 0xA8, 0x00}},
		{"FMOVE.L #$400,FPCR", []byte{0xF2, 0x3C, 0x90, 0x00, 0x00, 0x00, 0x04, 0x00}},
		{"FMOVE.L FPIAR,(A0)", []byte{0xF2, 0x10, 0xA4, 0x00}},
		{"FMOVE FPCR,D0", []byte{0xF2, 0x00, 0xB0, 0x00}},
	} {
		if got := assembleForTarget(t, "\t"+tc.src+"\n", targetFPU); !bytes.Equal(got, tc.want) {
			t.Errorf("%s: got % X want % X", tc.src, got, tc.want)
		}
	}
	mustAssembleErr(t, "\tFMOVE.L FPCR/FPSR,(A0)\n", targetFPU)
}

func TestFMovemControlImmediate(t *testing.T) {
	got := assembleForTarget(t, "\tFMOVEM.L #$400,FPCR\n", targetFPU)
	if want := []byte{0xF2, 0x3C, 0x90, 0x00, 0x00, 0x00, 0x04, 0x00}; !bytes.Equal(got, want) {
		t.Fatalf("got % X want % X", got, want)
	}
}

func TestFPURoundAndSingleOps(t *testing.T) {
	for _, tc := range []struct {
		src  string
		want []byte
	}{
		{"FINT.X FP1,FP2", []byte{0xF2, 0x00, 0x05, 0x01}},
		{"FINTRZ.D #1.5,FP0", []byte{0xF2, 0x3C, 0x54, 0x03, 0x3F, 0xF8, 0, 0, 0, 0, 0, 0}},
		{"FSGLDIV.X FP3,FP4", []byte{0xF2, 0x00, 0x0E, 0x24}},
		{"FSGLMUL.S #2.0,FP1", []byte{0xF2, 0x3C, 0x44, 0xA7, 0x40, 0x00, 0x00, 0x00}},
	} {
		if got := assembleForTarget(t, "\t"+tc.src+"\n", targetFPU); !bytes.Equal(got, tc.want) {
			t.Errorf("%s: got % X want % X", tc.src, got, tc.want)
		}
	}
}
