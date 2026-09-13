package asm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// Expected bytes below were derived by hand from the 68020 full
// extension-word bit layout (cross-checked against GNU binutils'
// gas/config/tc-m68k.c, the POST/PRE/BASE case) and then confirmed
// against the actual assembler output before being pinned here.
func TestMemoryIndirectAddressing(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"PreIndexedNullBdOd", "MOVE.L ([0,A0],D1,0),D2\n", []byte{0x24, 0x30, 0x11, 0x11}},
		{"PostIndexedNullBdLongOd", "MOVE.L ([0,A0,D1],4),D2\n", []byte{0x24, 0x30, 0x11, 0x17, 0x00, 0x00, 0x00, 0x04}},
		{"PreIndexedNoIndexNumericOuter", "MOVE.L ([0,A0],100),D2\n", []byte{0x24, 0x30, 0x01, 0x53, 0x00, 0x00, 0x00, 0x64}},
		{"PreIndexedNoIndexNoOuter", "MOVE.L ([0,A0]),D2\n", []byte{0x24, 0x30, 0x01, 0x51}},
		{"JmpThroughMemoryIndirect", "JMP ([0,A0])\n", []byte{0x4E, 0xF0, 0x01, 0x51}},
		{"PCPostIndexedNullOuter", "LEA ([0,PC,D0],0),A1\n", []byte{0x43, 0xFB, 0x01, 0x15}},
		{"PCPreIndexedLongBdWordOuter", "LEA ([100.L,PC],A0,2.W),A1\n", []byte{0x43, 0xFB, 0x81, 0x32, 0x00, 0x00, 0x00, 0x64, 0x00, 0x02}},
		{"ExplicitLongBdEvenWhenZero", "mylabel:\nMOVE.L ([mylabel.L,A0],D1,4),D2\n", []byte{0x24, 0x30, 0x11, 0x33, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assembleForTarget(t, tc.src, target68020)
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("unexpected output: got % X want % X", got, tc.want)
			}
		})
	}
}

func TestMemoryIndirectForwardReferenceResolves(t *testing.T) {
	// The label is defined AFTER its use (forward reference); this only
	// works correctly if the explicit .w suffix keeps the instruction's
	// encoded length identical across both of the assembler's passes —
	// see parseSizedDisp's doc comment on why a suffix is mandatory here.
	src := "MOVE.L ([mylabel.W,A0],D1),D2\nNOP\nmylabel:\n"
	want := []byte{0x24, 0x30, 0x11, 0x21, 0x00, 0x08, 0x4E, 0x71}
	got := assembleForTarget(t, src, target68020)
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected output: got % X want % X", got, want)
	}
}

func TestMemoryIndirectRequires68020(t *testing.T) {
	src := "MOVE.L ([0,A0],D1),D2\n"
	_, err := asm.ParseWithOptions(strings.NewReader(src), asm.ParseOptions{Target: instructions.Target68000})
	if err == nil {
		t.Fatal("expected an error: memory-indirect addressing requires 68020+")
	} else if !strings.Contains(err.Error(), "68020") {
		t.Errorf("expected error to mention the 68020 requirement, got: %v", err)
	}
}

func TestMemoryIndirectSymbolRequiresExplicitSize(t *testing.T) {
	mustAssembleErr(t, "mylabel:\nMOVE.L ([mylabel,A0],D1,4),D2\n", target68020)
}

func TestBriefIndexedOverflowNowErrors(t *testing.T) {
	// Previously this silently truncated the displacement to a signed
	// byte instead of erroring — a correctness bug fixed alongside the
	// full-format addressing work (see parser_ea.go's indexedEA).
	mustAssembleErr(t, "MOVE.L (200,A0,D1.W),D2\n", target68020)
}

func TestBriefIndexedInRangeUnaffected(t *testing.T) {
	src := "MOVE.L (100,A0,D1.W),D2\n"
	want := []byte{0x24, 0x30, 0x10, 0x64}
	got := assembleForTarget(t, src, target68020)
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected output: got % X want % X", got, want)
	}
}

// TestMemoryIndirectWordSymbolRangeCheckedAtEncode covers the one case
// parseSizedDisp cannot range-check up front: a symbol given an explicit
// .w size whose resolved address doesn't actually fit in 16 bits. The
// guard lives in dispWords (ea.go), exercised here end to end rather
// than only at the unit level, since this is exactly the kind of
// resolved-late value the two-pass safety design in parseSizedDisp's
// doc comment exists to keep from silently truncating.
func TestMemoryIndirectWordSymbolRangeCheckedAtEncode(t *testing.T) {
	src := ".org $10000\nmylabel:\nMOVE.L ([mylabel.W,A0],D1),D2\n"
	mustAssembleErr(t, src, target68020)
}
