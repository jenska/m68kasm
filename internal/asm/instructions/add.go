package instructions

var DefADD = InstrDef{
	Op:       OP_ADD,
	Mnemonic: "ADD",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_B, SZ_W, SZ_L},
			OperKinds: []OperandKind{OPK_EA, OPK_Dn},
			Validate:  func(a *Args) error { return validateAddSub("ADD", a) },
			Steps: []EmitStep{
				{WordBits: 0xD000, Fields: []FieldRef{F_DnReg, F_SizeBits, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
			},
		},
	},
}
