package asm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
)

// Expected bytes below were derived by hand from GNU binutils' GAS m68k
// opcode table (opcodes/m68k-opc.c) and then confirmed against the
// actual assembler output before being pinned here.
func TestAssemble68020MiscInstructions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"ExtbLong", "EXTB.L D0\n", []byte{0x49, 0xC0}},
		{"TrapNoOperand", "TRAPT\n", []byte{0x50, 0xFC}},
		{"TrapWordOperand", "TRAPT.W #5\n", []byte{0x50, 0xFA, 0x00, 0x05}},
		{"TrapLongOperand", "TRAPT.L #100000\n", []byte{0x50, 0xFB, 0x00, 0x01, 0x86, 0xA0}},
		{"TrapHSMatchesConditionCode4", "TRAPHS\n", []byte{0x54, 0xFC}},
		{"TrapLECondition15", "TRAPLE\n", []byte{0x5F, 0xFC}},
		{"Chk2Byte", "CHK2.B (A0),D1\n", []byte{0x00, 0xD0, 0x18, 0x00}},
		{"Cmp2Word", "CMP2.W (A0),A2\n", []byte{0x02, 0xD0, 0xA0, 0x00}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assembleForTarget(t, tc.src, target68020)
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("unexpected output: got % X want % X", got, tc.want)
			}
		})
	}
}

// TestTrapccWordFormNotSwallowedByBareForm guards against a real bug this
// milestone found and fixed in selectForm (assemble.go): a zero-operand
// form (TRAPcc's bare form) and a same-size operand-bearing form
// (TRAPcc's word-immediate form) share one InstrDef, and selectForm used
// to skip the operand-kind check entirely whenever a form declared zero
// OperKinds — so the bare form matched first and silently discarded the
// "#5" operand. This is the regression test for that fix; the word-form
// byte pattern above (0x50FA, not 0x50FC) is the same assertion, but
// this test names the failure mode explicitly.
func TestTrapccWordFormNotSwallowedByBareForm(t *testing.T) {
	got := assembleForTarget(t, "TRAPT.W #5\n", target68020)
	if len(got) != 4 {
		t.Fatalf("TRAPT.W #5 produced %d bytes (% X), want 4: the bare no-operand form must not have matched", len(got), got)
	}
}

func TestCPU68020MiscInstructionsRequireTarget(t *testing.T) {
	srcs := []string{"EXTB.L D0\n", "TRAPT\n", "CHK2.B (A0),D1\n", "CMP2.W (A0),A2\n"}
	for _, src := range srcs {
		if _, err := asm.Parse(strings.NewReader(src)); err == nil {
			t.Errorf("%q: expected an error on the default (68000) target", src)
		}
	}
}

func TestChk2RequiresControlEA(t *testing.T) {
	mustAssembleErr(t, "CHK2.B #5,D1\n", target68020)
}

func TestChk2RequiresRegisterDestination(t *testing.T) {
	mustAssembleErr(t, "CHK2.B (A0),(A1)\n", target68020)
}

func TestTrapccWordOperandRangeChecked(t *testing.T) {
	mustAssembleErr(t, "TRAPT.W #100000\n", target68020)
}
