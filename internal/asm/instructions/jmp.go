package instructions

import "fmt"

func init() {
	registerInstrDef(&DefJMP)
}

var DefJMP = InstrDef{
	Mnemonic: "JMP",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateJMP,
			Steps: []EmitStep{
				{WordBits: 0x4EC0, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

func validateJMP(a *Args) error {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
		a.Src = EAExpr{}
	}
	switch a.Dst.Kind {
	case EAkAddrInd, EAkAddrDisp16, EAkIdxAnBrief, EAkAbsW, EAkAbsL, EAkPCDisp16, EAkIdxPCBrief:
		return nil
	case EAkNone:
		return fmt.Errorf("JMP requires destination")
	default:
		return fmt.Errorf("JMP requires control addressing mode")
	}
}
