package asm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
)

func TestSectionLongEvenAndDCPseudoOps(t *testing.T) {
	src := `
.section ".text"
.byte 1
.even
.section .data
.long $11223344
DC.B 5, 6
DC.W $7788
DC.L $99AABBCC
.section bss
.byte 0
`

	prog, err := asm.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	out, err := asm.Assemble(prog)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	want := []byte{
		0x01, 0x00,
		0x11, 0x22, 0x33, 0x44,
		0x05, 0x06,
		0x77, 0x88,
		0x99, 0xAA, 0xBB, 0xCC,
		0x00,
	}
	if !bytes.Equal(out, want) {
		t.Fatalf("unexpected output: got %x want %x", out, want)
	}
}
