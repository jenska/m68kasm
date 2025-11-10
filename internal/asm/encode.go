package asm

import "fmt"

type FieldRef int

const (
	F_SrcEA FieldRef = iota
	F_DstEA
	F_SizeBits
	F_AnReg
	F_DnReg
	F_Cond
	F_ImmLow8
	F_BranchLow8
	F_MoveDestEA
	F_MoveSize
)

type TrailerItem int

const (
	T_SrcEAExt TrailerItem = iota
	T_DstEAExt
	T_ImmSized
	T_SrcImm
	T_BranchWordIfNeeded
)

type Size int

const (
	SZ_B Size = iota
	SZ_W
	SZ_L
)

type Cond uint8

const (
	CondT Cond = 0x0
)

type OperandKind int

const (
	OPK_None OperandKind = iota
	OPK_Imm
	OPK_Dn
	OPK_An
	OPK_EA
	OPK_DispRel
)

type Instr struct {
	Op       Opcode
	Mnemonic string
	Size     Size
	Args     Args
	PC       uint32
	Line     int
}

func sizeToBits(sz Size) uint16 {
	switch sz {
	case SZ_B:
		return 0x0000
	case SZ_W:
		return 0x0040
	case SZ_L:
		return 0x0080
	default:
		return 0
	}
}

func appendWord(out []byte, v uint16) []byte {
	return append(out, byte(v>>8), byte(v))
}

type prepared struct {
	PC       uint32
	SizeBits uint16
	Size     Size

	Imm  int64
	Dn   int
	An   int
	Cond uint8

	SrcEA EAEncoded
	DstEA EAEncoded

	TargetPC  uint32
	BrUseWord bool
	BrDisp8   int8
	BrDisp16  int16
}

type EmitStep struct {
	WordBits uint16
	Fields   []FieldRef
	Trailer  []TrailerItem
}

func applyField(wordVal uint16, f FieldRef, p *prepared) uint16 {
	switch f {
	case F_SizeBits:
		return wordVal | p.SizeBits
	case F_SrcEA:
		return wordVal | (uint16(p.SrcEA.Mode&7) << 3) | uint16(p.SrcEA.Reg&7)
	case F_DstEA:
		return wordVal | (uint16(p.DstEA.Mode&7) << 3) | uint16(p.DstEA.Reg&7)
	case F_DnReg:
		return wordVal | (uint16(p.Dn&7) << 9)
	case F_AnReg:
		return wordVal | (uint16(p.An&7) << 9)
	case F_Cond:
		return wordVal | (uint16(p.Cond&0x0F) << 8)
	case F_ImmLow8:
		return wordVal | uint16(uint8(p.Imm))
	case F_BranchLow8:
		if !p.BrUseWord {
			return wordVal | uint16(uint8(p.BrDisp8))
		}
		return wordVal
	case F_MoveDestEA:
		return wordVal | (uint16(p.DstEA.Mode&7) << 6) | (uint16(p.DstEA.Reg&7) << 9)
	case F_MoveSize:
		switch p.Size {
		case SZ_B:
			return wordVal | 0x1000
		case SZ_W:
			return wordVal | 0x3000
		case SZ_L:
			return wordVal | 0x2000
		default:
			return wordVal
		}
	default:
		return wordVal
	}
}

func emitTrailer(out []byte, t TrailerItem, p *prepared) ([]byte, error) {
	switch t {
	case T_SrcEAExt:
		if len(p.SrcEA.Ext) == 0 {
			return out, nil
		}
		for _, w := range p.SrcEA.Ext {
			out = appendWord(out, w)
		}
		return out, nil
	case T_DstEAExt:
		if len(p.DstEA.Ext) == 0 {
			return out, nil
		}
		for _, w := range p.DstEA.Ext {
			out = appendWord(out, w)
		}
		return out, nil
	case T_ImmSized:
		return appendWord(out, uint16(int16(p.Imm))), nil
	case T_SrcImm:
		if p.SrcEA.Mode == 7 && p.SrcEA.Reg == 4 {
			switch p.Size {
			case SZ_B:
				return appendWord(out, uint16(uint8(p.Imm))), nil
			case SZ_W:
				return appendWord(out, uint16(uint16(p.Imm))), nil
			case SZ_L:
				u := uint32(int32(p.Imm))
				out = appendWord(out, uint16(u>>16))
				out = appendWord(out, uint16(u))
				return out, nil
			default:
				return appendWord(out, uint16(uint16(p.Imm))), nil
			}
		}
		return out, nil
	case T_BranchWordIfNeeded:
		if p.BrUseWord {
			return appendWord(out, uint16(p.BrDisp16)), nil
		}
		return out, nil
	}
	return out, nil
}

func Encode(def *InstrDef, form *FormDef, ins *Instr, sym map[string]uint32) ([]byte, error) {
	p := prepared{PC: ins.PC, Size: ins.Size, Imm: ins.Args.Imm, Dn: ins.Args.Dn, An: ins.Args.An, Cond: uint8(ins.Args.Cond)}
	var err error

	if ins.Args.Src.Kind != EAkNone {
		p.SrcEA, err = encodeEA(ins.Args.Src)
		if err != nil {
			return nil, err
		}
	}
	if ins.Args.Dst.Kind != EAkNone {
		p.DstEA, err = encodeEA(ins.Args.Dst)
		if err != nil {
			return nil, err
		}
	}

	p.SizeBits = sizeToBits(ins.Size)

	if ins.Args.Target != "" {
		addr, ok := sym[ins.Args.Target]
		if !ok {
			return nil, fmt.Errorf("undefined label: %s", ins.Args.Target)
		}
		p.TargetPC = addr
		// try short (PC+2), else word (PC+4)
		d8 := int32(addr) - int32(p.PC+2)
		if d8 >= -128 && d8 <= 127 && d8 != 0 {
			p.BrUseWord = false
			p.BrDisp8 = int8(d8)
		} else {
			d16 := int32(addr) - int32(p.PC+4)
			if d16 < -32768 || d16 > 32767 {
				return nil, fmt.Errorf("branch displacement out of range")
			}
			p.BrUseWord = true
			p.BrDisp16 = int16(d16)
		}
	}

	out := make([]byte, 0, 12)
	for _, step := range form.Steps {
		haveWord := (step.WordBits != 0) || (len(step.Fields) > 0)
		if haveWord {
			w := step.WordBits
			for _, f := range step.Fields {
				w = applyField(w, f, &p)
			}
			out = append(out, byte(w>>8), byte(w))
		}
		for _, tr := range step.Trailer {
			var err error
			out, err = emitTrailer(out, tr, &p)
			if err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}
