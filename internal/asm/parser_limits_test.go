package asm_test

import (
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
)

func TestParseORGRejectsExcessivePadding(t *testing.T) {
	src := ".org 100000000\n"
	if _, err := asm.Parse(strings.NewReader(src)); err == nil {
		t.Fatalf("expected .org padding beyond limit to fail")
	} else if !strings.Contains(err.Error(), "maximum program size") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseALIGNRejectsExcessivePadding(t *testing.T) {
	src := ".byte 0\n.align 100000000\n"
	if _, err := asm.Parse(strings.NewReader(src)); err == nil {
		t.Fatalf("expected .align padding beyond limit to fail")
	} else if !strings.Contains(err.Error(), "maximum program size") {
		t.Fatalf("unexpected error: %v", err)
	}
}
