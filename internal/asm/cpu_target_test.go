package asm_test

import (
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
	"github.com/jenska/m68kasm/internal/asm/instructions"
)

func TestScaledIndexRejectedOnDefault68000Target(t *testing.T) {
	src := "MOVE.L (A0,D0.W*2),D1\n"

	_, err := asm.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("expected an error: scale factor *2 requires 68020+, and the default target is a bare 68000")
	}
	if !strings.Contains(err.Error(), "68020") {
		t.Errorf("expected error to mention the 68020 requirement, got: %v", err)
	}
}

func TestScaledIndexAcceptedOn68020Target(t *testing.T) {
	src := "MOVE.L (A0,D0.W*2),D1\n"
	opts := asm.ParseOptions{Target: instructions.Target{CPU: instructions.CPU68020}}

	prog, err := asm.ParseWithOptions(strings.NewReader(src), opts)
	if err != nil {
		t.Fatalf("expected scale factor *2 to be accepted on a 68020 target, got: %v", err)
	}

	if _, err := asm.Assemble(prog); err != nil {
		t.Fatalf("assemble failed: %v", err)
	}
}

func TestScaleFactorOfOneAllowedOnDefaultTarget(t *testing.T) {
	src := "MOVE.L (A0,D0.W*1),D1\n"

	if _, err := asm.Parse(strings.NewReader(src)); err != nil {
		t.Fatalf("scale factor *1 is representable on a plain 68000 and should not require gating, got: %v", err)
	}
}

func TestParseOptionsTargetFlowsToProgram(t *testing.T) {
	target := instructions.Target{CPU: instructions.CPU68020}
	prog, err := asm.ParseWithOptions(strings.NewReader("NOP\n"), asm.ParseOptions{Target: target})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if prog.Target != target {
		t.Errorf("Program.Target = %+v, want %+v", prog.Target, target)
	}
}

func TestDefaultTargetIsBare68000(t *testing.T) {
	prog, err := asm.Parse(strings.NewReader("NOP\n"))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if prog.Target != instructions.Target68000 {
		t.Errorf("Program.Target = %+v, want the zero-value 68000 target", prog.Target)
	}
}
