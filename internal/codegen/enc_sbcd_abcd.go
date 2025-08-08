package codegen

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/jenska/m68kasm/internal/parser"
)

// SBCD/ABCD instruction encoder for both "Dn,Dn" and "-(An),-(An)" forms.
type sbcdAbcdEncoder struct {
	isABCD bool
}

func (e sbcdAbcdEncoder) Encode(instr parser.Instruction, ops []OperandInfo, symbols map[string]int) ([]byte, error) {
	if len(ops) != 2 {
		return nil, fmt.Errorf("%s at 0x%04X: requires 2 operands, got %d", instrName(e.isABCD), instr.Address, len(ops))
	}
	src, dst := ops[0], ops[1]

	var modeBit uint16 // 0=data reg, 1=address reg predecrement
	var srcn, dstn byte

	if src.Type == Register && dst.Type == Register &&
		strings.HasPrefix(src.Value, "D") && strings.HasPrefix(dst.Value, "D") {
		modeBit = 0
		srcn = src.Value[1] - '0'
		dstn = dst.Value[1] - '0'
	} else if src.Type == AddressRegisterIndirectPreDec && dst.Type == AddressRegisterIndirectPreDec {
		modeBit = 1
		srcn = src.Value[3] - '0'
		dstn = dst.Value[3] - '0'
	} else {
		return nil, fmt.Errorf("%s at 0x%04X: only supports Dn,Dn or -(An),-(An); got %s,%s", instrName(e.isABCD), instr.Address, src.Value, dst.Value)
	}

	var opc uint16
	if e.isABCD {
		opc = 0xC100 // ABCD
	} else {
		opc = 0x8100 // SBCD
	}
	opc |= (modeBit << 3)
	opc |= uint16(dstn) << 9
	opc |= uint16(srcn)
	code := make([]byte, 2)
	binary.BigEndian.PutUint16(code, opc)
	return code, nil
}

func instrName(isABCD bool) string {
	if isABCD {
		return "ABCD"
	}
	return "SBCD"
}

func init() {
	forms := []struct {
		src, dst OperandType
	}{
		{Register, Register},
		{AddressRegisterIndirectPreDec, AddressRegisterIndirectPreDec},
	}
	for _, form := range forms {
		RegisterPatternEncoder(
			Pattern{
				Mnemonic:     "SBCD",
				Size:         parser.SizeUnknown,
				OperandTypes: []OperandType{form.src, form.dst},
			},
			sbcdAbcdEncoder{isABCD: false},
		)
		RegisterPatternEncoder(
			Pattern{
				Mnemonic:     "ABCD",
				Size:         parser.SizeUnknown,
				OperandTypes: []OperandType{form.src, form.dst},
			},
			sbcdAbcdEncoder{isABCD: true},
		)
	}
}