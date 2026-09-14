package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// Expected bytes were derived by hand from GNU binutils' GAS m68k opcode
// table (opcodes/m68k-opc.c) and gas/config/tc-m68k.c's install_operand
// logic, then confirmed against real assembler output before being
// pinned here.
func TestAssembleFMOVEMControlRegisters(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"StoreSingleToDn", "FMOVEM FPIAR,D0\n", []byte{0xF2, 0x00, 0xA4, 0x00}},
		{"StoreSingleToIndirect", "FMOVEM FPIAR,(A0)\n", []byte{0xF2, 0x10, 0xA4, 0x00}},
		{"StoreTwoToIndirect", "FMOVEM FPIAR/FPSR,(A0)\n", []byte{0xF2, 0x10, 0xAC, 0x00}},
		{"StoreAllThreeToIndirect", "FMOVEM FPCR/FPSR/FPIAR,(A0)\n", []byte{0xF2, 0x10, 0xBC, 0x00}},
		{"LoadSingleFromIndirect", "FMOVEM (A0),FPIAR\n", []byte{0xF2, 0x10, 0x84, 0x00}},
		{"LoadTwoFromIndirect", "FMOVEM (A0),FPIAR/FPSR\n", []byte{0xF2, 0x10, 0x8C, 0x00}},
		{"LoadFromDn", "FMOVEM D0,FPIAR\n", []byte{0xF2, 0x00, 0x84, 0x00}},
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

func TestFMOVEMControlRegSelectorBits(t *testing.T) {
	// FPIAR=bit0, FPSR=bit1, FPCR=bit2 (bits 10-12 of word2) — GAS's own
	// assignment, confirmed independently for each register.
	tests := []struct {
		src  string
		want byte // word2 high byte
	}{
		{"FMOVEM FPIAR,(A0)\n", 0xA4},
		{"FMOVEM FPSR,(A0)\n", 0xA8},
		{"FMOVEM FPCR,(A0)\n", 0xB0},
	}
	for _, tc := range tests {
		got := assembleForTarget(t, tc.src, targetFPU)
		if got[2] != tc.want {
			t.Fatalf("%q: word2 high byte = %02X, want %02X", tc.src, got[2], tc.want)
		}
	}
}

func TestFMOVEMControlRegRequiresFPUFeature(t *testing.T) {
	mustAssembleErr(t, "FMOVEM FPIAR,(A0)\n", instructions.Target{CPU: instructions.CPU68020})
}

// TestFMOVEMMultiRegStoreRejectsRegisterDest guards the deliberate
// single-vs-multi split: GAS's own opcode table has a FIXME noting only
// a single named register may target Dn/An; a genuine multi-register
// combination must target memory (a single 32-bit Dn/An can't sensibly
// receive two distinct registers' values in one move).
func TestFMOVEMMultiRegStoreRejectsRegisterDest(t *testing.T) {
	mustAssembleErr(t, "FMOVEM FPIAR/FPSR,D0\n", targetFPU)
	mustAssembleErr(t, "FMOVEM FPIAR/FPSR,A0\n", targetFPU)
}

func TestFMOVEMSingleRegStoreAllowsAn(t *testing.T) {
	got := assembleForTarget(t, "FMOVEM FPIAR,A0\n", targetFPU)
	if len(got) != 4 {
		t.Fatalf("got %d bytes, want 4", len(got))
	}
}

func TestFMOVEMMultiRegStoreAllowsMemory(t *testing.T) {
	got := assembleForTarget(t, "FMOVEM FPIAR/FPSR/FPCR,$1000\n", targetFPU)
	if len(got) != 8 { // word1 + word2 + 4-byte absolute address
		t.Fatalf("got %d bytes, want 8", len(got))
	}
}

func TestFMOVEMControlRegRejectsUnknownName(t *testing.T) {
	mustAssembleErr(t, "FMOVEM FPXX,(A0)\n", targetFPU)
}
