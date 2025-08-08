package codegen

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/jenska/m68kasm/internal/parser"
)

// MOVEM encoder for both memory->register(s) and register(s)->memory forms
type movemEncoder struct{}

func (e movemEncoder) Encode(instr parser.Instruction, ops []OperandInfo, symbols map[string]int) ([]byte, error) {
	if len(ops) != 2 {
		return nil, fmt.Errorf("MOVEM at 0x%04X: requires 2 operands, got %d", instr.Address, len(ops))
	}

	src, dst := ops[0], ops[1]
	size := instr.Size

	if size != parser.SizeWord && size != parser.SizeLong {
		return nil, fmt.Errorf("MOVEM at 0x%04X: invalid size; must be .W or .L", instr.Address)
	}

	// Determine direction and which operand is the register list
	var (
		isRegToMem   bool
		reglistOp    OperandInfo
		memOp        OperandInfo
	)
	if src.Type == RegisterList {
		isRegToMem = true
		reglistOp = src
		memOp = dst
	} else if dst.Type == RegisterList {
		isRegToMem = false
		reglistOp = dst
		memOp = src
	} else {
		return nil, fmt.Errorf("MOVEM at 0x%04X: one operand must be a register list", instr.Address)
	}

	// Validate EA - must be memory (not immediate, not register direct)
	switch memOp.Type {
	case AddressRegisterIndirect, AddressRegisterIndirectPostInc, AddressRegisterIndirectPreDec,
		AddressRegisterIndirectDispl, AddressRegisterIndirectIndex,
		AbsoluteShort, AbsoluteLong, Label:
		// ok
	default:
		return nil, fmt.Errorf("MOVEM at 0x%04X: memory operand must be memory EA, got %s (type %v)", instr.Address, memOp.Value, memOp.Type)
	}

	mode, reg, ext, err := encodeEffectiveAddress(memOp, symbols, true)
	if err != nil {
		return nil, fmt.Errorf("MOVEM at 0x%04X: invalid memory operand '%s': %w", instr.Address, memOp.Value, err)
	}

	// Register mask: 16 bits, D0-D7 (0-7), A0-A7 (8-15)
	mask, err := parseMovemRegisterList(reglistOp.Value)
	if err != nil {
		return nil, fmt.Errorf("MOVEM at 0x%04X: invalid register list '%s': %w", instr.Address, reglistOp.Value, err)
	}

	// Special case: for register-to-memory with pre-decrement, mask bits are reversed!
	if isRegToMem && memOp.Type == AddressRegisterIndirectPreDec {
		mask = reverseMovemMask(mask)
	}

	var opc uint16
	if isRegToMem {
		// Register(s) -> memory: 0100 1001 1sz0 mmm rrr (sz = 1 for long, 0 for word)
		opc = 0x4880
		if size == parser.SizeLong {
			opc |= 0x0040
		}
		opc |= (mode << 3) | reg
	} else {
		// Memory -> register(s): 0100 1000 1sz0 mmm rrr
		opc = 0x4C80
		if size == parser.SizeLong {
			opc |= 0x0040
		}
		opc |= (mode << 3) | reg
	}

	// Compose output (opcode, extension word, EA extension if any)
	code := make([]byte, 4+2*len(ext))
	binary.BigEndian.PutUint16(code[0:2], opc)
	binary.BigEndian.PutUint16(code[2:4], mask)
	for i, w := range ext {
		binary.BigEndian.PutUint16(code[4+2*i:6+2*i], w)
	}
	return code, nil
}

// MOVEM register list parsing: "D0-D2/A6/A7" etc.
func parseMovemRegisterList(regStr string) (uint16, error) {
	regStr = strings.ReplaceAll(regStr, " ", "")
	if regStr == "" {
		return 0, fmt.Errorf("register list is empty")
	}
	var mask uint16
	parts := strings.Split(regStr, "/")
	for _, part := range parts {
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			if len(bounds) != 2 {
				return 0, fmt.Errorf("invalid register range: %s", part)
			}
			startIdx, err := movemRegIndex(bounds[0])
			if err != nil {
				return 0, err
			}
			endIdx, err := movemRegIndex(bounds[1])
			if err != nil {
				return 0, err
			}
			if startIdx > endIdx {
				return 0, fmt.Errorf("invalid register range: %s", part)
			}
			for i := startIdx; i <= endIdx; i++ {
				mask |= 1 << i
			}
		} else {
			idx, err := movemRegIndex(part)
			if err != nil {
				return 0, err
			}
			mask |= 1 << idx
		}
	}
	return mask, nil
}

// D0-D7 = 0-7, A0-A7 = 8-15
func movemRegIndex(reg string) (int, error) {
	reg = strings.ToUpper(reg)
	if strings.HasPrefix(reg, "D") {
		n, err := strconv.Atoi(reg[1:])
		if err != nil || n < 0 || n > 7 {
			return 0, fmt.Errorf("invalid data register: %s", reg)
		}
		return n, nil
	} else if strings.HasPrefix(reg, "A") {
		n, err := strconv.Atoi(reg[1:])
		if err != nil || n < 0 || n > 7 {
			return 0, fmt.Errorf("invalid address register: %s", reg)
		}
		return n + 8, nil
	}
	return 0, fmt.Errorf("invalid register in list: %s", reg)
}

// For MOVEM to memory with predecrement, bits are reversed
func reverseMovemMask(mask uint16) uint16 {
	var rev uint16
	for i := 0; i < 16; i++ {
		if (mask & (1 << uint(i))) != 0 {
			rev |= 1 << uint(15-i)
		}
	}
	return rev
}

func init() {
	// Register both forms: reglist,ea (store), ea,reglist (load)
	memEAtypes := []OperandType{
		AddressRegisterIndirect, AddressRegisterIndirectPostInc, AddressRegisterIndirectPreDec,
		AddressRegisterIndirectDispl, AddressRegisterIndirectIndex,
		AbsoluteShort, AbsoluteLong, Label,
	}
	for _, sz := range []parser.OperandSize{parser.SizeWord, parser.SizeLong} {
		for _, ea := range memEAtypes {
			// reglist,ea
			RegisterPatternEncoder(
				Pattern{
					Mnemonic:     "MOVEM",
					Size:         sz,
					OperandTypes: []OperandType{RegisterList, ea},
				},
				movemEncoder{},
			)
			// ea,reglist
			RegisterPatternEncoder(
				Pattern{
					Mnemonic:     "MOVEM",
					Size:         sz,
					OperandTypes: []OperandType{ea, RegisterList},
				},
				movemEncoder{},
			)
		}
	}
}