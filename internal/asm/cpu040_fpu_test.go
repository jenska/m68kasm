package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

var targetM68040FPU = instructions.Target{CPU: instructions.CPU68040, Features: instructions.FeatFPU}

// The FSxxx/FDxxx instructions encode like the general instruction they
// round; only the opmode (bits 6-0 of the command word) differs.
func TestAssemble68040FPURounding(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		// FP1,FP0: command word = $0400 | opmode.
		{"FSMOVE", "FSMOVE FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x40}},
		{"FDMOVE", "FDMOVE FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x44}},
		{"FSSQRT", "FSSQRT FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x41}},
		{"FDSQRT", "FDSQRT FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x45}},
		{"FSABS", "FSABS FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x58}},
		{"FDABS", "FDABS FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x5C}},
		{"FSNEG", "FSNEG FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x5A}},
		{"FDNEG", "FDNEG FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x5E}},
		{"FSDIV", "FSDIV FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x60}},
		{"FDDIV", "FDDIV FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x64}},
		{"FSADD", "FSADD FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x62}},
		{"FDADD", "FDADD FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x66}},
		{"FSMUL", "FSMUL FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x63}},
		{"FDMUL", "FDMUL FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x67}},
		{"FSSUB", "FSSUB FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x68}},
		{"FDSUB", "FDSUB FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x6C}},

		// Examples from the task description.
		{"FSADDRegReg", "FSADD.X FP1,FP2\n", []byte{0xF2, 0x00, 0x05, 0x62}},
		{"FDMULDoubleImmediate", "FDMUL.D #2.0,FP0\n",
			[]byte{0xF2, 0x3C, 0x54, 0x67, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},

		// <ea>,FPn with the source formats.
		{"FSMOVESingleMemory", "FSMOVE.S (A0),FP1\n", []byte{0xF2, 0x10, 0x44, 0xC0}},
		{"FDMOVEExtendedMemory", "FDMOVE.X (A0),FP0\n", []byte{0xF2, 0x10, 0x48, 0x44}},
		{"FDABSLongDn", "FDABS.L D1,FP2\n", []byte{0xF2, 0x01, 0x41, 0x5C}},
		{"FDNEGWordImmediate", "FDNEG.W #5,FP0\n", []byte{0xF2, 0x3C, 0x50, 0x5E, 0x00, 0x05}},
		{"FDDIVByteMemory", "FDDIV.B (A1),FP7\n", []byte{0xF2, 0x11, 0x5B, 0xE4}},
		{"FSSUBDoubleMemory", "FSSUB.D (4,A0),FP3\n", []byte{0xF2, 0x28, 0x55, 0xE8, 0x00, 0x04}},
		{"FSMULSingleImmediate", "FSMUL.S #1.0,FP0\n", []byte{0xF2, 0x3C, 0x44, 0x63, 0x3F, 0x80, 0x00, 0x00}},
		{"FDADDPackedMemory", "FDADD.P (A0),FP0\n", []byte{0xF2, 0x10, 0x4C, 0x66}},

		// Monadic forms: FPn alone means FPn,FPn.
		{"FSSQRTShorthand", "FSSQRT FP3\n", []byte{0xF2, 0x00, 0x0D, 0xC1}},
		{"FDSQRTShorthand", "FDSQRT FP0\n", []byte{0xF2, 0x00, 0x00, 0x45}},
		{"FSABSShorthand", "FSABS FP7\n", []byte{0xF2, 0x00, 0x1F, 0xD8}},
		{"FDNEGShorthand", "FDNEG FP2\n", []byte{0xF2, 0x00, 0x09, 0x5E}},

		// Mnemonics are case-insensitive.
		{"Lowercase", "fsadd.x fp1,fp2\n", []byte{0xF2, 0x00, 0x05, 0x62}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assembleForTarget(t, tc.src, targetM68040FPU)
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("unexpected output: got % X want % X", got, tc.want)
			}
		})
	}

	t.Run("MC68060", func(t *testing.T) {
		got := assembleForTarget(t, "FSADD.X FP1,FP2\n", instructions.Target{CPU: instructions.CPU68060, Features: instructions.FeatFPU})
		if want := []byte{0xF2, 0x00, 0x05, 0x62}; !bytes.Equal(got, want) {
			t.Fatalf("got % X want % X", got, want)
		}
	})
}

// The 68881/68882 take these opcodes as line-F, and without an FPU there
// is nothing to execute them.
func TestFPURoundingRequires68040(t *testing.T) {
	srcs := []string{
		"FSADD.X FP1,FP2\n",
		"FDMUL.D #2.0,FP0\n",
		"FSMOVE.S (A0),FP1\n",
		"FDSQRT FP0\n",
	}
	targets := map[string]instructions.Target{
		"68020+FPU":    targetFPU,
		"68030+FPU":    {CPU: instructions.CPU68030, Features: instructions.FeatFPU | instructions.FeatFPUFull},
		"68040 no FPU": targetM68040,
		"68000":        {},
	}
	for name, target := range targets {
		for _, src := range srcs {
			t.Run(name+"/"+src, func(t *testing.T) { mustAssembleErr(t, src, target) })
		}
	}
}

func TestFPURoundingBadOperands(t *testing.T) {
	for _, src := range []string{
		"FSADD.X FP0,(A0)\n",    // no store form
		"FSMOVE FP0\n",          // FMOVE has no single-operand form
		"FDADD.X D0,FP0\n",      // extended source needs memory
		"FDMUL.L A0,FP0\n",      // no address register source
		"FSABS.L #1.5,FP0\n",    // float literal with an integer size
		"FDSUB.X #1,(A0)\n",     // immediate destination
		"FSADD.X FP1,FP2,FP3\n", // too many operands
	} {
		t.Run(src, func(t *testing.T) { mustAssembleErr(t, src, targetM68040FPU) })
	}
}
