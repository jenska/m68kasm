package instructions

var DefBranch = InstrDef{
	Op:       OP_BCC,
	Mnemonic: "BRANCH",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_B, SZ_W},
			OperKinds: []OperandKind{OPK_DispRel},
			Validate:  nil,
			Steps: []EmitStep{
				{WordBits: 0x6000, Fields: []FieldRef{F_Cond, F_BranchLow8}},
				{Trailer: []TrailerItem{T_BranchWordIfNeeded}},
			},
		},
	},
}
