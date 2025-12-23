package instructions

var branchConditions = []string{
	"BRA", "BSR", "BHI", "BLS", "BHS", "BLO", "BNE", "BEQ",
	"BVC", "BVS", "BPL", "BMI", "BGE", "BLT", "BGT", "BLE",
}

var dbConditions = []string{
	"DBT", "DBRA", "DBHI", "DBLS", "DBHS", "DBLO", "DBNE", "DBEQ",
	"DBVC", "DBVS", "DBPL", "DBMI", "DBGE", "DBLT", "DBGT", "DBLE",
}

var sccConditions = []string{
	"ST", "SF", "SHI", "SLS", "SHS", "SLO", "SNE", "SEQ",
	"SVC", "SVS", "SPL", "SMI", "SGE", "SLT", "SGT", "SLE",
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

	for c, m := range dbConditions {
		registerInstrDef(&InstrDef{
			Mnemonic: m,
			Forms: []FormDef{
				{
					DefaultSize: WordSize,
					Sizes:       []Size{WordSize},
					OperKinds:   []OperandKind{OpkDn, OpkDispRel},
					Validate:    nil,
					Steps: []EmitStep{
						{WordBits: 0x50C8 | uint16(c)<<8, Fields: []FieldRef{FSrcDnReg}},
						{Trailer: []TrailerItem{TBranchWordIfNeeded}},
					},
				},
			},
		})
	}
	// DBF is an alias for DBRA (condition code 1)
	registerInstrDef(&InstrDef{
		Mnemonic: "DBF",
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkDn, OpkDispRel},
				Validate:    nil,
				Steps: []EmitStep{
					{WordBits: 0x51C8, Fields: []FieldRef{FSrcDnReg}},
					{Trailer: []TrailerItem{TBranchWordIfNeeded}},
				},
			},
		},
	})

	for i, c := range sccConditions {
		registerInstrDef(
			&InstrDef{
				Mnemonic: c,
				Forms: []FormDef{
					{
						DefaultSize: ByteSize,
						Sizes:       []Size{ByteSize},
						OperKinds:   []OperandKind{OpkEA},
						Validate:    validateDataAlterable(c),
						Steps: []EmitStep{
							{WordBits: 0x50C0 | (uint16(i) << 8), Fields: []FieldRef{FDstEA}},
							{Trailer: []TrailerItem{TDstEAExt}},
						},
					},
				},
			})
	}
}
