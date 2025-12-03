package instructions

import "fmt"

func init() {
	registerInstrDef(&defMOVEM)
}

var defMOVEM = InstrDef{
	Mnemonic: "MOVEM",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_EA, OPK_RegList},
			Validate:    validateMovemLoad,
			Steps: []EmitStep{
				{WordBits: 0x4C80, Fields: []FieldRef{F_MovemSize, F_SrcEA}},
				{Trailer: []TrailerItem{T_DstRegMask, T_SrcEAExt}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_RegList, OPK_EA},
			Validate:    validateMovemStore,
			Steps: []EmitStep{
				{WordBits: 0x4880, Fields: []FieldRef{F_MovemSize, F_DstEA}},
				{Trailer: []TrailerItem{T_SrcRegMask, T_DstEAExt}},
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
