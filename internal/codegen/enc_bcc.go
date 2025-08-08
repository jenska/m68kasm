package codegen

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/jenska/m68kasm/internal/parser"
)

// opcode table for all Bcc
var branchOpcodes = map[string]uint16{
	"BRA": 0x6000, "BSR": 0x6100, "BHI": 0x6200, "BLS": 0x6300,
	"BCC": 0x6400, "BCS": 0x6500, "BNE": 0x6600, "BEQ": 0x6700,
	"BVC": 0x6800, "BVS": 0x6900, "BPL": 0x6A00, "BMI": 0x6B00,
	"BGE": 0x6C00, "BLT": 0x6D00, "BGT": 0x6E00, "BLE": 0x6F00,
}

type bccEncoder struct{}

func (e bccEncoder) Encode(instr parser.Instruction, ops []OperandInfo, symbols map[string]int) ([]byte, error) {
	if len(ops) != 1 {
		return nil, fmt.Errorf("%s at 0x%04X: requires exactly 1 operand, got %d", instr.Mnemonic, instr.Address, len(ops))
	}
	target := ops[0]
	var targetAddr int
	switch target.Type {
	case Label:
		addr, ok := symbols[target.Value]
		if !ok {
			return nil, fmt.Errorf("%s at 0x%04X: unknown label: %s", instr.Mnemonic, instr.Address, target.Value)
		}
		targetAddr = addr
	case AbsoluteShort, AbsoluteLong:
		targetAddr = int(parseAbsolute(target.Value))
	default:
		return nil, fmt.Errorf("%s at 0x%04X: operand must be label or absolute address, got %s (type %v)", instr.Mnemonic, instr.Address, target.Value, target.Type)
	}

	pcAfterInstr := instr.Address + 2
	disp := targetAddr - pcAfterInstr
	opc, ok := branchOpcodes[strings.ToUpper(instr.Mnemonic)]
	if !ok {
		return nil, fmt.Errorf("%s at 0x%04X: unsupported branch mnemonic: %s", instr.Mnemonic, instr.Address, instr.Mnemonic)
	}

	switch instr.Size {
	case parser.SizeShort:
		if disp < -128 || disp > 127 {
			return nil, fmt.Errorf("%s at 0x%04X: displacement %d out of 8-bit range for %s.S", instr.Mnemonic, instr.Address, disp, instr.Mnemonic)
		}
		code := make([]byte, 2)
		binary.BigEndian.PutUint16(code, opc|uint16(uint8(disp)&0xFF))
		return code, nil
	case parser.SizeWord, parser.SizeUnknown:
		if disp < -32768 || disp > 32767 {
			return nil, fmt.Errorf("%s at 0x%04X: displacement %d out of 16-bit range for %s.W", instr.Mnemonic, instr.Address, disp, instr.Mnemonic)
		}
		code := make([]byte, 4)
		binary.BigEndian.PutUint16(code[0:2], opc)
		binary.BigEndian.PutUint16(code[2:4], uint16(int16(disp)))
		return code, nil
	case parser.SizeLong:
		code := make([]byte, 6)
		binary.BigEndian.PutUint16(code[0:2], opc|0xFF)
		binary.BigEndian.PutUint32(code[2:6], uint32(int32(disp)))
		return code, nil
	default:
		return nil, fmt.Errorf("%s at 0x%04X: unsupported size: %v", instr.Mnemonic, instr.Address, instr.Size)
	}
}

func init() {
	for _, mnemonic := range []string{
		"BRA", "BSR", "BHI", "BLS", "BCC", "BCS", "BNE", "BEQ",
		"BVC", "BVS", "BPL", "BMI", "BGE", "BLT", "BGT", "BLE",
	} {
		for _, sz := range []parser.OperandSize{
			parser.SizeShort, parser.SizeWord, parser.SizeLong, parser.SizeUnknown,
		} {
			for _, op := range []OperandType{Label, AbsoluteShort, AbsoluteLong} {
				RegisterPatternEncoder(
					Pattern{
						Mnemonic:     mnemonic,
						Size:         sz,
						OperandTypes: []OperandType{op},
					},
					bccEncoder{},
				)
			}
		}
	}
}