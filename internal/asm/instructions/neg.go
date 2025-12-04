package instructions

import "fmt"

func init() {
	registerInstrDef(&defNEG)
	registerInstrDef(&defNEGX)
}

var defNEG = newNegLikeDef("NEG", 0x4400)
var defNEGX = newNegLikeDef("NEGX", 0x4000)

func newNegLikeDef(name string, wordBits uint16) InstrDef {
	return InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{ByteSize, WordSize, LongSize},
				OperKinds:   []OperandKind{OpkEA},
				Validate:    validateNegLike(name),
				Steps: []EmitStep{
					{WordBits: wordBits, Fields: []FieldRef{FSizeBits, FDstEA}},
					{Trailer: []TrailerItem{TDstEAExt}},
				},
			},
		},
	}
}

func validateNegLike(name string) func(*Args) error {
	return func(a *Args) error {
		if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
			a.Dst = a.Src
			a.Src = EAExpr{}
		}
		switch a.Dst.Kind {
		case EAkNone:
			return fmt.Errorf("%s requires destination", name)
		case EAkImm, EAkPCDisp16, EAkIdxPCBrief:
			return fmt.Errorf("%s destination must be data alterable EA", name)
		case EAkAn:
			return fmt.Errorf("%s does not allow address register destination", name)
		default:
			return nil
		}
	}
}
