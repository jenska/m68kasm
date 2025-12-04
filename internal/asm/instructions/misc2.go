package instructions

import "fmt"

func init() {
	registerInstrDef(&defCLR)
	registerInstrDef(&defCHK)
	registerInstrDef(&defEXG)
	registerInstrDef(&defEXT)
	registerInstrDef(&defILLEGAL)
}

var defCHK = InstrDef{
	Mnemonic: "CHK",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OPK_EA, OPK_Dn},
			Validate:    validateCHK,
			Steps: []EmitStep{
				{WordBits: 0x4180, Fields: []FieldRef{F_DnReg, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt}},
			},
		},
	},
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

var defCLR = InstrDef{
	Mnemonic: "CLR",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateDataAlterable("CLR"),
			Steps: []EmitStep{
				{WordBits: 0x4200, Fields: []FieldRef{F_SizeBits, F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

var defEXG = InstrDef{Mnemonic: "EXG", Forms: newEXGForms()}

var defEXT = InstrDef{
	Mnemonic: "EXT",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize, LongSize},
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
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
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

func newEXGForms() []FormDef {
	forms := []struct {
		operKinds []OperandKind
		validate  func(*Args) error
		wordBits  uint16
		fields    []FieldRef
	}{
		{[]OperandKind{OPK_Dn, OPK_Dn}, validateEXGData, 0xC140, []FieldRef{F_DnReg, F_SrcDnReg}},
		{[]OperandKind{OPK_An, OPK_An}, validateEXGAddr, 0xC148, []FieldRef{F_AnReg, F_SrcAnReg}},
		{[]OperandKind{OPK_Dn, OPK_An}, validateEXGMixed, 0xC188, []FieldRef{F_SrcDnRegHi, F_DstRegLow}},
	}

	formDefs := make([]FormDef, 0, len(forms))
	for _, f := range forms {
		formDefs = append(formDefs, FormDef{
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   f.operKinds,
			Validate:    f.validate,
			Steps:       []EmitStep{{WordBits: f.wordBits, Fields: f.fields}},
		})
	}

	return formDefs
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
	if a.Size == ByteSize {
		return fmt.Errorf("EXT does not support byte size")
	}
	return nil
}
