package codegen

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/jenska/m68kasm/internal/parser"
)

// JMP instruction encoder for JMP <ea>
// Only supports control alterable addressing modes: (An), (An)+, -(An), (d16,An), (d8,An,Xn), (xxx).W, (xxx).L, and label

type jmpEncoder struct{}

func (e jmpEncoder) Encode(instr parser.Instruction, ops []OperandInfo, symbols map[string]int) ([]byte, error) {
	if len(ops) != 1 {
		return nil, fmt.Errorf("JMP at 0x%04X: requires exactly 1 operand, got %d", instr.Address, len(ops))
	}
	op := ops[0]

	// Only control alterable EA modes
	switch op.Type {
	case AddressRegisterIndirect, AddressRegisterIndirectPostInc, AddressRegisterIndirectPreDec,
		AddressRegisterIndirectDispl, AddressRegisterIndirectIndex,
		AbsoluteShort, AbsoluteLong, Label:
		// ok
	default:
		return nil, fmt.Errorf("JMP at 0x%04X: invalid operand '%s' (type %v): only control alterable addressing modes are allowed", instr.Address, op.Value, op.Type)
	}

	mode, reg, ext, err := encodeEffectiveAddress(op, symbols, true)
	if err != nil {
		return nil, fmt.Errorf("JMP at 0x%04X: failed to encode operand '%s': %w", instr.Address, op.Value, err)
	}
	opc := uint16(0x4EC0) | (mode << 3) | reg
	code := make([]byte, 2+2*len(ext))
	binary.BigEndian.PutUint16(code, opc)
	for i, w := range ext {
		binary.BigEndian.PutUint16(code[2+2*i:4+2*i], w)
	}
	return code, nil
}

func init() {
	for _, op := range []OperandType{
		AddressRegisterIndirect, AddressRegisterIndirectPostInc, AddressRegisterIndirectPreDec,
		AddressRegisterIndirectDispl, AddressRegisterIndirectIndex,
		AbsoluteShort, AbsoluteLong, Label,
	} {
		RegisterPatternEncoder(
			Pattern{
				Mnemonic:     "JMP",
				Size:         parser.SizeUnknown,
				OperandTypes: []OperandType{op},
			},
			jmpEncoder{},
		)
	}
}