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
			OperKinds:   []OperandKind{OpkEA, OpkAn},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0x41C0, Fields: []FieldRef{FAnReg, FSrcEA}},
				{Trailer: []TrailerItem{TSrcEAExt}},
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
			OperKinds:   []OperandKind{OpkEA},
			Validate:    func(a *Args) error { return validateControlEA("PEA", a) },
			Steps: []EmitStep{
				{WordBits: 0x4840, Fields: []FieldRef{FDstEA}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
	},
}
