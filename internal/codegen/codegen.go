package codegen

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/jenska/m68kasm/internal/parser"
)

var (
	reDReg           = regexp.MustCompile(`^D[0-7]$`)
	reAReg           = regexp.MustCompile(`^A[0-7]$`)
	reAbsLong        = regexp.MustCompile(`^\$[0-9A-Fa-f]{7,8}$`)
	reAbsShort       = regexp.MustCompile(`^\$[0-9A-Fa-f]{1,4}$`)
	reAn             = regexp.MustCompile(`^\(A[0-7]\)$`)
	reAnPostInc      = regexp.MustCompile(`^\(A[0-7]\)\+$`)
	reAnPreDec       = regexp.MustCompile(`^-\(A[0-7]\)$`)
	reAnDispl        = regexp.MustCompile(`^-?\d+\(A[0-7]\)$`)
	reAnIndex        = regexp.MustCompile(`^-?\d+\(A[0-7],[DA][0-7](?:\.[WL])?\)$`)
	rePCDispl        = regexp.MustCompile(`^-?\d+\(PC\)$`)
	rePCIndex        = regexp.MustCompile(`^-?\d+\(PC,[DA][0-7](?:\.[WL])?\)$`)
)

func EmitToBuffer(instrs []parser.Instruction, symbols map[string]int) ([]byte, error) {
	var buf bytes.Buffer
	for _, instr := range instrs {
		ops := parseOperands(instr)
		encoder := FindEncoder(instr, ops)
		if encoder == nil {
			return nil, fmt.Errorf("No encoder for %s.%v at %04X", instr.Mnemonic, instr.Size, instr.Address)
		}
		code, err := encoder.Encode(instr, ops, symbols)
		if err != nil {
			return nil, fmt.Errorf("Encoding error at %04X: %v", instr.Address, err)
		}
		buf.Write(code)
	}
	return buf.Bytes(), nil
}

func parseOperands(instr parser.Instruction) []OperandInfo {
	var ops []OperandInfo
	for _, op := range instr.Operands {
		oi := OperandInfo{Type: detectOperandType(op.Value), Value: op.Value}
		ops = append(ops, oi)
	}
	return ops
}

func detectOperandType(op string) OperandType {
	switch {
	case len(op) > 0 && op[0] == '#':
		return Immediate
	case reDReg.MatchString(op) || reAReg.MatchString(op):
		return Register
	case reAn.MatchString(op):
		return AddressRegisterIndirect
	case reAnPostInc.MatchString(op):
		return AddressRegisterIndirectPostInc
	case reAnPreDec.MatchString(op):
		return AddressRegisterIndirectPreDec
	case reAnDispl.MatchString(op):
		return AddressRegisterIndirectDispl
	case reAnIndex.MatchString(op):
		return AddressRegisterIndirectIndex
	case reAbsLong.MatchString(op):
		return AbsoluteLong
	case reAbsShort.MatchString(op):
		return AbsoluteShort
	case rePCDispl.MatchString(op):
		return ProgramCounterDispl
	case rePCIndex.MatchString(op):
		return ProgramCounterIndex
	case isLabel(op):
		return Label
	default:
		return Unknown
	}
}

func isLabel(op string) bool {
	if len(op) == 0 || op[0] == '#' || op[0] == '$' {
		return false
	}
	if reDReg.MatchString(op) || reAReg.MatchString(op) {
		return false
	}
	return !strings.ContainsAny(op, " ()[]+-")
}

func parseImmediate(s string) int {
	if len(s) > 1 && s[0] == '#' {
		s = s[1:]
	}
	var v int
	if len(s) > 1 && s[0] == '$' {
		fmt.Sscanf(s[1:], "%x", &v)
	} else {
		fmt.Sscanf(s, "%d", &v)
	}
	return v
}