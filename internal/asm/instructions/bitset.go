package instructions

import "fmt"

func init() {
	registerInstrDef(newBitSetDef("BSET", 0x01C0, 0x08C0))
	registerInstrDef(newBitSetDef("BCLR", 0x0180, 0x0880))
	registerInstrDef(newBitSetDef("BCHG", 0x0140, 0x0840))
	registerInstrDef(&defBTST)
}

func newBitSetDef(name string, op1, op2 uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: LongSize,
				Sizes:       []Size{LongSize},
				OperKinds:   []OperandKind{OPK_Dn, OPK_EA},
				Validate:    func(a *Args) error { return validateBitReg(name, a) },
				Steps: []EmitStep{
					{WordBits: op1, Fields: []FieldRef{F_SrcDnRegHi, F_DstEA}},
					{Trailer: []TrailerItem{T_DstEAExt}},
				},
			},
			{
				DefaultSize: ByteSize,
				Sizes:       []Size{ByteSize},
				OperKinds:   []OperandKind{OPK_Imm, OPK_EA},
				Validate:    func(a *Args) error { return validateBitImm(name, a) },
				Steps: []EmitStep{
					{WordBits: op2, Fields: []FieldRef{F_DstEA}},
					{Trailer: []TrailerItem{T_DstEAExt, T_SrcImm}},
				},
			},
		},
	}

}

var defBTST = InstrDef{
	Mnemonic: "BTST",
	Forms: []FormDef{
		{
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OPK_Dn, OPK_EA},
			Validate:    validateBitTestReg,
			Steps: []EmitStep{
				{WordBits: 0x0100, Fields: []FieldRef{F_SrcDnRegHi, F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt}},
			},
		},
		{
			DefaultSize: ByteSize,
			Sizes:       []Size{ByteSize},
			OperKinds:   []OperandKind{OPK_Imm, OPK_EA},
			Validate:    validateBitTestImm,
			Steps: []EmitStep{
				{WordBits: 0x0800, Fields: []FieldRef{F_DstEA}},
				{Trailer: []TrailerItem{T_DstEAExt, T_SrcImm}},
			},
		},
	},
}

func validateBitReg(name string, a *Args) error {
	if a.Src.Kind != EAkDn {
		return fmt.Errorf("%s requires Dn source", name)
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

func validateBitImm(name string, a *Args) error {
	if a.Src.Kind != EAkImm {
		return fmt.Errorf("%s requires immediate source", name)
	}
	if err := checkImmediateRange(a.Src.Imm, ByteSize); err != nil {
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

func validateBitTestReg(a *Args) error {
	if a.Src.Kind != EAkDn {
		return fmt.Errorf("BTST requires Dn source")
	}
	switch a.Dst.Kind {
	case EAkDn, EAkAddrInd, EAkAddrPostinc, EAkAddrPredec, EAkAddrDisp16, EAkIdxAnBrief, EAkAbsW, EAkAbsL, EAkPCDisp16, EAkIdxPCBrief:
		return nil
	case EAkNone:
		return fmt.Errorf("BTST requires destination")
	default:
		return fmt.Errorf("BTST destination must be data register or memory EA")
	}
}

func validateBitTestImm(a *Args) error {
	if a.Src.Kind != EAkImm {
		return fmt.Errorf("BTST requires immediate source")
	}
	if err := checkImmediateRange(a.Src.Imm, ByteSize); err != nil {
		return err
	}
	switch a.Dst.Kind {
	case EAkDn, EAkAddrInd, EAkAddrPostinc, EAkAddrPredec, EAkAddrDisp16, EAkIdxAnBrief, EAkAbsW, EAkAbsL, EAkPCDisp16, EAkIdxPCBrief:
		return nil
	case EAkNone:
		return fmt.Errorf("BTST requires destination")
	default:
		return fmt.Errorf("BTST destination must be data register or memory EA")
	}
}
