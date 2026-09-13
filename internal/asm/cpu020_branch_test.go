package asm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
	"github.com/jenska/m68kasm/internal/asm/instructions"
)

var target68020 = instructions.Target{CPU: instructions.CPU68020}

func TestLongBranchRequires68020(t *testing.T) {
	src := "BRA.L target\nNOP\ntarget:\n"
	if _, err := asm.Parse(strings.NewReader(src)); err == nil {
		t.Fatal("BRA.L should be rejected on the default (68000) target")
	}
}

func TestLongBranchForward(t *testing.T) {
	// BRA.L (6 bytes) then NOP (2 bytes) then target: basePC = 0+2 = 2,
	// target address = 6+2 = 8, displacement = 8-2 = 6.
	src := "BRA.L target\nNOP\ntarget:\n"
	got := assembleForTarget(t, src, target68020)
	want := []byte{0x60, 0xFF, 0x00, 0x00, 0x00, 0x06, 0x4E, 0x71}
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected output: got % X want % X", got, want)
	}
}

func TestLongBranchSubroutineCall(t *testing.T) {
	src := "BSR.L sub\nNOP\nsub:\nRTS\n"
	got := assembleForTarget(t, src, target68020)
	want := []byte{0x61, 0xFF, 0x00, 0x00, 0x00, 0x06, 0x4E, 0x71, 0x4E, 0x75}
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected output: got % X want % X", got, want)
	}
}

func TestLongBranchBackward(t *testing.T) {
	// start:(0) NOP(2) NOP(2) BEQ.L start(4, 6 bytes). basePC = 4+2 = 6,
	// target = 0, displacement = 0-6 = -6.
	src := "start:\nNOP\nNOP\nBEQ.L start\n"
	got := assembleForTarget(t, src, target68020)
	want := []byte{0x4E, 0x71, 0x4E, 0x71, 0x67, 0xFF, 0xFF, 0xFF, 0xFF, 0xFA}
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected output: got % X want % X", got, want)
	}
}

func TestPlainBranchUnaffectedByLongForm(t *testing.T) {
	// A bare BRA with no size suffix must still pick the byte form, on
	// both a 68000 and a 68020 target — adding the gated .L form must not
	// disturb the pre-existing default-size resolution.
	src := "BRA target\nNOP\ntarget:\n"
	want := []byte{0x60, 0x02, 0x4E, 0x71}

	if got := assembleForTarget(t, src, instructions.Target68000); !bytes.Equal(got, want) {
		t.Errorf("68000 target: got % X want % X", got, want)
	}
	if got := assembleForTarget(t, src, target68020); !bytes.Equal(got, want) {
		t.Errorf("68020 target: got % X want % X", got, want)
	}
}

func TestShortAndWordBranchesUnaffectedOn68020(t *testing.T) {
	// BRA.W / BRA.S must still work exactly as before once a 68020-only
	// .L form exists alongside them in the same InstrDef.
	src := "BRA.W target\nNOP\ntarget:\n"
	want := []byte{0x60, 0x00, 0x00, 0x04, 0x4E, 0x71}
	if got := assembleForTarget(t, src, target68020); !bytes.Equal(got, want) {
		t.Errorf("got % X want % X", got, want)
	}
}
