package asm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
	"github.com/jenska/m68kasm/internal/asm/instructions"
)

var targetCPU32 = instructions.Target{CPU: instructions.CPU32}

// TestCPU32SharesConfirmed68020Additions locks in the "68020-era
// additions CPU32 backported" side of the lattice model — verified
// against GNU binutils' gas/config/tc-m68k.c, which tags each of these
// with both m68020up and cpu32 (or, for scale factors, whose own error
// message says "needs cpu32 or 68020 or higher").
func TestCPU32SharesConfirmed68020Additions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"LongBranch", "BRA.L target\nNOP\ntarget:\n", []byte{0x60, 0xFF, 0x00, 0x00, 0x00, 0x06, 0x4E, 0x71}},
		{"ScaleFactor", "MOVE.L (0,A0,D1.W*2),D2\n", []byte{0x24, 0x30, 0x12, 0x00}},
		{"Chk2", "CHK2.B (A0),D1\n", []byte{0x00, 0xD0, 0x18, 0x00}},
		{"Cmp2", "CMP2.W (A0),A2\n", []byte{0x02, 0xD0, 0xA0, 0x00}},
		{"Extb", "EXTB.L D0\n", []byte{0x49, 0xC0}},
		{"Trapcc", "TRAPT\n", []byte{0x50, 0xFC}},
		{"Movec68010Tier", "MOVEC VBR,A0\n", []byte{0x4E, 0x7A, 0x88, 0x01}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assembleForTarget(t, tc.src, targetCPU32)
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("unexpected output: got % X want % X", got, tc.want)
			}
		})
	}
}

// TestCPU32LacksMemoryIndirect is the other half of the lattice: unlike
// the plain-ordering cases above, 68020 memory-indirect addressing is
// explicitly rejected on cpu32 by GAS ("needs 68020 or higher"), so
// Target{CPU: CPU68020} — not Target{CPU: CPU32} — is the correct tag
// for it, and it must stay denied on a CPU32 target even though CPU32
// now has scale factors and Bcc.L.
func TestCPU32LacksMemoryIndirect(t *testing.T) {
	mustAssembleErr(t, "MOVE.L ([0,A0],D1),D2\n", targetCPU32)
}

// TestBGNDIsExclusiveToCPU32 is the key proof for the featCPU32Only
// mechanism: BGND must assemble on CPU32 but be REJECTED on 68020+,
// even though 68020's CPUKind value is numerically higher than CPU32's
// — the one relationship a plain "t.CPU >= req.CPU" floor comparison
// cannot express on its own (see Target.Supports's doc comment).
func TestBGNDIsExclusiveToCPU32(t *testing.T) {
	got := assembleForTarget(t, "BGND\n", targetCPU32)
	want := []byte{0x4A, 0xFA}
	if !bytes.Equal(got, want) {
		t.Fatalf("BGND on cpu32: got % X want % X", got, want)
	}

	for _, target := range []instructions.Target{instructions.Target68000, {CPU: instructions.CPU68010}, {CPU: instructions.CPU68020}, {CPU: instructions.CPU68060}} {
		if _, err := asm.ParseWithOptions(strings.NewReader("BGND\n"), asm.ParseOptions{Target: target}); err == nil {
			t.Errorf("BGND should be rejected on %s (CPU32-exclusive)", target.CPU)
		}
	}
}

func TestCPU68010DoesNotGetSharedCPU32Additions(t *testing.T) {
	target68010 := instructions.Target{CPU: instructions.CPU68010}
	srcs := []string{"BRA.L target\nNOP\ntarget:\n", "MOVE.L (0,A0,D1.W*2),D2\n", "CHK2.B (A0),D1\n"}
	for _, src := range srcs {
		if _, err := asm.ParseWithOptions(strings.NewReader(src), asm.ParseOptions{Target: target68010}); err == nil {
			t.Errorf("%q: expected an error on a 68010 target", src)
		}
	}
}
