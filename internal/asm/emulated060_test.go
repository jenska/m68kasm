package asm_test

import (
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// assembleWithWarnings is assembleForTarget but also returns
// Program.Warnings, for tests checking checkEmulatedOn68060.
func assembleWithWarnings(t *testing.T, src string, target instructions.Target) []string {
	t.Helper()
	prog, err := asm.ParseWithOptions(strings.NewReader(src), asm.ParseOptions{Target: target})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if _, err := asm.Assemble(prog); err != nil {
		t.Fatalf("assemble failed: %v", err)
	}
	return prog.Warnings
}

var target68060 = instructions.Target{CPU: instructions.CPU68060}

func TestEmulatedOn68060Warnings(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		wantWarn  bool
		substring string
	}{
		{"CAS2Warns", "CAS2.L D0:D1,D2:D3,(A0):(A1)\n", true, "CAS2"},
		{"MOVEPWarns", "MOVEP.L (0,A0),D0\n", true, "MOVEP"},
		{"CHK2Warns", "CHK2.L (A0),D0\n", true, "CHK2"},
		{"CMP2Warns", "CMP2.L (A0),D0\n", true, "CMP2"},
		{"DIVSLWideWarns", "DIVSL.L (A0),D0:D1\n", true, "DIVSL"},
		{"DIVSLNarrowNoWarn", "DIVSL.L (A0),D0\n", false, ""},
		{"BFEXTUDynamicOffsetWarns", "BFEXTU (A0){D1:8},D2\n", true, "BFEXTU"},
		{"BFEXTULiteralNoWarn", "BFEXTU (A0){0:8},D2\n", false, ""},
		{"PlainMoveNoWarn", "MOVE.L D0,D1\n", false, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			warnings := assembleWithWarnings(t, tc.src, target68060)
			if tc.wantWarn {
				if len(warnings) != 1 {
					t.Fatalf("got %d warnings, want 1: %v", len(warnings), warnings)
				}
				if !strings.Contains(warnings[0], tc.substring) {
					t.Fatalf("warning %q does not mention %q", warnings[0], tc.substring)
				}
				if !strings.Contains(warnings[0], "line 1") {
					t.Fatalf("warning %q missing line number", warnings[0])
				}
			} else if len(warnings) != 0 {
				t.Fatalf("got unexpected warnings: %v", warnings)
			}
		})
	}
}

func TestEmulatedOn68060WarningsOnlyOn68060(t *testing.T) {
	// The same emulated forms must NOT warn on any other CPU tier —
	// this is a 68060-specific software-emulation fact, not a general
	// deprecation.
	targets := []instructions.Target{
		{CPU: instructions.CPU68020},
		{CPU: instructions.CPU68030},
		{CPU: instructions.CPU68040},
	}
	for _, target := range targets {
		warnings := assembleWithWarnings(t, "CAS2.L D0:D1,D2:D3,(A0):(A1)\n", target)
		if len(warnings) != 0 {
			t.Fatalf("%s: got unexpected warnings: %v", target.CPU, warnings)
		}
	}
}

func TestEmulatedOn68060DoesNotBlockAssembly(t *testing.T) {
	// These forms must still assemble successfully — a warning, not an
	// error, matching real 68060 behavior (they execute correctly, just
	// slower).
	assembleWithWarnings(t, "CAS2.L D0:D1,D2:D3,(A0):(A1)\n", target68060)
}
