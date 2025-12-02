package instructions

var branchCondMap = map[string]cond{
	"BRA": condT,
	"BSR": condBSR,
	"BHI": condHI,
	"BLS": condLS,
	"BCC": condCC,
	"BHS": condCC,
	"BCS": condCS,
	"BLO": condCS,
	"BNE": condNE,
	"BEQ": condEQ,
	"BVC": condVC,
	"BVS": condVS,
	"BPL": condPL,
	"BMI": condMI,
	"BGE": condGE,
	"BLT": condLT,
	"BGT": condGT,
	"BLE": condLE,
}

func init() {
	for b, c := range branchCondMap {
		sz := SZ_B
		if b == "BSR" {
			sz = SZ_W
		}
		registerInstrDef(&InstrDef{
			Mnemonic: b,
			Forms: []FormDef{
				{
					DefaultSize: sz,
					Sizes:       []Size{SZ_B, SZ_W},
					OperKinds:   []OperandKind{OPK_DispRel},
					Validate:    nil,
					Steps: []EmitStep{
						{WordBits: 0x6000 | uint16(c)<<8, Fields: []FieldRef{F_BranchLow8}},
						{Trailer: []TrailerItem{T_BranchWordIfNeeded}},
					},
				},
			},
		})
	}
}
