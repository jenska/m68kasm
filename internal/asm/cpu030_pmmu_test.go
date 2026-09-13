package asm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
	"github.com/jenska/m68kasm/internal/asm/instructions"
)

var targetPMMU = instructions.Target{CPU: instructions.CPU68030, Features: instructions.FeatPMMU}

// Expected bytes were derived by hand from GNU binutils' GAS m68k opcode
// table (opcodes/m68k-opc.c) and gas/config/tc-m68k.c's install_operand
// logic for the PMMU control-register cases, then confirmed against real
// assembler output before being pinned here. Only PMOVE's TC-register
// form and PFLUSHA are implemented — see cpu030_pmmu.go's doc comment
// for why the rest of the PMMU/PMOVE/PFLUSH/Pcc family is out of scope.
func TestAssemblePMMUInstructions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"PmoveLoadTCFromAnIndirect", "PMOVE.L (A0),TC\n", []byte{0xF0, 0x10, 0x40, 0x00}},
		{"PmoveStoreTCToAnIndirect", "PMOVE.L TC,(A0)\n", []byte{0xF0, 0x10, 0x42, 0x00}},
		{"PmoveLoadTCFromAbsLong", "PMOVE.L $1000,TC\n", []byte{0xF0, 0x39, 0x40, 0x00, 0x00, 0x00, 0x10, 0x00}},
		{"Pflusha", "PFLUSHA\n", []byte{0xF0, 0x00, 0x24, 0x00}},
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

// TestPmoveStoreDirectionBit guards the load/store direction bit itself
// (bit 9 of word2: 0x4000 for load, 0x4200 for store) rather than just
// the two forms happening to differ somewhere.
func TestPmoveStoreDirectionBit(t *testing.T) {
	load := assembleForTarget(t, "PMOVE.L (A0),TC\n", targetPMMU)
	store := assembleForTarget(t, "PMOVE.L TC,(A0)\n", targetPMMU)
	if load[2] != 0x40 || load[3] != 0x00 {
		t.Fatalf("load word2 = % X, want 4000", load[2:4])
	}
	if store[2] != 0x42 || store[3] != 0x00 {
		t.Fatalf("store word2 = % X, want 4200", store[2:4])
	}
}

func TestPMMUInstructionsRequirePMMUFeature(t *testing.T) {
	srcs := []string{"PMOVE.L (A0),TC\n", "PMOVE.L TC,(A0)\n", "PFLUSHA\n"}
	for _, src := range srcs {
		// 68030 alone (no FeatPMMU) must still reject these.
		mustAssembleErr(t, src, instructions.Target{CPU: instructions.CPU68030})
	}
}

func TestPMMUInstructionsWorkOnAnyCPUTier(t *testing.T) {
	// PMMU gating is feature-only, like FPU: a bare 68000 with FeatPMMU
	// set (e.g. an external 68851) must assemble PMOVE/PFLUSHA fine.
	target := instructions.Target{CPU: instructions.CPU68000, Features: instructions.FeatPMMU}
	got := assembleForTarget(t, "PFLUSHA\n", target)
	want := []byte{0xF0, 0x00, 0x24, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X want % X", got, want)
	}
}

func TestPmoveRejectsAddressRegisterSource(t *testing.T) {
	mustAssembleErr(t, "PMOVE.L A0,TC\n", targetPMMU)
}

func TestPmoveRejectsAddressRegisterDest(t *testing.T) {
	mustAssembleErr(t, "PMOVE.L TC,A0\n", targetPMMU)
}

func TestPmoveRejectsImmediateDest(t *testing.T) {
	mustAssembleErr(t, "PMOVE.L TC,#1\n", targetPMMU)
}

func TestPflushaRejectsOperands(t *testing.T) {
	mustAssembleErr(t, "PFLUSHA D0\n", targetPMMU)
}

func TestPmoveOnlySupportsLongSize(t *testing.T) {
	mustAssembleErr(t, "PMOVE.W (A0),TC\n", targetPMMU)
}

func TestPmoveRejectsUnknownSecondRegister(t *testing.T) {
	// TC, CRP, SRP, TT0, TT1, MMUSR/PSR, DRP, CAL, VAL, SCC, AC, and
	// PCSR are all implemented now (see cpu030_pmmu2.go, cpu030_pmmu3.go);
	// BAD/BAC (8 numbered instances each) remain deliberately deferred
	// — see cpu030_pmmu3.go's header comment — and must still fail
	// cleanly rather than silently mis-assemble.
	_, err := asm.ParseWithOptions(strings.NewReader("PMOVE.L (A0),BAD0\n"), asm.ParseOptions{Target: targetPMMU})
	if err == nil {
		t.Fatalf("expected an error for unsupported PMOVE register BAD0")
	}
}
