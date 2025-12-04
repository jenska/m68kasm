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
	case instructions.ByteSize:
		return 0x0000
	case instructions.WordSize:
		return 0x0040
	case instructions.LongSize:
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

	Imm        int64
	SrcReg     int
	DstReg     int
	SrcRegMask uint16
	DstRegMask uint16

	SrcEA instructions.EAEncoded
	DstEA instructions.EAEncoded

	TargetPC  uint32
	BrUseWord bool
	BrDisp8   int8
	BrDisp16  int16
}

func applyField(wordVal uint16, f instructions.FieldRef, p *prepared) uint16 {
	switch f {
	case instructions.FSizeBits:
		return wordVal | p.SizeBits
	case instructions.FSrcEA:
		return wordVal | (uint16(p.SrcEA.Mode&7) << 3) | uint16(p.SrcEA.Reg&7)
	case instructions.FDstEA:
		return wordVal | (uint16(p.DstEA.Mode&7) << 3) | uint16(p.DstEA.Reg&7)
	case instructions.FDnReg:
		return wordVal | (uint16(p.DstReg&7) << 9)
	case instructions.FAnReg:
		return wordVal | (uint16(p.DstReg&7) << 9)
	case instructions.FImmLow8:
		return wordVal | uint16(uint8(p.Imm))
	case instructions.FBranchLow8:
		if !p.BrUseWord {
			return wordVal | uint16(uint8(p.BrDisp8))
		}
		return wordVal
	case instructions.FMoveDestEA:
		return wordVal | (uint16(p.DstEA.Mode&7) << 6) | (uint16(p.DstEA.Reg&7) << 9)
	case instructions.FMoveSize:
		switch p.Size {
		case instructions.ByteSize:
			return wordVal | 0x1000
		case instructions.WordSize:
			return wordVal | 0x3000
		case instructions.LongSize:
			return wordVal | 0x2000
		default:
			return wordVal
		}
	case instructions.FQuickData:
		quick := uint16(p.Imm)
		if quick == 8 {
			quick = 0
		}
		return wordVal | (quick&7)<<9
	case instructions.FSrcDnReg:
		return wordVal | uint16(p.SrcReg&7)
	case instructions.FSrcAnReg:
		return wordVal | uint16(p.SrcReg&7)
	case instructions.FDstRegLow:
		return wordVal | uint16(p.DstReg&7)
	case instructions.FMovemSize:
		if p.Size == instructions.LongSize {
			return wordVal | 0x0040
		}
		return wordVal
	case instructions.FAddaSize:
		if p.Size == instructions.LongSize {
			return wordVal | 0x0100
		}
		return wordVal
	case instructions.FSrcDnRegHi:
		return wordVal | (uint16(p.SrcReg&7) << 9)
	default:
		return wordVal
	}
}

func emitTrailer(out []byte, t instructions.TrailerItem, p *prepared) ([]byte, error) {
	switch t {
	case instructions.TSrcEAExt:
		if len(p.SrcEA.Ext) == 0 {
			return out, nil
		}
		for _, w := range p.SrcEA.Ext {
			out = appendWord(out, w)
		}
		return out, nil
	case instructions.TDstEAExt:
		if len(p.DstEA.Ext) == 0 {
			return out, nil
		}
		for _, w := range p.DstEA.Ext {
			out = appendWord(out, w)
		}
		return out, nil
	case instructions.TImmSized:
		return appendWord(out, uint16(int16(p.Imm))), nil
	case instructions.TSrcImm:
		if p.SrcEA.Mode == 7 && p.SrcEA.Reg == 4 {
			switch p.Size {
			case instructions.ByteSize:
				return appendWord(out, uint16(uint8(p.Imm))), nil
			case instructions.WordSize:
				return appendWord(out, uint16(uint16(p.Imm))), nil
			case instructions.LongSize:
				u := uint32(int32(p.Imm))
				out = appendWord(out, uint16(u>>16))
				out = appendWord(out, uint16(u))
				return out, nil
			default:
				return appendWord(out, uint16(uint16(p.Imm))), nil
			}
		}
		return out, nil
	case instructions.TBranchWordIfNeeded:
		if p.BrUseWord {
			return appendWord(out, uint16(p.BrDisp16)), nil
		}
		return out, nil
	case instructions.TSrcRegMask:
		mask := p.SrcRegMask
		if p.DstEA.Mode == 4 {
			mask = reverse16(mask)
		}
		return appendWord(out, mask), nil
	case instructions.TDstRegMask:
		return appendWord(out, p.DstRegMask), nil
	}
	return out, nil
}

func Encode(def *instructions.InstrDef, form *instructions.FormDef, ins *Instr, sym map[string]uint32) ([]byte, error) {
	p := prepared{PC: ins.PC, Size: ins.Args.Size, Imm: ins.Args.Src.Imm, SrcReg: ins.Args.Src.Reg, DstReg: ins.Args.Dst.Reg, SrcRegMask: ins.Args.RegMaskSrc, DstRegMask: ins.Args.RegMaskDst}
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
		basePC := p.PC + 2
		switch ins.Args.Size {
		case instructions.ByteSize:
			d8 := int32(addr) - int32(basePC)
			if d8 < -128 || d8 > 127 {
				return nil, fmt.Errorf("branch displacement out of range for .S")
			}
			p.BrUseWord = false
			p.BrDisp8 = int8(d8)
		case instructions.WordSize:
			basePC += 2
			d16 := int32(addr) - int32(basePC)
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

func reverse16(v uint16) uint16 {
	v = (v >> 8) | (v << 8)
	v = ((v & 0xF0F0) >> 4) | ((v & 0x0F0F) << 4)
	v = ((v & 0xCCCC) >> 2) | ((v & 0x3333) << 2)
	v = ((v & 0xAAAA) >> 1) | ((v & 0x5555) << 1)
	return v
}
