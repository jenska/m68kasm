package instructions

import "fmt"

func init() {
	registerInstrDef(&defMOVEP)
}

var defMOVEP = InstrDef{
	Mnemonic: "MOVEP",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkEA, OpkDn},
			Validate:    validateMOVEPFromMem,
			Steps: []EmitStep{
				{WordBits: 0x0148, Fields: []FieldRef{FDnReg, FSrcAnReg}},
				{Trailer: []TrailerItem{TSrcEAExt}},
			},
		},
		{
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OpkEA, OpkDn},
			Validate:    validateMOVEPFromMem,
			Steps: []EmitStep{
				{WordBits: 0x01C8, Fields: []FieldRef{FDnReg, FSrcAnReg}},
				{Trailer: []TrailerItem{TSrcEAExt}},
			},
		},
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkDn, OpkEA},
			Validate:    validateMOVEPToMem,
			Steps: []EmitStep{
				{WordBits: 0x0108, Fields: []FieldRef{FSrcDnRegHi, FDstRegLow}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
		{
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OpkDn, OpkEA},
			Validate:    validateMOVEPToMem,
			Steps: []EmitStep{
				{WordBits: 0x0188, Fields: []FieldRef{FSrcDnRegHi, FDstRegLow}},
				{Trailer: []TrailerItem{TDstEAExt}},
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
