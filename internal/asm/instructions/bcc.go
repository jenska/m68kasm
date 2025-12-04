package instructions

var branchConditions = []string{
	"BRA", "BSR", "BHI", "BLS", "BHS", "BLO", "BNE", "BEQ",
	"BVC", "BVS", "BPL", "BMI", "BGE", "BLT", "BGT", "BLE",
}

func init() {
	for c, b := range branchConditions {
		sz := ByteSize
		if b == "BSR" {
			sz = WordSize
		}
		registerInstrDef(&InstrDef{
			Mnemonic: b,
			Forms: []FormDef{
				{
					DefaultSize: sz,
					Sizes:       []Size{ByteSize, WordSize},
					OperKinds:   []OperandKind{OpkDispRel},
					Validate:    nil,
					Steps: []EmitStep{
						{WordBits: 0x6000 | uint16(c)<<8, Fields: []FieldRef{FBranchLow8}},
						{Trailer: []TrailerItem{TBranchWordIfNeeded}},
					},
				},
			},
		})
	}
}
