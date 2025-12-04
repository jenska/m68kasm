package instructions

import "fmt"

func init() {
	registerInstrDef(&defMOVEQ)
}

var defMOVEQ = InstrDef{
	Mnemonic: "MOVEQ",
	Forms: []FormDef{
		{
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
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
	return checkImmediateRange(a.Src.Imm, ByteSize)
}
