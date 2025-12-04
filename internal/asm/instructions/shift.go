package instructions

import "fmt"

func init() {
	registerInstrDef(&defASR)
	registerInstrDef(&defASL)
	registerInstrDef(&defLSR)
	registerInstrDef(&defLSL)
	registerInstrDef(&defROXR)
	registerInstrDef(&defROXL)
	registerInstrDef(&defROR)
	registerInstrDef(&defROL)
}

var defASR = InstrDef{
	Mnemonic: "ASR",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_ImmQuick, OPK_Dn},
			Validate:    validateShiftImmediate,
			Steps: []EmitStep{
				{WordBits: 0xE000, Fields: []FieldRef{F_QuickData, F_SizeBits, F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_Dn},
			Validate:    validateShiftRegister,
			Steps: []EmitStep{
				{WordBits: 0xE020, Fields: []FieldRef{F_SrcDnRegHi, F_SizeBits, F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateShiftMemory,
			Steps: []EmitStep{
				{WordBits: 0xE0C0, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

var defASL = InstrDef{
	Mnemonic: "ASL",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_ImmQuick, OPK_Dn},
			Validate:    validateShiftImmediate,
			Steps: []EmitStep{
				{WordBits: 0xE100, Fields: []FieldRef{F_QuickData, F_SizeBits, F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_Dn},
			Validate:    validateShiftRegister,
			Steps: []EmitStep{
				{WordBits: 0xE120, Fields: []FieldRef{F_SrcDnRegHi, F_SizeBits, F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateShiftMemory,
			Steps: []EmitStep{
				{WordBits: 0xE1C0, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

var defLSR = InstrDef{
	Mnemonic: "LSR",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_ImmQuick, OPK_Dn},
			Validate:    validateShiftImmediate,
			Steps: []EmitStep{
				{WordBits: 0xE008, Fields: []FieldRef{F_QuickData, F_SizeBits, F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_Dn},
			Validate:    validateShiftRegister,
			Steps: []EmitStep{
				{WordBits: 0xE028, Fields: []FieldRef{F_SrcDnRegHi, F_SizeBits, F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateShiftMemory,
			Steps: []EmitStep{
				{WordBits: 0xE2C0, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

var defLSL = InstrDef{
	Mnemonic: "LSL",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_ImmQuick, OPK_Dn},
			Validate:    validateShiftImmediate,
			Steps: []EmitStep{
				{WordBits: 0xE108, Fields: []FieldRef{F_QuickData, F_SizeBits, F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_Dn},
			Validate:    validateShiftRegister,
			Steps: []EmitStep{
				{WordBits: 0xE128, Fields: []FieldRef{F_SrcDnRegHi, F_SizeBits, F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateShiftMemory,
			Steps: []EmitStep{
				{WordBits: 0xE3C0, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

var defROXR = InstrDef{
	Mnemonic: "ROXR",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_ImmQuick, OPK_Dn},
			Validate:    validateShiftImmediate,
			Steps: []EmitStep{
				{WordBits: 0xE010, Fields: []FieldRef{F_QuickData, F_SizeBits, F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_Dn},
			Validate:    validateShiftRegister,
			Steps: []EmitStep{
				{WordBits: 0xE030, Fields: []FieldRef{F_SrcDnRegHi, F_SizeBits, F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateShiftMemory,
			Steps: []EmitStep{
				{WordBits: 0xE4C0, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

var defROXL = InstrDef{
	Mnemonic: "ROXL",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_ImmQuick, OPK_Dn},
			Validate:    validateShiftImmediate,
			Steps: []EmitStep{
				{WordBits: 0xE110, Fields: []FieldRef{F_QuickData, F_SizeBits, F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_Dn},
			Validate:    validateShiftRegister,
			Steps: []EmitStep{
				{WordBits: 0xE130, Fields: []FieldRef{F_SrcDnRegHi, F_SizeBits, F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateShiftMemory,
			Steps: []EmitStep{
				{WordBits: 0xE5C0, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

var defROR = InstrDef{
	Mnemonic: "ROR",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_ImmQuick, OPK_Dn},
			Validate:    validateShiftImmediate,
			Steps: []EmitStep{
				{WordBits: 0xE018, Fields: []FieldRef{F_QuickData, F_SizeBits, F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_Dn},
			Validate:    validateShiftRegister,
			Steps: []EmitStep{
				{WordBits: 0xE038, Fields: []FieldRef{F_SrcDnRegHi, F_SizeBits, F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateShiftMemory,
			Steps: []EmitStep{
				{WordBits: 0xE6C0, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

var defROL = InstrDef{
	Mnemonic: "ROL",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_ImmQuick, OPK_Dn},
			Validate:    validateShiftImmediate,
			Steps: []EmitStep{
				{WordBits: 0xE118, Fields: []FieldRef{F_QuickData, F_SizeBits, F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_Dn},
			Validate:    validateShiftRegister,
			Steps: []EmitStep{
				{WordBits: 0xE138, Fields: []FieldRef{F_SrcDnRegHi, F_SizeBits, F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateShiftMemory,
			Steps: []EmitStep{
				{WordBits: 0xE7C0, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

func validateShiftImmediate(a *Args) error {
	if !a.HasImmQuick {
		return fmt.Errorf("shift requires immediate count")
	}
	if a.Src.Imm < 1 || a.Src.Imm > 8 {
		return fmt.Errorf("shift count out of range: %d", a.Src.Imm)
	}
	if a.Dst.Kind != EAkDn {
		return fmt.Errorf("shift destination must be data register")
	}
	return nil
}

func validateShiftRegister(a *Args) error {
	if a.Src.Kind != EAkDn || a.Dst.Kind != EAkDn {
		return fmt.Errorf("shift with register count requires Dn source and destination")
	}
	return nil
}

func validateShiftMemory(a *Args) error {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
		a.Src = EAExpr{}
	}
	if a.Src.Kind != EAkNone {
		return fmt.Errorf("memory shift takes single operand")
	}
	switch a.Dst.Kind {
	case EAkAddrInd, EAkAddrPostinc, EAkAddrDisp16, EAkAddrPredec, EAkIdxAnBrief, EAkAbsW, EAkAbsL:
		return nil
	case EAkNone:
		return fmt.Errorf("shift requires destination")
	default:
		return fmt.Errorf("memory shift requires alterable memory EA")
	}
}
