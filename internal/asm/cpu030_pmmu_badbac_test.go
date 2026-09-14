package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// Expected bytes were derived by hand from GNU binutils' GAS m68k opcode
// table (opcodes/m68k-opc.c) and gas/config/tc-m68k.c's install_operand
// logic (the "case 'X':" register-number/family dispatch), then
// confirmed against real assembler output before being pinned here.
func TestAssemblePMOVEBadBac(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"Bad0Load", "PMOVE.L (A0),BAD0\n", []byte{0xF0, 0x10, 0x72, 0x00}},
		{"Bad0Store", "PMOVE.L BAD0,(A0)\n", []byte{0xF0, 0x10, 0x70, 0x00}},
		{"Bad7Load", "PMOVE.L (A0),BAD7\n", []byte{0xF0, 0x10, 0x72, 0x1C}},
		{"Bad7Store", "PMOVE.L BAD7,(A0)\n", []byte{0xF0, 0x10, 0x70, 0x1C}},
		{"Bac0Load", "PMOVE.L (A0),BAC0\n", []byte{0xF0, 0x10, 0x76, 0x00}},
		{"Bac0Store", "PMOVE.L BAC0,(A0)\n", []byte{0xF0, 0x10, 0x74, 0x00}},
		{"Bac3Load", "PMOVE.L (A0),BAC3\n", []byte{0xF0, 0x10, 0x76, 0x0C}},
		{"Bac3Store", "PMOVE.L BAC3,(A0)\n", []byte{0xF0, 0x10, 0x74, 0x0C}},
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

// TestPmoveBadBacDirectionIsInverted guards the one genuinely unusual
// detail of this milestone: unlike every other PMOVE register (where
// bit 9 set means store), BAD/BAC's bit 9 set means LOAD — confirmed
// directly against GAS's own table rather than assumed symmetric with
// TC/CRP/SRP/etc.
func TestPmoveBadBacDirectionIsInverted(t *testing.T) {
	load := assembleForTarget(t, "PMOVE.L (A0),BAD0\n", targetPMMU)
	store := assembleForTarget(t, "PMOVE.L BAD0,(A0)\n", targetPMMU)
	if load[2] != 0x72 {
		t.Fatalf("load word2 high byte = %02X, want 72 (bit 9 set = load, inverted vs every other PMOVE register)", load[2])
	}
	if store[2] != 0x70 {
		t.Fatalf("store word2 high byte = %02X, want 70", store[2])
	}
}

func TestPmoveBadBacRequirePMMUFeature(t *testing.T) {
	srcs := []string{"PMOVE.L (A0),BAD0\n", "PMOVE.L (A0),BAC0\n"}
	for _, src := range srcs {
		mustAssembleErr(t, src, instructions.Target{CPU: instructions.CPU68030})
	}
}

func TestPmoveBadRejectsInvalidNumber(t *testing.T) {
	mustAssembleErr(t, "PMOVE.L (A0),BAD8\n", targetPMMU)
}

func TestPmoveBadAndBacAreNotInterchangeable(t *testing.T) {
	mustAssembleErr(t, "PMOVE.L BAD0,BAC0\n", targetPMMU)
}
