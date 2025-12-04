package instructions

import "fmt"

func init() {
	registerInstrDef(&defTST)
}

var defTST = InstrDef{
	Mnemonic: "TST",
	Forms: []FormDef{
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

func validateTst(a *Args) error {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
		a.Src = EAExpr{}
	}
	switch a.Dst.Kind {
	case EAkNone:
		return fmt.Errorf("TST requires destination")
	case EAkImm:
		return fmt.Errorf("TST does not allow immediate operand")
	case EAkAn:
		return fmt.Errorf("TST does not allow address register operand")
	default:
		return nil
	}
}
