package instructions

var DefMULTI = InstrDef{
	Op:       OP_MULTI,
	Mnemonic: "MULTI",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_W},
			OperKinds: []OperandKind{OPK_EA, OPK_Dn},
			Validate:  validateMULTI,
			Steps: []EmitStep{
				{WordBits: 0xC1C0, Fields: []FieldRef{F_DnReg, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt}},
			},
		},
	},
}
