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
// pinned here — including the coprocessor-ID bit (see fpuWord1Base's
// doc comment in cpu020_fpu.go) every one of these shares with the rest
// of the FPU instruction set.
func TestAssembleFPUConditionalAndMisc(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"FBeqWordZeroDisplacement", "FBEQ.W here\nhere:\n", []byte{0xF2, 0x81, 0x00, 0x02}},
		{"FBeqLongZeroDisplacement", "FBEQ.L here\nhere:\n", []byte{0xF2, 0xC1, 0x00, 0x00, 0x00, 0x04}},
		{"FDBeqZeroDisplacement", "FDBEQ D0,here\nhere:\n", []byte{0xF2, 0x48, 0x00, 0x01, 0x00, 0x04}},
		{"FDBeqDn3", "FDBEQ D3,here\nhere:\n", []byte{0xF2, 0x4B, 0x00, 0x01, 0x00, 0x04}},
		{"FSeqOnD0", "FSEQ D0\n", []byte{0xF2, 0x40, 0x00, 0x01}},
		{"FTrapeqBare", "FTRAPEQ\n", []byte{0xF2, 0x7C, 0x00, 0x01}},
		{"FTrapeqWordImm", "FTRAPEQ.W #5\n", []byte{0xF2, 0x7A, 0x00, 0x01, 0x00, 0x05}},
		{"FTrapeqLongImm", "FTRAPEQ.L #5\n", []byte{0xF2, 0x7B, 0x00, 0x01, 0x00, 0x00, 0x00, 0x05}},
		{"FMovecrPi", "FMOVECR #1,FP0\n", []byte{0xF2, 0x00, 0x5C, 0x01}},
		{"FMovecrFP3", "FMOVECR #1,FP3\n", []byte{0xF2, 0x00, 0x5D, 0x81}},
		{"FSaveIndirect", "FSAVE (A0)\n", []byte{0xF1, 0x10}},
		{"FSavePredec", "FSAVE -(A0)\n", []byte{0xF1, 0x20}},
		{"FRestoreIndirect", "FRESTORE (A0)\n", []byte{0xF1, 0x50}},
		{"FRestorePostinc", "FRESTORE (A0)+\n", []byte{0xF1, 0x58}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assembleForTarget(t, tc.src, targetFPU)
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("unexpected output: got % X want % X", got, tc.want)
			}
		})
	}
}

// TestFSaveRestoreDedicatedFormMatchesGASLiteral guards the equivalence
// this codebase relies on instead of implementing GAS's two-row-per-
// mnemonic table literally (see newFSaveRestoreDef's doc comment):
// FSAVE's predecrement literal (0xF120) and FRESTORE's postincrement
// literal (0xF158) must fall out of the single general-form WordBits
// (0xF100/0xF140) combined with FSrcEA's own mode/reg computation,
// without a second, separately-baked Form.
func TestFSaveRestoreDedicatedFormMatchesGASLiteral(t *testing.T) {
	save := assembleForTarget(t, "FSAVE -(A0)\n", targetFPU)
	if len(save) != 2 || save[0] != 0xF1 || save[1] != 0x20 {
		t.Fatalf("FSAVE -(A0) = % X, want F1 20", save)
	}
	restore := assembleForTarget(t, "FRESTORE (A0)+\n", targetFPU)
	if len(restore) != 2 || restore[0] != 0xF1 || restore[1] != 0x58 {
		t.Fatalf("FRESTORE (A0)+ = % X, want F1 58", restore)
	}
}

func TestFPUConditionalInstructionsRequireFPUFeature(t *testing.T) {
	srcs := []string{"FBEQ.W here\nhere:\n", "FDBEQ D0,here\nhere:\n", "FSEQ D0\n", "FTRAPEQ\n", "FMOVECR #1,FP0\n", "FSAVE (A0)\n", "FRESTORE (A0)\n"}
	for _, src := range srcs {
		mustAssembleErr(t, src, instructions.Target{CPU: instructions.CPU68020})
	}
}

func TestFSaveRejectsDataRegister(t *testing.T) {
	mustAssembleErr(t, "FSAVE D0\n", targetFPU)
}

func TestFSaveRejectsPostincrement(t *testing.T) {
	// FSAVE writes its state frame descending, so only -(An) is its
	// dedicated predecrement-class addressing mode, not (An)+.
	mustAssembleErr(t, "FSAVE (A0)+\n", targetFPU)
}

func TestFRestoreRejectsPredecrement(t *testing.T) {
	// FRESTORE reads its state frame ascending, so only (An)+ is its
	// dedicated postincrement-class addressing mode, not -(An).
	mustAssembleErr(t, "FRESTORE -(A0)\n", targetFPU)
}

func TestFMovecrRejectsOutOfRangeConstant(t *testing.T) {
	mustAssembleErr(t, "FMOVECR #128,FP0\n", targetFPU)
}

func TestFSeqRejectsAddressRegister(t *testing.T) {
	mustAssembleErr(t, "FSEQ A0\n", targetFPU)
}

func TestFBccHasNoByteForm(t *testing.T) {
	// Unlike integer Bcc, FBcc always needs a full extension word — GAS's
	// own table has no 8-bit-inline displacement form for it.
	_, err := asm.ParseWithOptions(strings.NewReader("FBEQ.S here\nhere:\n"), asm.ParseOptions{Target: targetFPU})
	if err == nil {
		t.Fatalf("expected an error for FBEQ.S (no byte-displacement form)")
	}
}
