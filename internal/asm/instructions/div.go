package instructions

var DefDIV = InstrDef{
	Op:       OP_DIV,
	Mnemonic: "DIV",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_W},
			OperKinds: []OperandKind{OPK_EA, OPK_Dn},
			Validate:  validateDIV,
			Steps: []EmitStep{
				{WordBits: 0x80C0, Fields: []FieldRef{F_DnReg, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt}},
			},
		},
	},
}
