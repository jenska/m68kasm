package instructions

var DefMOVEQ = InstrDef{
	Op:       OP_MOVEQ,
	Mnemonic: "MOVEQ",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_L},
			OperKinds: []OperandKind{OPK_Imm, OPK_Dn},
			Validate:  validateMOVEQ,
			Steps: []EmitStep{
				{WordBits: 0x7000, Fields: []FieldRef{F_DnReg, F_ImmLow8}},
			},
		},
	},
}
