package asm

import (
	"fmt"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

type Instr struct {
	Def  *instructions.InstrDef
	Args instructions.Args
	PC   uint32
	Line int
}

func sizeToBits(sz instructions.Size) uint16 {
	switch sz {
	case instructions.SZ_B:
		return 0x0000
	case instructions.SZ_W:
		return 0x0040
	case instructions.SZ_L:
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
	Size     instructions.Size

	Imm int64
	Reg int

	SrcEA instructions.EAEncoded
	DstEA instructions.EAEncoded

	TargetPC  uint32
	BrUseWord bool
	BrDisp8   int8
	BrDisp16  int16
}

func applyField(wordVal uint16, f instructions.FieldRef, p *prepared) uint16 {
	switch f {
	case instructions.F_SizeBits:
		return wordVal | p.SizeBits
	case instructions.F_SrcEA:
		return wordVal | (uint16(p.SrcEA.Mode&7) << 3) | uint16(p.SrcEA.Reg&7)
	case instructions.F_DstEA:
		return wordVal | (uint16(p.DstEA.Mode&7) << 3) | uint16(p.DstEA.Reg&7)
	case instructions.F_DnReg:
		return wordVal | (uint16(p.Reg&7) << 9)
	case instructions.F_AnReg:
		return wordVal | (uint16(p.Reg&7) << 9)
	case instructions.F_ImmLow8:
		return wordVal | uint16(uint8(p.Imm))
	case instructions.F_BranchLow8:
		if !p.BrUseWord {
			return wordVal | uint16(uint8(p.BrDisp8))
		}
		return wordVal
	case instructions.F_MoveDestEA:
		return wordVal | (uint16(p.DstEA.Mode&7) << 6) | (uint16(p.DstEA.Reg&7) << 9)
	case instructions.F_MoveSize:
		switch p.Size {
		case instructions.SZ_B:
			return wordVal | 0x1000
		case instructions.SZ_W:
			return wordVal | 0x3000
		case instructions.SZ_L:
			return wordVal | 0x2000
		default:
			return wordVal
		}
	default:
		return wordVal
	}
}

func emitTrailer(out []byte, t instructions.TrailerItem, p *prepared) ([]byte, error) {
	switch t {
	case instructions.T_SrcEAExt:
		if len(p.SrcEA.Ext) == 0 {
			return out, nil
		}
		for _, w := range p.SrcEA.Ext {
			out = appendWord(out, w)
		}
		return out, nil
	case instructions.T_DstEAExt:
		if len(p.DstEA.Ext) == 0 {
			return out, nil
		}
		for _, w := range p.DstEA.Ext {
			out = appendWord(out, w)
		}
		return out, nil
	case instructions.T_ImmSized:
		return appendWord(out, uint16(int16(p.Imm))), nil
	case instructions.T_SrcImm:
		if p.SrcEA.Mode == 7 && p.SrcEA.Reg == 4 {
			switch p.Size {
			case instructions.SZ_B:
				return appendWord(out, uint16(uint8(p.Imm))), nil
			case instructions.SZ_W:
				return appendWord(out, uint16(uint16(p.Imm))), nil
			case instructions.SZ_L:
				u := uint32(int32(p.Imm))
				out = appendWord(out, uint16(u>>16))
				out = appendWord(out, uint16(u))
				return out, nil
			default:
				return appendWord(out, uint16(uint16(p.Imm))), nil
			}
		}
		return out, nil
	case instructions.T_BranchWordIfNeeded:
		if p.BrUseWord {
			return appendWord(out, uint16(p.BrDisp16)), nil
		}
		return out, nil
	}
	return out, nil
}

func Encode(def *instructions.InstrDef, form *instructions.FormDef, ins *Instr, sym map[string]uint32) ([]byte, error) {
	p := prepared{PC: ins.PC, Size: ins.Args.Size, Imm: ins.Args.Src.Imm, Reg: ins.Args.Dst.Reg}
	var err error

	if ins.Args.Src.Kind != instructions.EAkNone {
		p.SrcEA, err = instructions.EncodeEA(ins.Args.Src)
		if err != nil {
			return nil, err
		}
	}
	if ins.Args.Dst.Kind != instructions.EAkNone {
		p.DstEA, err = instructions.EncodeEA(ins.Args.Dst)
		if err != nil {
			return nil, err
		}
	}

	p.SizeBits = sizeToBits(ins.Args.Size)

	if ins.Args.Target != "" {
		addr, ok := sym[ins.Args.Target]
		if !ok {
			return nil, fmt.Errorf("undefined label: %s", ins.Args.Target)
		}
		p.TargetPC = addr
		switch ins.Args.Size {
		case instructions.SZ_B:
			d8 := int32(addr) - int32(p.PC+2)
			if d8 < -128 || d8 > 127 {
				return nil, fmt.Errorf("branch displacement out of range for .S")
			}
			p.BrUseWord = false
			p.BrDisp8 = int8(d8)
		case instructions.SZ_W:
			// TODO
			d16 := int32(addr) - int32(p.PC+2)
			if d16 < -32768 || d16 > 32767 {
				return nil, fmt.Errorf("branch displacement out of range for .W")
			}
			p.BrUseWord = true
			p.BrDisp16 = int16(d16)
		default:
			return nil, fmt.Errorf("unsupported branch size")
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
