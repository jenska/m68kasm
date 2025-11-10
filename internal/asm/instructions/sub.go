package instructions

var DefSUB = InstrDef{
	Op:       OP_SUB,
	Mnemonic: "SUB",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_B, SZ_W, SZ_L},
			OperKinds: []OperandKind{OPK_EA, OPK_Dn},
			Validate:  func(a *Args) error { return validateAddSub("SUB", a) },
			Steps: []EmitStep{
				{WordBits: 0x9000, Fields: []FieldRef{F_DnReg, F_SizeBits, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
			},
		},
	},
}
