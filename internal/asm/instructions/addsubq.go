package instructions

import "fmt"

func init() {
	registerInstrDef(&DefADDQ)
	registerInstrDef(&DefSUBQ)
}

var DefADDQ = InstrDef{
	Mnemonic: "ADDQ",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_ImmQuick, OPK_EA},
			Validate:    func(a *Args) error { return validateAddSubQuick("ADDQ", a) },
			Steps: []EmitStep{
				{WordBits: 0x5000, Fields: []FieldRef{F_QuickData, F_SizeBits, F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

var DefSUBQ = InstrDef{
	Mnemonic: "SUBQ",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_ImmQuick, OPK_EA},
			Validate:    func(a *Args) error { return validateAddSubQuick("SUBQ", a) },
			Steps: []EmitStep{
				{WordBits: 0x5100, Fields: []FieldRef{F_QuickData, F_SizeBits, F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

func validateAddSubQuick(name string, a *Args) error {
	if a.Src.Imm < 1 || a.Src.Imm > 8 {
		return fmt.Errorf("%s immediate out of range: %d", name, a.Src.Imm)
	}
	if a.Dst.Kind == EAkNone {
		return fmt.Errorf("%s requires destination", name)
	}
	if isPCRelativeKind(a.Dst.Kind) || a.Dst.Kind == EAkImm {
		return fmt.Errorf("%s requires alterable destination", name)
	}
	if a.Dst.Kind == EAkAn && a.Size == SZ_B {
		return fmt.Errorf("%s.B not allowed for address register", name)
	}
	return nil
}
