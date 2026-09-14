package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

var target68040PMMU = instructions.Target{CPU: instructions.CPU68040, Features: instructions.FeatPMMU}
var target68060PMMU = instructions.Target{CPU: instructions.CPU68060, Features: instructions.FeatPMMU}

// Expected bytes were hand-derived from GNU binutils' GAS m68k opcode
// table's single-word 68040 PMMU rows (pflusha=0xF518, pflushan=0xF510,
// pflushn=0xF500|An, pflush=0xF508|An, ptestr=0xF568|An,
// ptestw=0xF548|An — see cpu040_pmmu.go's own doc comment), then
// confirmed against this assembler's own CLI output.
func TestAssemble68040PMMU(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"PflushaBare", "PFLUSHA\n", []byte{0xF5, 0x18}},
		{"Pflushan", "PFLUSHAN\n", []byte{0xF5, 0x10}},
		{"PflushnIndirect", "PFLUSHN (A1)\n", []byte{0xF5, 0x01}},
		{"PflushnBareReg", "PFLUSHN A2\n", []byte{0xF5, 0x02}},
		{"PflushIndirect", "PFLUSH (A3)\n", []byte{0xF5, 0x0B}},
		{"PtestrIndirect", "PTESTR (A4)\n", []byte{0xF5, 0x6C}},
		{"PtestwIndirect", "PTESTW (A5)\n", []byte{0xF5, 0x4D}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assembleForTarget(t, tc.src, target68040PMMU)
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("unexpected output: got % X want % X", got, tc.want)
			}
		})
	}
}

// TestPflusha68040FormWinsOnPrepend guards the ordering hazard this
// codebase's own doc comment (cpu040_pmmu.go) calls out explicitly:
// PFLUSHA's 68040 form and its pre-existing 68030/68851 two-word form
// share identical (empty) OperKinds, so only Form order — the 68040
// form prepended, not appended — makes a 68040+ target pick the
// single-word encoding instead of the two-word one.
func TestPflusha68040FormWinsOnPrepend(t *testing.T) {
	got := assembleForTarget(t, "PFLUSHA\n", target68040PMMU)
	if len(got) != 2 || got[0] != 0xF5 || got[1] != 0x18 {
		t.Fatalf("PFLUSHA on a 68040 target = % X, want F5 18 (the single-word 68040 form)", got)
	}
}

// TestPflusha68030FormStillWorks guards the converse: a pre-68040
// target must still get the original two-word coprocessor encoding —
// confirming Table.ForTarget's own Requires-based filtering, not just
// Form order, is what keeps the 68040 form from leaking onto earlier
// tiers.
func TestPflusha68030FormStillWorks(t *testing.T) {
	got := assembleForTarget(t, "PFLUSHA\n", targetPMMU) // targetPMMU = CPU68030
	want := []byte{0xF0, 0x00, 0x24, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("PFLUSHA on a 68030 target = % X, want % X (the two-word coprocessor form)", got, want)
	}
}

func Test68040PMMURequiresPMMUFeature(t *testing.T) {
	mustAssembleErr(t, "PFLUSHA\n", targetM68040) // no FeatPMMU
	mustAssembleErr(t, "PFLUSHN (A0)\n", targetM68040)
}

func Test68040PMMURequires68040Floor(t *testing.T) {
	mustAssembleErr(t, "PFLUSHAN\n", targetPMMU) // targetPMMU = CPU68030
	mustAssembleErr(t, "PFLUSHN (A0)\n", targetPMMU)
}

// TestPtestr68040FormExcludes68060 guards requireM68040Only: unlike
// PFLUSH/PFLUSHA/PFLUSHAN/PFLUSHN (tagged m68040up in GAS's own table,
// extending to the 68060), PTESTR/PTESTW's single-word form is tagged
// the bare m68040 — the 68060 dropped it — so a plain CPU-floor
// Requires would have wrongly kept accepting it on a 68060 target.
func TestPtestr68040FormExcludes68060(t *testing.T) {
	got := assembleForTarget(t, "PTESTR (A0)\n", target68040PMMU)
	if len(got) != 2 || got[0] != 0xF5 || got[1] != 0x68 {
		t.Fatalf("PTESTR (A0) on a 68040 target = % X, want F5 68", got)
	}
	mustAssembleErr(t, "PTESTR (A0)\n", target68060PMMU)
	mustAssembleErr(t, "PTESTW (A0)\n", target68060PMMU)
}

// TestPflusha68040FormIncludes68060 is the converse of
// TestPtestr68040FormExcludes68060: PFLUSHA/PFLUSHAN/PFLUSHN/PFLUSH's
// own single-word forms ARE tagged m68040up, so they must keep working
// on a 68060 target, unlike PTESTR/PTESTW.
func TestPflusha68040FormIncludes68060(t *testing.T) {
	got := assembleForTarget(t, "PFLUSHA\n", target68060PMMU)
	if len(got) != 2 || got[0] != 0xF5 || got[1] != 0x18 {
		t.Fatalf("PFLUSHA on a 68060 target = % X, want F5 18", got)
	}
}

func TestPflushnRejectsNonAddressRegister(t *testing.T) {
	mustAssembleErr(t, "PFLUSHN D0\n", target68040PMMU)
	mustAssembleErr(t, "PFLUSHN #5\n", target68040PMMU)
	mustAssembleErr(t, "PFLUSHN\n", target68040PMMU)
}

// TestPflushnAcceptsBothAddressRegisterSpellings guards GAS's own
// dual-row shape for this operand: "(An)" and bare "An" must produce
// byte-identical output, since real hardware only ever sees the
// register number regardless of how it was spelled.
func TestPflushnAcceptsBothAddressRegisterSpellings(t *testing.T) {
	indirect := assembleForTarget(t, "PFLUSHN (A3)\n", target68040PMMU)
	bare := assembleForTarget(t, "PFLUSHN A3\n", target68040PMMU)
	if !bytes.Equal(indirect, bare) {
		t.Fatalf("PFLUSHN (A3) = % X, PFLUSHN A3 = % X, want identical", indirect, bare)
	}
}
