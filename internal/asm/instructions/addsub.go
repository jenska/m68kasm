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
				OperKinds:   []OperandKind{OPK_EA, OPK_Dn},
				Validate:    func(a *Args) error { return validateAddSubToDn(name, a) },
				Steps: []EmitStep{
					{WordBits: toDnBits, Fields: []FieldRef{F_DnReg, F_SizeBits, F_SrcEA}},
					{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
				},
			},
			{
				DefaultSize: WordSize,
				Sizes:       []Size{ByteSize, WordSize, LongSize},
				OperKinds:   []OperandKind{OPK_Dn, OPK_EA},
				Validate:    func(a *Args) error { return validateAddSubDnToEA(name, a) },
				Steps: []EmitStep{
					{WordBits: dnToEABits, Fields: []FieldRef{F_SrcDnRegHi, F_SizeBits, F_DstEA}},
					{Trailer: []TrailerItem{T_DstEAExt}},
				},
			},
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize, LongSize},
				OperKinds:   []OperandKind{OPK_EA, OPK_An},
				Validate:    func(a *Args) error { return validateAddSubAddr(name, a) },
				Steps: []EmitStep{
					{WordBits: addrBits, Fields: []FieldRef{F_AddaSize, F_AnReg, F_SrcEA}},
					{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
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
				OperKinds:   []OperandKind{OPK_ImmQuick, OPK_EA},
				Validate:    func(a *Args) error { return validateAddSubQuick(name, a) },
				Steps: []EmitStep{
					{WordBits: wordBits, Fields: []FieldRef{F_QuickData, F_SizeBits, F_DstEA}},
					{Trailer: []TrailerItem{T_DstEAExt}},
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
				OperKinds:   []OperandKind{OPK_Imm, OPK_EA},
				Validate:    func(a *Args) error { return validateAddSubImm(name, a) },
				Steps: []EmitStep{
					{WordBits: wordBits, Fields: []FieldRef{F_SizeBits, F_DstEA}},
					{Trailer: []TrailerItem{T_DstEAExt, T_SrcImm}},
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
				OperKinds:   []OperandKind{OPK_EA, OPK_An},
				Validate:    func(a *Args) error { return validateAddSubAddr(name, a) },
				Steps: []EmitStep{
					{WordBits: wordBits, Fields: []FieldRef{F_AddaSize, F_AnReg, F_SrcEA}},
					{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
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
				OperKinds:   []OperandKind{OPK_Dn, OPK_Dn},
				Validate:    func(a *Args) error { return validateAddSubX(name, a, false) },
				Steps: []EmitStep{
					{WordBits: regBits, Fields: []FieldRef{F_DnReg, F_SizeBits, F_SrcDnReg}},
				},
			},
			{
				DefaultSize: WordSize,
				Sizes:       []Size{ByteSize, WordSize, LongSize},
				OperKinds:   []OperandKind{OPK_PredecAn, OPK_PredecAn},
				Validate:    func(a *Args) error { return validateAddSubX(name, a, true) },
				Steps: []EmitStep{
					{WordBits: predecBits, Fields: []FieldRef{F_AnReg, F_SizeBits, F_SrcAnReg}},
				},
			},
		},
	}
}

func validateAddSubToDn(name string, a *Args) error {
	if a.Src.Kind == EAkNone || a.Dst.Kind != EAkDn {
		return fmt.Errorf("%s requires Dn destination", name)
	}
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

func validateAddSubAddr(name string, a *Args) error {
	if a.Src.Kind == EAkNone || a.Dst.Kind != EAkAn {
		return fmt.Errorf("%s requires An destination", name)
	}
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
	if a.Dst.Kind == EAkNone {
		return fmt.Errorf("%s requires destination", name)
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
	if a.Src.Kind == EAkNone || a.Dst.Kind == EAkNone {
		return fmt.Errorf("%s requires both source and destination", name)
	}
	if predec {
		if a.Src.Kind != EAkAddrPredec || a.Dst.Kind != EAkAddrPredec {
			return fmt.Errorf("%s requires predecrement addressing for memory form", name)
		}
		return nil
	}
	if a.Src.Kind != EAkDn || a.Dst.Kind != EAkDn {
		return fmt.Errorf("%s requires Dn registers for register form", name)
	}
	return nil
}

func validateAddSubImm(name string, a *Args) error {
	if a.Src.Kind != EAkImm {
		return fmt.Errorf("%s requires immediate source", name)
	}
	if err := checkImmediateRange(a.Src.Imm, a.Size); err != nil {
		return err
	}
	switch a.Dst.Kind {
	case EAkDn, EAkAddrInd, EAkAddrPostinc, EAkAddrPredec, EAkAddrDisp16, EAkIdxAnBrief, EAkAbsW, EAkAbsL:
		return nil
	case EAkNone:
		return fmt.Errorf("%s requires destination", name)
	default:
		return fmt.Errorf("%s destination must be data alterable EA", name)
	}
}
