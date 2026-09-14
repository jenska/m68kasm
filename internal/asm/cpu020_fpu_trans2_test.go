package asm_test

import (
	"bytes"
	"testing"
)

// Expected bytes match the exact FFPFormat/FFPSrcReg10/FFPDstReg7
// placement already verified for FABS/FADD/FSIN — this file's
// mnemonics reuse newFPMonadicDef/newFPBinaryDef unchanged, just with
// their own opBase and requireFPUFull.
func TestAssembleFPUMathExtensions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"FgetexpRegisterToRegister", "FGETEXP FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x1E}},
		{"FgetexpMemorySourceExtended", "FGETEXP.X (A0),FP0\n", []byte{0xF2, 0x10, 0x08, 0x1E}},
		{"Fgetman", "FGETMAN FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x1F}},
		{"FscaleRegisterToRegister", "FSCALE FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x26}},
		{"FscaleMemorySourceExtended", "FSCALE.X (A0),FP0\n", []byte{0xF2, 0x10, 0x08, 0x26}},
		{"Fmod", "FMOD FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x21}},
		{"Frem", "FREM FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x25}},
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

func TestFPUMathExtensionsRequireFPUFull(t *testing.T) {
	srcs := []string{"FGETEXP FP1,FP0\n", "FSCALE FP1,FP0\n", "FMOD FP1,FP0\n", "FREM FP1,FP0\n"}
	for _, src := range srcs {
		mustAssembleErr(t, src, targetFPU) // targetFPU = FeatFPU only, cpu020_fpu_test.go
	}
}

// TestFscaleRejectsStore guards allowStore=false: unlike FMOVE,
// FSCALE/FMOD/FREM's result only ever goes to an FPn register, never
// out to memory.
func TestFscaleRejectsStore(t *testing.T) {
	mustAssembleErr(t, "FSCALE FP0,(A0)\n", targetFPUFull)
}
