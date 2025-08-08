package codegen

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/jenska/m68kasm/internal/parser"
)

// MULU and DIVU instruction encoder for MULU.W <ea>,Dn and DIVU.W <ea>,Dn
type muluDivuEncoder struct {
	isMULU bool
}

func (e muluDivuEncoder) Encode(instr parser.Instruction, ops []OperandInfo, symbols map[string]int) ([]byte, error) {
	if len(ops) != 2 {
		return nil, fmt.Errorf("%s at 0x%04X: requires 2 operands, got %d", instrName(e.isMULU), instr.Address, len(ops))
	}
	src, dst := ops[0], ops[1]

	if instr.Size != parser.SizeWord {
		return nil, fmt.Errorf("%s at 0x%04X: only supports .W (word) size", instrName(e.isMULU), instr.Address)
	}
	if dst.Type != Register || !strings.HasPrefix(dst.Value, "D") {
		return nil, fmt.Errorf("%s at 0x%04X: destination must be Dn, got %s", instrName(e.isMULU), instr.Address, dst.Value)
	}
	dn := dst.Value[1] - '0'
	if src.Type == Immediate {
		return nil, fmt.Errorf("%s at 0x%04X: source cannot be immediate", instrName(e.isMULU), instr.Address)
	}
	srcMode, srcReg, srcExt, err := encodeEffectiveAddress(src, symbols, true)
	if err != nil {
		return nil, fmt.Errorf("%s at 0x%04X: invalid source operand '%s': %w", instrName(e.isMULU), instr.Address, src.Value, err)
	}
	var opc uint16
	if e.isMULU {
		opc = 0xC0C0 | (uint16(dn) << 9) | (srcMode << 3) | srcReg
	} else {
		opc = 0x80C0 | (uint16(dn) << 9) | (srcMode << 3) | srcReg
	}
	code := make([]byte, 2+2*len(srcExt))
	binary.BigEndian.PutUint16(code[:2], opc)
	for i, w := range srcExt {
		binary.BigEndian.PutUint16(code[2+2*i:4+2*i], w)
	}
	return code, nil
}

func instrName(isMULU bool) string {
	if isMULU {
		return "MULU"
	}
	return "DIVU"
}

func init() {
	dataAlterable := []OperandType{
		Register, AddressRegisterIndirect, AddressRegisterIndirectPostInc,
		AddressRegisterIndirectPreDec, AddressRegisterIndirectDispl,
		AddressRegisterIndirectIndex, AbsoluteShort, AbsoluteLong, Label,
	}
	for _, ea := range dataAlterable {
		RegisterPatternEncoder(
			Pattern{
				Mnemonic:     "MULU",
				Size:         parser.SizeWord,
				OperandTypes: []OperandType{ea, Register},
			},
			muluDivuEncoder{isMULU: true},
		)
		RegisterPatternEncoder(
			Pattern{
				Mnemonic:     "DIVU",
				Size:         parser.SizeWord,
				OperandTypes: []OperandType{ea, Register},
			},
			muluDivuEncoder{isMULU: false},
		)
	}
}