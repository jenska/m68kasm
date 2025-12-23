package asm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
)

func assembleSource(t *testing.T, src string) []byte {
	t.Helper()
	prog, err := asm.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	out, err := asm.Assemble(prog)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}
	return out
}

func TestAssembleCoreInstructions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"TrapOverflow", "TRAPV\n", []byte{0x4E, 0x76}},
		{"DBRADefault", "loop:\nDBRA D0, loop\n", []byte{0x51, 0xC8, 0xFF, 0xFC}},
		{"DBNEForward", "DBNE D1, target\nNOP\nNOP\ntarget:\n", []byte{0x56, 0xC9, 0x00, 0x04, 0x4E, 0x71, 0x4E, 0x71}},
		{"ABCDDataRegs", "ABCD D1,D0\n", []byte{0xC1, 0x01}},
		{"ABCDPredecrement", "ABCD -(A1),-(A0)\n", []byte{0xC1, 0x09}},
		{"SBCDDataRegs", "SBCD D2,D3\n", []byte{0x87, 0x02}},
		{"SBCDPredecrement", "SBCD -(A3),-(A5)\n", []byte{0x8B, 0x0B}},
		{"AddByteToMemory", "ADD.B D0,(A1)\n", []byte{0xD1, 0x11}},
		{"AddLongToAddressReg", "ADD.L #1,A3\n", []byte{0xD7, 0xFC, 0x00, 0x00, 0x00, 0x01}},
		{"SubWordToMemory", "SUB.W D2,(A3)\n", []byte{0x95, 0x53}},
		{"MovemStorePredec", "MOVEM.L D0-D1/A6,-(A7)\n", []byte{0x48, 0xE7, 0xC0, 0x02}},
		{"MovemLoadPostinc", "MOVEM.W (A0)+,D0-D1/A6\n", []byte{0x4C, 0x98, 0x40, 0x03}},
		{"NoOperation", "NOP\n", []byte{0x4E, 0x71}},
		{"Reset", "RESET\n", []byte{0x4E, 0x70}},
		{"ReturnFromSubroutine", "RTS\n", []byte{0x4E, 0x75}},
		{"ReturnFromTrap", "RTR\n", []byte{0x4E, 0x77}},
		{"ReturnFromException", "RTE\n", []byte{0x4E, 0x73}},
		{"TrapVector", "TRAP #9\n", []byte{0x4E, 0x49}},
		{"MoveRegisterByte", "MOVE.B D1,D0\n", []byte{0x10, 0x01}},
		{"MoveImmediateLong", "MOVE.L #$12345678,D1\n", []byte{0x22, 0x3C, 0x12, 0x34, 0x56, 0x78}},
		{"MoveImmediateByteToMem", "MOVE.B #$12,(A0)\n", []byte{0x10, 0xBC, 0x00, 0x12}},
		{"MoveQuickSigned", "MOVEQ #-1,D0\n", []byte{0x70, 0xFF}},
		{"MoveAddressMnemonic", "MOVEA.W (A0),A1\n", []byte{0x32, 0x50}},
		{"MoveAddressAbsLong", "MOVEA.W $100,A1\n", []byte{0x32, 0x79, 0x00, 0x00, 0x01, 0x00}},
		{"MoveAddressAbsWord", "MOVEA.W $100.w,A1\n", []byte{0x32, 0x78, 0x01, 0x00}},
		{"AddWord", "ADD.W D1,D0\n", []byte{0xD0, 0x41}},
		{"AddImmediate", "ADD.W #1,D0\n", []byte{0xD0, 0x7C, 0x00, 0x01}},
		{"AddImmediateMnemonic", "ADDI.B #1,D0\n", []byte{0x06, 0x00, 0x00, 0x01}},
		{"AddQuickByte", "ADDQ.B #1,D0\n", []byte{0x52, 0x00}},
		{"AddQuickLongPredec", "ADDQ.L #8,-(A7)\n", []byte{0x50, 0xA7}},
		{"AndWord", "AND.W D1,D0\n", []byte{0xC0, 0x41}},
		{"OrByteToMemory", "OR.B D0,(A1)\n", []byte{0x81, 0x11}},
		{"ExclusiveOrLong", "EOR.L D2,D3\n", []byte{0xB5, 0x83}},
		{"NotWordPostIncrement", "NOT.W (A0)+\n", []byte{0x46, 0x58}},
		{"SubLong", "SUB.L (A1),D3\n", []byte{0x96, 0x91}},
		{"SubQuickWordToAn", "SUBQ.W #3,A1\n", []byte{0x57, 0x49}},
		{"SubImmediateMnemonic", "SUBI.W #1,(A1)\n", []byte{0x04, 0x51, 0x00, 0x01}},
		{"CmpByte", "CMP.B (16,A0),D2\n", []byte{0xB4, 0x28, 0x00, 0x10}},
		{"CmpByteDispBeforeParen", "CMP.B 16(A0),D2\n", []byte{0xB4, 0x28, 0x00, 0x10}},
		{"CompareAddressMnemonic", "CMPA.L #1,A0\n", []byte{0xB1, 0xFC, 0x00, 0x00, 0x00, 0x01}},
		{"AddAddressMnemonic", "ADDA.L D1,A0\n", []byte{0xD1, 0xC1}},
		{"SubAddressMnemonic", "SUBA.W (A1),A2\n", []byte{0x94, 0xD1}},
		{"CheckBounds", "CHK (A0),D1\n", []byte{0x43, 0x90}},
		{"Negate", "NEG.W D2\n", []byte{0x44, 0x42}},
		{"SwapWord", "SWAP D2\n", []byte{0x48, 0x42}},
		{"NbcdPredecrement", "NBCD -(A3)\n", []byte{0x48, 0x23}},
		{"TestAndSetByte", "TAS D0\n", []byte{0x4A, 0xC0}},
		{"MultiplyWordUnsigned", "MULU (A1),D0\n", []byte{0xC0, 0xD1}},
		{"MultiplyWordSigned", "MULS (A1),D0\n", []byte{0xC1, 0xD1}},
		{"DivideWordUnsigned", "DIVU (A2),D1\n", []byte{0x82, 0xD2}},
		{"DivideWordSigned", "DIVS (A2),D1\n", []byte{0x83, 0xD2}},
		{"MoveToSRImmediate", "MOVE #$2700,SR\n", []byte{0x46, 0xFC, 0x27, 0x00}},
		{"MoveFromSRToDn", "MOVE SR,D0\n", []byte{0x40, 0xC0}},
		{"MoveUSPToA0", "MOVE USP,A0\n", []byte{0x4E, 0x68}},
		{"MoveAToUSP", "MOVE A1,USP\n", []byte{0x4E, 0x61}},
		{"MoveUSPToSSP", "MOVE USP,SSP\n", []byte{0x4E, 0x6F}},
		{"MoveSSPToUSP", "MOVE SSP,USP\n", []byte{0x4E, 0x67}},
		{"MoveLongToSSP", "MOVE.L D0,SSP\n", []byte{0x2E, 0x40}},
		{"StopWithStatus", "STOP #$2700\n", []byte{0x4E, 0x72, 0x27, 0x00}},
		{"JumpAddressIndirect", "JMP (A0)\n", []byte{0x4E, 0xD0}},
		{"ORIToCCR", "ORI #1,CCR\n", []byte{0x00, 0x3C, 0x00, 0x01}},
		{"ANDIToSR", "ANDI #$FF00,SR\n", []byte{0x02, 0x7C, 0xFF, 0x00}},
		{"MoveByteLabel", "label:\n.WORD 0\nMOVE.B label,D0\n", []byte{0x00, 0x00, 0x10, 0x39, 0x00, 0x00, 0x00, 0x00}},
		{"BitSetImmediateToDn", "BSET #1,D0\n", []byte{0x08, 0xC0, 0x00, 0x01}},
		{"BitClearImmediateToMem", "BCLR #7,(A1)\n", []byte{0x08, 0x91, 0x00, 0x07}},
		{"BitChangeRegisterSource", "BCHG D2,(A3)\n", []byte{0x05, 0x53}},
		{"BitTestImmediate", "BTST #3,D1\n", []byte{0x08, 0x01, 0x00, 0x03}},
		{"BitTestRegisterToMemory", "BTST D2,(A3)\n", []byte{0x05, 0x13}},
		{"CompareImmediateByte", "CMPI.B #1,D2\n", []byte{0x0C, 0x02, 0x00, 0x01}},
		{"SetIfNotEqual", "SNE (A1)\n", []byte{0x56, 0xD1}},
		{"BranchAlwaysShort", "BRA target\n.WORD 0\ntarget:\n", []byte{0x60, 0x02, 0x00, 0x00}},
		{"BranchLocalLabels", "BRA 1f\n.WORD 0\n1:\nBRA 1b\n", []byte{0x60, 0x02, 0x00, 0x00, 0x60, 0xFE}},
		{"BranchConditionWord", "BNE.W target\n.WORD 0\n.WORD 0\ntarget:\n", []byte{0x66, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00}},
		{"BranchSynonymShort", "BHS target\n.WORD 0\ntarget:\n", []byte{0x64, 0x02, 0x00, 0x00}},
		{"BSRWordDefault", "BSR target\n.WORD 0\ntarget:\n", []byte{0x61, 0x00, 0x00, 0x02, 0x00, 0x00}},
		{"BSRShortExplicit", "BSR.S target\n.WORD 0\ntarget:\n", []byte{0x61, 0x02, 0x00, 0x00}},
		{"LEAAdrIndToSP", "LEA  (A1), SP\n.WORD 0\ntarget:\n", []byte{0x4f, 0xd1, 0x00, 0x00}},
		{"LEAdrIndToA7", "LEA  (A1), A7\n", []byte{0x4f, 0xd1}},
		{"Comment", "; comment\nLEA  (A1), A7\n", []byte{0x4f, 0xd1}},
		{"MovepLoadWord", "MOVEP (6,A1),D0\n", []byte{0x01, 0x49, 0x00, 0x06}},
		{"MovepStoreLong", "MOVEP.L D2,(8,A3)\n", []byte{0x05, 0x8b, 0x00, 0x08}},
		{"TestDataLong", "TST.L D3\n", []byte{0x4A, 0x83}},
		{"NegateWithExtend", "NEGX.L D3\n", []byte{0x40, 0x83}},
		{"SubExtendWordRegisters", "SUBX.W D1,D0\n", []byte{0x91, 0x41}},
		{"SubExtendLongPredec", "SUBX.L -(A2),-(A3)\n", []byte{0x97, 0x8A}},
		{"AddExtendByteRegisters", "ADDX.B D4,D5\n", []byte{0xDB, 0x04}},
		{"AddExtendLongPredec", "ADDX.L -(A6),-(A7)\n", []byte{0xDF, 0x8E}},
		{"JumpToSubroutine", "JSR (A2)\n", []byte{0x4E, 0x92}},
		{"JumpToSubroutinePCDisp", "JSR (12,PC)\n", []byte{0x4E, 0xBA, 0x00, 0x0C}},
		{"PushEffectiveAddress", "PEA (A1)\n", []byte{0x48, 0x51}},
		{"LinkFrame", "LINK A6,#-4\n", []byte{0x4E, 0x56, 0xFF, 0xFC}},
		{"UnlinkFrame", "UNLK A6\n", []byte{0x4E, 0x5E}},
		{"ExtWord", "EXT.W D0\n", []byte{0x48, 0x80}},
		{"ExtLong", "EXT.L D1\n", []byte{0x48, 0xC1}},
		{"ClrByte", "CLR.B D0\n", []byte{0x42, 0x00}},
		{"ClrWord", "CLR.W D1\n", []byte{0x42, 0x41}},
		{"ClrLong", "CLR.L (A0)\n", []byte{0x42, 0x90}},
		{"ExgData", "EXG D0,D1\n", []byte{0xC1, 0x41}},
		{"ExgAddr", "EXG A0,A1\n", []byte{0xC1, 0x49}},
		{"ExgMixed", "EXG D0,A1\n", []byte{0xC1, 0x89}},
		{"Illegal", "ILLEGAL\n", []byte{0x4A, 0xFC}},
		{"AslImm", "ASL.W #1,D0\n", []byte{0xE3, 0x40}},
		{"AslReg", "ASL.L D1,D0\n", []byte{0xE3, 0xA0}},
		{"AslMem", "ASL.W (A0)\n", []byte{0xE1, 0xD0}},
		{"LsrImm", "LSR.B #4,D1\n", []byte{0xE8, 0x09}},
		{"RolReg", "ROL.W D2,D3\n", []byte{0xE5, 0x7B}},
		{"RoxrMem", "ROXR.W (A1)\n", []byte{0xE4, 0xD1}},
		{"MoveIndex", "MOVE.W (4,A0,D1.W),D1\n", []byte{0x32, 0x30, 0x10, 0x04}},
		{"MoveAbsWord", "MOVE.W $2000.W,D0\n", []byte{0x30, 0x38, 0x20, 0x00}},
		{"MoveAbsLong", "MOVE.W $20000.L,D0\n", []byte{0x30, 0x39, 0x00, 0x02, 0x00, 0x00}},
		{"MovePCDisp", "MOVE.W (16,PC),D0\n", []byte{0x30, 0x3A, 0x00, 0x10}},
		{"MovePCIndex", "MOVE.W (4,PC,D1.W),D0\n", []byte{0x30, 0x3B, 0x10, 0x04}},
		{"MoveToAbsWord", "MOVE.W D0,$2000.W\n", []byte{0x31, 0xC0, 0x20, 0x00}},
		{"MoveToAbsLong", "MOVE.W D0,$20000.L\n", []byte{0x33, 0xC0, 0x00, 0x02, 0x00, 0x00}},
		{"LeaIndex", "LEA (4,A0,D1.L),A1\n", []byte{0x43, 0xF0, 0x18, 0x04}},
		{"PeaIndirect", "PEA (A2)\n", []byte{0x48, 0x52}},
		{"JmpPCIndex", "JMP (4,PC,D0.W)\n", []byte{0x4E, 0xFB, 0x00, 0x04}},
		{"ClrAbs", "CLR.B $1000.W\n", []byte{0x42, 0x38, 0x10, 0x00}},
		{"TstIndex", "TST.B (2,A0,D1.W)\n", []byte{0x4A, 0x30, 0x10, 0x02}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assembleSource(t, tt.src)
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("unexpected encoding for %s: got %x want %x", tt.src, got, tt.want)
			}
		})
	}
}

func TestBranchPCIncludesExtensionWords(t *testing.T) {
	src := "BSR target\nNOP\nNOP\ntarget:\nNOP\n"
	prog, err := asm.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	addr, ok := prog.Labels["target"]
	if !ok {
		t.Fatalf("label not recorded")
	}
	if got := int(addr - prog.Origin); got != 8 {
		t.Fatalf("unexpected target address: got %d want 8", got)
	}

	if _, err := asm.Assemble(prog); err != nil {
		t.Fatalf("assemble failed: %v", err)
	}
}

func TestAddSubQuickAndMoveQValidation(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{"AddQuickImmediateRange", "ADDQ #9,D0\n", "immediate out of range"},
		{"SubQuickByteToAn", "SUBQ.B #1,A0\n", "not allowed"},
		{"MoveQImmediateRange", "MOVEQ #200,D0\n", "immediate out of range"},
		{"AndAddressRegisterSource", "AND.W A0,D1\n", "address register source"},
		{"NotAddressRegisterDestination", "NOT.W A0\n", "address register destination"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := asm.Parse(strings.NewReader(tt.src))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if _, err := asm.Assemble(prog); err == nil {
				t.Fatalf("expected error but assembly succeeded")
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewInstructionValidation(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{"NegxAddressRegister", "NEGX.W A0\n", "address register"},
		{"NegxImmediate", "NEGX #1\n", "data alterable EA"},
		{"TstImmediate", "TST #1\n", "immediate"},
		{"TstAddressRegister", "TST.L A1\n", "address register"},
		{"JsrRequiresControlEA", "JSR D0\n", "control addressing mode"},
		{"PeaRequiresControlEA", "PEA D0\n", "control addressing mode"},
		{"LinkImmediateRange", "LINK A6,#70000\n", "out of range"},
		{"AddImmediateAddressRegister", "ADDI.B #1,A0\n", "data alterable"},
		{"ChkImmediateInvalid", "CHK #1,D0\n", "immediate source"},
		{"NbcdDestinationRequired", "NBCD (A0)\n", "predecrement address"},
		{"SetConditionImmediate", "SNE #1\n", "data alterable"},
		{"CompareImmediateToAddress", "CMPI.B #1,A0\n", "data alterable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := asm.Parse(strings.NewReader(tt.src))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if _, err := asm.Assemble(prog); err == nil {
				t.Fatalf("expected error but assembly succeeded")
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestOrgSetsOriginAndFillsGaps(t *testing.T) {
	src := ".org 4\n.byte 1\n.org 8\n.byte 2\n"
	prog, err := asm.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if prog.Origin != 4 {
		t.Fatalf("origin not recorded: got %d want 4", prog.Origin)
	}

	out, err := asm.Assemble(prog)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}
	want := []byte{0x01, 0x00, 0x00, 0x00, 0x02}
	if !bytes.Equal(out, want) {
		t.Fatalf("unexpected output: got %x want %x", out, want)
	}
}
