package asm_test

import (
	"bytes"
	"testing"
)

// Expected bytes were hand-derived from GNU binutils' GAS m68k opcode
// table: PFLUSHS shares PFLUSH's own bit layout with exactly one extra
// bit (0x0400) set in every row, and PFLUSHR's word2 (0xA000) is fully
// fixed — see newPFlushDef/defPFLUSHR's own doc comments in
// cpu030_pmmu_ptest.go for the full derivation — then confirmed against
// this assembler's own CLI output.
func TestAssemblePflushsPflushr(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"PflushsSfcMaskZero", "PFLUSHS SFC,#0\n", []byte{0xF0, 0x00, 0x34, 0x00}},
		{"PflushsSfcWithEA", "PFLUSHS SFC,#0,(A0)\n", []byte{0xF0, 0x10, 0x3C, 0x00}},
		{"PflushrIndirect", "PFLUSHR (A0)\n", []byte{0xF0, 0x10, 0xA0, 0x00}},
		{"PflushrDisplacement", "PFLUSHR (100,A0)\n", []byte{0xF0, 0x28, 0xA0, 0x00, 0x00, 0x64}},
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

func TestPflushsPflushrRequirePMMUFeature(t *testing.T) {
	mustAssembleErr(t, "PFLUSHS SFC,#0\n", targetFPU)
	mustAssembleErr(t, "PFLUSHR (A0)\n", targetFPU)
}

func TestPflushrRejectsNonMemoryEA(t *testing.T) {
	mustAssembleErr(t, "PFLUSHR D0\n", targetPMMU)
	mustAssembleErr(t, "PFLUSHR #5\n", targetPMMU)
}

func TestPflushrRequiresOperand(t *testing.T) {
	mustAssembleErr(t, "PFLUSHR\n", targetPMMU)
}
