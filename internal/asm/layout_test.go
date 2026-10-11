package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// TestLayoutMatchesEncoding checks that labels after an instruction are
// placed for the bytes the instruction really encodes to. Single-operand
// instructions with extension words and FPU immediates used to be laid out
// too short, so every label after them was wrong.
func TestLayoutMatchesEncoding(t *testing.T) {
	fpu := instructions.Target{CPU: instructions.CPU68020, Features: instructions.FeatFPU}
	for _, tc := range []struct {
		ins    string
		size   int
		target instructions.Target
	}{
		{"TST.W -16(A0)", 4, instructions.Target68000},
		{"TST.W $1234", 6, instructions.Target68000},
		{"CLR.L $12345678", 6, instructions.Target68000},
		{"NEG.W 4(A0)", 4, instructions.Target68000},
		{"NEGX.B 2(A1,D0.W)", 4, instructions.Target68000},
		{"NOT.L 4(A0)", 4, instructions.Target68000},
		{"NBCD 4(A0)", 4, instructions.Target68000},
		{"TAS 4(A0)", 4, instructions.Target68000},
		{"ASL.W 4(A0)", 4, instructions.Target68000},
		{"ROR.W $1234", 6, instructions.Target68000},
		{"SEQ 4(A0)", 4, instructions.Target68000},
		{"PEA $12345678", 6, instructions.Target68000},
		{"JMP 4(A0)", 4, instructions.Target68000},
		{"JSR $12345678", 6, instructions.Target68000},
		{"FMOVE.S #1.5,FP0", 8, fpu},
		{"FMOVE.D #1.5,FP0", 12, fpu},
		{"FMOVE.X #1.5,FP0", 16, fpu},
		{"FADD.L #3,FP1", 8, fpu},
	} {
		forward := assembleForTarget(t, "\tBRA.S l\n\t"+tc.ins+"\nl:\tNOP\n", tc.target)
		if len(forward) != 2+tc.size+2 || forward[1] != byte(tc.size) {
			t.Errorf("%s: BRA.S over it = % X, want displacement %d", tc.ins, forward, tc.size)
		}
		backward := assembleForTarget(t, "l:\t"+tc.ins+"\n\tBRA.S l\n", tc.target)
		if len(backward) != tc.size+2 || backward[tc.size+1] != byte(-(tc.size+2)) {
			t.Errorf("%s: BRA.S back over it = % X, want displacement %d", tc.ins, backward, -(tc.size + 2))
		}
	}
}

// TestSFP004PollingLoop assembles the loop an Atari SFP004 library uses to
// wait for the memory-mapped 68881; the library itself spells it as words.
func TestSFP004PollingLoop(t *testing.T) {
	got := assembleForTarget(t, "wait:\tTST.W -16(A0)\n\tBMI.S wait\n", instructions.Target68000)
	if want := []byte{0x4A, 0x68, 0xFF, 0xF0, 0x6B, 0xFA}; !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}
