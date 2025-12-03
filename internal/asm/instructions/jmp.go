package instructions

import "fmt"

func init() {
	registerInstrDef(&defJMP)
	registerInstrDef(&defJSR)
	registerInstrDef(&defPEA)
}

var defJMP = InstrDef{
	Mnemonic: "JMP",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    validateJMP,
			Steps: []EmitStep{
				{WordBits: 0x4EC0, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

var defJSR = InstrDef{
	Mnemonic: "JSR",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    func(a *Args) error { return validateControlEA("JSR", a) },
			Steps: []EmitStep{
				{WordBits: 0x4E80, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

var defPEA = InstrDef{
	Mnemonic: "PEA",
	Forms: []FormDef{
		{
			DefaultSize: SZ_L,
			Sizes:       []Size{SZ_L},
			OperKinds:   []OperandKind{OPK_EA},
			Validate:    func(a *Args) error { return validateControlEA("PEA", a) },
			Steps: []EmitStep{
				{WordBits: 0x4840, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
	},
}

func validateJMP(a *Args) error {
	return validateControlEA("JMP", a)
}

func validateControlEA(name string, a *Args) error {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
		a.Src = EAExpr{}
	}
	switch a.Dst.Kind {
	case EAkAddrInd, EAkAddrDisp16, EAkIdxAnBrief, EAkAbsW, EAkAbsL, EAkPCDisp16, EAkIdxPCBrief:
		return nil
	case EAkNone:
		return fmt.Errorf("%s requires destination", name)
	default:
		return fmt.Errorf("%s requires control addressing mode", name)
	}
}
