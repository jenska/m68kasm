package instructions

func init() {
	registerInstrDef(&defABCD)
	registerInstrDef(&defSBCD)
}

var defABCD = InstrDef{
	Mnemonic: "ABCD",
	Forms: []FormDef{
		{
			DefaultSize: SZ_B,
			Sizes:       []Size{SZ_B},
			OperKinds:   []OperandKind{OPK_Dn, OPK_Dn},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0xC100, Fields: []FieldRef{F_DnReg, F_SrcDnReg}},
			},
		},
		{
			DefaultSize: SZ_B,
			Sizes:       []Size{SZ_B},
			OperKinds:   []OperandKind{OPK_PredecAn, OPK_PredecAn},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0xC108, Fields: []FieldRef{F_AnReg, F_SrcAnReg}},
			},
		},
	},
}

var defSBCD = InstrDef{
	Mnemonic: "SBCD",
	Forms: []FormDef{
		{
			DefaultSize: SZ_B,
			Sizes:       []Size{SZ_B},
			OperKinds:   []OperandKind{OPK_Dn, OPK_Dn},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0x8100, Fields: []FieldRef{F_DnReg, F_SrcDnReg}},
			},
		},
		{
			DefaultSize: SZ_B,
			Sizes:       []Size{SZ_B},
			OperKinds:   []OperandKind{OPK_PredecAn, OPK_PredecAn},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0x8108, Fields: []FieldRef{F_AnReg, F_SrcAnReg}},
			},
		},
	},
}
