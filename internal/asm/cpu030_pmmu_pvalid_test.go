package asm_test

import (
	"bytes"
	"testing"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

var target68851 = instructions.Target{CPU: instructions.CPU68020, Features: instructions.FeatPMMU}

// Expected bytes follow the MC68851 User's Manual, appendix A.
func TestAssembleMC68851Forms(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		want []byte
	}{
		{"PvalidVal", "PVALID VAL,(A0)\n", []byte{0xF0, 0x10, 0x28, 0x00}},
		{"PvalidAn", "PVALID A3,(A0)\n", []byte{0xF0, 0x10, 0x2C, 0x03}},
		{"PvalidAbs", "PVALID VAL,$1234.W\n", []byte{0xF0, 0x38, 0x28, 0x00, 0x12, 0x34}},
		{"PflushFc15", "PFLUSH #15,#15\n", []byte{0xF0, 0x00, 0x31, 0xFF}},
		{"PloadFc9", "PLOADW #9,(A0)\n", []byte{0xF0, 0x10, 0x20, 0x19}},
		{"PtestFc9", "PTESTR #9,(A0),#7\n", []byte{0xF0, 0x10, 0x9E, 0x19}},
		{"PmoveImmTC", "PMOVE #$80C86600,TC\n", []byte{0xF0, 0x3C, 0x40, 0x00, 0x80, 0xC8, 0x66, 0x00}},
		{"PmoveImmAC", "PMOVE #$0090,AC\n", []byte{0xF0, 0x3C, 0x5C, 0x00, 0x00, 0x90}},
		{"PmoveImmCAL", "PMOVE #$40,CAL\n", []byte{0xF0, 0x3C, 0x50, 0x00, 0x00, 0x40}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := assembleForTarget(t, tc.src, target68851); !bytes.Equal(got, tc.want) {
				t.Fatalf("got % X want % X", got, tc.want)
			}
		})
	}
}

func TestPvalidOnlyOnMC68851(t *testing.T) {
	mustAssembleErr(t, "PVALID VAL,(A0)\n", instructions.Target{CPU: instructions.CPU68030, Features: instructions.FeatPMMU})
	mustAssembleErr(t, "PVALID VAL,(A0)\n", instructions.Target{CPU: instructions.CPU68020})
	mustAssembleErr(t, "PVALID VAL,D0\n", target68851)
}

func TestFunctionCodeRangeOnMC68030(t *testing.T) {
	mustAssembleErr(t, "PFLUSH #8,#7\n", instructions.Target{CPU: instructions.CPU68030, Features: instructions.FeatPMMU})
}

func TestCpDBccBranchFromDisplacementWord(t *testing.T) {
	// The displacement counts from its own word, two bytes after the
	// condition word: back to the PDBcc itself is -4.
	got := assembleForTarget(t, "top:\nPDBIC D1,top\n", target68851)
	if want := []byte{0xF0, 0x49, 0x00, 0x0B, 0xFF, 0xFC}; !bytes.Equal(got, want) {
		t.Fatalf("got % X want % X", got, want)
	}
}
