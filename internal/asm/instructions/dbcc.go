package instructions

var dbConditions = []string{
	"DBT", "DBRA", "DBHI", "DBLS", "DBHS", "DBLO", "DBNE", "DBEQ",
	"DBVC", "DBVS", "DBPL", "DBMI", "DBGE", "DBLT", "DBGT", "DBLE",
}

func init() {
	for c, m := range dbConditions {
		registerInstrDef(&InstrDef{
			Mnemonic: m,
			Forms: []FormDef{
				{
					DefaultSize: WordSize,
					Sizes:       []Size{WordSize},
					OperKinds:   []OperandKind{OPK_Dn, OPK_DispRel},
					Validate:    nil,
					Steps: []EmitStep{
						{WordBits: 0x50C8 | uint16(c)<<8, Fields: []FieldRef{F_SrcDnReg}},
						{Trailer: []TrailerItem{T_BranchWordIfNeeded}},
					},
				},
			},
		})
	}
}
