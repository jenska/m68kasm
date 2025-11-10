package instructions

var DefLEA = InstrDef{
	Op:       OP_LEA,
	Mnemonic: "LEA",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_L},
			OperKinds: []OperandKind{OPK_EA, OPK_An},
			Validate:  func(a *Args) error { return nil },
			Steps: []EmitStep{
				{WordBits: 0x41C0, Fields: []FieldRef{F_AnReg, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt}},
			},
		},
	},
}
