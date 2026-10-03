package instructions

import "fmt"

func init() {
	registerInstrDef(newDivMulDef("MULU", 0xC0C0, newMulLongForm("MULU", 0x0000)))
	registerInstrDef(newDivMulDef("MULS", 0xC1C0, newMulLongForm("MULS", 0x0800)))
	registerInstrDef(newDivMulDef("DIVU", 0x80C0, newDivLongForm("DIVU", 0x0000, true)))
	registerInstrDef(newDivMulDef("DIVS", 0x81C0, newDivLongForm("DIVS", 0x0800, true)))
}

// newDivMulDef builds the 68000's 16-bit MULU/MULS/DIVU/DIVS plus the
// CPU32/68020+ long form.
func newDivMulDef(name string, wordBits uint16, long FormDef) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkEA, OpkDn},
				Validate:    func(a *Args) error { return validateDivMul(name, a) },
				Steps: []EmitStep{
					{WordBits: wordBits, Fields: []FieldRef{FDnReg, FSrcEA}},
					{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
				},
			},
			long,
		},
	}
}

// newMulLongForm builds MULU.L/MULS.L <ea>,Dl (32x32 -> 32) and
// <ea>,Dh:Dl (32x32 -> 64): 0100 1100 00 eeeeee, extension word
// 0lll sz00 0000 0hhh with s = signBit (0x0800) and z = 64-bit product.
func newMulLongForm(name string, signBit uint16) FormDef {
	return FormDef{
		DefaultSize: LongSize,
		Sizes:       []Size{LongSize},
		OperKinds:   []OperandKind{OpkEA, OpkRegPair},
		Validate:    func(a *Args) error { return validateDiv(name, a) },
		Requires:    requireDiv,
		Steps: []EmitStep{
			{WordBits: 0x4C00, Fields: []FieldRef{FSrcEA}},
			{WordBits: signBit, Fields: []FieldRef{FDivWide, FDivRegQ, FMulRegH}},
			{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
		},
	}
}

func validateDivMul(name string, a *Args) error {
	if a.Size != WordSize {
		return fmt.Errorf("%s operates on word size", name)
	}
	if a.Src.Kind == EAkAn {
		return fmt.Errorf("%s does not allow address register source", name)
	}
	return nil
}
