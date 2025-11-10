package instructions

var DefCMP = InstrDef{
	Op:       OP_CMP,
	Mnemonic: "CMP",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_B, SZ_W, SZ_L},
			OperKinds: []OperandKind{OPK_EA, OPK_Dn},
			Validate:  validateCMP,
			Steps: []EmitStep{
				{WordBits: 0xB000, Fields: []FieldRef{F_DnReg, F_SizeBits, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
			},
		},
	},
}
