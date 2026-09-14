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
// actual assembler output before being pinned here. Three real bugs
// were found and fixed while doing that confirmation — see
// TestFPUFixedZeroWordEmitted, TestFPUOpmodeWordBeforeEAExtension, and
// TestFPURMBit.
func TestAssembleFPUInstructions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"AddMemorySourceExtended", "FADD.X (A0),FP0\n", []byte{0xF2, 0x10, 0x48, 0x22}},
		{"AddRegisterToRegister", "FADD FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x22}},
		{"MoveIntegerLongFromDn", "FMOVE.L D0,FP0\n", []byte{0xF2, 0x00, 0x40, 0x00}},
		{"MoveExtendedToMemory", "FMOVE.X FP2,(A0)\n", []byte{0xF2, 0x10, 0x69, 0x00}},
		{"AbsSingleOperandShorthand", "FABS FP0\n", []byte{0xF2, 0x00, 0x00, 0x18}},
		{"TstMemoryExtended", "FTST.X (A0)\n", []byte{0xF2, 0x10, 0x48, 0x3A}},
		{"Nop", "FNOP\n", []byte{0xF2, 0x80, 0x00, 0x00}},
		{"SubRegisterToRegister", "FSUB FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x28}},
		{"MulRegisterToRegister", "FMUL FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x23}},
		{"DivRegisterToRegister", "FDIV FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x20}},
		{"CmpRegisterToRegister", "FCMP FP1,FP0\n", []byte{0xF2, 0x00, 0x04, 0x38}},
		{"NegSingleOperandShorthand", "FNEG FP0\n", []byte{0xF2, 0x00, 0x00, 0x1A}},
		{"SqrtSingleOperandShorthand", "FSQRT FP0\n", []byte{0xF2, 0x00, 0x00, 0x04}},
		{"AddLongImmediate", "FADD.L #100,FP0\n", []byte{0xF2, 0x3C, 0x40, 0x22, 0x00, 0x00, 0x00, 0x64}},
		{"MoveWordImmediate", "FMOVE.W #5,FP0\n", []byte{0xF2, 0x3C, 0x50, 0x00, 0x00, 0x05}},
		{"AddExtendedWithDisplacement", "FADD.X (100,A0),FP0\n", []byte{0xF2, 0x28, 0x48, 0x22, 0x00, 0x64}},
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
	want := []byte{0xF2, 0x28, 0x48, 0x22, 0x00, 0x64}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X want % X (opmode word 4822 must precede the displacement word 0064)", got, want)
	}
}

// TestFPURMBit guards a real, severe encoding bug found while
// researching packed BCD's own format code (milestone 28): every FPU
// "general instruction" word2's bit 14 (the R/M bit — see fpRMBit's own
// doc comment in encode.go) was never set for an <ea>-sourced operand,
// across every instruction using FFPFormat (FADD/FSUB/FMUL/FDIV/FCMP,
// FMOVE's load and store directions, FABS/FNEG/FSQRT, FTST, every
// transcendental function, and FSINCOS). This wasn't a cosmetic
// difference: a word2 with R/M=0 matches the *register-to-register*
// form's own fixed-bit pattern instead, so real 68881 hardware (or any
// opcode-table-driven disassembler) would have silently read the wrong
// source register and ignored the <ea>/extension words entirely, not
// merely disagreed on a don't-care bit.
func TestFPURMBit(t *testing.T) {
	got := assembleForTarget(t, "FADD.X (A0),FP0\n", targetFPU)
	word2 := uint16(got[2])<<8 | uint16(got[3])
	if word2&0x4000 == 0 {
		t.Fatalf("word2 = %04X, want bit 14 (R/M) set for a memory-sourced operand", word2)
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

// TestFPUAllowsIntegerImmediateAgainstFloatSize guards the milestone-27
// gap closed once float literals became real: before, an INTEGER
// literal against a floating-point size ("FADD.S #1,FP0") was *also*
// rejected, not just a genuinely fractional one — see
// cpu020_fpu_floatimm_test.go for the full float-literal coverage this
// change enabled.
func TestFPUAllowsIntegerImmediateAgainstFloatSize(t *testing.T) {
	got := assembleForTarget(t, "FADD.S #1,FP0\n", targetFPU)
	if len(got) != 8 {
		t.Fatalf("got %d bytes, want 8", len(got))
	}
}

// TestFPURejectsFractionalImmediateAgainstIntegerSize guards the
// converse: a genuine floating-point literal still can't be used with
// an integer size (.b/.w/.l), since a fraction has no integer
// representation.
func TestFPURejectsFractionalImmediateAgainstIntegerSize(t *testing.T) {
	mustAssembleErr(t, "FADD.L #1.5,FP0\n", targetFPU)
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

// TestFPUCoprocessorIDBit guards a real encoding bug found while
// researching FBcc's condition-code layout: every FPU instruction's
// word1 must carry coprocessor ID 1 in bits 11-9 (0x0200), not 0 — GAS's
// tc-m68k.c always synthesizes an implicit COP1 operand for plain float
// mnemonics (m68k_ip's "fake a first entry of type COP#1"), and its
// disassembler treats cpid=1 as the silent default, printing "(cpid=N)"
// only when it differs. This was originally shipped as a bare 0xF000
// base (cpid=0) — see fpuWord1Base's doc comment in cpu020_fpu.go.
func TestFPUCoprocessorIDBit(t *testing.T) {
	got := assembleForTarget(t, "FADD.X (A0),FP0\n", targetFPU)
	if len(got) < 1 || got[0] != 0xF2 {
		t.Fatalf("word1 high byte = %02X, want F2 (coprocessor ID 1 in bits 11-9)", got[0])
	}
}
