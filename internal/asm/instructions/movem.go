package instructions

import "fmt"

func init() {
	registerInstrDef(&defMOVEM)
}

var defMOVEM = InstrDef{
	Mnemonic: "MOVEM",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize, LongSize},
			OperKinds:   []OperandKind{OpkEA, OpkRegList},
			Validate:    validateMovemLoad,
			Steps: []EmitStep{
				{WordBits: 0x4C80, Fields: []FieldRef{FMovemSize, FSrcEA}},
				{Trailer: []TrailerItem{TDstRegMask, TSrcEAExt}},
			},
		},
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize, LongSize},
			OperKinds:   []OperandKind{OpkRegList, OpkEA},
			Validate:    validateMovemStore,
			Steps: []EmitStep{
				{WordBits: 0x4880, Fields: []FieldRef{FMovemSize, FDstEA}},
				{Trailer: []TrailerItem{TSrcRegMask, TDstEAExt}},
			},
		},
	},
}

func validateMovemLoad(a *Args) error {
	if a.RegMaskDst == 0 {
		return fmt.Errorf("MOVEM requires register list destination")
	}
	if a.Src.Kind == EAkImm {
		return fmt.Errorf("MOVEM cannot use immediate source")
	}
	if a.Src.Kind == EAkNone {
		return fmt.Errorf("MOVEM requires source EA")
	}
	switch a.Src.Kind {
	case EAkPCDisp16, EAkIdxPCBrief, EAkAddrInd, EAkAddrPostinc, EAkAddrDisp16, EAkAddrPredec, EAkIdxAnBrief, EAkAbsW, EAkAbsL:
		return nil
	default:
		return fmt.Errorf("MOVEM source must be memory or PC-relative")
	}
}

func validateMovemStore(a *Args) error {
	if a.RegMaskSrc == 0 {
		return fmt.Errorf("MOVEM requires register list source")
	}
	switch a.Dst.Kind {
	case EAkAddrInd, EAkAddrPostinc, EAkAddrDisp16, EAkAddrPredec, EAkIdxAnBrief, EAkAbsW, EAkAbsL:
		return nil
	default:
		return fmt.Errorf("MOVEM destination must be memory alterable EA")
	}
}
