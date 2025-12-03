package instructions

import "fmt"

func init() {
	registerInstrDef(&DefMULU)
	registerInstrDef(&DefMULS)
	registerInstrDef(&DefDIVU)
	registerInstrDef(&DefDIVS)
}

var (
	DefDIVU = newDivMulDef("DIVU", 0x80C0)
	DefDIVS = newDivMulDef("DIVS", 0x81C0)
	DefMULU = newDivMulDef("MULU", 0xC0C0)
	DefMULS = newDivMulDef("MULS", 0xC1C0)
)

func newDivMulDef(name string, wordBits uint16) InstrDef {
	return InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: SZ_W,
				Sizes:       []Size{SZ_W},
				OperKinds:   []OperandKind{OPK_EA, OPK_Dn},
				Validate:    func(a *Args) error { return validateDivMul(name, a) },
				Steps: []EmitStep{
					{WordBits: wordBits, Fields: []FieldRef{F_DnReg, F_SrcEA}},
					{Trailer: []TrailerItem{T_SrcEAExt}},
				},
			},
		},
	}
}

func validateDivMul(name string, a *Args) error {
	if a.Dst.Kind != EAkDn {
		return fmt.Errorf("%s destination must be data register", name)
	}
	if a.Size != SZ_W {
		return fmt.Errorf("%s operates on word size", name)
	}
	if a.Src.Kind == EAkImm {
		return fmt.Errorf("%s does not allow immediate source", name)
	}
	if a.Src.Kind == EAkAn {
		return fmt.Errorf("%s does not allow address register source", name)
	}
	return nil
}
