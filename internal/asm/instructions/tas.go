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
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateDataAlterable("TAS"),
			Steps: []EmitStep{
				{WordBits: 0x4AC0, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}
