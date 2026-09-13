package asm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
	"github.com/jenska/m68kasm/internal/asm/instructions"
)

var target68010 = instructions.Target{CPU: instructions.CPU68010}

func assembleForTarget(t *testing.T, src string, target instructions.Target) []byte {
	t.Helper()
	prog, err := asm.ParseWithOptions(strings.NewReader(src), asm.ParseOptions{Target: target})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	out, err := asm.Assemble(prog)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}
	return out
}

// Expected bytes below were cross-checked against GNU binutils' GAS m68k
// backend (opcodes/m68k-opc.c and gas/config/tc-m68k.c) rather than typed
// from memory, since a wrong bit placement would silently miscompile.
func TestAssemble68010Instructions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"MovecControlToAddrReg", "MOVEC VBR,A0\n", []byte{0x4E, 0x7A, 0x88, 0x01}},
		{"MovecDataRegToUSP", "MOVEC D0,USP\n", []byte{0x4E, 0x7B, 0x08, 0x00}},
		{"MovecAddrRegToSFC", "MOVEC A1,SFC\n", []byte{0x4E, 0x7B, 0x90, 0x00}},
		{"MovecAddrRegToDFC", "MOVEC A2,DFC\n", []byte{0x4E, 0x7B, 0xA0, 0x01}},
		{"MovesLongDataRegToIndirect", "MOVES.L D0,(A0)\n", []byte{0x0E, 0x90, 0x08, 0x00}},
		{"MovesWordIndirectToAddrReg", "MOVES.W (A0),A2\n", []byte{0x0E, 0x50, 0xA0, 0x00}},
		{"MovesByteDataRegToDisp16", "MOVES.B D3,$10(A0)\n", []byte{0x0E, 0x28, 0x38, 0x00, 0x00, 0x10}},
		{"Rtd", "RTD #8\n", []byte{0x4E, 0x74, 0x00, 0x08}},
		{"RtdNegativeDisplacement", "RTD #-4\n", []byte{0x4E, 0x74, 0xFF, 0xFC}},
		{"Bkpt", "BKPT #3\n", []byte{0x48, 0x4B}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assembleForTarget(t, tc.src, target68010)
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("unexpected output: got % X want % X", got, tc.want)
			}
		})
	}
}

func TestCPU68010InstructionsRequireTarget(t *testing.T) {
	srcs := []string{"MOVEC VBR,A0\n", "MOVES.L D0,(A0)\n", "RTD #8\n", "BKPT #3\n"}
	for _, src := range srcs {
		if _, err := asm.Parse(strings.NewReader(src)); err == nil {
			t.Errorf("%q: expected an error on the default (68000) target", src)
		}
	}
}

// mustAssembleErr parses and assembles src for target, expecting a failure
// at either stage (operand-kind mismatches surface at parse time, since
// tryParseForm's per-form dispatch already rejects the wrong shape; range
// and Validate checks run later, during Assemble).
func mustAssembleErr(t *testing.T, src string, target instructions.Target) {
	t.Helper()
	prog, err := asm.ParseWithOptions(strings.NewReader(src), asm.ParseOptions{Target: target})
	if err != nil {
		return
	}
	if _, err := asm.Assemble(prog); err == nil {
		t.Errorf("%q: expected an error, got none", src)
	}
}

func TestBKPTVectorRange(t *testing.T) {
	mustAssembleErr(t, "BKPT #8\n", target68010)

	prog, err := asm.ParseWithOptions(strings.NewReader("BKPT #0\n"), asm.ParseOptions{Target: target68010})
	if err != nil {
		t.Fatalf("BKPT #0 should be valid: %v", err)
	}
	if _, err := asm.Assemble(prog); err != nil {
		t.Fatalf("assemble failed: %v", err)
	}
}

func TestRTDDisplacementRange(t *testing.T) {
	mustAssembleErr(t, "RTD #40000\n", target68010)
}

func TestMOVECRejectsMemoryOperand(t *testing.T) {
	mustAssembleErr(t, "MOVEC (A0),D0\n", target68010)
}

func TestMOVESRejectsRegisterDestination(t *testing.T) {
	mustAssembleErr(t, "MOVES.L D0,D1\n", target68010)
}
