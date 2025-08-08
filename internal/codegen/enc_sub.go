package codegen

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/jenska/m68kasm/internal/parser"
)

type subEncoder struct{}

func (e subEncoder) Encode(instr parser.Instruction, ops []OperandInfo, symbols map[string]int) ([]byte, error) {
	if len(ops) != 2 {
		return nil, fmt.Errorf("SUB at 0x%04X: requires 2 operands, got %d", instr.Address, len(ops))
	}
	src, dst := ops[0], ops[1]
	size := instr.Size

	dnDst := dst.Type == Register && strings.HasPrefix(dst.Value, "D")
	dnSrc := src.Type == Register && strings.HasPrefix(src.Value, "D")

	var opc uint16
	var code []byte
	var extWords []uint16

	var sizeBits uint16
	switch size {
	case parser.SizeByte:
		sizeBits = 0
	case parser.SizeWord:
		sizeBits = 1
	case parser.SizeLong:
		sizeBits = 2
	default:
		return nil, fmt.Errorf("SUB at 0x%04X: invalid size %v (must be .B, .W, or .L)", instr.Address, size)
	}

	if dnDst {
		// Form: SUB <ea>,Dn
		if src.Type == Immediate {
			return nil, fmt.Errorf("SUB at 0x%04X: immediate not allowed as source in SUB <ea>,Dn", instr.Address)
		}
		dstn := dst.Value[1] - '0'
		srcMode, srcReg, ext, err := encodeEffectiveAddress(src, symbols, true)
		if err != nil {
			return nil, fmt.Errorf("SUB at 0x%04X: invalid source operand '%s': %w", instr.Address, src.Value, err)
		}
		extWords = ext
		opc = 0x9000 | (uint16(dstn)<<9) | (sizeBits<<6) | (srcMode<<3) | srcReg
		code = make([]byte, 2+2*len(extWords))
		binary.BigEndian.PutUint16(code[:2], opc)
		for i, w := range extWords {
			binary.BigEndian.PutUint16(code[2+2*i:4+2*i], w)
		}
		return code, nil
	} else if dnSrc {
		// Form: SUB Dn,<ea>
		if dst.Type == Immediate {
			return nil, fmt.Errorf("SUB at 0x%04X: immediate not allowed as destination in SUB Dn,<ea>", instr.Address)
		}
		srcn := src.Value[1] - '0'
		dstMode, dstReg, ext, err := encodeEffectiveAddress(dst, symbols, false)
		if err != nil {
			return nil, fmt.Errorf("SUB at 0x%04X: invalid destination operand '%s': %w", instr.Address, dst.Value, err)
		}
		extWords = ext
		opc = 0x9100 | (uint16(srcn)<<9) | (sizeBits<<6) | (dstMode<<3) | dstReg
		code = make([]byte, 2+2*len(extWords))
		binary.BigEndian.PutUint16(code[:2], opc)
		for i, w := range extWords {
			binary.BigEndian.PutUint16(code[2+2*i:4+2*i], w)
		}
		return code, nil
	} else {
		return nil, fmt.Errorf("SUB at 0x%04X: operands must be Dn,<ea> or <ea>,Dn; got %s,%s", instr.Address, src.Value, dst.Value)
	}
}

func init() {
	for _, sz := range []parser.OperandSize{parser.SizeByte, parser.SizeWord, parser.SizeLong} {
		for _, dst := range []OperandType{
			Register, AddressRegisterIndirect, AddressRegisterIndirectPostInc,
			AddressRegisterIndirectPreDec, AddressRegisterIndirectDispl, AddressRegisterIndirectIndex,
			AbsoluteShort, AbsoluteLong, Label,
		} {
			RegisterPatternEncoder(
				Pattern{
					Mnemonic:     "SUB",
					Size:         sz,
					OperandTypes: []OperandType{Register, dst},
				},
				subEncoder{},
			)
		}
		for _, src := range []OperandType{
			Register, AddressRegisterIndirect, AddressRegisterIndirectPostInc,
			AddressRegisterIndirectPreDec, AddressRegisterIndirectDispl, AddressRegisterIndirectIndex,
			AbsoluteShort, AbsoluteLong, Label,
		} {
			RegisterPatternEncoder(
				Pattern{
					Mnemonic:     "SUB",
					Size:         sz,
					OperandTypes: []OperandType{src, Register},
				},
				subEncoder{},
			)
		}
	}
}