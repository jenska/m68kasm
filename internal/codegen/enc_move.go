package codegen

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/jenska/m68kasm/internal/parser"
)

// MOVE instruction encoder supporting all major m68000 addressing modes.
type moveEncoder struct{}

func (e moveEncoder) Encode(instr parser.Instruction, ops []OperandInfo, symbols map[string]int) ([]byte, error) {
	if len(ops) != 2 {
		return nil, fmt.Errorf("MOVE at 0x%04X: requires 2 operands, got %d", instr.Address, len(ops))
	}

	src, dst := ops[0], ops[1]
	size := instr.Size

	if dst.Type == Immediate || dst.Type == ProgramCounterDispl || dst.Type == ProgramCounterIndex {
		return nil, fmt.Errorf("MOVE at 0x%04X: invalid destination operand '%s' (type %v)", instr.Address, dst.Value, dst.Type)
	}

	var sizeBits uint16
	switch size {
	case parser.SizeByte:
		sizeBits = 1
	case parser.SizeWord:
		sizeBits = 3
	case parser.SizeLong:
		sizeBits = 2
	default:
		return nil, fmt.Errorf("MOVE at 0x%04X: invalid size %v (must be .B, .W, or .L)", instr.Address, size)
	}

	srcMode, srcReg, srcExt, err := encodeEffectiveAddress(src, symbols, true)
	if err != nil {
		return nil, fmt.Errorf("MOVE at 0x%04X: invalid source operand '%s': %w", instr.Address, src.Value, err)
	}
	dstMode, dstReg, dstExt, err := encodeEffectiveAddress(dst, symbols, false)
	if err != nil {
		return nil, fmt.Errorf("MOVE at 0x%04X: invalid destination operand '%s': %w", instr.Address, dst.Value, err)
	}
	extWords := append(srcExt, dstExt...)

	// MOVE opcode: 15-12=0001, 11-10=size, 9-6=dest EA, 5-3=src mode, 2-0=src reg
	opc := uint16(0x1000) |
		(sizeBits << 12) |
		(dstMode << 6) | (dstReg << 9) |
		(srcMode << 3) | srcReg

	code := make([]byte, 2+2*len(extWords))
	binary.BigEndian.PutUint16(code[:2], opc)
	for i, w := range extWords {
		binary.BigEndian.PutUint16(code[2+2*i:4+2*i], w)
	}
	return code, nil
}

func init() {
	// Register all MOVE.[BWL] forms with valid src/dst operand types
	for _, sz := range []parser.OperandSize{parser.SizeByte, parser.SizeWord, parser.SizeLong} {
		for _, src := range []OperandType{
			Register, AddressRegisterIndirect, AddressRegisterIndirectPostInc,
			AddressRegisterIndirectPreDec, AddressRegisterIndirectDispl, AddressRegisterIndirectIndex,
			AbsoluteShort, AbsoluteLong, ProgramCounterDispl, ProgramCounterIndex, Label,
		} {
			for _, dst := range []OperandType{
				Register, AddressRegisterIndirect, AddressRegisterIndirectPostInc,
				AddressRegisterIndirectPreDec, AddressRegisterIndirectDispl, AddressRegisterIndirectIndex,
				AbsoluteShort, AbsoluteLong, Label,
			} {
				RegisterPatternEncoder(
					Pattern{
						Mnemonic:     "MOVE",
						Size:         sz,
						OperandTypes: []OperandType{src, dst},
					},
					moveEncoder{},
				)
			}
		}
	}
}