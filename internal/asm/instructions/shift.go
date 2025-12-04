package instructions

import "fmt"

func init() {
	registerInstrDef(newShiftDef("ASR", 0xE000, 0xE020, 0xE0C0))
	registerInstrDef(newShiftDef("ASL", 0xE100, 0xE120, 0xE1C0))
	registerInstrDef(newShiftDef("LSR", 0xE008, 0xE028, 0xE2C0))
	registerInstrDef(newShiftDef("LSL", 0xE108, 0xE128, 0xE3C0))
	registerInstrDef(newShiftDef("ROXR", 0xE010, 0xE030, 0xE4C0))
	registerInstrDef(newShiftDef("ROXL", 0xE110, 0xE130, 0xE5C0))
	registerInstrDef(newShiftDef("ROR", 0xE018, 0xE038, 0xE6C0))
	registerInstrDef(newShiftDef("ROL", 0xE118, 0xE138, 0xE7C0))
}

func newShiftDef(name string, immBits, regBits, memBits uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{ByteSize, WordSize, LongSize},
				OperKinds:   []OperandKind{OPK_ImmQuick, OPK_Dn},
				Validate:    validateShiftImmediate,
				Steps: []EmitStep{
					{WordBits: immBits, Fields: []FieldRef{F_QuickData, F_SizeBits, F_DstRegLow}},
				},
			},
			{
				DefaultSize: WordSize,
				Sizes:       []Size{ByteSize, WordSize, LongSize},
				OperKinds:   []OperandKind{OPK_Dn, OPK_Dn},
				Validate:    validateShiftRegister,
				Steps: []EmitStep{
					{WordBits: regBits, Fields: []FieldRef{F_SrcDnRegHi, F_SizeBits, F_DstRegLow}},
				},
			},
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OPK_EA},
				Validate:    validateShiftMemory,
				Steps: []EmitStep{
					{WordBits: memBits, Fields: []FieldRef{F_DstEA}},
					{Trailer: []TrailerItem{T_DstEAExt}},
				},
			},
		},
	}
}

func validateShiftImmediate(a *Args) error {
	if !a.HasImmQuick {
		return fmt.Errorf("shift requires immediate count")
	}
	if a.Src.Imm < 1 || a.Src.Imm > 8 {
		return fmt.Errorf("shift count out of range: %d", a.Src.Imm)
	}
	if a.Dst.Kind != EAkDn {
		return fmt.Errorf("shift destination must be data register")
	}
	return nil
}

func validateShiftRegister(a *Args) error {
	if a.Src.Kind != EAkDn || a.Dst.Kind != EAkDn {
		return fmt.Errorf("shift with register count requires Dn source and destination")
	}
	return nil
}

func validateShiftMemory(a *Args) error {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
		a.Src = EAExpr{}
	}
	if a.Src.Kind != EAkNone {
		return fmt.Errorf("memory shift takes single operand")
	}
	switch a.Dst.Kind {
	case EAkAddrInd, EAkAddrPostinc, EAkAddrDisp16, EAkAddrPredec, EAkIdxAnBrief, EAkAbsW, EAkAbsL:
		return nil
	case EAkNone:
		return fmt.Errorf("shift requires destination")
	default:
		return fmt.Errorf("memory shift requires alterable memory EA")
	}
}
