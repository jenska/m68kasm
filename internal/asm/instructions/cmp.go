package instructions

import "fmt"

func init() {
	registerInstrDef(&defCMP)
	registerInstrDef(&defCMPM)
	registerInstrDef(&defCMPI)
	registerInstrDef(&defCMPA)
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

var defCMPI = InstrDef{
	Mnemonic: "CMPI",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OpkImm, OpkEA},
			Validate:    validateCMPI,
			Steps: []EmitStep{
				{WordBits: 0x0C00, Fields: []FieldRef{FSizeBits, FDstEA}},
				{Trailer: []TrailerItem{TDstEAExt, TSrcImm}},
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
	if a.Src.Kind == EAkNone || a.Dst.Kind != EAkDn {
		return fmt.Errorf("CMP requires Dn destination")
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
	if a.Src.Kind != EAkImm {
		return fmt.Errorf("CMPI requires immediate source")
	}
	if err := checkImmediateRange(a.Src.Imm, a.Size); err != nil {
		return err
	}
	switch a.Dst.Kind {
	case EAkDn, EAkAddrInd, EAkAddrPostinc, EAkAddrPredec, EAkAddrDisp16, EAkIdxAnBrief, EAkAbsW, EAkAbsL:
		return nil
	case EAkNone:
		return fmt.Errorf("CMPI requires destination")
	default:
		return fmt.Errorf("CMPI destination must be data alterable EA")
	}
}

func validateCMPA(a *Args) error {
	if a.Src.Kind == EAkNone || a.Dst.Kind != EAkAn {
		return fmt.Errorf("CMPA requires An destination and source")
	}
	if a.Size == ByteSize {
		return fmt.Errorf("CMPA does not support byte size")
	}
	if a.Src.Kind == EAkImm {
		return checkImmediateRange(a.Src.Imm, a.Size)
	}
	return nil
}
