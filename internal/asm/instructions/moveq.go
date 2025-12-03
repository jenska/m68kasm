package instructions

import "fmt"

func init() {
	registerInstrDef(&DefMOVEQ)
}

var DefMOVEQ = InstrDef{
	Mnemonic: "MOVEQ",
	Forms: []FormDef{
		{
			DefaultSize: SZ_L,
			Sizes:       []Size{SZ_L},
			OperKinds:   []OperandKind{OPK_Imm, OPK_Dn},
			Validate:    validateMOVEQ,
			Steps: []EmitStep{
				{WordBits: 0x7000, Fields: []FieldRef{F_DnReg, F_ImmLow8}},
			},
		},
	},
}

func validateMOVEQ(a *Args) error {
	if a.Src.Kind != EAkImm {
		return fmt.Errorf("MOVEQ needs immediate")
	}
	if a.Src.Imm < -128 || a.Src.Imm > 127 {
		return fmt.Errorf("immediate out of range for MOVEQ: %d", a.Src.Imm)
	}
	return nil
}
