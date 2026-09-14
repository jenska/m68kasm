package asm_test

import (
	"bytes"
	"testing"
)

// Expected bytes were hand-derived from GNU binutils' GAS m68k opcode
// table's two "fsincosx" rows (word2 0x0030 | FFPFormat | sine-at-bits9-7
// | cosine-at-bits2-0 — see cpu020_fpu_sincos.go's own doc comment for
// the full derivation), then confirmed against this assembler's own CLI
// output.
func TestAssembleFsincos(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"FsincosRegisterToRegister", "FSINCOS FP1,FP2:FP3\n", []byte{0xF2, 0x00, 0x05, 0xB2}},
		{"FsincosMemorySourceExtended", "FSINCOS.X (A0),FP2:FP3\n", []byte{0xF2, 0x10, 0x49, 0xB2}},
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

func TestFsincosRequiresFPUFull(t *testing.T) {
	mustAssembleErr(t, "FSINCOS FP1,FP2:FP3\n", targetFPU) // targetFPU has no FeatFPUFull bit
}

// TestFsincosCosineFirstSineSecond guards the field/position mapping
// this codebase relies on instead of a symmetric "either order" pair:
// Motorola's own documented syntax is "FSINCOS <ea>,FPc:FPs" (cosine
// first, sine second), with cosine landing in bits 2-0 and sine sharing
// every other FPU instruction's usual bits 9-7 destination field. Colon
// order must not be interchangeable with the other functions' plain
// destination register.
func TestFsincosCosineFirstSineSecond(t *testing.T) {
	got := assembleForTarget(t, "FSINCOS FP0,FP1:FP2\n", targetFPUFull)
	word2 := uint16(got[2])<<8 | uint16(got[3])
	cos := word2 & 0x7
	sin := (word2 >> 7) & 0x7
	if cos != 1 {
		t.Fatalf("cosine register (bits 2-0) = %d, want 1 (FP1, written first)", cos)
	}
	if sin != 2 {
		t.Fatalf("sine register (bits 9-7) = %d, want 2 (FP2, written second)", sin)
	}
}

func TestFsincosRequiresColonPair(t *testing.T) {
	mustAssembleErr(t, "FSINCOS FP0,FP1\n", targetFPUFull)
}

func TestFsincosDestinationMustBeFPRegisters(t *testing.T) {
	mustAssembleErr(t, "FSINCOS FP0,FP1:D0\n", targetFPUFull)
	mustAssembleErr(t, "FSINCOS FP0,D0:FP1\n", targetFPUFull)
}

func TestFsincosRejectsDataRegisterSource(t *testing.T) {
	mustAssembleErr(t, "FSINCOS D0,FP1:FP2\n", targetFPUFull)
}
