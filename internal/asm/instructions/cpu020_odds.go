package instructions

import "fmt"

// This file adds the remaining 68020 instructions scoped for milestone
// 8: PACK, UNPK, CAS, CALLM, and RTM. Encodings were cross-checked
// against GNU binutils' GAS m68k opcode table (opcodes/m68k-opc.c) and
// its architecture-bit header (include/opcode/m68k.h), per the same
// discipline used throughout this project.
//
// CAS2 is deliberately NOT included: its six operands (two compare
// registers, two update registers, two memory pointers) go well beyond
// what even this file's new three-operand Aux mechanism supports, and
// generalizing further for one rare instruction isn't justified — see
// docs/design/cpu-family-support.md.
//
// PACK/UNPK and CAS are tagged m68020up in GAS (68020 and every later
// tier, not CPU32 — unlike milestone 7's Bcc.L/CHK2/EXTB/TRAPcc, GAS
// never tags these cpu32). CALLM/RTM are tagged the single m68020 bit,
// not the m68020up union — confirmed against include/opcode/m68k.h,
// where m68020up = m68020|m68030up|... is a strictly larger set — so
// they are gated on requireM68020Only: real 68030+ silicon dropped
// them, unlike everything else 68020 added.
var require68020 = Target{CPU: CPU68020}

func init() {
	registerInstrDef(newPackUnpkDef("PACK", 0x8140, 0x8148))
	registerInstrDef(newPackUnpkDef("UNPK", 0x8180, 0x8188))
	registerInstrDef(newCasDef())
	registerInstrDef(&defCALLM)
	registerInstrDef(&defRTM)
}

// newPackUnpkDef builds PACK/UNPK's register and predecrement forms.
// Both share ABCD/SBCD's exact register-field layout (bcd.go's
// newBcdDef: dest at FDnReg/FAnReg, source at FSrcDnReg/FSrcAnReg) —
// PACK/UNPK just add a third operand, the #adjustment word, via the new
// Aux mechanism (TAuxImmWord).
func newPackUnpkDef(name string, regBits, memBits uint16) *InstrDef {
	validate := func(a *Args) error { return checkImmediateRange(a.Aux.Imm, WordSize) }
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkDn, OpkDn, OpkImm},
				Validate:    validate,
				Requires:    require68020,
				Steps: []EmitStep{
					{WordBits: regBits, Fields: []FieldRef{FDnReg, FSrcDnReg}},
					{Trailer: []TrailerItem{TAuxImmWord}},
				},
			},
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkPredecAn, OpkPredecAn, OpkImm},
				Validate:    validate,
				Requires:    require68020,
				Steps: []EmitStep{
					{WordBits: memBits, Fields: []FieldRef{FAnReg, FSrcAnReg}},
					{Trailer: []TrailerItem{TAuxImmWord}},
				},
			},
		},
	}
}

// newCasDef builds one size of CAS Dc,Du,<ea>: the compare register (Dc)
// occupies the same bits-2-0 slot as any other "Src" register field
// (FSrcDnReg), the update register (Du) occupies bits 8-6 (FDstReg6 —
// no other instruction in this codebase uses that position), and <ea>
// is the third (Aux) operand, encoded via FAuxEA/TAuxEAExt exactly like
// Src/Dst would be for a two-operand instruction.
func newCasDef() *InstrDef {
	form := func(sz Size, base uint16) FormDef {
		return FormDef{
			DefaultSize: sz,
			Sizes:       []Size{sz},
			OperKinds:   []OperandKind{OpkDn, OpkDn, OpkEA},
			Validate:    validateCas,
			Requires:    require68020,
			Steps: []EmitStep{
				{WordBits: base, Fields: []FieldRef{FAuxEA}},
				{WordBits: 0x0000, Fields: []FieldRef{FDstReg6, FSrcDnReg}},
				{Trailer: []TrailerItem{TAuxEAExt}},
			},
		}
	}
	return &InstrDef{
		Mnemonic: "CAS",
		Forms: []FormDef{
			form(ByteSize, 0x0AC0),
			form(WordSize, 0x0CC0),
			form(LongSize, 0x0EC0),
		},
	}
}

func validateCas(a *Args) error {
	if !memoryAlterableEA[a.Aux.Kind] {
		if a.Aux.Kind == EAkNone {
			return fmt.Errorf("CAS requires a destination address")
		}
		return fmt.Errorf("CAS destination must be memory-alterable")
	}
	return nil
}

var defCALLM = InstrDef{
	Mnemonic: "CALLM",
	Forms: []FormDef{
		{
			// CALLM #vector,<ea>: <ea> must be control-addressable (it's
			// the called module descriptor's address, not data), and the
			// vector is a plain unsigned byte — restricting its Validate
			// range to 0-255 (rather than the usual signed-byte -128..255)
			// sidesteps a real ambiguity in how the trailer word would
			// otherwise need to sign- vs zero-extend a negative value.
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkImm, OpkEA},
			Validate:    validateCallm,
			Requires:    requireM68020Only,
			Steps: []EmitStep{
				{WordBits: 0x06C0, Fields: []FieldRef{FDstEA}},
				{Trailer: []TrailerItem{TImmSized, TDstEAExt}},
			},
		},
	},
}

func validateCallm(a *Args) error {
	if a.Src.Kind != EAkImm || a.Src.Imm < 0 || a.Src.Imm > 255 {
		return fmt.Errorf("CALLM requires an immediate vector in 0-255")
	}
	if !controlAlterableEA[a.Dst.Kind] {
		return fmt.Errorf("CALLM requires control addressing mode")
	}
	return nil
}

var defRTM = InstrDef{
	Mnemonic: "RTM",
	Forms: []FormDef{
		{
			// RTM Rn: Rn is any data or address register — the same
			// "register-specifier nibble" MOVEC/MOVES/CHK2/CMP2 use, here
			// at bits 3-0 instead of bits 15-12. Reusing generic OpkEA plus
			// Validate (rather than two Dn/An-specific forms) matches how
			// MOVEC's own general-register operand is handled.
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkEA},
			Validate:    validateRtm,
			Requires:    requireM68020Only,
			Steps: []EmitStep{
				{WordBits: 0x06C0, Fields: []FieldRef{FSrcRegNibble0}},
			},
		},
	},
}

func validateRtm(a *Args) error {
	if a.Src.Kind != EAkDn && a.Src.Kind != EAkAn {
		return fmt.Errorf("RTM requires a data or address register")
	}
	return nil
}
