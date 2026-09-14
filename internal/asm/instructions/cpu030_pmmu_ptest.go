package instructions

import "fmt"

// This file adds PFLUSH, PFLUSHS, PFLUSHR, PLOADR/PLOADW, and PTESTR/
// PTESTW — the 68030/68851 PMMU cache/TLB management instructions
// beyond PFLUSHA (milestone 12), chosen by the maintainer from the open
// items list. PFLUSHAN/PFLUSHN and PTESTR/PTESTW's own 68040-only
// single-word forms remain out of scope here — the 68040 PMMU interface
// is simplified/re-encoded relative to 68030/68851 in ways that don't
// share these instructions' bit layout at all — see cpu040_pmmu.go.
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
// defPFLUSH is captured in a package-level var, unlike PFLUSHS's own
// newPFlushDef call, so cpu040_pmmu.go can append the 68040's own
// single-word PFLUSH form to defPFLUSH.Forms directly — the same
// direct-variable-reference pattern defPTESTR/defPTESTW use just below.
var defPFLUSH = newPFlushDef("PFLUSH", 0x0000)

func init() {
	registerInstrDef(defPFLUSH)
	registerInstrDef(newPFlushDef("PFLUSHS", 0x0400))
	registerInstrDef(&defPFLUSHR)
	registerInstrDef(newPLoadDef("PLOADR", 0x2200))
	registerInstrDef(newPLoadDef("PLOADW", 0x2000))
	registerInstrDef(defPTESTR)
	registerInstrDef(defPTESTW)
}

// newPFlushDef builds PFLUSH or PFLUSHS: "FC,#mask" and "FC,#mask,<ea>".
// extraBit is 0 for PFLUSH, 0x0400 for PFLUSHS — GAS's own opcode table
// gives PFLUSHS (68851-exclusive; unlike PFLUSH, not available on a bare
// 68030's on-chip PMMU) the identical bit layout as PFLUSH with exactly
// that one extra bit set (PFLUSH's own T3T9 row literal 0x3010 vs
// PFLUSHS's 0x3410, and so on for every row pair) — confirmed by
// decomposing all six PFLUSH/PFLUSHS row pairs, not just the one shown
// here.
func newPFlushDef(name string, extraBit uint16) *InstrDef {
	validateMask := func(a *Args) error {
		if a.Dst.Imm < 0 || a.Dst.Imm > 31 {
			return fmt.Errorf("%s mask must be 0-31", name)
		}
		return nil
	}
	validateMaskAndEA := func(a *Args) error {
		if err := validateMask(a); err != nil {
			return err
		}
		if !memoryAlterableEA[a.Aux.Kind] {
			return fmt.Errorf("%s <ea> must be a memory addressing mode", name)
		}
		return nil
	}
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				// FC,#mask
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkFCSpec, OpkImm},
				Validate:    validateMask,
				Requires:    requirePMMU,
				Steps: []EmitStep{
					{WordBits: 0xF000},
					{WordBits: 0x3000 | extraBit, Fields: []FieldRef{FFCSpecWord, FDstImmShift5}},
				},
			},
			{
				// FC,#mask,<ea>
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkFCSpec, OpkImm, OpkEA},
				Validate:    validateMaskAndEA,
				Requires:    requirePMMU,
				Steps: []EmitStep{
					{WordBits: 0xF000, Fields: []FieldRef{FAuxEA}},
					{WordBits: 0x3800 | extraBit, Fields: []FieldRef{FFCSpecWord, FDstImmShift5}},
					{Trailer: []TrailerItem{TAuxEAExt}},
				},
			},
		},
	}
}

// defPFLUSHR is "PFLUSHR <ea>" — flush by root-pointer descriptor,
// 68851-exclusive. Unlike PFLUSH/PFLUSHS, GAS's own row
// (two(0xf000,0xa000), mask two(0xffc0,0xffff)) has a FULLY fixed
// word2: no function-code specifier, no mask, just a plain memory <ea>.
var defPFLUSHR = InstrDef{
	Mnemonic: "PFLUSHR",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkEA},
			Validate:    validatePflushrEA,
			Requires:    requirePMMU,
			Steps: []EmitStep{
				{WordBits: 0xF000, Fields: []FieldRef{FSrcEA}},
				{WordBits: 0xA000},
				{Trailer: []TrailerItem{TSrcEAExt}},
			},
		},
	},
}

func validatePflushrEA(a *Args) error {
	if !memoryAlterableEA[a.Src.Kind] {
		if a.Src.Kind == EAkNone {
			return fmt.Errorf("PFLUSHR requires an operand")
		}
		return fmt.Errorf("PFLUSHR requires a memory addressing mode")
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

// defPTESTR and defPTESTW are captured in package-level vars, unlike
// PLOADR/PLOADW's own newPLoadDef calls, so cpu040_pmmu.go can append
// the 68040's own single-word PTESTR/PTESTW form to defPTESTR.Forms/
// defPTESTW.Forms directly — the same direct-variable-reference pattern
// defPMOVE/defFMOVE/defPFLUSHA already use for exactly this reason.
var defPTESTR = newPTestDef("PTESTR", 0x8200)
var defPTESTW = newPTestDef("PTESTW", 0x8000)

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
