package asm_test

import (
	"bytes"
	"testing"
)

// Expected bytes were hand-derived from GNU binutils' GAS m68k opcode
// table's "fmovep" rows (word2 0x4C00 for the load direction — the
// same fpRMBit|format-3|opBase shape every other size already uses via
// FFPFormat — and 0x6C00/0x7C00 for the static/dynamic store
// directions, see cpu020_fpu_packed.go's own doc comment for the full
// derivation), then confirmed against this assembler's own CLI output.
func TestAssembleFPUPacked(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"MoveLoadFromMemory", "FMOVE.P (A0),FP0\n", []byte{0xF2, 0x10, 0x4C, 0x00}},
		{"AddLoadFromMemory", "FADD.P (A0),FP0\n", []byte{0xF2, 0x10, 0x4C, 0x22}},
		{"StoreStaticKFactor", "FMOVE.P FP3,(A0){#-5}\n", []byte{0xF2, 0x10, 0x6D, 0xFB}},
		{"StoreStaticKFactorZero", "FMOVE.P FP0,(A0){#0}\n", []byte{0xF2, 0x10, 0x6C, 0x00}},
		{"StoreStaticKFactorSeventeen", "FMOVE.P FP0,(A0){#17}\n", []byte{0xF2, 0x10, 0x6C, 0x11}},
		{"StoreDynamicKFactor", "FMOVE.P FP0,(A0){D1}\n", []byte{0xF2, 0x10, 0x7C, 0x10}},
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

func TestFPUPackedRequiresFPUFeature(t *testing.T) {
	mustAssembleErr(t, "FMOVE.P (A0),FP0\n", targetPMMU) // targetPMMU has no FeatFPU bit
	mustAssembleErr(t, "FMOVE.P FP0,(A0){#5}\n", targetPMMU)
}

// TestFPUPackedStoreRequiresKFactor guards real hardware's own
// requirement: unlike every other FMOVE store size, packed has no
// implicit default precision, so the "{...}" suffix is mandatory, not
// optional.
func TestFPUPackedStoreRequiresKFactor(t *testing.T) {
	mustAssembleErr(t, "FMOVE.P FP0,(A0)\n", targetFPU)
}

func TestFPUPackedKFactorOutOfRange(t *testing.T) {
	mustAssembleErr(t, "FMOVE.P FP0,(A0){#18}\n", targetFPU)
	mustAssembleErr(t, "FMOVE.P FP0,(A0){#-65}\n", targetFPU)
}

func TestFPUPackedKFactorRejectsNonDataRegister(t *testing.T) {
	mustAssembleErr(t, "FMOVE.P FP0,(A0){A1}\n", targetFPU)
}

// TestFPUPackedStoreDestinationMustBeMemory guards that the k-factor
// suffix doesn't accidentally loosen FMOVE.P's own destination
// restriction: Dn is not a valid packed-BCD destination on real
// hardware (packed is a digit-string format, memory-only).
func TestFPUPackedStoreDestinationMustBeMemory(t *testing.T) {
	mustAssembleErr(t, "FMOVE.P FP0,D0{#5}\n", targetFPU)
}

// TestFPUPackedNoImmediateLiteral guards the deliberate scope cut: a
// packed BCD literal parsed from decimal digits in source text (as
// opposed to a memory operand already holding packed BCD bytes) is not
// supported.
func TestFPUPackedNoImmediateLiteral(t *testing.T) {
	mustAssembleErr(t, "FMOVE.P #3.14,FP0\n", targetFPU)
}
