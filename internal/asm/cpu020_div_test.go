package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// Expected bytes below were derived by hand from GNU binutils' GAS m68k
// opcode table (opcodes/m68k-opc.c) and its install_operand/
// print_insn_arg logic (which register bit position is Dq vs Dr — not
// settled by the opcode table's mask alone), then confirmed against the
// actual assembler output before being pinned here.
func TestAssembleDivsl(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"WideFormMemorySource", "DIVSL.L (A0),D0:D1\n", []byte{0x4C, 0x50, 0x1C, 0x00}},
		{"ShorthandSameRegister", "DIVUL.L D2,D3\n", []byte{0x4C, 0x42, 0x30, 0x03}},
		{"WideFormWithDisplacement", "DIVUL.L (100,A0),D4:D5\n", []byte{0x4C, 0x68, 0x54, 0x04, 0x00, 0x64}},
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

func TestDivslRequiresTarget(t *testing.T) {
	src := "DIVSL.L (A0),D0:D1\n"
	mustAssembleErr(t, src, instructions.Target68000)
	mustAssembleErr(t, src, target68010)
}

// TestDivslKeptOnCPU32 confirms DIVSL/DIVUL are tagged m68020up|cpu32 in
// GAS (like milestone 7's Bcc.L/CHK2/EXTB/TRAPcc), unlike milestone 8's
// CALLM/RTM.
func TestDivslKeptOnCPU32(t *testing.T) {
	got := assembleForTarget(t, "DIVSL.L (A0),D0:D1\n", targetCPU32)
	want := []byte{0x4C, 0x50, 0x1C, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X want % X", got, want)
	}
}

func TestDivslRejectsAddressRegisterSource(t *testing.T) {
	mustAssembleErr(t, "DIVSL.L A0,D0:D1\n", target68020)
}
