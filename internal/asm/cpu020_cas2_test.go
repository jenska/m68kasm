package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// Expected bytes below were derived by hand from GNU binutils' GAS m68k
// opcode table (opcodes/m68k-opc.c) and its install_operand logic
// (gas/config/tc-m68k.c — including the "case '6'" comment that first
// confirmed CAS2 is three words, not two), then confirmed against the
// actual assembler output before being pinned here.
func TestAssembleCas2(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"Word", "CAS2.W D0:D1,D2:D3,(A0):(A1)\n", []byte{0x0C, 0xFC, 0x00, 0x80, 0x10, 0xC1}},
		{"Long", "CAS2.L D4:D5,D6:D7,(A2):(A3)\n", []byte{0x0E, 0xFC, 0x21, 0x84, 0x31, 0xC5}},
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

func TestCas2RequiresTarget(t *testing.T) {
	src := "CAS2.W D0:D1,D2:D3,(A0):(A1)\n"
	mustAssembleErr(t, src, instructions.Target68000)
	mustAssembleErr(t, src, target68010)
}

// TestCas2NotOnCPU32 is the contrasting case to milestone 7's Bcc.L/
// CHK2/EXTB/TRAPcc and milestone 10's DIVSL/DIVUL: GAS tags CAS2 plain
// m68020up, without cpu32, unlike its single-location sibling CAS
// (milestone 8) — this must stay rejected on a CPU32 target.
func TestCas2NotOnCPU32(t *testing.T) {
	mustAssembleErr(t, "CAS2.W D0:D1,D2:D3,(A0):(A1)\n", targetCPU32)
}

func TestCas2KeptOn68030(t *testing.T) {
	got := assembleForTarget(t, "CAS2.W D0:D1,D2:D3,(A0):(A1)\n", target68030)
	want := []byte{0x0C, 0xFC, 0x00, 0x80, 0x10, 0xC1}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X want % X", got, want)
	}
}

func TestCas2RequiresComparePairColon(t *testing.T) {
	mustAssembleErr(t, "CAS2.W D0,D2:D3,(A0):(A1)\n", target68020)
}

func TestCas2RequiresUpdatePairColon(t *testing.T) {
	mustAssembleErr(t, "CAS2.W D0:D1,D2,(A0):(A1)\n", target68020)
}
