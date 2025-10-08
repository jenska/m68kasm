package asm

import "fmt"

var defMOVEQ = InstrDef{
	Op:       OP_MOVEQ,
	Mnemonic: "MOVEQ",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_L},
			OperKinds: []OperandKind{OPK_Imm, OPK_Dn},
			Validate: func(a *Args) error {
				if !a.HasImm {
					return fmt.Errorf("MOVEQ needs immediate")
				}
				if a.Imm < -128 || a.Imm > 127 {
					return fmt.Errorf("MOVEQ immediate out of range")
				}
				return nil
			},
			Steps: []EmitStep{
				{WordBits: 0x7000, Fields: []FieldRef{F_DnReg, F_ImmLow8}},
			},
		},
	},
}

var defLEA = InstrDef{
	Op:       OP_LEA,
	Mnemonic: "LEA",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_L},
			OperKinds: []OperandKind{OPK_EA, OPK_An},
			Validate:  func(a *Args) error { return nil },
			Steps: []EmitStep{
				{WordBits: 0x41C0, Fields: []FieldRef{F_AnReg, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt}},
			},
		},
	},
}

var defBRA = InstrDef{
	Op:       OP_BCC,
	Mnemonic: "BRA",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_B, SZ_W},
			OperKinds: []OperandKind{OPK_DispRel},
			Validate:  nil,
			Steps: []EmitStep{
				{WordBits: 0x6000, Fields: []FieldRef{F_Cond, F_BranchLow8}},
				{Trailer: []TrailerItem{T_BranchWordIfNeeded}},
			},
		},
	},
}

var InstrTable = []InstrDef{defMOVEQ, defLEA, defBRA}
