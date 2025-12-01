package instructions

import "fmt"

func init() {
	registerInstrDef(&DefMOVE)
}

var DefMOVE = InstrDef{
	Mnemonic: "MOVE",
	Forms: []FormDef{
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
