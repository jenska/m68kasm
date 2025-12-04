package instructions

import "fmt"

func init() {
	registerInstrDef(&defCMP)
	registerInstrDef(&defCMPM)
}

var defCMP = InstrDef{
	Mnemonic: "CMP",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OPK_EA, OPK_Dn},
			Validate:    validateCMP,
			Steps: []EmitStep{
				{WordBits: 0xB000, Fields: []FieldRef{F_DnReg, F_SizeBits, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
			},
		},
	},
}

var defCMPM = InstrDef{
	Mnemonic: "CMPM",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OPK_EA, OPK_EA},
			Validate:    validateCMPM,
			Steps: []EmitStep{
				{WordBits: 0xB108, Fields: []FieldRef{F_AnReg, F_SizeBits, F_SrcAnReg}},
			},
		},
	},
}

func validateCMP(a *Args) error {
	if a.Src.Kind == EAkNone || a.Dst.Kind != EAkDn {
		return fmt.Errorf("CMP requires Dn destination")
	}
	if a.Src.Kind == EAkImm {
		if err := checkImmediateRange(a.Src.Imm, a.Size); err != nil {
			return err
		}
	}
	return nil
}

func validateCMPM(a *Args) error {
	if a.Src.Kind != EAkAddrPostinc || a.Dst.Kind != EAkAddrPostinc {
		return fmt.Errorf("CMPM requires post-increment address operands")
	}
	return nil
}
