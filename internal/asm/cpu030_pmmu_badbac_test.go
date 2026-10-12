package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// Expected bytes follow the MC68851 User's Manual, PMOVE format 2: bit 9
// (R/W) set transfers the register to <ea>, as for every other PMOVE.
// GAS's opcode table has the direction of BAD/BAC reversed.
func TestAssemblePMOVEBadBac(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"Bad0Load", "PMOVE.W (A0),BAD0\n", []byte{0xF0, 0x10, 0x70, 0x00}},
		{"Bad0Store", "PMOVE.W BAD0,(A0)\n", []byte{0xF0, 0x10, 0x72, 0x00}},
		{"Bad7Load", "PMOVE.W (A0),BAD7\n", []byte{0xF0, 0x10, 0x70, 0x1C}},
		{"Bad7Store", "PMOVE.W BAD7,(A0)\n", []byte{0xF0, 0x10, 0x72, 0x1C}},
		{"Bac0Load", "PMOVE.W (A0),BAC0\n", []byte{0xF0, 0x10, 0x74, 0x00}},
		{"Bac0Store", "PMOVE.W BAC0,(A0)\n", []byte{0xF0, 0x10, 0x76, 0x00}},
		{"Bac3Load", "PMOVE.W (A0),BAC3\n", []byte{0xF0, 0x10, 0x74, 0x0C}},
		{"Bac3Store", "PMOVE.W BAC3,(A0)\n", []byte{0xF0, 0x10, 0x76, 0x0C}},
		{"Bad1Immediate", "PMOVE #$4E71,BAD1\n", []byte{0xF0, 0x3C, 0x70, 0x04, 0x4E, 0x71}},
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

// TestPmoveBadBacDirection guards BAD/BAC's direction bit, which follows
// the MC68851 manual (bit 9 set = store) rather than GAS's reversed table.
func TestPmoveBadBacDirection(t *testing.T) {
	load := assembleForTarget(t, "PMOVE.W (A0),BAD0\n", targetPMMU)
	store := assembleForTarget(t, "PMOVE.W BAD0,(A0)\n", targetPMMU)
	if load[2] != 0x70 {
		t.Fatalf("load word2 high byte = %02X, want 70", load[2])
	}
	if store[2] != 0x72 {
		t.Fatalf("store word2 high byte = %02X, want 72 (bit 9 set = store)", store[2])
	}
}

func TestPmoveBadBacRequirePMMUFeature(t *testing.T) {
	srcs := []string{"PMOVE.W (A0),BAD0\n", "PMOVE.W (A0),BAC0\n"}
	for _, src := range srcs {
		mustAssembleErr(t, src, instructions.Target{CPU: instructions.CPU68030})
	}
}

func TestPmoveBadRejectsInvalidNumber(t *testing.T) {
	mustAssembleErr(t, "PMOVE.W (A0),BAD8\n", targetPMMU)
}

func TestPmoveBadAndBacAreNotInterchangeable(t *testing.T) {
	mustAssembleErr(t, "PMOVE.W BAD0,BAC0\n", targetPMMU)
}
