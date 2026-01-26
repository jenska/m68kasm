package asm_test

import (
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
)

// TestMovemSizeDistinction verifies that MOVEM.L and MOVEM.W with identical
// operands produce different opcodes, specifically that the size bit (bit 6)
// is correctly set/cleared based on the size suffix. This test catches the bug
// where both variants produced identical opcodes.
func TestMovemSizeDistinction(t *testing.T) {
	tests := []struct {
		name    string
		sizeL   string
		sizeW   string
		operand string
	}{
		// Various operand combinations - store forms
		{"StoreIndirect", "D0", "D0", "(A0)"},
		{"StoreMultiReg", "D0/A0", "D0/A0", "(A1)"},
		{"StorePreDecSimple", "D0", "D0", "-(A7)"},
		{"StorePreDecMultiple", "D0-D1/A6", "D0-D1/A6", "-(A7)"},

		// Load forms
		{"LoadPostIncSimple", "(A0)+,D0", "(A0)+,D0", "D0"},
		{"LoadPostIncMultiple", "(A0)+,D0-D1/A6", "(A0)+,D0-D1/A6", "D0-D1/A6"},
		{"LoadIndirectSimple", "(A1),D0", "(A1),D0", "D0"},
		{"LoadIndirectMultiReg", "(A1),D0/A0", "(A1),D0/A0", "D0/A0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Determine if this is a load or store
			var srcL, srcW string
			if strings.HasPrefix(tt.sizeL, "(") {
				// Load form: (EA),Regs
				srcL = "MOVEM.L " + tt.sizeL + "\n"
				srcW = "MOVEM.W " + tt.sizeW + "\n"
			} else {
				// Store form: Regs,(EA)
				srcL = "MOVEM.L " + tt.sizeL + "," + tt.operand + "\n"
				srcW = "MOVEM.W " + tt.sizeW + "," + tt.operand + "\n"
			}

			progL, err := asm.Parse(strings.NewReader(srcL))
			if err != nil {
				t.Fatalf("parse .L failed: %v", err)
			}
			outL, err := asm.Assemble(progL)
			if err != nil {
				t.Fatalf("assemble .L failed: %v", err)
			}

			progW, err := asm.Parse(strings.NewReader(srcW))
			if err != nil {
				t.Fatalf("parse .W failed: %v", err)
			}
			outW, err := asm.Assemble(progW)
			if err != nil {
				t.Fatalf("assemble .W failed: %v", err)
			}

			// First words should differ
			if len(outL) < 2 || len(outW) < 2 {
				t.Fatalf("output too short")
			}

			// Check that the opcodes are different
			if outL[0] == outW[0] && outL[1] == outW[1] {
				t.Errorf("MOVEM.L and MOVEM.W produced same opcode: 0x%02X%02X",
					outL[0], outL[1])
			}

			// Check that size bit (bit 6 / 0x0040) differs
			sizeL := (uint16(outL[0])<<8 | uint16(outL[1])) & 0x0040
			sizeW := (uint16(outW[0])<<8 | uint16(outW[1])) & 0x0040

			if sizeL == 0 {
				t.Errorf("MOVEM.L size bit not set (0x%02X%02X)", outL[0], outL[1])
			}
			if sizeW != 0 {
				t.Errorf("MOVEM.W size bit should not be set (0x%02X%02X)", outW[0], outW[1])
			}
		})
	}
}

// TestMovemLoadStoreOpcodes verifies that load and store forms use different
// base opcodes (0x4Cxx for load, 0x48xx for store) and that size bits are
// correctly encoded in both forms.
func TestMovemLoadStoreOpcodes(t *testing.T) {
	tests := []struct {
		name           string
		src            string
		wantBaseOpcode byte // First byte: should be 0x4C for load, 0x48 for store
		wantSizeBit    bool // Whether size bit (bit 6) should be set
	}{
		// Store forms (base 0x48)
		{"StoreLongIndirect", "MOVEM.L D0,(A0)\n", 0x48, true},
		{"StoreWordIndirect", "MOVEM.W D0,(A0)\n", 0x48, false},
		{"StoreLongPreDec", "MOVEM.L D0,-(A0)\n", 0x48, true},
		{"StoreWordPreDec", "MOVEM.W D0,-(A0)\n", 0x48, false},

		// Load forms (base 0x4C)
		{"LoadLongPostInc", "MOVEM.L (A0)+,D0\n", 0x4C, true},
		{"LoadWordPostInc", "MOVEM.W (A0)+,D0\n", 0x4C, false},
		{"LoadLongIndirect", "MOVEM.L (A0),D0\n", 0x4C, true},
		{"LoadWordIndirect", "MOVEM.W (A0),D0\n", 0x4C, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := asm.Parse(strings.NewReader(tt.src))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			out, err := asm.Assemble(prog)
			if err != nil {
				t.Fatalf("assemble failed: %v", err)
			}

			if len(out) < 2 {
				t.Fatalf("output too short")
			}

			// Check base opcode
			if out[0] != tt.wantBaseOpcode {
				t.Errorf("opcode base mismatch: got 0x%02X, want 0x%02X",
					out[0], tt.wantBaseOpcode)
			}

			// Check size bit (bit 6 of the full opcode word)
			opcode := uint16(out[0])<<8 | uint16(out[1])
			sizeBit := (opcode & 0x0040) != 0

			if sizeBit != tt.wantSizeBit {
				t.Errorf("size bit mismatch: got %v, want %v", sizeBit, tt.wantSizeBit)
			}
		})
	}
}
