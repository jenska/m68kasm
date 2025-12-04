package instructions

import "fmt"

func init() {
	registerInstrDef(newLogicDef("AND", 0xC000, 0xC100))
	registerInstrDef(newLogicDef("OR", 0x8000, 0x8100))
	registerInstrDef(newLogicEorDef())
	registerInstrDef(newLogicNotDef())
}

func newLogicDef(name string, toDnBits, dnToMemBits uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{ByteSize, WordSize, LongSize},
				OperKinds:   []OperandKind{OpkEA, OpkDn},
				Validate:    func(a *Args) error { return validateLogicToDn(name, a) },
				Steps: []EmitStep{
					{WordBits: toDnBits, Fields: []FieldRef{FDnReg, FSizeBits, FSrcEA}},
					{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
				},
			},
			{
				DefaultSize: WordSize,
				Sizes:       []Size{ByteSize, WordSize, LongSize},
				OperKinds:   []OperandKind{OpkDn, OpkEA},
				Validate:    func(a *Args) error { return validateLogicDnToMemory(name, a) },
				Steps: []EmitStep{
					{WordBits: dnToMemBits, Fields: []FieldRef{FSrcDnRegHi, FSizeBits, FDstEA}},
					{Trailer: []TrailerItem{TDstEAExt}},
				},
			},
		},
	}
}

func newLogicEorDef() *InstrDef {
	return &InstrDef{
		Mnemonic: "EOR",
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{ByteSize, WordSize, LongSize},
				OperKinds:   []OperandKind{OpkDn, OpkEA},
				Validate:    validateEor,
				Steps: []EmitStep{
					{WordBits: 0xB100, Fields: []FieldRef{FSrcDnRegHi, FSizeBits, FDstEA}},
					{Trailer: []TrailerItem{TDstEAExt}},
				},
			},
		},
	}
}

func newLogicNotDef() *InstrDef {
	return &InstrDef{
		Mnemonic: "NOT",
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{ByteSize, WordSize, LongSize},
				OperKinds:   []OperandKind{OpkEA},
				Validate:    validateNot,
				Steps: []EmitStep{
					{WordBits: 0x4600, Fields: []FieldRef{FSizeBits, FDstEA}},
					{Trailer: []TrailerItem{TDstEAExt}},
				},
			},
		},
	}
}

func validateLogicToDn(name string, a *Args) error {
	if a.Src.Kind == EAkNone || a.Dst.Kind != EAkDn {
		return fmt.Errorf("%s requires Dn destination", name)
	}
	if a.Src.Kind == EAkAn {
		return fmt.Errorf("%s does not allow address register source", name)
	}
	if a.Src.Kind == EAkImm {
		if err := checkImmediateRange(a.Src.Imm, a.Size); err != nil {
			return err
		}
	}
	return nil
}

func validateLogicDnToMemory(name string, a *Args) error {
	if a.Src.Kind != EAkDn || a.Dst.Kind == EAkNone {
		return fmt.Errorf("%s requires Dn source", name)
	}
	switch a.Dst.Kind {
	case EAkAddrInd, EAkAddrPostinc, EAkAddrDisp16, EAkAddrPredec, EAkIdxAnBrief, EAkAbsW, EAkAbsL:
		return nil
	default:
		return fmt.Errorf("%s destination must be memory alterable EA", name)
	}
}

func validateEor(a *Args) error {
	if a.Src.Kind != EAkDn {
		return fmt.Errorf("EOR requires Dn source")
	}
	switch a.Dst.Kind {
	case EAkDn, EAkAddrInd, EAkAddrPostinc, EAkAddrDisp16, EAkAddrPredec, EAkIdxAnBrief, EAkAbsW, EAkAbsL:
		return nil
	case EAkNone:
		return fmt.Errorf("EOR requires destination")
	default:
		return fmt.Errorf("EOR destination must be data alterable EA")
	}
}

func validateNot(a *Args) error {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
		a.Src = EAExpr{}
	}
	switch a.Dst.Kind {
	case EAkNone:
		return fmt.Errorf("NOT requires destination")
	case EAkImm, EAkPCDisp16, EAkIdxPCBrief:
		return fmt.Errorf("NOT destination must be data alterable EA")
	case EAkAn:
		return fmt.Errorf("NOT does not allow address register destination")
	default:
		return nil
	}
}
