package asm_test

import (
	"bytes"
	"testing"
)

// Expected bytes were hand-derived from GNU binutils' GAS m68k opcode
// table's three "pmovefd" rows (word2 0x4100 for TC, 0x4100|(sel<<10)
// for DRP/SRP/CRP, 0x0900/0x0D00 for TT0/TT1 — see
// cpu030_pmmu_pmovefd.go's own doc comment for the full derivation),
// then confirmed against this assembler's own CLI output.
func TestAssemblePmovefd(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"PmovefdTC", "PMOVEFD.L (A0),TC\n", []byte{0xF0, 0x10, 0x41, 0x00}},
		{"PmovefdDRP", "PMOVEFD.L (A0),DRP\n", []byte{0xF0, 0x10, 0x45, 0x00}},
		{"PmovefdSRP", "PMOVEFD.L (A0),SRP\n", []byte{0xF0, 0x10, 0x49, 0x00}},
		{"PmovefdCRP", "PMOVEFD.L (A0),CRP\n", []byte{0xF0, 0x10, 0x4D, 0x00}},
		{"PmovefdTT0", "PMOVEFD.L (A0),TT0\n", []byte{0xF0, 0x10, 0x09, 0x00}},
		{"PmovefdTT1", "PMOVEFD.L (A0),TT1\n", []byte{0xF0, 0x10, 0x0D, 0x00}},
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

func TestPmovefdRequiresPMMUFeature(t *testing.T) {
	mustAssembleErr(t, "PMOVEFD.L (A0),TC\n", targetFPU)
}

// TestPmovefdIsLoadOnly guards GAS's own table shape: unlike PMOVE,
// PMOVEFD has no store-direction row at all (its whole purpose is
// disabling function-code lookup while *loading* a translation
// register), so "PMOVEFD REG,<ea>" must not parse as any known form.
func TestPmovefdIsLoadOnly(t *testing.T) {
	mustAssembleErr(t, "PMOVEFD.L TC,(A0)\n", targetPMMU)
}

func TestPmovefdCrpSrpDrpRequireMemoryEA(t *testing.T) {
	mustAssembleErr(t, "PMOVEFD.L D0,CRP\n", targetPMMU)
	mustAssembleErr(t, "PMOVEFD.L D0,SRP\n", targetPMMU)
	mustAssembleErr(t, "PMOVEFD.L D0,DRP\n", targetPMMU)
}

// TestPmovefdWordMatchesPmoveLoadPlusFDBit guards the +0x0100 relation
// this codebase relies on instead of a fourth hardcoded literal per
// register: PMOVEFD's word2 must equal PMOVE's own load-direction word2
// for the same register, with bit 8 additionally set.
func TestPmovefdWordMatchesPmoveLoadPlusFDBit(t *testing.T) {
	pmove := assembleForTarget(t, "PMOVE.L (A0),TC\n", targetPMMU)
	pmovefd := assembleForTarget(t, "PMOVEFD.L (A0),TC\n", targetPMMU)
	pmoveWord2 := uint16(pmove[2])<<8 | uint16(pmove[3])
	pmovefdWord2 := uint16(pmovefd[2])<<8 | uint16(pmovefd[3])
	if pmovefdWord2 != pmoveWord2|0x0100 {
		t.Fatalf("PMOVEFD word2 = %04X, want PMOVE word2 (%04X) | 0x0100", pmovefdWord2, pmoveWord2)
	}
}
