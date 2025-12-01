package instructions

import "fmt"

func init() {
	registerInstrDef(&DefMUL)
	registerInstrDef(&DefDIV)
}

var DefDIV = InstrDef{
	Mnemonic: "DIV",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_EA, OPK_Dn},
			Validate:    func(a *Args) error { return validateDivMul("DIV", a) },
			Steps: []EmitStep{
				{WordBits: 0x80C0, Fields: []FieldRef{F_DnReg, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt}},
			},
		},
	},
}

var DefMUL = InstrDef{
	Mnemonic: "MUL",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_EA, OPK_Dn},
			Validate:    func(a *Args) error { return validateDivMul("MUL", a) },
			Steps: []EmitStep{
				{WordBits: 0xC1C0, Fields: []FieldRef{F_DnReg, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt}},
			},
		},
	},
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
