package asm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
)

func TestConstDefinitionWithEquals(t *testing.T) {
	src := "FOO = 42\n.BYTE FOO\n"

	prog, err := asm.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if got := prog.Labels["FOO"]; got != 42 {
		t.Fatalf("unexpected constant value: got %d want 42", got)
	}

	out, err := asm.Assemble(prog)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	want := []byte{0x2A}
	if !bytes.Equal(out, want) {
		t.Fatalf("unexpected output: got %x want %x", out, want)
	}
}

func TestConstDefinitionWithEqu(t *testing.T) {
	src := "BAR .equ 0x1234\n.WORD BAR\n"

	prog, err := asm.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if got := prog.Labels["BAR"]; got != 0x1234 {
		t.Fatalf("unexpected constant value: got %d want %d", got, 0x1234)
	}

	out, err := asm.Assemble(prog)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	want := []byte{0x12, 0x34}
	if !bytes.Equal(out, want) {
		t.Fatalf("unexpected output: got %x want %x", out, want)
	}
}
