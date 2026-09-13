package instructions

import "fmt"

// This file adds PFLUSH, PLOADR/PLOADW, and PTESTR/PTESTW — the
// 68030/68851 PMMU cache/TLB management instructions beyond PFLUSHA
// (milestone 12), chosen by the maintainer from the open items list.
// Deliberately scoped to the 68030|68851 forms only: PFLUSHR/PFLUSHS
// (68851-exclusive alternate flush forms), PFLUSHAN/PFLUSHN (68040's
// own, differently-encoded flush variants), and PTESTR/PTESTW's
// 68040-only single-word forms all remain out of scope, each its own
// future milestone rather than folded in here — the 68040 PMMU
// interface is simplified/re-encoded relative to 68030/68851 in ways
// that don't share these instructions' bit layout at all.
//
// All three share a new operand kind, PFLUSH/PLOAD/PTEST's "function
// code specifier" (OpkFCSpec/EAkFCSpec): GAS accepts this operand in
// three alternate spellings — SFC, DFC, a plain Dn, or "#<imm>" — each
// contributing a 2-bit "mode" marker (00 = named SFC/DFC, 01 = Dn,
// 10 = immediate) at bits 4-3 plus a 3-bit value (the SFC/DFC selector,
// the Dn number, or the immediate itself) at bits 2-0, confirmed
// directly from GAS's own opcode-table literals (0x3000 f-mode/0x3008
// D-mode/0x3010 T-mode all sharing a base) rather than assumed from
// its "case 'f'"/"case 'D'"/"case 'T'" install_operand dispatches
// alone. See FFCSpecWord (encode.go/types.go).
//
// PTEST's full form needs a genuine fourth operand — "PTESTR
// FC,<ea>,#level,An" — the first instruction in this codebase to need
// one; Args gained an Aux2 field for exactly this (types.go), used by
// nothing else.
func init() {
	registerInstrDef(&defPFLUSH)
	registerInstrDef(newPLoadDef("PLOADR", 0x2200))
	registerInstrDef(newPLoadDef("PLOADW", 0x2000))
	registerInstrDef(newPTestDef("PTESTR", 0x8200))
	registerInstrDef(newPTestDef("PTESTW", 0x8000))
}

var defPFLUSH = InstrDef{
	Mnemonic: "PFLUSH",
	Forms: []FormDef{
		{
			// PFLUSH FC,#mask
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkFCSpec, OpkImm},
			Validate:    validatePflushMask,
			Requires:    requirePMMU,
			Steps: []EmitStep{
				{WordBits: 0xF000},
				{WordBits: 0x3000, Fields: []FieldRef{FFCSpecWord, FDstImmShift5}},
			},
		},
		{
			// PFLUSH FC,#mask,<ea>
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkFCSpec, OpkImm, OpkEA},
			Validate:    validatePflushMaskAndEA,
			Requires:    requirePMMU,
			Steps: []EmitStep{
				{WordBits: 0xF000, Fields: []FieldRef{FAuxEA}},
				{WordBits: 0x3800, Fields: []FieldRef{FFCSpecWord, FDstImmShift5}},
				{Trailer: []TrailerItem{TAuxEAExt}},
			},
		},
	},
}

func validatePflushMask(a *Args) error {
	if a.Dst.Imm < 0 || a.Dst.Imm > 31 {
		return fmt.Errorf("PFLUSH mask must be 0-31")
	}
	return nil
}

func validatePflushMaskAndEA(a *Args) error {
	if err := validatePflushMask(a); err != nil {
		return err
	}
	if !memoryAlterableEA[a.Aux.Kind] {
		return fmt.Errorf("PFLUSH <ea> must be a memory addressing mode")
	}
	return nil
}

// newPLoadDef builds PLOADR or PLOADW: "PLOADR/W FC,<ea>". base is the
// word2 literal (0x2200 for PLOADR, 0x2000 for PLOADW — GAS's own R/W
// bit, bit 9) with the function-code mode/value ORed on top by
// FFCSpecWord.
func newPLoadDef(name string, base uint16) *InstrDef {
	validate := func(a *Args) error {
		if !memoryAlterableEA[a.Dst.Kind] {
			if a.Dst.Kind == EAkNone {
				return fmt.Errorf("%s requires a destination", name)
			}
			return fmt.Errorf("%s <ea> must be a memory addressing mode", name)
		}
		return nil
	}
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkFCSpec, OpkEA},
				Validate:    validate,
				Requires:    requirePMMU,
				Steps: []EmitStep{
					{WordBits: 0xF000, Fields: []FieldRef{FDstEA}},
					{WordBits: base, Fields: []FieldRef{FFCSpecWord}},
					{Trailer: []TrailerItem{TDstEAExt}},
				},
			},
		},
	}
}

// newPTestDef builds PTESTR or PTESTW, each with two forms: "FC,<ea>,
// #level" and "FC,<ea>,#level,An" (the optional trailing result
// register). base is the word2 literal (0x8200 for PTESTR, 0x8000 for
// PTESTW — the same R/W bit PLOADR/PLOADW use).
func newPTestDef(name string, base uint16) *InstrDef {
	validateEA := func(a *Args) error {
		if !memoryAlterableEA[a.Dst.Kind] {
			if a.Dst.Kind == EAkNone {
				return fmt.Errorf("%s requires a destination", name)
			}
			return fmt.Errorf("%s <ea> must be a memory addressing mode", name)
		}
		if a.Aux.Imm < 0 || a.Aux.Imm > 7 {
			return fmt.Errorf("%s level must be 0-7", name)
		}
		return nil
	}
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkFCSpec, OpkEA, OpkImm},
				Validate:    validateEA,
				Requires:    requirePMMU,
				Steps: []EmitStep{
					{WordBits: 0xF000, Fields: []FieldRef{FDstEA}},
					{WordBits: base, Fields: []FieldRef{FFCSpecWord, FAuxImmShift10}},
					{Trailer: []TrailerItem{TDstEAExt}},
				},
			},
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkFCSpec, OpkEA, OpkImm, OpkAn},
				Validate:    validateEA,
				Requires:    requirePMMU,
				Steps: []EmitStep{
					{WordBits: 0xF000, Fields: []FieldRef{FDstEA}},
					{WordBits: base, Fields: []FieldRef{FFCSpecWord, FAuxImmShift10, FAux2RegShift5}},
					{Trailer: []TrailerItem{TDstEAExt}},
				},
			},
		},
	}
}
