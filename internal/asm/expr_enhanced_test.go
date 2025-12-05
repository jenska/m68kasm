package asm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
)

func TestEnhancedExpressions(t *testing.T) {
	src := `.byte 5%3, 1&&0, 1||0, !1, 3==3, 4!=4, 3<2, 3>=2
START: .byte 0
MID: .byte 0
DIFF: .word MID-START
`

	prog, err := asm.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	out, err := asm.Assemble(prog)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	want := []byte{2, 0, 1, 0, 1, 0, 0, 1, 0, 0, 0, 1}
	if !bytes.Equal(out, want) {
		t.Fatalf("unexpected output: got %x want %x", out, want)
	}
}
