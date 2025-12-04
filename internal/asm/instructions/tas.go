package instructions

func init() {
	registerInstrDef(&defTAS)
}

var defTAS = InstrDef{
	Mnemonic: "TAS",
	Forms: []FormDef{
		{
			DefaultSize: ByteSize,
			Sizes:       []Size{ByteSize},
			OperKinds:   []OperandKind{OpkEA},
			Validate:    validateDataAlterable("TAS"),
			Steps: []EmitStep{
				{WordBits: 0x4AC0, Fields: []FieldRef{FDstEA}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
	},
}
