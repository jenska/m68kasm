package instructions

func init() {
	for _, c := range sccConditions {
		registerInstrDef(newSccDef(c.Mnemonic, c.Code))
	}
}

type sccCond struct {
	Mnemonic string
	Code     uint16
}

var sccConditions = []sccCond{
	{Mnemonic: "ST", Code: 0},
	{Mnemonic: "SF", Code: 1},
	{Mnemonic: "SHI", Code: 2},
	{Mnemonic: "SLS", Code: 3},
	{Mnemonic: "SCC", Code: 4},
	{Mnemonic: "SHS", Code: 4},
	{Mnemonic: "SCS", Code: 5},
	{Mnemonic: "SLO", Code: 5},
	{Mnemonic: "SNE", Code: 6},
	{Mnemonic: "SEQ", Code: 7},
	{Mnemonic: "SVC", Code: 8},
	{Mnemonic: "SVS", Code: 9},
	{Mnemonic: "SPL", Code: 10},
	{Mnemonic: "SMI", Code: 11},
	{Mnemonic: "SGE", Code: 12},
	{Mnemonic: "SLT", Code: 13},
	{Mnemonic: "SGT", Code: 14},
	{Mnemonic: "SLE", Code: 15},
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
