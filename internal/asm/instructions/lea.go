package instructions

func init() {
	registerInstrDef(&defLEA)
	registerInstrDef(&defPEA)
}

var defLEA = InstrDef{
	Mnemonic: "LEA",
	Forms: []FormDef{
		{
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OPK_EA, OPK_An},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0x41C0, Fields: []FieldRef{F_AnReg, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt}},
			},
		},
	},
}

var defPEA = InstrDef{
	Mnemonic: "PEA",
	Forms: []FormDef{
		{
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    func(a *Args) error { return validateControlEA("PEA", a) },
			Steps: []EmitStep{
				{WordBits: 0x4840, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}
