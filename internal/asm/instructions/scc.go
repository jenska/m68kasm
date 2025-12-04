package instructions

func init() {
	for i, c := range sccConditions {
		registerInstrDef(newSccDef(c, uint16(i)))
	}
}

var sccConditions = []string{
	"ST", "SF", "SHI", "SLS", "SHS", "SLO", "SNE", "SEQ",
	"SVC", "SVS", "SPL", "SMI", "SGE", "SLT", "SGT", "SLE",
}

func newSccDef(name string, cond uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: ByteSize,
				Sizes:       []Size{ByteSize},
				OperKinds:   []OperandKind{OpkEA},
				Validate:    validateDataAlterable(name),
				Steps: []EmitStep{
					{WordBits: 0x50C0 | (cond << 8), Fields: []FieldRef{FDstEA}},
					{Trailer: []TrailerItem{TDstEAExt}},
				},
			},
		},
	}
}
