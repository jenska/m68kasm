package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// Expected bytes here are NOT cross-checked against a working GAS
// reference (unlike every other test file in this codebase) — GAS's
// own opcode table has dormant TBL entries with no corresponding
// parser or disassembler support (see the caveat in
// internal/asm/instructions/cpu32_tbl.go). These are instead derived
// by hand from the TBL1 macro's raw opcode/mask literals
// (opcodes/m68k-opc.c) plus this codebase's own encoder, and pinned so
// a future change can't silently alter them without review.
func TestAssembleTBL(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"TblsByteMemory", "TBLS.B (A0),D0\n", []byte{0xF8, 0x10, 0x09, 0x00}},
		{"TblsWordMemory", "TBLS.W (A0),D0\n", []byte{0xF8, 0x10, 0x09, 0x40}},
		{"TblsLongMemory", "TBLS.L (A0),D0\n", []byte{0xF8, 0x10, 0x09, 0x80}},
		{"TblsnByteMemory", "TBLSN.B (A0),D1\n", []byte{0xF8, 0x10, 0x1D, 0x00}},
		{"TbluByteMemory", "TBLU.B (A0),D0\n", []byte{0xF8, 0x10, 0x01, 0x00}},
		{"TblunByteMemory", "TBLUN.B (A0),D0\n", []byte{0xF8, 0x10, 0x05, 0x00}},
		{"TblsRegPair", "TBLS.B D2:D3,D4\n", []byte{0xF8, 0x02, 0x48, 0x03}},
		{"TblsnRegPair", "TBLSN.B D0:D1,D2\n", []byte{0xF8, 0x00, 0x2C, 0x01}},
		{"TbluRegPair", "TBLU.B D0:D1,D2\n", []byte{0xF8, 0x00, 0x20, 0x01}},
		{"TblunRegPair", "TBLUN.B D0:D1,D2\n", []byte{0xF8, 0x00, 0x24, 0x01}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assembleForTarget(t, tc.src, instructions.Target{CPU: instructions.CPU32})
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("unexpected output: got % X want % X", got, tc.want)
			}
		})
	}
}

func TestTBLIsCPU32Exclusive(t *testing.T) {
	// Unlike Bcc.L/CHK2/EXTB/TRAPcc (which CPU32 shares with 68020+),
	// GAS tags TBL plain "cpu32" with no m68020up union — it's not
	// available even on 68020+.
	srcs := []string{"TBLS.B (A0),D0\n", "TBLS.B D0:D1,D2\n"}
	for _, src := range srcs {
		mustAssembleErr(t, src, instructions.Target{CPU: instructions.CPU68020})
	}
}

// TestTBLRegPairRequiresColon guards a real bug found while writing
// this milestone: the register-pair form is deliberately listed BEFORE
// the memory form (see newTblDef's comment) so that "TBLS.B D0,D2"
// hits this form's mandatory-colon rejection specifically, rather than
// silently matching the memory form's <ea> with Dn (an earlier draft
// allowed that, via readableDataEA, and "D0,D2" assembled as if D0 were
// a memory location — wrong, and with no error at all).
func TestTBLRegPairRequiresColon(t *testing.T) {
	mustAssembleErr(t, "TBLS.B D0,D2\n", instructions.Target{CPU: instructions.CPU32})
}

func TestTBLMemoryFormRejectsImmediate(t *testing.T) {
	mustAssembleErr(t, "TBLS.B #5,D0\n", instructions.Target{CPU: instructions.CPU32})
}
