package instructions

func init() {
	registerInstrDef(&DefLEA)
}

var DefLEA = InstrDef{
	Mnemonic: "LEA",
	Forms: []FormDef{
		{
			DefaultSize: SZ_L,
			Sizes:       []Size{SZ_L},
			OperKinds:   []OperandKind{OPK_EA, OPK_An},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0x41C0, Fields: []FieldRef{F_AnReg, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt}},
			},
		},
	},
}
