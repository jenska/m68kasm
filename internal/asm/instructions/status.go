package instructions

import "fmt"

func init() {
	registerInstrDef(newStatusDef("ORI", 0x003c, 0x007c, 0x0000))
	registerInstrDef(newStatusDef("ANDI", 0x023C, 0x027C, 0x0200))
	registerInstrDef(newStatusDef("EORI", 0x0A3C, 0x0A7C, 0x0A00))
}

func newStatusDef(name string, wordb, wordw, immWord uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: ByteSize,
				Sizes:       []Size{ByteSize},
				OperKinds:   []OperandKind{OpkImm, OpkCCR},
				Validate:    validateCCRImmediate,
				Steps: []EmitStep{
					{WordBits: wordb},
					{Trailer: []TrailerItem{TImmSized}},
				},
			},
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkImm, OpkSR},
				Validate:    validateSRImmediate,
				Steps: []EmitStep{
					{WordBits: wordw},
					{Trailer: []TrailerItem{TImmSized}},
				},
			},
			{
				DefaultSize: WordSize,
				Sizes:       []Size{ByteSize, WordSize, LongSize},
				OperKinds:   []OperandKind{OpkImm, OpkEA},
				Validate:    validateImmediateLogicEA,
				Steps: []EmitStep{
					{WordBits: immWord, Fields: []FieldRef{FSizeBits, FDstEA}},
					{Trailer: []TrailerItem{TDstEAExt, TSrcImm}},
				},
			},
		},
	}
}

func validateSRImmediate(a *Args) error {
	if a.Dst.Kind != EAkSR {
		return fmt.Errorf("requires SR destination")
	}
	return checkImmediateRange(a.Src.Imm, WordSize)
}

func validateCCRImmediate(a *Args) error {
	if a.Dst.Kind != EAkCCR {
		return fmt.Errorf("requires CCR destination")
	}
	return checkImmediateRange(a.Src.Imm, ByteSize)
}

func validateImmediateLogicEA(a *Args) error {
	if err := checkImmediateRange(a.Src.Imm, a.Size); err != nil {
		return err
	}
	switch a.Dst.Kind {
	case EAkDn, EAkAddrInd, EAkAddrPostinc, EAkAddrPredec, EAkAddrDisp16, EAkIdxAnBrief, EAkAbsW, EAkAbsL:
		return nil
	case EAkNone:
		return fmt.Errorf("requires destination")
	default:
		return fmt.Errorf("destination must be data alterable EA")
	}
}
