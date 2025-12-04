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
				OperKinds:   []OperandKind{OPK_Dn, OPK_Dn},
				Steps: []EmitStep{
					{WordBits: regBits, Fields: []FieldRef{F_DnReg, F_SrcDnReg}},
				},
			},
			{
				DefaultSize: ByteSize,
				Sizes:       []Size{ByteSize},
				OperKinds:   []OperandKind{OPK_PredecAn, OPK_PredecAn},
				Steps: []EmitStep{
					{WordBits: memBits, Fields: []FieldRef{F_AnReg, F_SrcAnReg}},
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
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateNbcd,
			Steps: []EmitStep{
				{WordBits: 0x4800, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
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
