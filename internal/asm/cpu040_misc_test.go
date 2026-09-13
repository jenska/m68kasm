package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

var targetM68040 = instructions.Target{CPU: instructions.CPU68040}
var targetM68060 = instructions.Target{CPU: instructions.CPU68060}

// Expected bytes were derived by hand from GNU binutils' GAS m68k opcode
// table (opcodes/m68k-opc.c) and gas/config/tc-m68k.c's install_operand
// logic, then confirmed against real assembler output before being
// pinned here.
func TestAssemble68040Instructions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"Move16RegToReg", "MOVE16 (A0)+,(A1)+\n", []byte{0xF6, 0x20, 0x90, 0x00}},
		{"Move16PostincToAbs", "MOVE16 (A0)+,$1000\n", []byte{0xF6, 0x00, 0x00, 0x00, 0x10, 0x00}},
		{"Move16AbsToPostinc", "MOVE16 $1000,(A0)+\n", []byte{0xF6, 0x08, 0x00, 0x00, 0x10, 0x00}},
		{"Move16IndToAbs", "MOVE16 (A0),$1000\n", []byte{0xF6, 0x10, 0x00, 0x00, 0x10, 0x00}},
		{"Move16AbsToInd", "MOVE16 $1000,(A0)\n", []byte{0xF6, 0x18, 0x00, 0x00, 0x10, 0x00}},
		{"Cinva", "CINVA BC\n", []byte{0xF4, 0xD8}},
		{"CinvlDataCache", "CINVL DC,(A0)\n", []byte{0xF4, 0x48}},
		{"CinvpInstrCache", "CINVP IC,(A3)\n", []byte{0xF4, 0x93}},
		{"Cpusha", "CPUSHA BC\n", []byte{0xF4, 0xF8}},
		{"CpushlDataCache", "CPUSHL DC,(A0)\n", []byte{0xF4, 0x68}},
		{"CpushpInstrCache", "CPUSHP IC,(A3)\n", []byte{0xF4, 0xB3}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assembleForTarget(t, tc.src, targetM68040)
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("unexpected output: got % X want % X", got, tc.want)
			}
		})
	}
}

// TestMove16AbsFormPicksWordByRuntimeMode guards the one-Form design
// for MOVE16's "(An),abs" pairing: word1 (0xF610 vs 0xF618) must be
// picked from which operand is actually (An) at runtime, not from
// which Form matched (both directions share identical OperKinds).
func TestMove16AbsFormPicksWordByRuntimeMode(t *testing.T) {
	indFirst := assembleForTarget(t, "MOVE16 (A0),$1000\n", targetM68040)
	absFirst := assembleForTarget(t, "MOVE16 $1000,(A0)\n", targetM68040)
	if indFirst[0] != 0xF6 || indFirst[1] != 0x10 {
		t.Fatalf("(A0),$1000 word1 = % X, want F6 10", indFirst[:2])
	}
	if absFirst[0] != 0xF6 || absFirst[1] != 0x18 {
		t.Fatalf("$1000,(A0) word1 = % X, want F6 18", absFirst[:2])
	}
}

func Test68040InstructionsWorkOn68060Too(t *testing.T) {
	got := assembleForTarget(t, "MOVE16 (A0)+,(A1)+\n", targetM68060)
	want := []byte{0xF6, 0x20, 0x90, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X want % X", got, want)
	}
}

func Test68040InstructionsRequire68040Floor(t *testing.T) {
	srcs := []string{"MOVE16 (A0)+,(A1)+\n", "CINVA BC\n", "CPUSHA BC\n"}
	for _, src := range srcs {
		mustAssembleErr(t, src, instructions.Target{CPU: instructions.CPU68030})
	}
}

func TestMove16RejectsBothIndirect(t *testing.T) {
	mustAssembleErr(t, "MOVE16 (A0),(A1)\n", targetM68040)
}

func TestCinvlRequiresIndirectAn(t *testing.T) {
	mustAssembleErr(t, "CINVL DC,A0\n", targetM68040)
}

func TestCacheSelectorRejectsUnknownName(t *testing.T) {
	mustAssembleErr(t, "CINVA XX\n", targetM68040)
}
