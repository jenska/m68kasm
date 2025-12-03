package instructions

import "fmt"

func init() {
	registerInstrDef(&defBSET)
	registerInstrDef(&defBCLR)
	registerInstrDef(&defBCHG)
}

var defBSET = InstrDef{
	Mnemonic: "BSET",
	Forms: []FormDef{
		{
			DefaultSize: SZ_L,
			Sizes:       []Size{SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_EA},
			Validate:    func(a *Args) error { return validateBitReg("BSET", a) },
			Steps: []EmitStep{
				{WordBits: 0x01C0, Fields: []FieldRef{F_SrcDnRegHi, F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
		{
			DefaultSize: SZ_B,
			Sizes:       []Size{SZ_B},
			OperKinds:   []OperandKind{OPK_Imm, OPK_EA},
			Validate:    func(a *Args) error { return validateBitImm("BSET", a) },
			Steps: []EmitStep{
				{WordBits: 0x08C0, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt, T_SrcImm}},
			},
		},
	},
}

var defBCLR = InstrDef{
	Mnemonic: "BCLR",
	Forms: []FormDef{
		{
			DefaultSize: SZ_L,
			Sizes:       []Size{SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_EA},
			Validate:    func(a *Args) error { return validateBitReg("BCLR", a) },
			Steps: []EmitStep{
				{WordBits: 0x0180, Fields: []FieldRef{F_SrcDnRegHi, F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
		{
			DefaultSize: SZ_B,
			Sizes:       []Size{SZ_B},
			OperKinds:   []OperandKind{OPK_Imm, OPK_EA},
			Validate:    func(a *Args) error { return validateBitImm("BCLR", a) },
			Steps: []EmitStep{
				{WordBits: 0x0880, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt, T_SrcImm}},
			},
		},
	},
}

var defBCHG = InstrDef{
	Mnemonic: "BCHG",
	Forms: []FormDef{
		{
			DefaultSize: SZ_L,
			Sizes:       []Size{SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_EA},
			Validate:    func(a *Args) error { return validateBitReg("BCHG", a) },
			Steps: []EmitStep{
				{WordBits: 0x0140, Fields: []FieldRef{F_SrcDnRegHi, F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
		{
			DefaultSize: SZ_B,
			Sizes:       []Size{SZ_B},
			OperKinds:   []OperandKind{OPK_Imm, OPK_EA},
			Validate:    func(a *Args) error { return validateBitImm("BCHG", a) },
			Steps: []EmitStep{
				{WordBits: 0x0840, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt, T_SrcImm}},
			},
		},
	},
}

func validateBitReg(name string, a *Args) error {
	if a.Src.Kind != EAkDn {
		return fmt.Errorf("%s requires Dn source", name)
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

func validateBitImm(name string, a *Args) error {
	if a.Src.Kind != EAkImm {
		return fmt.Errorf("%s requires immediate source", name)
	}
	if err := checkImmediateRange(a.Src.Imm, SZ_B); err != nil {
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
