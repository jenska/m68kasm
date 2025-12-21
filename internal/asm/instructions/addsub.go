package instructions

import "fmt"

func init() {
	registerInstrDef(newAddSubDef("ADD", 0xD000, 0xD100, 0xD0C0))
	registerInstrDef(newAddSubDef("SUB", 0x9000, 0x9100, 0x90C0))
	registerInstrDef(newAddSubQuickDef("ADDQ", 0x5000))
	registerInstrDef(newAddSubQuickDef("SUBQ", 0x5100))
	registerInstrDef(newAddSubImmDef("ADDI", 0x0600))
	registerInstrDef(newAddSubImmDef("SUBI", 0x0400))
	registerInstrDef(newAddSubAddrDef("ADDA", 0xD0C0))
	registerInstrDef(newAddSubAddrDef("SUBA", 0x90C0))
	registerInstrDef(newAddSubXDef("SUBX", 0x9100, 0x9108))
	registerInstrDef(newAddSubXDef("ADDX", 0xD100, 0xD108))
}

func newAddSubDef(name string, toDnBits, dnToEABits, addrBits uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{ByteSize, WordSize, LongSize},
				OperKinds:   []OperandKind{OpkEA, OpkDn},
				Validate:    func(a *Args) error { return validateAddSubToDn(name, a) },
				Steps: []EmitStep{
					{WordBits: toDnBits, Fields: []FieldRef{FDnReg, FSizeBits, FSrcEA}},
					{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
				},
			},
			{
				DefaultSize: WordSize,
				Sizes:       []Size{ByteSize, WordSize, LongSize},
				OperKinds:   []OperandKind{OpkDn, OpkEA},
				Validate:    func(a *Args) error { return validateAddSubDnToEA(name, a) },
				Steps: []EmitStep{
					{WordBits: dnToEABits, Fields: []FieldRef{FSrcDnRegHi, FSizeBits, FDstEA}},
					{Trailer: []TrailerItem{TDstEAExt}},
				},
			},
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize, LongSize},
				OperKinds:   []OperandKind{OpkEA, OpkAn},
				Validate:    func(a *Args) error { return validateAddSubAddr(name, a) },
				Steps: []EmitStep{
					{WordBits: addrBits, Fields: []FieldRef{FAddaSize, FAnReg, FSrcEA}},
					{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
				},
			},
		},
	}
}

func newAddSubQuickDef(name string, wordBits uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{ByteSize, WordSize, LongSize},
				OperKinds:   []OperandKind{OpkImmQuick, OpkEA},
				Validate:    func(a *Args) error { return validateAddSubQuick(name, a) },
				Steps: []EmitStep{
					{WordBits: wordBits, Fields: []FieldRef{FQuickData, FSizeBits, FDstEA}},
					{Trailer: []TrailerItem{TDstEAExt}},
				},
			},
		},
	}
}

func newAddSubImmDef(name string, wordBits uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{ByteSize, WordSize, LongSize},
				OperKinds:   []OperandKind{OpkImm, OpkEA},
				Validate:    func(a *Args) error { return validateAddSubImm(name, a) },
				Steps: []EmitStep{
					{WordBits: wordBits, Fields: []FieldRef{FSizeBits, FDstEA}},
					{Trailer: []TrailerItem{TDstEAExt, TSrcImm}},
				},
			},
		},
	}
}

func newAddSubAddrDef(name string, wordBits uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize, LongSize},
				OperKinds:   []OperandKind{OpkEA, OpkAn},
				Validate:    func(a *Args) error { return validateAddSubAddr(name, a) },
				Steps: []EmitStep{
					{WordBits: wordBits, Fields: []FieldRef{FAddaSize, FAnReg, FSrcEA}},
					{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
				},
			},
		},
	}
}

func newAddSubXDef(name string, regBits, predecBits uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{ByteSize, WordSize, LongSize},
				OperKinds:   []OperandKind{OpkDn, OpkDn},
				Validate:    func(a *Args) error { return validateAddSubX(name, a, false) },
				Steps: []EmitStep{
					{WordBits: regBits, Fields: []FieldRef{FDnReg, FSizeBits, FSrcDnReg}},
				},
			},
			{
				DefaultSize: WordSize,
				Sizes:       []Size{ByteSize, WordSize, LongSize},
				OperKinds:   []OperandKind{OpkPredecAn, OpkPredecAn},
				Validate:    func(a *Args) error { return validateAddSubX(name, a, true) },
				Steps: []EmitStep{
					{WordBits: predecBits, Fields: []FieldRef{FAnReg, FSizeBits, FSrcAnReg}},
				},
			},
		},
	}
}

func validateAddSubToDn(name string, a *Args) error {
	// Operand types (Src=EA, Dst=Dn) are enforced by FormDef.OperKinds.
	// We only need to validate specific constraints like immediate ranges or size restrictions.
	if a.Size == ByteSize && a.Src.Kind == EAkAn {
		return fmt.Errorf("%s.B does not allow address register source", name)
	}
	if a.Src.Kind == EAkImm {
		if err := checkImmediateRange(a.Src.Imm, a.Size); err != nil {
			return err
		}
	}
	return nil
}

func validateAddSubDnToEA(name string, a *Args) error {
	// Src=Dn is enforced by OperKinds. Check if Dst is a memory alterable EA.
	switch a.Dst.Kind {
	case EAkAddrInd, EAkAddrPostinc, EAkAddrDisp16, EAkAddrPredec, EAkIdxAnBrief, EAkAbsW, EAkAbsL:
		return nil
	default:
		return fmt.Errorf("%s destination must be memory alterable EA", name)
	}
}

func validateAddSubAddr(name string, a *Args) error {
	if a.Size == ByteSize {
		return fmt.Errorf("%s does not support byte size for address destination", name)
	}
	if a.Src.Kind == EAkImm {
		if err := checkImmediateRange(a.Src.Imm, a.Size); err != nil {
			return err
		}
	}
	return nil
}

func validateAddSubQuick(name string, a *Args) error {
	if a.Src.Imm < 1 || a.Src.Imm > 8 {
		return fmt.Errorf("%s immediate out of range: %d", name, a.Src.Imm)
	}
	if isPCRelativeKind(a.Dst.Kind) || a.Dst.Kind == EAkImm {
		return fmt.Errorf("%s requires alterable destination", name)
	}
	if a.Dst.Kind == EAkAn && a.Size == ByteSize {
		return fmt.Errorf("%s.B not allowed for address register", name)
	}
	return nil
}

func validateAddSubX(name string, a *Args, predec bool) error {
	// All constraints (Dn,Dn or -(An),-(An)) are fully covered by OperKinds.
	return nil
}

func validateAddSubImm(name string, a *Args) error {
	if err := checkImmediateRange(a.Src.Imm, a.Size); err != nil {
		return err
	}
	switch a.Dst.Kind {
	case EAkDn, EAkAddrInd, EAkAddrPostinc, EAkAddrPredec, EAkAddrDisp16, EAkIdxAnBrief, EAkAbsW, EAkAbsL:
		return nil
	default:
		return fmt.Errorf("%s destination must be data alterable EA", name)
	}
}
