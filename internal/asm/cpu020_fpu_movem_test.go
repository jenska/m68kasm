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
func TestAssembleFMOVEM(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"StaticStorePredec", "FMOVEM.X FP0-FP3,-(A7)\n", []byte{0xF2, 0x27, 0xE0, 0xF0}},
		{"StaticStoreGeneral", "FMOVEM.X FP0-FP3,(A0)\n", []byte{0xF2, 0x10, 0xF0, 0x0F}},
		{"StaticLoadPostinc", "FMOVEM.X (A0)+,FP0-FP3\n", []byte{0xF2, 0x18, 0xD0, 0x0F}},
		{"DynamicStorePredec", "FMOVEM.X D0,-(A7)\n", []byte{0xF2, 0x27, 0xE8, 0x00}},
		{"DynamicStoreGeneral", "FMOVEM.X D0,(A0)\n", []byte{0xF2, 0x10, 0xF8, 0x00}},
		{"DynamicLoadPostinc", "FMOVEM.X (A0)+,D0\n", []byte{0xF2, 0x18, 0xD8, 0x00}},
		{"DynamicLoadGeneral", "FMOVEM.X (A0),D0\n", []byte{0xF2, 0x10, 0xD8, 0x00}},
		{"StaticStoreSingleReg", "FMOVEM.X FP5,-(A7)\n", []byte{0xF2, 0x27, 0xE0, 0x04}},
		{"StaticStoreSlashList", "FMOVEM.X FP1/FP4/FP6,(A0)\n", []byte{0xF2, 0x10, 0xF0, 0x52}},
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

// TestFMovemStorePredecReversesListVsGeneral guards the one Form
// design's central claim: the predecrement and general store cases
// must differ in BOTH the word2 base bit AND the list's bit order,
// computed purely from the Dst EA's runtime mode (FFPMovemStoreWord2),
// not from which Form matched — see its doc comment in types.go.
func TestFMovemStorePredecReversesListVsGeneral(t *testing.T) {
	general := assembleForTarget(t, "FMOVEM.X FP0-FP3,(A0)\n", targetFPU)
	predec := assembleForTarget(t, "FMOVEM.X FP0-FP3,-(A7)\n", targetFPU)
	if general[2] != 0xF0 || general[3] != 0x0F {
		t.Fatalf("general store word2 = % X, want F0 0F (unreversed mask)", general[2:4])
	}
	if predec[2] != 0xE0 || predec[3] != 0xF0 {
		t.Fatalf("predecrement store word2 = % X, want E0 F0 (reversed mask)", predec[2:4])
	}
}

func TestFMOVEMRequiresFPUFeature(t *testing.T) {
	srcs := []string{"FMOVEM.X FP0-FP3,-(A7)\n", "FMOVEM.X (A0)+,FP0-FP3\n", "FMOVEM.X D0,(A0)\n"}
	for _, src := range srcs {
		mustAssembleErr(t, src, instructions.Target{CPU: instructions.CPU68020})
	}
}

func TestFMovemStoreRejectsPostincrement(t *testing.T) {
	// Postincrement is load-only for FMOVEM, exactly like integer MOVEM.
	mustAssembleErr(t, "FMOVEM.X FP0-FP3,(A0)+\n", targetFPU)
}

func TestFMovemLoadRejectsPredecrement(t *testing.T) {
	// Predecrement is store-only for FMOVEM, exactly like integer MOVEM.
	mustAssembleErr(t, "FMOVEM.X -(A0),FP0-FP3\n", targetFPU)
}

func TestFMovemDynamicStoreRejectsPostincrement(t *testing.T) {
	mustAssembleErr(t, "FMOVEM.X D0,(A0)+\n", targetFPU)
}

func TestFMovemDynamicLoadRejectsPredecrement(t *testing.T) {
	mustAssembleErr(t, "FMOVEM.X -(A0),D0\n", targetFPU)
}

func TestFMovemRejectsDescendingRange(t *testing.T) {
	mustAssembleErr(t, "FMOVEM.X FP3-FP0,(A0)\n", targetFPU)
}

func TestFMovemRejectsNonFPnInList(t *testing.T) {
	mustAssembleErr(t, "FMOVEM.X D0-D3,(A0)\n", targetFPU)
}
