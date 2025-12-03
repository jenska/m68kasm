package instructions

import "fmt"

func init() {
	registerInstrDef(&DefMOVEP)
}

var DefMOVEP = InstrDef{
	Mnemonic: "MOVEP",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_EA, OPK_Dn},
			Validate:    validateMOVEPFromMem,
			Steps: []EmitStep{
				{WordBits: 0x0148, Fields: []FieldRef{F_DnReg, F_SrcAnReg}},
				{Trailer: []TrailerItem{T_SrcEAExt}},
			},
		},
		{
			DefaultSize: SZ_L,
			Sizes:       []Size{SZ_L},
			OperKinds:   []OperandKind{OPK_EA, OPK_Dn},
			Validate:    validateMOVEPFromMem,
			Steps: []EmitStep{
				{WordBits: 0x01C8, Fields: []FieldRef{F_DnReg, F_SrcAnReg}},
				{Trailer: []TrailerItem{T_SrcEAExt}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_Dn, OPK_EA},
			Validate:    validateMOVEPToMem,
			Steps: []EmitStep{
				{WordBits: 0x0108, Fields: []FieldRef{F_SrcDnRegHi, F_DstRegLow}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
		{
			DefaultSize: SZ_L,
			Sizes:       []Size{SZ_L},
			OperKinds:   []OperandKind{OPK_Dn, OPK_EA},
			Validate:    validateMOVEPToMem,
			Steps: []EmitStep{
				{WordBits: 0x0188, Fields: []FieldRef{F_SrcDnRegHi, F_DstRegLow}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

func validateMOVEPToMem(a *Args) error {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
		a.Src = EAExpr{}
	}
	if a.Src.Kind != EAkDn {
		return fmt.Errorf("MOVEP requires Dn source")
	}
	if a.Dst.Kind != EAkAddrDisp16 {
		return fmt.Errorf("MOVEP destination must be (d16,An)")
	}
	return nil
}

func validateMOVEPFromMem(a *Args) error {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
		a.Src = EAExpr{}
	}
	if a.Src.Kind != EAkAddrDisp16 {
		return fmt.Errorf("MOVEP source must be (d16,An)")
	}
	if a.Dst.Kind != EAkDn {
		return fmt.Errorf("MOVEP requires Dn destination")
	}
	return nil
}
