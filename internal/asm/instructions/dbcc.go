package instructions

var dbCondMap = map[string]cond{
	"DBT":  condT,
	"DBRA": condF,
	"DBF":  condF,
	"DBHI": condHI,
	"DBLS": condLS,
	"DBCC": condCC,
	"DBHS": condCC,
	"DBCS": condCS,
	"DBLO": condCS,
	"DBNE": condNE,
	"DBEQ": condEQ,
	"DBVC": condVC,
	"DBVS": condVS,
	"DBPL": condPL,
	"DBMI": condMI,
	"DBGE": condGE,
	"DBLT": condLT,
	"DBGT": condGT,
	"DBLE": condLE,
}

func init() {
	for m, c := range dbCondMap {
		registerInstrDef(&InstrDef{
			Mnemonic: m,
			Forms: []FormDef{
				{
					DefaultSize: SZ_W,
					Sizes:       []Size{SZ_W},
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
