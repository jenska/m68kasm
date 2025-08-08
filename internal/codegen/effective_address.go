package codegen

import (
	"fmt"
	"strings"

	"github.com/jenska/m68kasm/internal/parser"
)

// encodeEffectiveAddress encodes the effective address field for a 68k instruction.
// It returns the mode and register bits (ea), any extension words (as a byte slice), and an error if unsupported.
// ins: the full instruction, for error reporting and context
// op: the operand to encode
// Returns: eaBits (3-bit mode << 3 | 3-bit reg), extension words ([]byte), error
func encodeEffectiveAddress(ins parser.Instruction, op parser.OperandInfo, symtab map[string]int, verbose bool) (uint8, []byte, error) {
	opstr := strings.TrimSpace(op.Value)
	switch op.Type {
	case parser.Register:
		// Data register direct: Dn
		if strings.HasPrefix(strings.ToUpper(opstr), "D") {
			regNum := opstr[1] - '0'
			if regNum < 0 || regNum > 7 {
				return 0, nil, fmt.Errorf("invalid data register %q", opstr)
			}
			ea := 0x00 | (regNum & 0x7) // mode=000, reg=Dn
			if verbose {
				fmt.Printf("    [EA] Dn: %s -> ea=0x%02X\n", opstr, ea)
			}
			return ea, nil, nil
		}
		// Address register direct: An
		if strings.HasPrefix(strings.ToUpper(opstr), "A") {
			regNum := opstr[1] - '0'
			if regNum < 0 || regNum > 7 {
				return 0, nil, fmt.Errorf("invalid address register %q", opstr)
			}
			ea := 0x08 | (regNum & 0x7) // mode=001, reg=An
			if verbose {
				fmt.Printf("    [EA] An: %s -> ea=0x%02X\n", opstr, ea)
			}
			return ea, nil, nil
		}
	case parser.AddressRegisterIndirect:
		// (An)
		regNum := opstr[2] - '0'
		if regNum < 0 || regNum > 7 {
			return 0, nil, fmt.Errorf("invalid address register indirect %q", opstr)
		}
		ea := 0x10 | (regNum & 0x7) // mode=010, reg=An
		if verbose {
			fmt.Printf("    [EA] (An): %s -> ea=0x%02X\n", opstr, ea)
		}
		return ea, nil, nil
	case parser.AddressRegisterIndirectPostInc:
		// (An)+
		regNum := opstr[2] - '0'
		if regNum < 0 || regNum > 7 {
			return 0, nil, fmt.Errorf("invalid address register indirect postinc %q", opstr)
		}
		ea := 0x18 | (regNum & 0x7) // mode=011, reg=An
		if verbose {
			fmt.Printf("    [EA] (An)+: %s -> ea=0x%02X\n", opstr, ea)
		}
		return ea, nil, nil
	case parser.AddressRegisterIndirectPreDec:
		// -(An)
		regNum := opstr[3] - '0'
		if regNum < 0 || regNum > 7 {
			return 0, nil, fmt.Errorf("invalid address register indirect predec %q", opstr)
		}
		ea := 0x20 | (regNum & 0x7) // mode=100, reg=An
		if verbose {
			fmt.Printf("    [EA] -(An): %s -> ea=0x%02X\n", opstr, ea)
		}
		return ea, nil, nil
	case parser.AddressRegisterIndirectDispl:
		// (d16,An)
		i := strings.Index(opstr, ",A")
		if i < 0 || len(opstr) < i+3 {
			return 0, nil, fmt.Errorf("invalid address register indirect w/ displacement %q", opstr)
		}
		regNum := opstr[i+2] - '0'
		if regNum < 0 || regNum > 7 {
			return 0, nil, fmt.Errorf("invalid An in (d16,An): %q", opstr)
		}
		dispStr := opstr[1:i]
		disp, err := parser.EvaluateExpr(dispStr, parser.EvalContext{Symbols: symtab})
		if err != nil {
			return 0, nil, fmt.Errorf("invalid displacement in %q: %v", opstr, err)
		}
		ea := 0x28 | (regNum & 0x7) // mode=101, reg=An
		ext := []byte{byte(disp >> 8), byte(disp)}
		if verbose {
			fmt.Printf("    [EA] (d16,An): %s -> ea=0x%02X, ext=% X\n", opstr, ea, ext)
		}
		return ea, ext, nil
	case parser.AddressRegisterIndirectIndex:
		// (d8,An,Xn)
		// e.g. (4,A0,D1.L)
		paren := strings.Trim(opstr, "()")
		parts := strings.Split(paren, ",")
		if len(parts) != 3 {
			return 0, nil, fmt.Errorf("invalid address register indirect w/ index %q", opstr)
		}
		regPart := strings.ToUpper(parts[1])
		if !strings.HasPrefix(regPart, "A") {
			return 0, nil, fmt.Errorf("expected An in %q", opstr)
		}
		regNum := regPart[1] - '0'
		if regNum < 0 || regNum > 7 {
			return 0, nil, fmt.Errorf("invalid An in (d8,An,Xn): %q", opstr)
		}
		ea := 0x30 | (regNum & 0x7) // mode=110, reg=An
		disp, err := parser.EvaluateExpr(parts[0], parser.EvalContext{Symbols: symtab})
		if err != nil {
			return 0, nil, fmt.Errorf("invalid d8 in %q: %v", opstr, err)
		}
		// Index reg/mode/size
		xn := strings.ToUpper(parts[2])
		var idxReg uint8
		var idxSize uint8
		if strings.HasPrefix(xn, "D") {
			idxReg = 0
		} else if strings.HasPrefix(xn, "A") {
			idxReg = 1
		} else {
			return 0, nil, fmt.Errorf("invalid index register %q", xn)
		}
		idxNum := xn[1] - '0'
		if idxNum < 0 || idxNum > 7 {
			return 0, nil, fmt.Errorf("bad index register %q", xn)
		}
		idxSize = 0 // default: .W
		if strings.HasSuffix(xn, ".L") {
			idxSize = 1
		}
		ext := uint16(idxReg<<15 | uint16(idxNum)<<12 | idxSize<<11 | (disp & 0xFF))
		extBytes := []byte{byte(ext >> 8), byte(ext)}
		if verbose {
			fmt.Printf("    [EA] (d8,An,Xn): %s -> ea=0x%02X, ext=% X\n", opstr, ea, extBytes)
		}
		return ea, extBytes, nil
	case parser.AbsoluteShort:
		// $xxxx
		addrVal, err := parser.EvaluateExpr(opstr, parser.EvalContext{Symbols: symtab})
		if err != nil {
			return 0, nil, fmt.Errorf("invalid absolute short address %q: %v", opstr, err)
		}
		ea := 0x38 // mode=111, reg=000
		ext := []byte{byte(addrVal >> 8), byte(addrVal)}
		if verbose {
			fmt.Printf("    [EA] $xxxx: %s -> ea=0x%02X, ext=% X\n", opstr, ea, ext)
		}
		return ea, ext, nil
	case parser.AbsoluteLong:
		// $xxxxxxxx
		addrVal, err := parser.EvaluateExpr(opstr, parser.EvalContext{Symbols: symtab})
		if err != nil {
			return 0, nil, fmt.Errorf("invalid absolute long address %q: %v", opstr, err)
		}
		ea := 0x39 // mode=111, reg=001
		ext := []byte{byte(addrVal >> 24), byte(addrVal >> 16), byte(addrVal >> 8), byte(addrVal)}
		if verbose {
			fmt.Printf("    [EA] $xxxxxxxx: %s -> ea=0x%02X, ext=% X\n", opstr, ea, ext)
		}
		return ea, ext, nil
	case parser.Immediate:
		// #imm
		immStr := strings.TrimPrefix(opstr, "#")
		immVal, err := parser.EvaluateExpr(immStr, parser.EvalContext{Symbols: symtab})
		if err != nil {
			return 0, nil, fmt.Errorf("invalid immediate %q: %v", opstr, err)
		}
		ea := 0x3C // mode=111, reg=100
		ext := []byte{byte(immVal >> 8), byte(immVal)}
		if verbose {
			fmt.Printf("    [EA] #imm: %s -> ea=0x%02X, ext=% X\n", opstr, ea, ext)
		}
		return ea, ext, nil
	case parser.Label:
		// Use as absolute short (can be improved if label is 32 bits)
		addrVal, err := parser.EvaluateExpr(opstr, parser.EvalContext{Symbols: symtab})
		if err != nil {
			return 0, nil, fmt.Errorf("invalid label/address %q: %v", opstr, err)
		}
		ea := 0x38 // mode=111, reg=000 (absolute short)
		ext := []byte{byte(addrVal >> 8), byte(addrVal)}
		if verbose {
			fmt.Printf("    [EA] label: %s -> ea=0x%02X, ext=% X\n", opstr, ea, ext)
		}
		return ea, ext, nil
	default:
		return 0, nil, fmt.Errorf("unsupported addressing mode for operand %q (type=%v)", opstr, op.Type)
	}
}