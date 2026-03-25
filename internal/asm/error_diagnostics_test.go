package asm_test

import (
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
)

func TestParseErrorIncludesSourceContext(t *testing.T) {
	src := "FOOBAR D0, D1\n"

	_, err := asm.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatalf("expected parse to fail")
	}

	msg := err.Error()
	for _, want := range []string{
		"unknown mnemonic",
		"FOOBAR D0, D1",
		"^",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected error to contain %q, got:\n%s", want, msg)
		}
	}
}

func TestAssembleErrorIncludesSourceContext(t *testing.T) {
	src := "BRA missing\n"

	prog, err := asm.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	_, err = asm.Assemble(prog)
	if err == nil {
		t.Fatalf("expected assembly to fail")
	}

	msg := err.Error()
	for _, want := range []string{
		"undefined label: missing",
		"BRA missing",
		"^",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected error to contain %q, got:\n%s", want, msg)
		}
	}
}

func TestParseErrorUsesReadableTokenNames(t *testing.T) {
	src := "MOVE D0 D1\n"

	_, err := asm.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatalf("expected parse to fail")
	}

	msg := err.Error()
	if !strings.Contains(msg, "expected comma, got identifier") {
		t.Fatalf("expected readable token names, got:\n%s", msg)
	}
}
