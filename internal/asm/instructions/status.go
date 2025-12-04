package instructions

import "fmt"

func init() {
	registerInstrDef(newStatusDef("ORI", 0x003c, 0x007c))
	registerInstrDef(newStatusDef("ANDI", 0x023C, 0x027C))
	registerInstrDef(newStatusDef("EORI", 0x0A3C, 0x0A7C))
}

func newStatusDef(name string, wordb, wordw uint16) *InstrDef {
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
