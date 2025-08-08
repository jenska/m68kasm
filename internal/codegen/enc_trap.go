package codegen

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/jenska/m68kasm/internal/parser"
)

// TRAP instruction encoder for TRAP #vector (vector = 0..15)
type trapEncoder struct{}

func (e trapEncoder) Encode(instr parser.Instruction, ops []OperandInfo, symbols map[string]int) ([]byte, error) {
	if len(ops) != 1 {
		return nil, fmt.Errorf("TRAP at 0x%04X: requires exactly 1 operand", instr.Address)
	}
	op := ops[0]
	var vec int
	if op.Type == Immediate {
		val := op.Value
		if strings.HasPrefix(val, "#") {
			val = val[1:]
		}
		if strings.HasPrefix(val, "$") {
			_, err := fmt.Sscanf(val, "$%x", &vec)
			if err != nil {
				return nil, fmt.Errorf("TRAP at 0x%04X: invalid hex trap vector: %s", instr.Address, op.Value)
			}
		} else {
			_, err := fmt.Sscanf(val, "%d", &vec)
			if err != nil {
				return nil, fmt.Errorf("TRAP at 0x%04X: invalid decimal trap vector: %s", instr.Address, op.Value)
			}
		}
	} else {
		return nil, fmt.Errorf("TRAP at 0x%04X: operand must be immediate, got %s", instr.Address, op.Value)
	}
	if vec < 0 || vec > 15 {
		return nil, fmt.Errorf("TRAP at 0x%04X: vector must be 0..15, got %d", instr.Address, vec)
	}
	opc := 0x4E40 | uint16(vec)
	code := make([]byte, 2)
	binary.BigEndian.PutUint16(code, opc)
	return code, nil
}

func init() {
	RegisterPatternEncoder(
		Pattern{
			Mnemonic:     "TRAP",
			Size:         parser.SizeUnknown,
			OperandTypes: []OperandType{Immediate},
		},
		trapEncoder{},
	)
}