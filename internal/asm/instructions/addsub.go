package instructions

import "fmt"

func init() {
	registerInstrDef(&defADD)
	registerInstrDef(&defSUB)
}

var defADD = InstrDef{
	Mnemonic: "ADD",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_EA, OPK_Dn},
			Validate:    func(a *Args) error { return validateAddSub("ADD", a) },
			Steps: []EmitStep{
				{WordBits: 0xD000, Fields: []FieldRef{F_DnReg, F_SizeBits, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
			},
		},
	},
}

var defSUB = InstrDef{
	Mnemonic: "SUB",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_EA, OPK_Dn},
			Validate:    func(a *Args) error { return validateAddSub("SUB", a) },
			Steps: []EmitStep{
				{WordBits: 0x9000, Fields: []FieldRef{F_DnReg, F_SizeBits, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
			},
		},
	},
}

func validateAddSub(name string, a *Args) error {
	if a.Src.Kind == EAkNone || a.Dst.Kind != EAkDn {
		return fmt.Errorf("%s requires Dn destination", name)
	}
	if a.Size == SZ_B && a.Src.Kind == EAkAn {
		return fmt.Errorf("%s.B does not allow address register source", name)
	}
	if a.Src.Kind == EAkImm {
		if err := checkImmediateRange(a.Src.Imm, a.Size); err != nil {
			return err
		}
	}
	return nil
}
