package instructions

import "fmt"

func init() {
	registerInstrDef(&defMOVE)
	registerInstrDef(&defMOVEA)
}

var defMOVE = InstrDef{
	Mnemonic: "MOVE",
	Forms: append(newMoveUSPForms(), []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkEA, OpkSR},
			Validate:    validateMoveToSR,
			Steps: []EmitStep{
				{WordBits: 0x46C0, Fields: []FieldRef{FSrcEA}},
				{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
			},
		},
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkSR, OpkEA},
			Validate:    validateMoveFromSR,
			Steps: []EmitStep{
				{WordBits: 0x40C0, Fields: []FieldRef{FDstEA}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkEA, OpkCCR},
			Validate:    validateMoveToCCR,
			Steps: []EmitStep{
				{WordBits: 0x44C0, Fields: []FieldRef{FSrcEA}},
				{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
			},
		},
		{
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OpkEA, OpkEA},
			Validate:    validateMOVE,
			Steps: []EmitStep{
				{WordBits: 0x0000, Fields: []FieldRef{FMoveSize, FMoveDestEA, FSrcEA}},
				{Trailer: []TrailerItem{TSrcEAExt, TSrcImm, TDstEAExt}},
			},
		},
	}...),
}

var defMOVEA = InstrDef{
	Mnemonic: "MOVEA",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize, LongSize},
			OperKinds:   []OperandKind{OpkEA, OpkAn},
			Validate:    validateMOVEA,
			Steps: []EmitStep{
				{WordBits: 0x0040, Fields: []FieldRef{FMoveSize, FMoveDestEA, FSrcEA}},
				{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
			},
		},
	},
}

func validateMOVE(a *Args) error {
	if a.Src.Kind == EAkSR || a.Src.Kind == EAkCCR || a.Src.Kind == EAkUSP || a.Dst.Kind == EAkSR || a.Dst.Kind == EAkCCR || a.Dst.Kind == EAkUSP {
		return fmt.Errorf("MOVE special registers require dedicated forms")
	}
	if a.Dst.Kind == EAkImm {
		return fmt.Errorf("MOVE destination cannot be immediate")
	}
	if isPCRelativeKind(a.Dst.Kind) {
		return fmt.Errorf("MOVE destination cannot be PC-relative")
	}
	if a.Size == ByteSize {
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
	switch a.Src.Kind {
	case EAkDn, EAkAddrInd, EAkAddrPostinc, EAkAddrPredec, EAkAddrDisp16, EAkIdxAnBrief, EAkPCDisp16, EAkIdxPCBrief, EAkAbsW, EAkAbsL, EAkImm:
		if a.Src.Kind == EAkImm {
			return checkImmediateRange(a.Src.Imm, WordSize)
		}
		return nil
	case EAkNone:
		return fmt.Errorf("MOVE to SR requires source")
	default:
		return fmt.Errorf("MOVE to SR requires data addressing source")
	}
}

func validateMoveFromSR(a *Args) error {
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

func validateMoveToCCR(a *Args) error {
	switch a.Src.Kind {
	case EAkDn, EAkAddrInd, EAkAddrPostinc, EAkAddrPredec, EAkAddrDisp16, EAkIdxAnBrief, EAkPCDisp16, EAkIdxPCBrief, EAkAbsW, EAkAbsL, EAkImm:
		if a.Src.Kind == EAkImm {
			return checkImmediateRange(a.Src.Imm, WordSize)
		}
		return nil
	case EAkNone:
		return fmt.Errorf("MOVE to CCR requires source")
	default:
		return fmt.Errorf("MOVE to CCR requires data addressing source")
	}
}

func validateMOVEA(a *Args) error {
	if a.Size == ByteSize {
		return fmt.Errorf("MOVEA does not support byte size")
	}
	if a.Src.Kind == EAkImm {
		return checkImmediateRange(a.Src.Imm, a.Size)
	}
	return nil
}

func newMoveUSPForms() []FormDef {
	return []FormDef{
		{
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OpkUSP, OpkAn},
			Validate:    validateMoveUSP,
			Steps:       []EmitStep{{WordBits: 0x4E68, Fields: []FieldRef{FDstRegLow}}},
		},
		{
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OpkAn, OpkUSP},
			Validate:    validateMoveUSP,
			Steps:       []EmitStep{{WordBits: 0x4E60, Fields: []FieldRef{FSrcAnReg}}},
		},
	}
}
