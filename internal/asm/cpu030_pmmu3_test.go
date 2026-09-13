package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// Expected bytes were derived by hand from GNU binutils' GAS m68k opcode
// table (opcodes/m68k-opc.c) and gas/config/tc-m68k.c's install_operand
// logic (the "case '2':" and "case '1':" register-selector dispatches),
// then confirmed against real assembler output before being pinned here.
func TestAssemblePMOVERemainingRegisters(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"DrpLoad", "PMOVE.L (A0),DRP\n", []byte{0xF0, 0x10, 0x44, 0x00}},
		{"DrpStore", "PMOVE.L DRP,(A0)\n", []byte{0xF0, 0x10, 0x46, 0x00}},
		{"CalLoad", "PMOVE.B (A0),CAL\n", []byte{0xF0, 0x10, 0x50, 0x00}},
		{"CalStore", "PMOVE.B CAL,(A0)\n", []byte{0xF0, 0x10, 0x52, 0x00}},
		{"ValLoad", "PMOVE.B (A0),VAL\n", []byte{0xF0, 0x10, 0x54, 0x00}},
		{"ValStore", "PMOVE.B VAL,(A0)\n", []byte{0xF0, 0x10, 0x56, 0x00}},
		{"SccLoad", "PMOVE.B (A0),SCC\n", []byte{0xF0, 0x10, 0x58, 0x00}},
		{"SccStore", "PMOVE.B SCC,(A0)\n", []byte{0xF0, 0x10, 0x5A, 0x00}},
		{"AcLoad", "PMOVE.W (A0),AC\n", []byte{0xF0, 0x10, 0x5C, 0x00}},
		{"AcStore", "PMOVE.W AC,(A0)\n", []byte{0xF0, 0x10, 0x5E, 0x00}},
		{"PsrAliasForMmusr", "PMOVE.W (A0),PSR\n", []byte{0xF0, 0x10, 0x60, 0x00}},
		{"PcsrStore", "PMOVE.W PCSR,(A0)\n", []byte{0xF0, 0x10, 0x66, 0x00}},
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

// TestPsrAndMmusrProduceIdenticalBytes guards the deliberate aliasing:
// "PSR" (the 68851's own name) and "MMUSR" (the 68030's name for the
// exact same register) must produce byte-identical output, since
// they're the same real hardware register under two names, not two
// different Forms.
func TestPsrAndMmusrProduceIdenticalBytes(t *testing.T) {
	psr := assembleForTarget(t, "PMOVE.W (A0),PSR\n", targetPMMU)
	mmusr := assembleForTarget(t, "PMOVE.W (A0),MMUSR\n", targetPMMU)
	if !bytes.Equal(psr, mmusr) {
		t.Fatalf("PSR and MMUSR produced different bytes: % X vs % X", psr, mmusr)
	}
}

func TestPmoveDrpRejectsDn(t *testing.T) {
	mustAssembleErr(t, "PMOVE.L D0,DRP\n", targetPMMU)
}

func TestPmoveCalAllowsDn(t *testing.T) {
	// CAL/VAL/SCC/AC share TC's own broader restriction, unlike
	// DRP/CRP/SRP's memory-only one.
	got := assembleForTarget(t, "PMOVE.B D0,CAL\n", targetPMMU)
	if len(got) != 4 {
		t.Fatalf("got %d bytes, want 4", len(got))
	}
}

// TestPcsrIsStoreOnly guards GAS's own table having no load-direction
// row for PCSR at all.
func TestPcsrIsStoreOnly(t *testing.T) {
	mustAssembleErr(t, "PMOVE.W (A0),PCSR\n", targetPMMU)
}

func TestPmoveRemainingRegistersRequirePMMUFeature(t *testing.T) {
	srcs := []string{"PMOVE.L (A0),DRP\n", "PMOVE.B (A0),CAL\n", "PMOVE.W (A0),AC\n", "PMOVE.W PCSR,(A0)\n"}
	for _, src := range srcs {
		mustAssembleErr(t, src, instructions.Target{CPU: instructions.CPU68030})
	}
}
