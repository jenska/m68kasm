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
			OperKinds:   []OperandKind{OpkImm, OpkDn},
			Validate:    validateMOVEQ,
			Steps: []EmitStep{
				{WordBits: 0x7000, Fields: []FieldRef{FDnReg, FImmLow8}},
			},
		},
	},
}

func validateMOVEQ(a *Args) error {
	if a.Src.Kind != EAkImm {
		return fmt.Errorf("MOVEQ needs immediate")
	}
	if a.Src.Imm < -128 || a.Src.Imm > 127 {
		return fmt.Errorf("immediate out of range for .b: %d", a.Src.Imm)
	}
	return nil
}
