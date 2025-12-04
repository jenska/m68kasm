package instructions

func init() {
	registerInstrDef(&defJMP)
	registerInstrDef(&defJSR)

}

var defJMP = InstrDef{
	Mnemonic: "JMP",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkEA},
			Validate: func(a *Args) error {
				return validateControlEA("JMP", a)
			},
			Steps: []EmitStep{
				{WordBits: 0x4EC0, Fields: []FieldRef{FDstEA}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
	},
}

var defJSR = InstrDef{
	Mnemonic: "JSR",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkEA},
			Validate:    func(a *Args) error { return validateControlEA("JSR", a) },
			Steps: []EmitStep{
				{WordBits: 0x4E80, Fields: []FieldRef{FDstEA}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
	},
}
