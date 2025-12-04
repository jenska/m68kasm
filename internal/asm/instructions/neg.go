package instructions

import "fmt"

func init() {
	registerInstrDef(&defNEG)
	registerInstrDef(&defNEGX)
}

var defNEG = InstrDef{
	Mnemonic: "NEG",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateNeg,
			Steps: []EmitStep{
				{WordBits: 0x4400, Fields: []FieldRef{F_SizeBits, F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}
var defNEGX = InstrDef{
	Mnemonic: "NEGX",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateNegx,
			Steps: []EmitStep{
				{WordBits: 0x4000, Fields: []FieldRef{F_SizeBits, F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
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
