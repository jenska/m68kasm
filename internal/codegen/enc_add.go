package codegen

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/jenska/m68kasm/internal/parser"
)

// ADD instruction encoder for ADD.[BWL] Dn,<ea> and <ea>,Dn forms (Data or Address register as destination).
// This implementation supports the most common 68000 ADD forms.

type addEncoder struct{}

func (e addEncoder) Encode(instr parser.Instruction, ops []OperandInfo, symbols map[string]int) ([]byte, error) {
	if len(ops) != 2 {
		return nil, fmt.Errorf("ADD requires 2 operands")
	}
	src, dst := ops[0], ops[1]
	size := instr.Size

	// Determine if this is "ADD Dn,<ea>" or "ADD <ea>,Dn"
	// 1. ADD Dn,<ea> (ea alterable, Dn is source, <ea> is dest)
	// 2. ADD <ea>,Dn (Dn is dest, <ea> is source)

	// Detect which form: if dst is Dn, then it's Dn = Dn + <ea>
	dnDst := dst.Type == Register && strings.HasPrefix(dst.Value, "D")
	dnSrc := src.Type == Register && strings.HasPrefix(src.Value, "D")

	var opc uint16
	var code []byte
	var extWords []uint16

	// Only size.B, size.W, size.L
	var sizeBits uint16
	switch size {
	case parser.SizeByte:
		sizeBits = 0
	case parser.SizeWord:
		sizeBits = 1
	case parser.SizeLong:
		sizeBits = 2
	default:
		return nil, fmt.Errorf("invalid size for ADD")
	}

	if dnDst {
		// ADD <ea>,Dn
		// 1101|nnn|ss|mmm|rrr
		// nnn = Dn dest, ss = size, mmm/rrr = EA src

		// EA cannot be immediate, An, etc. Only data alterable sources.
		if src.Type == Immediate {
			return nil, fmt.Errorf("immediate not allowed as source for this ADD form")
		}
		dstn := dst.Value[1] - '0'
		srcMode, srcReg, ext, err := encodeEffectiveAddress(src, symbols, true)
		if err != nil {
			return nil, fmt.Errorf("invalid source for ADD: %w", err)
		}
		extWords = ext
		opc = 0xD000 | (uint16(dstn)<<9) | (sizeBits<<6) | (srcMode<<3) | srcReg
		code = make([]byte, 2+2*len(extWords))
		binary.BigEndian.PutUint16(code[:2], opc)
		for i, w := range extWords {
			binary.BigEndian.PutUint16(code[2+2*i:4+2*i], w)
		}
		return code, nil
	} else if dnSrc {
		// ADD Dn,<ea>
		// 1101|rrr|1ss|mmm|nnn
		// rrr = Dn src, ss = size, mmm/nnn = EA dest

		// EA must be data alterable destination (not immediate, PC, An, etc.)
		if dst.Type == Immediate {
			return nil, fmt.Errorf("immediate not allowed as destination for this ADD form")
		}
		srcn := src.Value[1] - '0'
		dstMode, dstReg, ext, err := encodeEffectiveAddress(dst, symbols, false)
		if err != nil {
			return nil, fmt.Errorf("invalid destination for ADD: %w", err)
		}
		extWords = ext
		// The "direction" bit is 1, size shifted one more to the left
		opc = 0xD100 | (uint16(srcn)<<9) | (sizeBits<<6) | (dstMode<<3) | dstReg
		code = make([]byte, 2+2*len(extWords))
		binary.BigEndian.PutUint16(code[:2], opc)
		for i, w := range extWords {
			binary.BigEndian.PutUint16(code[2+2*i:4+2*i], w)
		}
		return code, nil
	} else {
		return nil, fmt.Errorf("ADD must be Dn,<ea> or <ea>,Dn")
	}
}

func init() {
	// Register ADD.[BWL] for both Dn,<ea> and <ea>,Dn
	for _, sz := range []parser.OperandSize{parser.SizeByte, parser.SizeWord, parser.SizeLong} {
		// Dn,<ea> (Dn is source)
		for _, dst := range []OperandType{
			Register, AddressRegisterIndirect, AddressRegisterIndirectPostInc,
			AddressRegisterIndirectPreDec, AddressRegisterIndirectDispl, AddressRegisterIndirectIndex,
			AbsoluteShort, AbsoluteLong, Label,
		} {
			RegisterPatternEncoder(
				Pattern{
					Mnemonic:     "ADD",
					Size:         sz,
					OperandTypes: []OperandType{Register, dst},
				},
				addEncoder{},
			)
		}
		// <ea>,Dn (Dn is destination)
		for _, src := range []OperandType{
			Register, AddressRegisterIndirect, AddressRegisterIndirectPostInc,
			AddressRegisterIndirectPreDec, AddressRegisterIndirectDispl, AddressRegisterIndirectIndex,
			AbsoluteShort, AbsoluteLong, Label,
		} {
			RegisterPatternEncoder(
				Pattern{
					Mnemonic:     "ADD",
					Size:         sz,
					OperandTypes: []OperandType{src, Register},
				},
				addEncoder{},
			)
		}
	}
}