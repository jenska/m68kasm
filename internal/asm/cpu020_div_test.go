package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// Expected bytes follow the M68000 Family Programmer's Reference Manual:
// extension word 0qqq sz00 0000 0rrr (Dq in bits 14-12, Dr in bits 2-0,
// s = signed, z = 64-bit dividend). DIVUL/DIVSL divide 32 bits and keep
// the remainder; DIVU.L/DIVS.L with Dr:Dq divide 64 bits.
func TestAssembleDivsl(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"DivslMemorySource", "DIVSL.L (A0),D0:D1\n", []byte{0x4C, 0x50, 0x18, 0x00}},
		{"ShorthandSameRegister", "DIVUL.L D2,D3\n", []byte{0x4C, 0x42, 0x30, 0x03}},
		{"DivulWithDisplacement", "DIVUL.L (100,A0),D4:D5\n", []byte{0x4C, 0x68, 0x50, 0x04, 0x00, 0x64}},
		{"DivsLong64BitDividend", "DIVS.L (A0),D0:D1\n", []byte{0x4C, 0x50, 0x1C, 0x00}},
		{"DivuLong64BitDividend", "DIVU.L (100,A0),D4:D5\n", []byte{0x4C, 0x68, 0x54, 0x04, 0x00, 0x64}},
		{"DivuLong32Bit", "DIVU.L D1,D2\n", []byte{0x4C, 0x41, 0x20, 0x02}},
		{"DivsLongImmediate", "DIVS.L #7,D2\n", []byte{0x4C, 0x7C, 0x28, 0x02, 0x00, 0x00, 0x00, 0x07}},
		{"MuluLong32Bit", "MULU.L D1,D0\n", []byte{0x4C, 0x01, 0x00, 0x00}},
		{"MulsLong64Bit", "MULS.L (A0),D2:D3\n", []byte{0x4C, 0x10, 0x3C, 0x02}},
		{"MuluLongImmediate", "MULU.L #$10000,D4\n", []byte{0x4C, 0x3C, 0x40, 0x00, 0x00, 0x01, 0x00, 0x00}},
		{"ChkLong", "CHK.L (A1),D3\n", []byte{0x47, 0x11}},
		{"ChkLongImmediate", "CHK.L #100000,D0\n", []byte{0x41, 0x3C, 0x00, 0x01, 0x86, 0xA0}},
		{"LinkLong", "LINK.L A6,#-100000\n", []byte{0x48, 0x0E, 0xFF, 0xFE, 0x79, 0x60}},
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
	want := []byte{0x4C, 0x50, 0x18, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X want % X", got, want)
	}
}

func TestDivslRejectsAddressRegisterSource(t *testing.T) {
	mustAssembleErr(t, "DIVSL.L A0,D0:D1\n", target68020)
}

func TestLongMultiplyDivideRequireTarget(t *testing.T) {
	for _, src := range []string{"MULU.L D1,D0\n", "DIVS.L D1,D2:D3\n", "CHK.L D1,D0\n", "LINK.L A6,#-8\n"} {
		mustAssembleErr(t, src, instructions.Target68000)
		mustAssembleErr(t, src, target68010)
	}
}
