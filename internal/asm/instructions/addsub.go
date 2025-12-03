package instructions

import "fmt"

func init() {
	registerInstrDef(&defADD)
	registerInstrDef(&defSUB)
	registerInstrDef(&defADDQ)
	registerInstrDef(&defSUBQ)
}

var defADD = InstrDef{
	Mnemonic: "ADD",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_EA, OPK_Dn},
			Validate:    func(a *Args) error { return validateAddSubToDn("ADD", a) },
			Steps: []EmitStep{
				{WordBits: 0xD000, Fields: []FieldRef{F_DnReg, F_SizeBits, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_EA},
			Validate:    func(a *Args) error { return validateAddSubDnToEA("ADD", a) },
			Steps: []EmitStep{
				{WordBits: 0xD100, Fields: []FieldRef{F_SrcDnRegHi, F_SizeBits, F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_EA, OPK_An},
			Validate:    func(a *Args) error { return validateAddSubAddr("ADD", a) },
			Steps: []EmitStep{
				{WordBits: 0xD0C0, Fields: []FieldRef{F_AddaSize, F_AnReg, F_SrcEA}},
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
			Validate:    func(a *Args) error { return validateAddSubToDn("SUB", a) },
			Steps: []EmitStep{
				{WordBits: 0x9000, Fields: []FieldRef{F_DnReg, F_SizeBits, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_EA},
			Validate:    func(a *Args) error { return validateAddSubDnToEA("SUB", a) },
			Steps: []EmitStep{
				{WordBits: 0x9100, Fields: []FieldRef{F_SrcDnRegHi, F_SizeBits, F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_EA, OPK_An},
			Validate:    func(a *Args) error { return validateAddSubAddr("SUB", a) },
			Steps: []EmitStep{
				{WordBits: 0x90C0, Fields: []FieldRef{F_AddaSize, F_AnReg, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
			},
		},
	},
}

func validateAddSubToDn(name string, a *Args) error {
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

func validateAddSubDnToEA(name string, a *Args) error {
	if a.Src.Kind != EAkDn || a.Dst.Kind == EAkNone {
		return fmt.Errorf("%s requires Dn source", name)
	}
	switch a.Dst.Kind {
	case EAkAddrInd, EAkAddrPostinc, EAkAddrDisp16, EAkAddrPredec, EAkIdxAnBrief, EAkAbsW, EAkAbsL:
		return nil
	default:
		return fmt.Errorf("%s destination must be memory alterable EA", name)
	}
}

func validateAddSubAddr(name string, a *Args) error {
	if a.Src.Kind == EAkNone || a.Dst.Kind != EAkAn {
		return fmt.Errorf("%s requires An destination", name)
	}
	if a.Size == SZ_B {
		return fmt.Errorf("%s does not support byte size for address destination", name)
	}
	if a.Src.Kind == EAkImm {
		if err := checkImmediateRange(a.Src.Imm, a.Size); err != nil {
			return err
		}
	}
	return nil
}

var defADDQ = InstrDef{
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

var defSUBQ = InstrDef{
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
