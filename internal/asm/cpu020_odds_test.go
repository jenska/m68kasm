package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

var target68030 = instructions.Target{CPU: instructions.CPU68030}

// Expected bytes below were derived by hand from GNU binutils' GAS m68k
// opcode table (opcodes/m68k-opc.c) and then confirmed against the
// actual assembler output before being pinned here.
func TestAssemble68020OddsAndEnds(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"PackRegisters", "PACK D0,D1,#0\n", []byte{0x83, 0x40, 0x00, 0x00}},
		{"PackPredecrement", "PACK -(A0),-(A1),#0\n", []byte{0x83, 0x48, 0x00, 0x00}},
		{"UnpkRegisters", "UNPK D0,D1,#0\n", []byte{0x83, 0x80, 0x00, 0x00}},
		{"CasByteIndirect", "CAS.B D0,D1,(A0)\n", []byte{0x0A, 0xD0, 0x00, 0x40}},
		{"CasLongWithDisplacement", "CAS.L D0,D1,(100,A0)\n", []byte{0x0E, 0xE8, 0x00, 0x40, 0x00, 0x64}},
		{"Callm", "CALLM #5,(A0)\n", []byte{0x06, 0xD0, 0x00, 0x05}},
		{"RtmDataRegister", "RTM D0\n", []byte{0x06, 0xC0}},
		{"RtmAddressRegister", "RTM A2\n", []byte{0x06, 0xCA}},
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

func TestCPU68020OddsRequireTarget(t *testing.T) {
	srcs := []string{"PACK D0,D1,#0\n", "CAS.B D0,D1,(A0)\n", "CALLM #5,(A0)\n", "RTM D0\n"}
	for _, src := range srcs {
		mustAssembleErr(t, src, instructions.Target68000)
	}
}

// TestCallmRtmDroppedFrom68030 is the key proof for requireM68020Only:
// CALLM/RTM are tagged the single m68020 bit in GAS (not the m68020up
// union other 68020 additions use), confirming real 68030+ silicon
// dropped them — they must be rejected on 68030 even though its
// CPUKind value is numerically higher than CPU68020's.
func TestCallmRtmDroppedFrom68030(t *testing.T) {
	mustAssembleErr(t, "CALLM #5,(A0)\n", target68030)
	mustAssembleErr(t, "RTM D0\n", target68030)
}

// TestPackCasKeptOn68030 is the contrasting case: PACK/UNPK/CAS are
// tagged m68020up (kept on every later tier), so — unlike CALLM/RTM —
// they must still work on 68030.
func TestPackCasKeptOn68030(t *testing.T) {
	got := assembleForTarget(t, "PACK D0,D1,#0\n", target68030)
	want := []byte{0x83, 0x40, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X want % X", got, want)
	}
	got = assembleForTarget(t, "CAS.B D0,D1,(A0)\n", target68030)
	want = []byte{0x0A, 0xD0, 0x00, 0x40}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X want % X", got, want)
	}
}

func TestCasRequiresMemoryDestination(t *testing.T) {
	mustAssembleErr(t, "CAS.B D0,D1,D2\n", target68020)
}

func TestCallmRequiresVectorInByteRange(t *testing.T) {
	mustAssembleErr(t, "CALLM #300,(A0)\n", target68020)
}

func TestRtmRequiresRegisterOperand(t *testing.T) {
	mustAssembleErr(t, "RTM #5\n", target68020)
}
