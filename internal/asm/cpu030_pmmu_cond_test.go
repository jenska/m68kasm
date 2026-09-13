package asm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// Expected bytes were derived by hand from GNU binutils' GAS m68k opcode
// table (opcodes/m68k-opc.c) and gas/config/tc-m68k.c's install_operand
// logic, then confirmed against real assembler output before being
// pinned here.
func TestAssemblePMMUConditional(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"PBccWordZeroDisplacement", "PBCC.W here\nhere:\n", []byte{0xF0, 0x8F, 0x00, 0x02}},
		{"PBccLongZeroDisplacement", "PBCC.L here\nhere:\n", []byte{0xF0, 0xCF, 0x00, 0x00, 0x00, 0x04}},
		{"PDBccZeroDisplacement", "PDBCC D0,here\nhere:\n", []byte{0xF0, 0x48, 0x00, 0x0F, 0x00, 0x04}},
		{"PDBccDn3", "PDBCC D3,here\nhere:\n", []byte{0xF0, 0x4B, 0x00, 0x0F, 0x00, 0x04}},
		{"PSccOnD0", "PSCC D0\n", []byte{0xF0, 0x40, 0x00, 0x0F}},
		{"PTrapccBare", "PTRAPCC\n", []byte{0xF0, 0x7C, 0x00, 0x0F}},
		{"PTrapccWordImm", "PTRAPCC.W #5\n", []byte{0xF0, 0x7A, 0x00, 0x0F, 0x00, 0x05}},
		{"PTrapccLongImm", "PTRAPCC.L #5\n", []byte{0xF0, 0x7B, 0x00, 0x0F, 0x00, 0x00, 0x00, 0x05}},
		{"PBbsCondZero", "PBBS.W here\nhere:\n", []byte{0xF0, 0x80, 0x00, 0x02}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assembleForTarget(t, tc.src, targetPMMU)
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("unexpected output: got % X want % X", got, tc.want)
			}
		})
	}
}

func TestPMMUConditionalRequiresPMMUFeature(t *testing.T) {
	srcs := []string{"PBCC.W here\nhere:\n", "PDBCC D0,here\nhere:\n", "PSCC D0\n", "PTRAPCC\n"}
	for _, src := range srcs {
		mustAssembleErr(t, src, instructions.Target{CPU: instructions.CPU68030})
	}
}

func TestPSccRejectsAddressRegister(t *testing.T) {
	mustAssembleErr(t, "PSCC A0\n", targetPMMU)
}

func TestPBccHasNoByteForm(t *testing.T) {
	// Unlike integer Bcc, PBcc has no 8-bit-inline displacement form —
	// GAS's own table has only word and long forms for it.
	_, err := asm.ParseWithOptions(strings.NewReader("PBCC.S here\nhere:\n"), asm.ParseOptions{Target: targetPMMU})
	if err == nil {
		t.Fatalf("expected an error for PBCC.S (no byte-displacement form)")
	}
}

func TestPDBccHasNoLongForm(t *testing.T) {
	_, err := asm.ParseWithOptions(strings.NewReader("PDBCC.L D0,here\nhere:\n"), asm.ParseOptions{Target: targetPMMU})
	if err == nil {
		t.Fatalf("expected an error for PDBCC.L (word displacement only)")
	}
}
