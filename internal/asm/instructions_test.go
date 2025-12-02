package asm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
)

func assembleSource(t *testing.T, src string) []byte {
	t.Helper()
	prog, err := asm.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	out, err := asm.Assemble(prog)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}
	return out
}

func TestAssembleCoreInstructions(t *testing.T) {
tests := []struct {
name string
src  string
want []byte
}{
{"ABCDDataRegs", "ABCD D1,D0\n", []byte{0xC1, 0x01}},
{"ABCDPredecrement", "ABCD -(A1),-(A0)\n", []byte{0xC1, 0x09}},
{"SBCDDataRegs", "SBCD D2,D3\n", []byte{0x87, 0x02}},
{"SBCDPredecrement", "SBCD -(A3),-(A5)\n", []byte{0x8B, 0x0B}},
{"AddByteToMemory", "ADD.B D0,(A1)\n", []byte{0xD1, 0x11}},
{"AddLongToAddressReg", "ADD.L #1,A3\n", []byte{0xD7, 0xFC, 0x00, 0x00, 0x00, 0x01}},
{"SubWordToMemory", "SUB.W D2,(A3)\n", []byte{0x95, 0x53}},
{"MovemStorePredec", "MOVEM.L D0-D1/A6,-(A7)\n", []byte{0x48, 0xE7, 0xC0, 0x02}},
{"MovemLoadPostinc", "MOVEM.W (A0)+,D0-D1/A6\n", []byte{0x4C, 0x98, 0x40, 0x03}},
{"NoOperation", "NOP\n", []byte{0x4E, 0x71}},
{"Reset", "RESET\n", []byte{0x4E, 0x70}},
{"ReturnFromSubroutine", "RTS\n", []byte{0x4E, 0x75}},
		{"ReturnFromException", "RTE\n", []byte{0x4E, 0x73}},
		{"TrapVector", "TRAP #9\n", []byte{0x4E, 0x49}},
		{"MoveRegisterByte", "MOVE.B D1,D0\n", []byte{0x10, 0x01}},
		{"MoveImmediateLong", "MOVE.L #$12345678,D1\n", []byte{0x22, 0x3C, 0x12, 0x34, 0x56, 0x78}},
		{"MoveImmediateByteToMem", "MOVE.B #$12,(A0)\n", []byte{0x10, 0xBC, 0x00, 0x12}},
		{"AddWord", "ADD.W D1,D0\n", []byte{0xD0, 0x41}},
		{"AddImmediate", "ADD.W #1,D0\n", []byte{0xD0, 0x7C, 0x00, 0x01}},
		{"SubLong", "SUB.L (A1),D3\n", []byte{0x96, 0x91}},
		{"CmpByte", "CMP.B (16,A0),D2\n", []byte{0xB4, 0x28, 0x00, 0x10}},
		{"MultiplyWord", "MUL (A1),D0\n", []byte{0xC1, 0xD1}},
		{"DivideWord", "DIV (A2),D1\n", []byte{0x82, 0xD2}},
		{"MoveByteLabel", "label:\n.WORD 0\nMOVE.B label,D0\n", []byte{0x00, 0x00, 0x10, 0x39, 0x00, 0x00, 0x00, 0x00}},
		{"BranchAlwaysShort", "BRA target\n.WORD 0\ntarget:\n", []byte{0x60, 0x02, 0x00, 0x00}},
		{"BranchConditionWord", "BNE.W target\n.WORD 0\n.WORD 0\ntarget:\n", []byte{0x66, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00}},
		{"BranchSynonymShort", "BHS target\n.WORD 0\ntarget:\n", []byte{0x64, 0x02, 0x00, 0x00}},
		{"BSRWordDefault", "BSR target\n.WORD 0\ntarget:\n", []byte{0x61, 0x00, 0x00, 0x02, 0x00, 0x00}},
		{"BSRShortExplicit", "BSR.S target\n.WORD 0\ntarget:\n", []byte{0x61, 0x02, 0x00, 0x00}},
		{"LEAAdrIndToSP", "LEA  (A1), SP\n.WORD 0\ntarget:\n", []byte{0x4f, 0xd1, 0x00, 0x00}},
		{"LEAdrIndToA7", "LEA  (A1), A7\n", []byte{0x4f, 0xd1}},
		{"LEAddr", "LEA  target, SP\n.WORD 0\ntarget:\n", []byte{0x4f, 0xd1, 0x00, 0x00}},
		{"Comment", "; comment\nLEA  (A1), A7\n", []byte{0x4f, 0xd1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assembleSource(t, tt.src)
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("unexpected encoding for %s: got %x want %x", tt.src, got, tt.want)
			}
		})
	}
}

func TestBranchPCIncludesExtensionWords(t *testing.T) {
	src := "BSR target\nNOP\nNOP\ntarget:\nNOP\n"
	prog, err := asm.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	addr, ok := prog.Labels["target"]
	if !ok {
		t.Fatalf("label not recorded")
	}
	if got := int(addr - prog.Origin); got != 8 {
		t.Fatalf("unexpected target address: got %d want 8", got)
	}

	if _, err := asm.Assemble(prog); err != nil {
		t.Fatalf("assemble failed: %v", err)
	}
}

func TestOrgSetsOriginAndFillsGaps(t *testing.T) {
	src := ".org 4\n.byte 1\n.org 8\n.byte 2\n"
	prog, err := asm.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if prog.Origin != 4 {
		t.Fatalf("origin not recorded: got %d want 4", prog.Origin)
	}

	out, err := asm.Assemble(prog)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}
	want := []byte{0x01, 0x00, 0x00, 0x00, 0x02}
	if !bytes.Equal(out, want) {
		t.Fatalf("unexpected output: got %x want %x", out, want)
	}
}
