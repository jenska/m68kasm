package instructions

import "fmt"

func init() {
	registerInstrDef(&defADDI)
	registerInstrDef(&defSUBI)
	registerInstrDef(&defADDA)
	registerInstrDef(&defSUBA)
	registerInstrDef(&defCHK)
	registerInstrDef(&defNEG)
	registerInstrDef(&defNBCD)
}

var defADDI = InstrDef{
	Mnemonic: "ADDI",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_Imm, OPK_EA},
			Validate:    func(a *Args) error { return validateAddSubImm("ADDI", a) },
			Steps: []EmitStep{
				{WordBits: 0x0600, Fields: []FieldRef{F_SizeBits, F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt, T_SrcImm}},
			},
		},
	},
}

var defSUBI = InstrDef{
	Mnemonic: "SUBI",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_Imm, OPK_EA},
			Validate:    func(a *Args) error { return validateAddSubImm("SUBI", a) },
			Steps: []EmitStep{
				{WordBits: 0x0400, Fields: []FieldRef{F_SizeBits, F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt, T_SrcImm}},
			},
		},
	},
}

var defADDA = InstrDef{
	Mnemonic: "ADDA",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_EA, OPK_An},
			Validate:    func(a *Args) error { return validateAddSubAddr("ADDA", a) },
			Steps: []EmitStep{
				{WordBits: 0xD0C0, Fields: []FieldRef{F_AddaSize, F_AnReg, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
			},
		},
	},
}

var defSUBA = InstrDef{
	Mnemonic: "SUBA",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_EA, OPK_An},
			Validate:    func(a *Args) error { return validateAddSubAddr("SUBA", a) },
			Steps: []EmitStep{
				{WordBits: 0x90C0, Fields: []FieldRef{F_AddaSize, F_AnReg, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
			},
		},
	},
}

var defCHK = InstrDef{
	Mnemonic: "CHK",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_EA, OPK_Dn},
			Validate:    validateCHK,
			Steps: []EmitStep{
				{WordBits: 0x4180, Fields: []FieldRef{F_DnReg, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt}},
			},
		},
	},
}

var defNEG = InstrDef{
	Mnemonic: "NEG",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateNeg,
			Steps: []EmitStep{
				{WordBits: 0x4400, Fields: []FieldRef{F_SizeBits, F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

var defNBCD = InstrDef{
	Mnemonic: "NBCD",
	Forms: []FormDef{
		{
			DefaultSize: SZ_B,
			Sizes:       []Size{SZ_B},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateNbcd,
			Steps: []EmitStep{
				{WordBits: 0x4800, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

func validateAddSubImm(name string, a *Args) error {
	if a.Src.Kind != EAkImm {
		return fmt.Errorf("%s requires immediate source", name)
	}
	if err := checkImmediateRange(a.Src.Imm, a.Size); err != nil {
		return err
	}
	switch a.Dst.Kind {
	case EAkDn, EAkAddrInd, EAkAddrPostinc, EAkAddrPredec, EAkAddrDisp16, EAkIdxAnBrief, EAkAbsW, EAkAbsL:
		return nil
	case EAkNone:
		return fmt.Errorf("%s requires destination", name)
	default:
		return fmt.Errorf("%s destination must be data alterable EA", name)
	}
}

func validateCHK(a *Args) error {
	if a.Src.Kind == EAkNone || a.Dst.Kind != EAkDn {
		return fmt.Errorf("CHK requires Dn destination and source")
	}
	if a.Src.Kind == EAkImm {
		return fmt.Errorf("CHK does not allow immediate source")
	}
	return nil
}

func validateNeg(a *Args) error {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
		a.Src = EAExpr{}
	}
	switch a.Dst.Kind {
	case EAkNone:
		return fmt.Errorf("NEG requires destination")
	case EAkImm, EAkPCDisp16, EAkIdxPCBrief:
		return fmt.Errorf("NEG destination must be data alterable EA")
	case EAkAn:
		return fmt.Errorf("NEG does not allow address register destination")
	default:
		return nil
	}
}

func validateNbcd(a *Args) error {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
		a.Src = EAExpr{}
	}
	switch a.Dst.Kind {
	case EAkDn, EAkAddrPredec:
		return nil
	case EAkNone:
		return fmt.Errorf("NBCD requires destination")
	default:
		return fmt.Errorf("NBCD destination must be Dn or predecrement address")
	}
}
