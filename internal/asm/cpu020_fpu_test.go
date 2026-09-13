package asm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
	"github.com/jenska/m68kasm/internal/asm/instructions"
)

var targetFPU = instructions.Target{CPU: instructions.CPU68020, Features: instructions.FeatFPU}

// Expected bytes below were derived by hand from GNU binutils' GAS m68k
// opcode table (opcodes/m68k-opc.c) and then confirmed against the
// actual assembler output before being pinned here. Two real bugs were
// found and fixed while doing that confirmation — see
// TestFPUFixedZeroWordEmitted and TestFPUOpmodeWordBeforeEAExtension.
func TestAssembleFPUInstructions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"AddMemorySourceExtended", "FADD.X (A0),FP0\n", []byte{0xF0, 0x10, 0x08, 0x22}},
		{"AddRegisterToRegister", "FADD FP1,FP0\n", []byte{0xF0, 0x00, 0x04, 0x22}},
		{"MoveIntegerLongFromDn", "FMOVE.L D0,FP0\n", []byte{0xF0, 0x00, 0x00, 0x00}},
		{"MoveExtendedToMemory", "FMOVE.X FP2,(A0)\n", []byte{0xF0, 0x10, 0x29, 0x00}},
		{"AbsSingleOperandShorthand", "FABS FP0\n", []byte{0xF0, 0x00, 0x00, 0x18}},
		{"TstMemoryExtended", "FTST.X (A0)\n", []byte{0xF0, 0x10, 0x08, 0x3A}},
		{"Nop", "FNOP\n", []byte{0xF2, 0x80, 0x00, 0x00}},
		{"SubRegisterToRegister", "FSUB FP1,FP0\n", []byte{0xF0, 0x00, 0x04, 0x28}},
		{"MulRegisterToRegister", "FMUL FP1,FP0\n", []byte{0xF0, 0x00, 0x04, 0x23}},
		{"DivRegisterToRegister", "FDIV FP1,FP0\n", []byte{0xF0, 0x00, 0x04, 0x20}},
		{"CmpRegisterToRegister", "FCMP FP1,FP0\n", []byte{0xF0, 0x00, 0x04, 0x38}},
		{"NegSingleOperandShorthand", "FNEG FP0\n", []byte{0xF0, 0x00, 0x00, 0x1A}},
		{"SqrtSingleOperandShorthand", "FSQRT FP0\n", []byte{0xF0, 0x00, 0x00, 0x04}},
		{"AddLongImmediate", "FADD.L #100,FP0\n", []byte{0xF0, 0x3C, 0x00, 0x22, 0x00, 0x00, 0x00, 0x64}},
		{"MoveWordImmediate", "FMOVE.W #5,FP0\n", []byte{0xF0, 0x3C, 0x10, 0x00, 0x00, 0x05}},
		{"AddExtendedWithDisplacement", "FADD.X (100,A0),FP0\n", []byte{0xF0, 0x28, 0x08, 0x22, 0x00, 0x64}},
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

// TestFPUFixedZeroWordEmitted guards a bug found while implementing
// FNOP: encode.go's (and parser_stmt.go's) haveWord heuristic treated
// any step with WordBits==0 and no Fields as "no word here" — correct
// for a step that exists purely to carry a Trailer, but wrong for
// FNOP's second word, a genuinely fixed 0x0000 literal with neither
// Fields nor a Trailer. It was silently dropped (2 bytes instead of 4)
// until the heuristic learned to tell the two cases apart.
func TestFPUFixedZeroWordEmitted(t *testing.T) {
	got := assembleForTarget(t, "FNOP\n", targetFPU)
	if len(got) != 4 {
		t.Fatalf("FNOP produced %d bytes (% X), want 4 (0xF280 0x0000)", len(got), got)
	}
}

// TestFPUOpmodeWordBeforeEAExtension guards a real word-ordering bug:
// the opmode/format/register word must be emitted before the EA's own
// extension words (a displacement or immediate value), not after — the
// first draft of cpu020_fpu.go's Steps had the Trailer ahead of the
// opmode word, which put an immediate or displacement before the opmode
// word instead of after it.
func TestFPUOpmodeWordBeforeEAExtension(t *testing.T) {
	got := assembleForTarget(t, "FADD.X (100,A0),FP0\n", targetFPU)
	want := []byte{0xF0, 0x28, 0x08, 0x22, 0x00, 0x64}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X want % X (opmode word 0822 must precede the displacement word 0064)", got, want)
	}
}

func TestFPUInstructionsRequireFPUFeature(t *testing.T) {
	srcs := []string{"FADD.X (A0),FP0\n", "FMOVE.L D0,FP0\n", "FNOP\n"}
	for _, src := range srcs {
		// 68020 alone (no FeatFPU) must still reject these.
		_, err := asm.ParseWithOptions(strings.NewReader(src), asm.ParseOptions{Target: instructions.Target{CPU: instructions.CPU68020}})
		if err == nil {
			t.Errorf("%q: expected an error without FeatFPU set", src)
		}
	}
}

func TestFPURejectsAddressRegister(t *testing.T) {
	mustAssembleErr(t, "FADD.L A0,FP0\n", targetFPU)
}

func TestFPURejectsFloatImmediate(t *testing.T) {
	// The lexer has no float-literal syntax; ".s" with an immediate must
	// be rejected by Validate (an integer literal would otherwise be
	// silently reinterpreted as if it were a single-precision float).
	mustAssembleErr(t, "FADD.S #1,FP0\n", targetFPU)
}

func TestFPUAllowsIntegerImmediate(t *testing.T) {
	got := assembleForTarget(t, "FADD.L #100,FP0\n", targetFPU)
	if len(got) != 8 {
		t.Fatalf("got %d bytes, want 8", len(got))
	}
}

func TestFPURejectsDnForFloatingSize(t *testing.T) {
	mustAssembleErr(t, "FADD.X D0,FP0\n", targetFPU)
}
