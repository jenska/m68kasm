package instructions

var DefMOVE = InstrDef{
	Op:       OP_MOVE,
	Mnemonic: "MOVE",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_B, SZ_W, SZ_L},
			OperKinds: []OperandKind{OPK_EA, OPK_EA},
			Validate:  validateMOVE,
			Steps: []EmitStep{
				{WordBits: 0x0000, Fields: []FieldRef{F_MoveSize, F_MoveDestEA, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm, T_DstEAExt}},
			},
		},
	},
}
