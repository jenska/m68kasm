package instructions

import "fmt"

func init() {
	registerInstrDef(&defLINK)
	registerInstrDef(&defUNLK)
}

var defLINK = InstrDef{
	Mnemonic: "LINK",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkAn, OpkImm},
			Validate:    validateLINK,
			Steps: []EmitStep{
				{WordBits: 0x4E50, Fields: []FieldRef{FDstRegLow}},
				{Trailer: []TrailerItem{TImmSized}},
			},
		},
	},
}

var defUNLK = InstrDef{
	Mnemonic: "UNLK",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkAn},
			Validate:    validateUNLK,
			Steps: []EmitStep{
				{WordBits: 0x4E58, Fields: []FieldRef{FDstRegLow}},
			},
		},
	},
}

func validateLINK(a *Args) error {
	if a.Src.Kind == EAkAn && a.Dst.Kind == EAkImm {
		a.Src, a.Dst = a.Dst, a.Src
	}
	if a.Src.Kind != EAkImm || a.Dst.Kind != EAkAn {
		return fmt.Errorf("LINK requires address register and immediate displacement")
	}
	return checkImmediateRange(a.Src.Imm, WordSize)
}

func validateUNLK(a *Args) error {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
		a.Src = EAExpr{}
	}
	if a.Dst.Kind != EAkAn {
		return fmt.Errorf("UNLK requires address register operand")
	}
	return nil
}
