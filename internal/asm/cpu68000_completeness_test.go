package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// TestConditionCodeCCandCS covers the CC/CS condition names the Motorola
// manuals use, alongside their HS/LO aliases.
func TestConditionCodeCCandCS(t *testing.T) {
	for _, tc := range []struct {
		src  string
		want []byte
	}{
		{"BCC target\n.WORD 0\ntarget:\n", []byte{0x64, 0x02, 0x00, 0x00}},
		{"BCS.W target\n.WORD 0\ntarget:\n", []byte{0x65, 0x00, 0x00, 0x04, 0x00, 0x00}},
		{"loop:\nDBCC D0,loop\n", []byte{0x54, 0xC8, 0xFF, 0xFE}},
		{"loop:\nDBCS D1,loop\n", []byte{0x55, 0xC9, 0xFF, 0xFE}},
		{"SCC D0\n", []byte{0x54, 0xC0}},
		{"SCS (A1)\n", []byte{0x55, 0xD1}},
	} {
		if got := assembleForTarget(t, tc.src, instructions.Target68000); !bytes.Equal(got, tc.want) {
			t.Errorf("%q: got % X want % X", tc.src, got, tc.want)
		}
	}
	for _, tc := range []struct {
		src  string
		want []byte
	}{
		{"TRAPCC\n", []byte{0x54, 0xFC}},
		{"TRAPCS\n", []byte{0x55, 0xFC}},
	} {
		if got := assembleForTarget(t, tc.src, instructions.Target{CPU: instructions.CPU68020}); !bytes.Equal(got, tc.want) {
			t.Errorf("%q: got % X want % X", tc.src, got, tc.want)
		}
	}
}

// TestTSTAddressingByCPU checks that TST takes PC-relative, immediate and
// address register operands only from the CPU32/68020 on.
func TestTSTAddressingByCPU(t *testing.T) {
	cpu020 := instructions.Target{CPU: instructions.CPU68020}
	for _, tc := range []struct {
		src  string
		want []byte
	}{
		{"TST.W (4,PC)\n", []byte{0x4A, 0x7A, 0x00, 0x04}},
		{"TST.L #$12345678\n", []byte{0x4A, 0xBC, 0x12, 0x34, 0x56, 0x78}},
		{"TST.B #5\n", []byte{0x4A, 0x3C, 0x00, 0x05}},
		{"TST.W A1\n", []byte{0x4A, 0x49}},
		{"TST.L D3\n", []byte{0x4A, 0x83}},
	} {
		if got := assembleForTarget(t, tc.src, cpu020); !bytes.Equal(got, tc.want) {
			t.Errorf("%q on 68020: got % X want % X", tc.src, got, tc.want)
		}
	}
	for _, src := range []string{"TST.W (4,PC)\n", "TST.L #1\n", "TST.W A1\n"} {
		mustAssembleErr(t, src, instructions.Target68000)
	}
	mustAssembleErr(t, "TST.B A1\n", cpu020)
}

// TestMOVECControlRegistersByCPU checks MOVEC's control registers against
// the CPUs that have them (MC68020/030/040/060 user's manuals).
func TestMOVECControlRegistersByCPU(t *testing.T) {
	cpus := map[string]instructions.CPUKind{
		"68010": instructions.CPU68010, "cpu32": instructions.CPU32, "68020": instructions.CPU68020,
		"68030": instructions.CPU68030, "68040": instructions.CPU68040, "68060": instructions.CPU68060,
	}
	for _, tc := range []struct {
		reg  string
		sel  uint16
		cpus []string
	}{
		{"VBR", 0x801, []string{"68010", "cpu32", "68020", "68030", "68040", "68060"}},
		{"CACR", 0x002, []string{"68020", "68030", "68040", "68060"}},
		{"CAAR", 0x802, []string{"68020", "68030"}},
		{"MSP", 0x803, []string{"68020", "68030", "68040"}},
		{"ISP", 0x804, []string{"68020", "68030", "68040"}},
		{"TC", 0x003, []string{"68040", "68060"}},
		{"ITT0", 0x004, []string{"68040", "68060"}},
		{"DTT1", 0x007, []string{"68040", "68060"}},
		{"MMUSR", 0x805, []string{"68040"}},
		{"URP", 0x806, []string{"68040", "68060"}},
		{"SRP", 0x807, []string{"68040", "68060"}},
		{"BUSCR", 0x008, []string{"68060"}},
		{"PCR", 0x808, []string{"68060"}},
	} {
		for name, cpu := range cpus {
			target := instructions.Target{CPU: cpu}
			has := false
			for _, c := range tc.cpus {
				has = has || c == name
			}
			load := "MOVEC " + tc.reg + ",D1\n"
			store := "MOVEC A2," + tc.reg + "\n"
			if !has {
				mustAssembleErr(t, load, target)
				mustAssembleErr(t, store, target)
				continue
			}
			wantLoad := []byte{0x4E, 0x7A, 0x10 | byte(tc.sel>>8), byte(tc.sel)}
			wantStore := []byte{0x4E, 0x7B, 0xA0 | byte(tc.sel>>8), byte(tc.sel)}
			if got := assembleForTarget(t, load, target); !bytes.Equal(got, wantLoad) {
				t.Errorf("%s on %s: got % X want % X", load, name, got, wantLoad)
			}
			if got := assembleForTarget(t, store, target); !bytes.Equal(got, wantStore) {
				t.Errorf("%s on %s: got % X want % X", store, name, got, wantStore)
			}
		}
	}
}

// TestFPUSingleInDataRegister checks that a single-precision operand may sit
// in a data register, like the integer formats, while .D/.X/.P need memory.
func TestFPUSingleInDataRegister(t *testing.T) {
	for _, tc := range []struct {
		src  string
		want []byte
	}{
		{"FMOVE.S D0,FP0\n", []byte{0xF2, 0x00, 0x44, 0x00}},
		{"FADD.S D3,FP2\n", []byte{0xF2, 0x03, 0x45, 0x22}},
		{"FMOVE.S FP1,D2\n", []byte{0xF2, 0x02, 0x64, 0x80}},
		{"FMOVE.L D1,FP0\n", []byte{0xF2, 0x01, 0x40, 0x00}},
	} {
		if got := assembleForTarget(t, tc.src, targetFPU); !bytes.Equal(got, tc.want) {
			t.Errorf("%q: got % X want % X", tc.src, got, tc.want)
		}
	}
	for _, src := range []string{"FMOVE.D D0,FP0\n", "FADD.X D1,FP0\n", "FMOVE.P FP0,D1\n"} {
		mustAssembleErr(t, src, targetFPU)
	}
}

// TestDBccDisplacementRelativeToExtensionWord pins DBcc's displacement to
// the address of its extension word (PC+2): a target right after that word
// is displacement 0, a loop back onto the instruction itself is -2.
func TestDBccDisplacementRelativeToExtensionWord(t *testing.T) {
	for _, tc := range []struct {
		src  string
		want []byte
	}{
		{"ORG $1000\nDBRA D0,$1002\n", []byte{0x51, 0xC8, 0x00, 0x00}},
		{"ORG $1000\nDBRA D0,$1000\n", []byte{0x51, 0xC8, 0xFF, 0xFE}},
		{"ORG $1000\nDBNE D3,$1010\n", []byte{0x56, 0xCB, 0x00, 0x0E}},
		{"loop:\nDBRA D0,loop\n", []byte{0x51, 0xC8, 0xFF, 0xFE}},
	} {
		if got := assembleForTarget(t, tc.src, instructions.Target68000); !bytes.Equal(got, tc.want) {
			t.Errorf("%q: got % X want % X", tc.src, got, tc.want)
		}
	}
}

// TestLateIntegerForms covers MOVE CCR,<ea> (68010+), CMPI with a
// PC-relative destination (CPU32/68020+) and CHK2/CMP2's word order: the
// register word directly follows the opcode, before the <ea> extension.
func TestLateIntegerForms(t *testing.T) {
	for _, tc := range []struct {
		src    string
		target instructions.Target
		want   []byte
	}{
		{"MOVE CCR,D3\n", target68010, []byte{0x42, 0xC3}},
		{"MOVE CCR,(4,A1)\n", target68010, []byte{0x42, 0xE9, 0x00, 0x04}},
		{"CMPI.W #5,(8,PC)\n", target68020, []byte{0x0C, 0x7A, 0x00, 0x05, 0x00, 0x08}},
		{"CMPI.B #1,D0\n", target68020, []byte{0x0C, 0x00, 0x00, 0x01}},
		{"CHK2.L (4,A1),A2\n", target68020, []byte{0x04, 0xE9, 0xA8, 0x00, 0x00, 0x04}},
		{"CMP2.W ($1234).W,D0\n", target68020, []byte{0x02, 0xF8, 0x00, 0x00, 0x12, 0x34}},
	} {
		if got := assembleForTarget(t, tc.src, tc.target); !bytes.Equal(got, tc.want) {
			t.Errorf("%q: got % X want % X", tc.src, got, tc.want)
		}
	}
	mustAssembleErr(t, "MOVE CCR,D0\n", instructions.Target68000)
	mustAssembleErr(t, "CMPI.W #5,(8,PC)\n", instructions.Target68000)
	mustAssembleErr(t, "CMPI.W #5,#6\n", target68020)
}

func TestCoprocessorRestoreFromPCRelative(t *testing.T) {
	if got, want := assembleForTarget(t, "FRESTORE (4,PC)\n", targetFPU), []byte{0xF3, 0x7A, 0x00, 0x04}; !bytes.Equal(got, want) {
		t.Errorf("FRESTORE (4,PC): got % X want % X", got, want)
	}
	mustAssembleErr(t, "FSAVE (4,PC)\n", targetFPU)
}
