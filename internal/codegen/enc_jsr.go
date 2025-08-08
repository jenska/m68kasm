package codegen

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/jenska/m68kasm/internal/parser"
)

// JSR instruction encoder for JSR <ea>
// Only supports control alterable addressing modes: (An), (d16,An), (d8,An,Xn), (xxx).W, (xxx).L, and label

type jsrEncoder struct{}

func (e jsrEncoder) Encode(instr parser.Instruction, ops []OperandInfo, symbols map[string]int) ([]byte, error) {
	if len(ops) != 1 {
		return nil, fmt.Errorf("JSR at 0x%04X: requires exactly 1 operand, got %d", instr.Address, len(ops))
	}
	op := ops[0]

	// Only control alterable EA modes (not (An)+ or -(An)!)
	switch op.Type {
	case AddressRegisterIndirect, AddressRegisterIndirectDispl, AddressRegisterIndirectIndex,
		AbsoluteShort, AbsoluteLong, Label:
		// ok
	case AddressRegisterIndirectPostInc, AddressRegisterIndirectPreDec:
		return nil, fmt.Errorf("JSR at 0x%04X: invalid operand '%s' (type %v): (An)+ and -(An) are not allowed for JSR", instr.Address, op.Value, op.Type)
	default:
		return nil, fmt.Errorf("JSR at 0x%04X: invalid operand '%s' (type %v): only control alterable addressing modes are allowed", instr.Address, op.Value, op.Type)
	}

	mode, reg, ext, err := encodeEffectiveAddress(op, symbols, true)
	if err != nil {
		return nil, fmt.Errorf("JSR at 0x%04X: failed to encode operand '%s': %w", instr.Address, op.Value, err)
	}
	opc := uint16(0x4E80) | (mode << 3) | reg
	code := make([]byte, 2+2*len(ext))
	binary.BigEndian.PutUint16(code, opc)
	for i, w := range ext {
		binary.BigEndian.PutUint16(code[2+2*i:4+2*i], w)
	}
	return code, nil
}

func init() {
	for _, op := range []OperandType{
		AddressRegisterIndirect, AddressRegisterIndirectDispl, AddressRegisterIndirectIndex,
		AbsoluteShort, AbsoluteLong, Label,
	} {
		RegisterPatternEncoder(
			Pattern{
				Mnemonic:     "JSR",
				Size:         parser.SizeUnknown,
				OperandTypes: []OperandType{op},
			},
			jsrEncoder{},
		)
	}
}