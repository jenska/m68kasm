package asm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
)

func TestMacroExpansionWithParameters(t *testing.T) {
	src := `
.macro BYTEPAIR a, b
.byte a, b
.endmacro

BYTEPAIR 1,2
BYTEPAIR 3+4, !0
`

	prog, err := asm.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	out, err := asm.Assemble(prog)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	want := []byte{0x01, 0x02, 0x07, 0x01}
	if !bytes.Equal(out, want) {
		t.Fatalf("unexpected output: got %x want %x", out, want)
	}
}

func TestNestedMacroUse(t *testing.T) {
	src := `
.macro PAIR a, b
.byte a, b
.endmacro

.macro DOUBLE start
PAIR start, start+1
PAIR start+2, start+3
.endmacro

DOUBLE 1
`

	prog, err := asm.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	out, err := asm.Assemble(prog)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	want := []byte{0x01, 0x02, 0x03, 0x04}
	if !bytes.Equal(out, want) {
		t.Fatalf("unexpected output: got %x want %x", out, want)
	}
}
