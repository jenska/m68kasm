package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// Expected bytes were derived by hand from GNU binutils' GAS m68k opcode
// table (opcodes/m68k-opc.c) and gas/config/tc-m68k.c's install_operand
// logic (the "case 'W':" and "case '3':" register-selector dispatches),
// then confirmed against real assembler output before being pinned here.
func TestAssemblePMOVEOtherRegisters(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"CrpLoad", "PMOVE.L (A0),CRP\n", []byte{0xF0, 0x10, 0x4C, 0x00}},
		{"CrpStore", "PMOVE.L CRP,(A0)\n", []byte{0xF0, 0x10, 0x4E, 0x00}},
		{"SrpLoad", "PMOVE.L (A0),SRP\n", []byte{0xF0, 0x10, 0x48, 0x00}},
		{"SrpStore", "PMOVE.L SRP,(A0)\n", []byte{0xF0, 0x10, 0x4A, 0x00}},
		{"Tt0Load", "PMOVE.L (A0),TT0\n", []byte{0xF0, 0x10, 0x08, 0x00}},
		{"Tt0Store", "PMOVE.L TT0,(A0)\n", []byte{0xF0, 0x10, 0x0A, 0x00}},
		{"Tt1Load", "PMOVE.L (A0),TT1\n", []byte{0xF0, 0x10, 0x0C, 0x00}},
		{"Tt1Store", "PMOVE.L TT1,(A0)\n", []byte{0xF0, 0x10, 0x0E, 0x00}},
		{"MmusrLoad", "PMOVE.W (A0),MMUSR\n", []byte{0xF0, 0x10, 0x60, 0x00}},
		{"MmusrStore", "PMOVE.W MMUSR,(A0)\n", []byte{0xF0, 0x10, 0x62, 0x00}},
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

// TestPmoveCrpSrpRejectDn guards the deliberate, stricter EA
// restriction CRP/SRP use compared to TC/TT0/TT1/MMUSR: GAS's own
// argument-type letters for these two rows ('|' load-side, '~'
// store-side) exclude Dn/An/immediate entirely — CRP/SRP are 64-bit
// "quad word" registers, and GAS's own source comments that it doesn't
// even support an immediate operand for that width.
func TestPmoveCrpSrpRejectDn(t *testing.T) {
	mustAssembleErr(t, "PMOVE.L D0,CRP\n", targetPMMU)
	mustAssembleErr(t, "PMOVE.L CRP,D0\n", targetPMMU)
	mustAssembleErr(t, "PMOVE.L D0,SRP\n", targetPMMU)
	mustAssembleErr(t, "PMOVE.L SRP,D0\n", targetPMMU)
}

// TestPmoveTT0AllowsDn guards the other side of that same distinction:
// TT0/TT1/MMUSR are ordinary-width registers and share TC's own
// broader restriction (readableDataEA/dataAlterableEA), which does
// include Dn.
func TestPmoveTT0AllowsDn(t *testing.T) {
	got := assembleForTarget(t, "PMOVE.L D0,TT0\n", targetPMMU)
	if len(got) != 4 {
		t.Fatalf("got %d bytes, want 4", len(got))
	}
}

// TestPmoveMmusrIsWordSized guards MMUSR's distinct size: GAS's own
// row uses a 'w' size hint (unlike TC/CRP/SRP/TT0/TT1's 'l'), matching
// MMUSR being a 16-bit status register on real hardware.
func TestPmoveMmusrIsWordSized(t *testing.T) {
	mustAssembleErr(t, "PMOVE.L (A0),MMUSR\n", targetPMMU)
	got := assembleForTarget(t, "PMOVE.W (A0),MMUSR\n", targetPMMU)
	if len(got) != 4 {
		t.Fatalf("got %d bytes, want 4", len(got))
	}
}

func TestPmoveOtherRegistersRequirePMMUFeature(t *testing.T) {
	srcs := []string{"PMOVE.L (A0),CRP\n", "PMOVE.L (A0),SRP\n", "PMOVE.L (A0),TT0\n", "PMOVE.L (A0),TT1\n", "PMOVE.W (A0),MMUSR\n"}
	for _, src := range srcs {
		mustAssembleErr(t, src, instructions.Target{CPU: instructions.CPU68030})
	}
}
