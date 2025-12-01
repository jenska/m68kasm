package instructions

import "fmt"

func init() {
	registerInstrDef(&DefCMP)
}

var DefCMP = InstrDef{
	Mnemonic: "CMP",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_B, SZ_W, SZ_L},
			OperKinds:   []OperandKind{OPK_EA, OPK_Dn},
			Validate:    validateCMP,
			Steps: []EmitStep{
				{WordBits: 0xB000, Fields: []FieldRef{F_DnReg, F_SizeBits, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
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
