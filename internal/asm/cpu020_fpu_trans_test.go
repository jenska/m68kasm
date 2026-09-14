package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

var targetFPUFull = instructions.Target{CPU: instructions.CPU68020, Features: instructions.FeatFPU | instructions.FeatFPUFull}

// Expected bytes match the exact FFPFormat/FFPSrcReg10/FFPDstReg7
// placement already verified for FABS/FADD in cpu020_fpu_test.go — the
// transcendental set reuses newFPMonadicDef unchanged, just with a
// different opBase and requireFPUFull instead of requireFPU.
func TestAssembleFPUTranscendental(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"FsinMemorySourceExtended", "FSIN.X (A0),FP0\n", []byte{0xF2, 0x10, 0x48, 0x0E}},
		{"FsinRegisterToRegister", "FSIN FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x0E}},
		{"FcosSingleOperandShorthand", "FCOS FP0\n", []byte{0xF2, 0x00, 0x00, 0x1D}},
		{"Ftan", "FTAN FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x0F}},
		{"Fatan", "FATAN FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x0A}},
		{"Fasin", "FASIN FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x0C}},
		{"Facos", "FACOS FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x1C}},
		{"Fatanh", "FATANH FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x0D}},
		{"Fsinh", "FSINH FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x02}},
		{"Fcosh", "FCOSH FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x19}},
		{"Ftanh", "FTANH FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x09}},
		{"Fetox", "FETOX FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x10}},
		{"Fetoxm1", "FETOXM1 FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x08}},
		{"Flogn", "FLOGN FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x14}},
		{"Flognp1", "FLOGNP1 FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x06}},
		{"Flog10", "FLOG10 FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x15}},
		{"Flog2", "FLOG2 FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x16}},
		{"Ftwotox", "FTWOTOX FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x11}},
		{"Ftentox", "FTENTOX FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x12}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assembleForTarget(t, tc.src, targetFPUFull)
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("unexpected output: got % X want % X", got, tc.want)
			}
		})
	}
}

// TestFPUTranscendentalRequiresFPUFull guards the point of this
// milestone's gating design: plain --fpu (FeatFPU alone, modeling an
// integrated/reduced FPU) is not enough for the transcendental set —
// --fpu-full (FeatFPU|FeatFPUFull, a real discrete 68881/68882) is
// required. This must not affect the plain FPU instructions
// (FADD/etc.), which still only need requireFPU.
func TestFPUTranscendentalRequiresFPUFull(t *testing.T) {
	mustAssembleErr(t, "FSIN FP0\n", targetFPU) // targetFPU = FeatFPU only, cpu020_fpu_test.go
}

func TestPlainFPUStillOnlyNeedsFeatFPU(t *testing.T) {
	got := assembleForTarget(t, "FADD FP1,FP0\n", targetFPU)
	if len(got) != 4 {
		t.Fatalf("got %d bytes, want 4", len(got))
	}
}

func TestFPUFullAloneAlsoGrantsPlainFPU(t *testing.T) {
	// --fpu-full implies --fpu at the CLI level (main.go sets both
	// bits); confirm the Target-level combination works the same way.
	got := assembleForTarget(t, "FADD FP1,FP0\n", targetFPUFull)
	if len(got) != 4 {
		t.Fatalf("got %d bytes, want 4", len(got))
	}
}

func TestFPUTranscendentalRequiresSomeFPUFeature(t *testing.T) {
	mustAssembleErr(t, "FSIN FP0\n", instructions.Target{CPU: instructions.CPU68020})
}
