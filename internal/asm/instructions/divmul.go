package instructions

import "fmt"

func init() {
	registerInstrDef(newDivMulDef("MULU", 0xC0C0))
	registerInstrDef(newDivMulDef("MULS", 0xC1C0))
	registerInstrDef(newDivMulDef("DIVU", 0x80C0))
	registerInstrDef(newDivMulDef("DIVS", 0x81C0))
}

func newDivMulDef(name string, wordBits uint16) *InstrDef {
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
					{Trailer: []TrailerItem{TSrcEAExt}},
				},
			},
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
