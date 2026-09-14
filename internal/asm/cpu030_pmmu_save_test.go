package asm_test

import (
	"bytes"
	"testing"
)

// Expected bytes were hand-derived from GNU binutils' GAS m68k opcode
// table ({"psave", one(0xf100), one(0xffc0), ">s", m68851} / similarly
// for prestore's 0xf140), then confirmed against this assembler's own
// CLI output before being pinned here.
func TestAssemblePsaveRestore(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"PsaveIndirect", "PSAVE (A0)\n", []byte{0xF1, 0x10}},
		{"PsavePredec", "PSAVE -(A0)\n", []byte{0xF1, 0x20}},
		{"PrestoreIndirect", "PRESTORE (A0)\n", []byte{0xF1, 0x50}},
		{"PrestorePostinc", "PRESTORE (A0)+\n", []byte{0xF1, 0x58}},
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

// TestPsaveRestoreDoNotShareFsaveRestoreEncoding guards against a
// regression of the exact bug found and fixed elsewhere in this same
// milestone (see fpuWord1Base's use in newFSaveRestoreDef,
// cpu020_fpu2.go): PSAVE/PRESTORE have no coprocessor-ID field at all
// (GAS's own mask, 0xffc0, fixes bits 11-9 at the table literal's own
// zero value), so their word1 must stay the bare 0xF100/0xF140 literal
// — NOT fpuWord1Base-adjusted like FSAVE/FRESTORE's 0xF300/0xF340.
func TestPsaveRestoreDoNotShareFsaveRestoreEncoding(t *testing.T) {
	got := assembleForTarget(t, "PSAVE -(A0)\n", targetPMMU)
	if got[0] != 0xF1 {
		t.Fatalf("PSAVE -(A0) word1 high byte = %02X, want F1 (no coprocessor-ID bit)", got[0])
	}
}

func TestPsaveRestoreRequirePMMUFeature(t *testing.T) {
	srcs := []string{"PSAVE (A0)\n", "PRESTORE (A0)\n"}
	for _, src := range srcs {
		mustAssembleErr(t, src, targetFPU) // targetFPU has no PMMU feature bit
	}
}

func TestPsaveRejectsDataRegister(t *testing.T) {
	mustAssembleErr(t, "PSAVE D0\n", targetPMMU)
}

func TestPsaveRejectsPostincrement(t *testing.T) {
	// PSAVE writes its state frame descending, so only -(An) is its
	// dedicated predecrement-class addressing mode, not (An)+.
	mustAssembleErr(t, "PSAVE (A0)+\n", targetPMMU)
}

func TestPrestoreRejectsPredecrement(t *testing.T) {
	// PRESTORE reads its state frame ascending, so only (An)+ is its
	// dedicated postincrement-class addressing mode, not -(An).
	mustAssembleErr(t, "PRESTORE -(A0)\n", targetPMMU)
}

func TestPsaveRestoreRejectMissingOperand(t *testing.T) {
	mustAssembleErr(t, "PSAVE\n", targetPMMU)
	mustAssembleErr(t, "PRESTORE\n", targetPMMU)
}
