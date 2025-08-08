package codegen

import (
	"github.com/jenska/m68kasm/internal/parser"
)

type OperandType int

const (
	Immediate OperandType = iota
	Register
	AddressRegisterIndirect
	AddressRegisterIndirectPostInc
	AddressRegisterIndirectPreDec
	AddressRegisterIndirectDispl
	AddressRegisterIndirectIndex
	AbsoluteShort
	AbsoluteLong
	ProgramCounterDispl
	ProgramCounterIndex
	Label
	Unknown
)

type OperandInfo struct {
	Type  OperandType
	Value string
}

type Pattern struct {
	Mnemonic     string
	Size         parser.OperandSize
	OperandTypes []OperandType
}

type InstructionEncoder interface {
	Encode(instr parser.Instruction, ops []OperandInfo, symbols map[string]int) ([]byte, error)
}

type patternEncoder struct {
	Pattern Pattern
	Encoder InstructionEncoder
}

var encoderRegistry []patternEncoder

func RegisterPatternEncoder(pattern Pattern, encoder InstructionEncoder) {
	encoderRegistry = append(encoderRegistry, patternEncoder{pattern, encoder})
}

func FindEncoder(instr parser.Instruction, ops []OperandInfo) InstructionEncoder {
	for _, entry := range encoderRegistry {
		if entry.Pattern.Mnemonic == instr.Mnemonic &&
			entry.Pattern.Size == instr.Size &&
			len(entry.Pattern.OperandTypes) == len(ops) {
			match := true
			for i, t := range entry.Pattern.OperandTypes {
				if ops[i].Type != t {
					match = false
					break
				}
			}
			if match {
				return entry.Encoder
			}
		}
	}
	return nil
}