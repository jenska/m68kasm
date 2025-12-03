package instructions

import "fmt"

func init() {
	registerInstrDef(&defNEGX)
	registerInstrDef(&defSUBX)
	registerInstrDef(&defADDX)
}

var defNEGX = InstrDef{
	Mnemonic: "NEGX",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateNegx,
			Steps: []EmitStep{
				{WordBits: 0x4000, Fields: []FieldRef{F_SizeBits, F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

var defSUBX = InstrDef{
	Mnemonic: "SUBX",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_Dn},
			Validate:    func(a *Args) error { return validateAddSubX("SUBX", a, false) },
			Steps: []EmitStep{
				{WordBits: 0x9100, Fields: []FieldRef{F_DnReg, F_SizeBits, F_SrcDnReg}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_PredecAn, OPK_PredecAn},
			Validate:    func(a *Args) error { return validateAddSubX("SUBX", a, true) },
			Steps: []EmitStep{
				{WordBits: 0x9108, Fields: []FieldRef{F_AnReg, F_SizeBits, F_SrcAnReg}},
			},
		},
	},
}

var defADDX = InstrDef{
	Mnemonic: "ADDX",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_Dn},
			Validate:    func(a *Args) error { return validateAddSubX("ADDX", a, false) },
			Steps: []EmitStep{
				{WordBits: 0xD100, Fields: []FieldRef{F_DnReg, F_SizeBits, F_SrcDnReg}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_PredecAn, OPK_PredecAn},
			Validate:    func(a *Args) error { return validateAddSubX("ADDX", a, true) },
			Steps: []EmitStep{
				{WordBits: 0xD108, Fields: []FieldRef{F_AnReg, F_SizeBits, F_SrcAnReg}},
			},
		},
	},
}

func validateNegx(a *Args) error {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
		a.Src = EAExpr{}
	}
	switch a.Dst.Kind {
	case EAkNone:
		return fmt.Errorf("NEGX requires destination")
	case EAkImm, EAkPCDisp16, EAkIdxPCBrief:
		return fmt.Errorf("NEGX destination must be data alterable EA")
	case EAkAn:
		return fmt.Errorf("NEGX does not allow address register destination")
	default:
		return nil
	}
}

func validateAddSubX(name string, a *Args, predec bool) error {
	if a.Src.Kind == EAkNone || a.Dst.Kind == EAkNone {
		return fmt.Errorf("%s requires both source and destination", name)
	}
	if predec {
		if a.Src.Kind != EAkAddrPredec || a.Dst.Kind != EAkAddrPredec {
			return fmt.Errorf("%s requires predecrement addressing for memory form", name)
		}
		return nil
	}
	if a.Src.Kind != EAkDn || a.Dst.Kind != EAkDn {
		return fmt.Errorf("%s requires Dn registers for register form", name)
	}
	return nil
}
