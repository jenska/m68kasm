package instructions

import "fmt"

func init() {
	registerInstrDef(newBcdDef("ABCD", 0xC100, 0xC108))
	registerInstrDef(newBcdDef("SBCD", 0x8100, 0x8108))
	registerInstrDef(&defNBCD)
}

func newBcdDef(name string, regBits, memBits uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: ByteSize,
				Sizes:       []Size{ByteSize},
				OperKinds:   []OperandKind{OpkDn, OpkDn},
				Steps: []EmitStep{
					{WordBits: regBits, Fields: []FieldRef{FDnReg, FSrcDnReg}},
				},
			},
			{
				DefaultSize: ByteSize,
				Sizes:       []Size{ByteSize},
				OperKinds:   []OperandKind{OpkPredecAn, OpkPredecAn},
				Steps: []EmitStep{
					{WordBits: memBits, Fields: []FieldRef{FAnReg, FSrcAnReg}},
				},
			},
		},
	}
}

var defNBCD = InstrDef{
	Mnemonic: "NBCD",
	Forms: []FormDef{
		{
			DefaultSize: ByteSize,
			Sizes:       []Size{ByteSize},
			OperKinds:   []OperandKind{OpkEA},
			Validate:    validateNbcd,
			Steps: []EmitStep{
				{WordBits: 0x4800, Fields: []FieldRef{FDstEA}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
	},
}

func validateNbcd(a *Args) error {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
		a.Src = EAExpr{}
	}
	switch a.Dst.Kind {
	case EAkDn, EAkAddrPredec:
		return nil
	case EAkNone:
		return fmt.Errorf("NBCD requires destination")
	default:
		return fmt.Errorf("NBCD destination must be Dn or predecrement address")
	}
}
