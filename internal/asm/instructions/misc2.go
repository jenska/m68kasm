package instructions

import "fmt"

func init() {
	registerInstrDef(&defCLR)
	registerInstrDef(&defCMPM)
	registerInstrDef(&defEXG)
	registerInstrDef(&defEXT)
	registerInstrDef(&defILLEGAL)
}

var defCLR = InstrDef{
	Mnemonic: "CLR",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateDataAlterable("CLR"),
			Steps: []EmitStep{
				{WordBits: 0x4200, Fields: []FieldRef{F_SizeBits, F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

var defCMPM = InstrDef{
	Mnemonic: "CMPM",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_EA, OPK_EA},
			Validate:    validateCMPM,
			Steps: []EmitStep{
				{WordBits: 0xB108, Fields: []FieldRef{F_AnReg, F_SizeBits, F_SrcAnReg}},
			},
		},
	},
}

var defEXG = InstrDef{
	Mnemonic: "EXG",
	Forms: []FormDef{
		{
			DefaultSize: SZ_L,
			Sizes:       []Size{SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_Dn},
			Validate:    validateEXGData,
			Steps: []EmitStep{
				{WordBits: 0xC140, Fields: []FieldRef{F_DnReg, F_SrcDnReg}},
			},
		},
		{
			DefaultSize: SZ_L,
			Sizes:       []Size{SZ_L},
			OperKinds:   []OperandKind{OPK_An, OPK_An},
			Validate:    validateEXGAddr,
			Steps: []EmitStep{
				{WordBits: 0xC148, Fields: []FieldRef{F_AnReg, F_SrcAnReg}},
			},
		},
		{
			DefaultSize: SZ_L,
			Sizes:       []Size{SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_An},
			Validate:    validateEXGMixed,
			Steps: []EmitStep{
				{WordBits: 0xC188, Fields: []FieldRef{F_SrcDnRegHi, F_DstRegLow}},
			},
		},
	},
}

var defEXT = InstrDef{
	Mnemonic: "EXT",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_Dn},
			Validate:    validateEXT,
			Steps: []EmitStep{
				{WordBits: 0x4800, Fields: []FieldRef{F_SizeBits, F_DstRegLow}},
			},
		},
	},
}

var defILLEGAL = InstrDef{
	Mnemonic: "ILLEGAL",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{},
			Validate:    nil,
			Steps:       []EmitStep{{WordBits: 0x4AFC}},
		},
	},
}

func validateDataAlterable(name string) func(*Args) error {
	return func(a *Args) error {
		if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
			a.Dst = a.Src
			a.Src = EAExpr{}
		}
		if a.Dst.Kind == EAkNone {
			return fmt.Errorf("%s requires destination", name)
		}
		switch a.Dst.Kind {
		case EAkImm, EAkPCDisp16, EAkIdxPCBrief, EAkAn:
			return fmt.Errorf("%s destination must be data alterable EA", name)
		default:
			return nil
		}
	}
}

func validateCMPM(a *Args) error {
	if a.Src.Kind != EAkAddrPostinc || a.Dst.Kind != EAkAddrPostinc {
		return fmt.Errorf("CMPM requires post-increment address operands")
	}
	return nil
}

func validateEXGData(a *Args) error {
	if a.Src.Kind != EAkDn || a.Dst.Kind != EAkDn {
		return fmt.Errorf("EXG requires data registers")
	}
	return nil
}

func validateEXGAddr(a *Args) error {
	if a.Src.Kind != EAkAn || a.Dst.Kind != EAkAn {
		return fmt.Errorf("EXG requires address registers")
	}
	return nil
}

func validateEXGMixed(a *Args) error {
	if a.Src.Kind != EAkDn || a.Dst.Kind != EAkAn {
		return fmt.Errorf("EXG mixed form requires Dn source and An destination")
	}
	return nil
}

func validateEXT(a *Args) error {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
		a.Src = EAExpr{}
	}
	if a.Dst.Kind != EAkDn {
		return fmt.Errorf("EXT requires Dn destination")
	}
	if a.Size == SZ_B {
		return fmt.Errorf("EXT does not support byte size")
	}
	return nil
}
