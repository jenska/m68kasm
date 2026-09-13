package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// Expected bytes were derived by hand from GNU binutils' GAS m68k opcode
// table (opcodes/m68k-opc.c) and gas/config/tc-m68k.c's install_operand
// logic (the "case 'f'"/"case 'D'"/"case 'T'" function-code-specifier
// dispatches and the "case '8'"/"case '9'"/"case '3'" bit placements),
// then confirmed against real assembler output before being pinned here.
func TestAssemblePflushPloadPtest(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"PflushSfcMaskZero", "PFLUSH SFC,#0\n", []byte{0xF0, 0x00, 0x30, 0x00}},
		{"PflushDfcMaskZero", "PFLUSH DFC,#0\n", []byte{0xF0, 0x00, 0x30, 0x01}},
		{"PflushDnMask", "PFLUSH D3,#5\n", []byte{0xF0, 0x00, 0x30, 0xAB}},
		{"PflushImmMask", "PFLUSH #2,#16\n", []byte{0xF0, 0x00, 0x32, 0x12}},
		{"PflushSfcWithEA", "PFLUSH SFC,#0,(A0)\n", []byte{0xF0, 0x10, 0x38, 0x00}},
		{"PflushDnWithEA", "PFLUSH D3,#5,(A0)\n", []byte{0xF0, 0x10, 0x38, 0xAB}},
		{"PloadrSfc", "PLOADR SFC,(A0)\n", []byte{0xF0, 0x10, 0x22, 0x00}},
		{"PloadwDfc", "PLOADW DFC,(A0)\n", []byte{0xF0, 0x10, 0x20, 0x01}},
		{"PloadrDn", "PLOADR D2,(A0)\n", []byte{0xF0, 0x10, 0x22, 0x0A}},
		{"PtestrSfcLevel", "PTESTR SFC,(A0),#3\n", []byte{0xF0, 0x10, 0x8E, 0x00}},
		{"PtestwDfcLevel", "PTESTW DFC,(A0),#5\n", []byte{0xF0, 0x10, 0x94, 0x01}},
		{"PtestrSfcLevelAn", "PTESTR SFC,(A0),#3,A2\n", []byte{0xF0, 0x10, 0x8E, 0x40}},
		{"PtestwDnLevelAn", "PTESTW D1,(A0),#7,A5\n", []byte{0xF0, 0x10, 0x9C, 0xA9}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assembleForTarget(t, tc.src, targetPMMU)
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("unexpected output: got % X want % X", got, tc.want)
			}
		})
	}
}

func TestPflushPloadPtestRequirePMMUFeature(t *testing.T) {
	srcs := []string{"PFLUSH SFC,#0\n", "PLOADR SFC,(A0)\n", "PTESTR SFC,(A0),#3\n"}
	for _, src := range srcs {
		mustAssembleErr(t, src, instructions.Target{CPU: instructions.CPU68030})
	}
}

func TestPflushRejectsMaskOutOfRange(t *testing.T) {
	mustAssembleErr(t, "PFLUSH SFC,#32\n", targetPMMU)
}

func TestPtestRejectsLevelOutOfRange(t *testing.T) {
	mustAssembleErr(t, "PTESTR SFC,(A0),#8\n", targetPMMU)
}

func TestFCSpecRejectsInvalidToken(t *testing.T) {
	mustAssembleErr(t, "PLOADR A0,(A0)\n", targetPMMU)
}

func TestPflushRejectsNonMemoryEA(t *testing.T) {
	mustAssembleErr(t, "PFLUSH SFC,#0,D0\n", targetPMMU)
}

func TestPloadRejectsNonMemoryEA(t *testing.T) {
	mustAssembleErr(t, "PLOADR SFC,D0\n", targetPMMU)
}

// TestPtestOptionalAnResultRegister guards the one genuinely new piece
// of Args-model plumbing this milestone needed: a fourth operand slot
// (Aux2), used only by PTEST's optional trailing An result register.
func TestPtestOptionalAnResultRegister(t *testing.T) {
	withoutAn := assembleForTarget(t, "PTESTR SFC,(A0),#3\n", targetPMMU)
	withAn := assembleForTarget(t, "PTESTR SFC,(A0),#3,A2\n", targetPMMU)
	if len(withoutAn) != 4 || len(withAn) != 4 {
		t.Fatalf("expected both forms to be 4 bytes, got %d and %d", len(withoutAn), len(withAn))
	}
	if bytes.Equal(withoutAn, withAn) {
		t.Fatalf("expected the An-result form to differ from the bare form")
	}
}
