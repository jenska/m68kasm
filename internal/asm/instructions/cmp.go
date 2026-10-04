package instructions

import "fmt"

func init() {
	registerInstrDef(&defCMP)
	registerInstrDef(&defCMPM)
	registerInstrDef(&defCMPI)
	registerInstrDef(&defCMPA)
	registerInstrDef(&defTST)
}

var defCMP = InstrDef{
	Mnemonic: "CMP",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OpkEA, OpkDn},
			Validate:    validateCMP,
			Steps: []EmitStep{
				{WordBits: 0xB000, Fields: []FieldRef{FDnReg, FSizeBits, FSrcEA}},
				{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
			},
		},
	},
}

var defCMPM = InstrDef{
	Mnemonic: "CMPM",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OpkEA, OpkEA},
			Validate:    validateCMPM,
			Steps: []EmitStep{
				{WordBits: 0xB108, Fields: []FieldRef{FAnReg, FSizeBits, FSrcAnReg}},
			},
		},
	},
}

// defCMPI's first form is the CPU32/68020+ one, whose destination may also
// be PC-relative; on the 68000 it must be data alterable.
var defCMPI = InstrDef{
	Mnemonic: "CMPI",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OpkImm, OpkEA},
			Requires:    require68020orCPU32,
			Validate:    validateCMPI020,
			Steps: []EmitStep{
				{WordBits: 0x0C00, Fields: []FieldRef{FSizeBits, FDstEA}},
				{Trailer: []TrailerItem{TSrcImm, TDstEAExt}},
			},
		},
		{
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OpkImm, OpkEA},
			Validate:    validateCMPI,
			Steps: []EmitStep{
				{WordBits: 0x0C00, Fields: []FieldRef{FSizeBits, FDstEA}},
				{Trailer: []TrailerItem{TSrcImm, TDstEAExt}},
			},
		},
	},
}

var defCMPA = InstrDef{
	Mnemonic: "CMPA",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize, LongSize},
			OperKinds:   []OperandKind{OpkEA, OpkAn},
			Validate:    validateCMPA,
			Steps: []EmitStep{
				{WordBits: 0xB0C0, Fields: []FieldRef{FAddaSize, FAnReg, FSrcEA}},
				{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
			},
		},
	},
}

func validateCMP(a *Args) error {
	if a.Size == ByteSize && a.Src.Kind == EAkAn {
		return fmt.Errorf("CMP.B does not allow address register source")
	}
	if a.Src.Kind == EAkImm {
		if err := checkImmediateRange(a.Src.Imm, a.Size); err != nil {
			return err
		}
	}
	return nil
}

func validateCMPM(a *Args) error {
	if a.Src.Kind != EAkAddrPostinc || a.Dst.Kind != EAkAddrPostinc {
		return fmt.Errorf("CMPM requires post-increment address operands")
	}
	return nil
}

func validateCMPI(a *Args) error {
	if err := checkImmediateRange(a.Src.Imm, a.Size); err != nil {
		return err
	}
	if !isDataAlterable(a.Dst.Kind) {
		if a.Dst.Kind == EAkNone {
			return fmt.Errorf("CMPI requires destination")
		}
		return fmt.Errorf("CMPI destination must be data alterable EA")
	}
	return nil
}

func validateCMPI020(a *Args) error {
	if err := checkImmediateRange(a.Src.Imm, a.Size); err != nil {
		return err
	}
	if !isDataAlterable(a.Dst.Kind) && !isPCRelativeKind(a.Dst.Kind) {
		if a.Dst.Kind == EAkNone {
			return fmt.Errorf("CMPI requires destination")
		}
		return fmt.Errorf("CMPI destination must be data addressing other than immediate")
	}
	return nil
}

func validateCMPA(a *Args) error {
	if a.Size == ByteSize {
		return fmt.Errorf("CMPA does not support byte size")
	}
	if a.Src.Kind == EAkImm {
		return checkImmediateRange(a.Src.Imm, a.Size)
	}
	return nil
}

// defTST's first form is the CPU32/68020+ one, which also reads PC-relative
// and immediate operands and (word/long) address registers; the 68000 form
// only takes data alterable operands.
var defTST = InstrDef{
	Mnemonic: "TST",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OpkEA},
			Requires:    require68020orCPU32,
			Validate:    validateTst020,
			Steps: []EmitStep{
				{WordBits: 0x4A00, Fields: []FieldRef{FSizeBits, FDstEA}},
				{Trailer: []TrailerItem{TDstEAExt, TDstImmSized}},
			},
		},
		{
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OpkEA},
			Validate:    validateTst,
			Steps: []EmitStep{
				{WordBits: 0x4A00, Fields: []FieldRef{FSizeBits, FDstEA}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
	},
}

func validateTst020(a *Args) error {
	swapSrcDstIfDstNone(a)
	switch {
	case a.Dst.Kind == EAkNone:
		return fmt.Errorf("TST requires destination")
	case a.Dst.Kind == EAkAn && a.Size == ByteSize:
		return fmt.Errorf("TST.B does not allow address register operand")
	case a.Dst.Kind == EAkImm:
		return checkImmediateRange(a.Dst.Imm, a.Size)
	}
	return nil
}

func validateTst(a *Args) error {
	swapSrcDstIfDstNone(a)
	switch a.Dst.Kind {
	case EAkNone:
		return fmt.Errorf("TST requires destination")
	case EAkImm:
		return fmt.Errorf("TST does not allow immediate operand")
	case EAkAn:
		return fmt.Errorf("TST does not allow address register operand")
	}
	if !isDataAlterable(a.Dst.Kind) {
		return fmt.Errorf("TST operand must be data alterable EA on the 68000")
	}
	return nil
}
