package asm

import "fmt"

type Opcode int

const (
	OP_MOVEQ Opcode = iota
	OP_MOVE
	OP_ADD
	OP_SUB
	OP_MULTI
	OP_DIV
	OP_CMP
	OP_LEA
	OP_BCC
)

type (
	InstrDef struct {
		Op       Opcode
		Mnemonic string
		Forms    []FormDef
	}

	FormDef struct {
		Sizes     []Size
		OperKinds []OperandKind
		Validate  func(*Args) error
		Steps     []EmitStep
	}
)

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

var defMOVE = InstrDef{
	Op:       OP_MOVE,
	Mnemonic: "MOVE",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_B, SZ_W, SZ_L},
			OperKinds: []OperandKind{OPK_EA, OPK_EA},
			Validate:  validateMOVE,
			Steps: []EmitStep{
				{WordBits: 0x0000, Fields: []FieldRef{F_MoveSize, F_MoveDestEA, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm, T_DstEAExt}},
			},
		},
	},
}

var defADD = InstrDef{
	Op:       OP_ADD,
	Mnemonic: "ADD",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_B, SZ_W, SZ_L},
			OperKinds: []OperandKind{OPK_EA, OPK_Dn},
			Validate:  func(a *Args) error { return validateAddSub("ADD", a) },
			Steps: []EmitStep{
				{WordBits: 0xD000, Fields: []FieldRef{F_DnReg, F_SizeBits, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
			},
		},
	},
}

var defSUB = InstrDef{
	Op:       OP_SUB,
	Mnemonic: "SUB",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_B, SZ_W, SZ_L},
			OperKinds: []OperandKind{OPK_EA, OPK_Dn},
			Validate:  func(a *Args) error { return validateAddSub("SUB", a) },
			Steps: []EmitStep{
				{WordBits: 0x9000, Fields: []FieldRef{F_DnReg, F_SizeBits, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
			},
		},
	},
}

var defMULTI = InstrDef{
	Op:       OP_MULTI,
	Mnemonic: "MULTI",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_W},
			OperKinds: []OperandKind{OPK_EA, OPK_Dn},
			Validate:  validateMULTI,
			Steps: []EmitStep{
				{WordBits: 0xC1C0, Fields: []FieldRef{F_DnReg, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt}},
			},
		},
	},
}

var defDIV = InstrDef{
	Op:       OP_DIV,
	Mnemonic: "DIV",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_W},
			OperKinds: []OperandKind{OPK_EA, OPK_Dn},
			Validate:  validateDIV,
			Steps: []EmitStep{
				{WordBits: 0x80C0, Fields: []FieldRef{F_DnReg, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt}},
			},
		},
	},
}

var defCMP = InstrDef{
	Op:       OP_CMP,
	Mnemonic: "CMP",
	Forms: []FormDef{
		{
			Sizes:     []Size{SZ_B, SZ_W, SZ_L},
			OperKinds: []OperandKind{OPK_EA, OPK_Dn},
			Validate:  validateCMP,
			Steps: []EmitStep{
				{WordBits: 0xB000, Fields: []FieldRef{F_DnReg, F_SizeBits, F_SrcEA}},
				{Trailer: []TrailerItem{T_SrcEAExt, T_SrcImm}},
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

var InstrTable = []InstrDef{defMOVEQ, defMOVE, defADD, defSUB, defMULTI, defDIV, defCMP, defLEA, defBRA}

func validateMOVE(a *Args) error {
	if a.Src.Kind == EAkNone || a.Dst.Kind == EAkNone {
		return fmt.Errorf("MOVE requires source and destination")
	}
	if a.Dst.Kind == EAkImm {
		return fmt.Errorf("MOVE destination cannot be immediate")
	}
	if isPCRelativeKind(a.Dst.Kind) {
		return fmt.Errorf("MOVE destination cannot be PC-relative")
	}
	if a.Size == SZ_B {
		if a.Src.Kind == EAkAn {
			return fmt.Errorf("MOVE.B cannot read from address register")
		}
		if a.Dst.Kind == EAkAn {
			return fmt.Errorf("MOVE.B cannot write to address register")
		}
	}
	if a.Src.Kind == EAkImm {
		if err := checkImmediateRange(a.Imm, a.Size); err != nil {
			return err
		}
	}
	return nil
}

func validateAddSub(name string, a *Args) error {
	if a.Src.Kind == EAkNone || a.Dst.Kind != EAkDn {
		return fmt.Errorf("%s requires Dn destination", name)
	}
	if a.Size == SZ_B && a.Src.Kind == EAkAn {
		return fmt.Errorf("%s.B does not allow address register source", name)
	}
	if a.Src.Kind == EAkImm {
		if err := checkImmediateRange(a.Imm, a.Size); err != nil {
			return err
		}
	}
	return nil
}

func validateMULTI(a *Args) error {
	if a.Dst.Kind != EAkDn {
		return fmt.Errorf("MULTI destination must be data register")
	}
	if a.Size != SZ_W {
		return fmt.Errorf("MULTI operates on word size")
	}
	if a.Src.Kind == EAkImm {
		return fmt.Errorf("MULTI does not allow immediate source")
	}
	if a.Src.Kind == EAkAn {
		return fmt.Errorf("MULTI does not allow address register source")
	}
	return nil
}

func validateDIV(a *Args) error {
	if a.Dst.Kind != EAkDn {
		return fmt.Errorf("DIV destination must be data register")
	}
	if a.Size != SZ_W {
		return fmt.Errorf("DIV operates on word size")
	}
	if a.Src.Kind == EAkImm {
		return fmt.Errorf("DIV does not allow immediate source")
	}
	if a.Src.Kind == EAkAn {
		return fmt.Errorf("DIV does not allow address register source")
	}
	return nil
}

func validateCMP(a *Args) error {
	if a.Src.Kind == EAkNone || a.Dst.Kind != EAkDn {
		return fmt.Errorf("CMP requires Dn destination")
	}
	if a.Src.Kind == EAkImm {
		if err := checkImmediateRange(a.Imm, a.Size); err != nil {
			return err
		}
	}
	return nil
}

func checkImmediateRange(v int64, sz Size) error {
	switch sz {
	case SZ_B:
		if v < -128 || v > 255 {
			return fmt.Errorf("immediate out of range for .b: %d", v)
		}
	case SZ_W:
		if v < -32768 || v > 65535 {
			return fmt.Errorf("immediate out of range for .w: %d", v)
		}
	case SZ_L:
		if v < -0x80000000 || v > 0xFFFFFFFF {
			return fmt.Errorf("immediate out of range for .l: %d", v)
		}
	}
	return nil
}

func isPCRelativeKind(k EAExprKind) bool {
	switch k {
	case EAkPCDisp16, EAkIdxPCBrief:
		return true
	default:
		return false
	}
}
