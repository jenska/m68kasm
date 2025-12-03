package instructions

import "fmt"

func init() {
	registerInstrDef(&defMOVE)
}

var defMOVE = InstrDef{
	Mnemonic: "MOVE",
	Forms: []FormDef{
		{
			DefaultSize: SZ_L,
			Sizes:       []Size{SZ_L},
			OperKinds:   []OperandKind{OPK_USP, OPK_An},
			Validate:    validateMoveUSP,
			Steps: []EmitStep{
				{WordBits: 0x4E68, Fields: []FieldRef{F_DstRegLow}},
			},
		},
		{
			DefaultSize: SZ_L,
			Sizes:       []Size{SZ_L},
			OperKinds:   []OperandKind{OPK_An, OPK_USP},
			Validate:    validateMoveUSP,
			Steps: []EmitStep{
				{WordBits: 0x4E60, Fields: []FieldRef{F_SrcAnReg}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_EA, OPK_SR},
			Validate:    validateMoveToSR,
			Steps: []EmitStep{
				{WordBits: 0x46C0, Fields: []FieldRef{F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_SR, OPK_EA},
			Validate:    validateMoveFromSR,
			Steps: []EmitStep{
				{WordBits: 0x40C0, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_EA, OPK_EA},
			Validate:    validateMOVE,
			Steps: []EmitStep{
				{WordBits: 0x0000, Fields: []FieldRef{F_MoveSize, F_MoveDestEA, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm, T_DstEAExt}},
			},
		},
	},
}

func validateMOVE(a *Args) error {
	if a.Src.Kind == EAkNone || a.Dst.Kind == EAkNone {
		return fmt.Errorf("MOVE requires source and destination")
	}
	if a.Src.Kind == EAkSR || a.Src.Kind == EAkCCR || a.Src.Kind == EAkUSP || a.Dst.Kind == EAkSR || a.Dst.Kind == EAkCCR || a.Dst.Kind == EAkUSP {
		return fmt.Errorf("MOVE special registers require dedicated forms")
	}
	if a.Dst.Kind == EAkImm {
		return fmt.Errorf("MOVE destination cannot be immediate")
	}
	if isPCRelativeKind(a.Dst.Kind) {
		return fmt.Errorf("MOVE destination cannot be PC-relative")
	}
	if a.Size == SZ_B {
		if a.Src.Kind == EAkAn {
			return fmt.Errorf("MOVE.B cannot read from address register")
		}
		if a.Dst.Kind == EAkAn {
			return fmt.Errorf("MOVE.B cannot write to address register")
		}
	}
	if a.Src.Kind == EAkImm {
		if err := checkImmediateRange(a.Src.Imm, a.Size); err != nil {
			return err
		}
	}
	return nil
}

func validateMoveToSR(a *Args) error {
	if a.Dst.Kind != EAkSR {
		return fmt.Errorf("MOVE to SR requires SR destination")
	}
	switch a.Src.Kind {
	case EAkDn, EAkAddrInd, EAkAddrPostinc, EAkAddrPredec, EAkAddrDisp16, EAkIdxAnBrief, EAkPCDisp16, EAkIdxPCBrief, EAkAbsW, EAkAbsL, EAkImm:
		if a.Src.Kind == EAkImm {
			return checkImmediateRange(a.Src.Imm, SZ_W)
		}
		return nil
	case EAkNone:
		return fmt.Errorf("MOVE to SR requires source")
	default:
		return fmt.Errorf("MOVE to SR requires data addressing source")
	}
}

func validateMoveFromSR(a *Args) error {
	if a.Src.Kind != EAkSR {
		return fmt.Errorf("MOVE from SR requires SR source")
	}
	switch a.Dst.Kind {
	case EAkDn, EAkAddrInd, EAkAddrPostinc, EAkAddrDisp16, EAkAddrPredec, EAkIdxAnBrief, EAkAbsW, EAkAbsL:
		return nil
	case EAkNone:
		return fmt.Errorf("MOVE from SR requires destination")
	default:
		return fmt.Errorf("MOVE from SR destination must be data alterable")
	}
}

func validateMoveUSP(a *Args) error {
	if (a.Src.Kind == EAkUSP && a.Dst.Kind == EAkAn) || (a.Src.Kind == EAkAn && a.Dst.Kind == EAkUSP) {
		return nil
	}
	return fmt.Errorf("MOVE USP requires USP and An operands")
}
