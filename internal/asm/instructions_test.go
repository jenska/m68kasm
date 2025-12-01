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
