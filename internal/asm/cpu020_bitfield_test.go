package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// Expected bytes below were derived by hand from GNU binutils' GAS m68k
// opcode table (opcodes/m68k-opc.c, for the per-instruction opcode word)
// and gas/config/tc-m68k.c's 'O' argument case (for the offset/width
// field format), then confirmed against the actual assembler output
// before being pinned here.
func TestAssembleBitFieldInstructions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"TstImmediateOffsetWidth", "BFTST D0{0:8}\n", []byte{0xE8, 0xC0, 0x00, 0x08}},
		{"TstRegisterOffsetWidth", "BFTST D0{D1:D2}\n", []byte{0xE8, 0xC0, 0x08, 0x62}},
		{"ExtuMemoryBase", "BFEXTU (A0){0:8},D3\n", []byte{0xE9, 0xD0, 0x30, 0x08}},
		{"InsFromDataRegister", "BFINS D3,(A0){0:8}\n", []byte{0xEF, 0xD0, 0x30, 0x08}},
		{"FfoWidth32WrapsToZero", "BFFFO D0{0:32},D1\n", []byte{0xED, 0xC0, 0x10, 0x00}},
		{"ClrWithDisplacementExtension", "BFCLR (100,A0){8:16}\n", []byte{0xEC, 0xE8, 0x02, 0x10, 0x00, 0x64}},
		{"SetSingleBit", "BFSET D0{0:1}\n", []byte{0xEE, 0xC0, 0x00, 0x01}},
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

func TestBitFieldInstructionsRequire68020(t *testing.T) {
	mustAssembleErr(t, "BFTST D0{0:8}\n", instructions.Target68000)
}

// TestBitFieldKeptOn68030 confirms BFxxx are tagged m68020up in GAS
// (kept on every later tier), unlike milestone 8's CALLM/RTM.
func TestBitFieldKeptOn68030(t *testing.T) {
	got := assembleForTarget(t, "BFTST D0{0:8}\n", target68030)
	want := []byte{0xE8, 0xC0, 0x00, 0x08}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X want % X", got, want)
	}
}

func TestBitFieldRejectsAutoIncrementBase(t *testing.T) {
	mustAssembleErr(t, "BFTST -(A0){0:8}\n", target68020)
	mustAssembleErr(t, "BFTST (A0)+{0:8}\n", target68020)
}

func TestBitFieldRequiresSpecifier(t *testing.T) {
	mustAssembleErr(t, "BFTST D0\n", target68020)
}

func TestBitFieldOffsetRangeChecked(t *testing.T) {
	mustAssembleErr(t, "BFTST D0{40:8}\n", target68020)
}

func TestBitFieldWidthRangeChecked(t *testing.T) {
	mustAssembleErr(t, "BFTST D0{0:0}\n", target68020)
	mustAssembleErr(t, "BFTST D0{0:33}\n", target68020)
}
