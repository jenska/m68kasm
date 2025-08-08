package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jenska/m68kasm/internal/parser"
)

// --- Directive metadata ---

type DirectiveHandler func(instr parser.Instruction, currentAddress *int, output *[]byte, symtab map[string]int, verbose bool) error

type DirectiveInfo struct {
	Handler     DirectiveHandler
	MinOperands int // minimum required operands
	MaxOperands int // maximum allowed operands, -1 means unlimited
}

var PseudoDirectiveTable = map[string]DirectiveInfo{
	"ORG":    {handleORG, 1, 1},
	".ORG":   {handleORG, 1, 1},
	"ALIGN":  {handleALIGN, 1, 1},
	".ALIGN": {handleALIGN, 1, 1},
	"EVEN":   {handleEVEN, 0, 0},
	".EVEN":  {handleEVEN, 0, 0},
	"DC.B":   {handleDCB, 1, -1},
	".DC.B":  {handleDCB, 1, -1},
	"DC.W":   {genericDCHandler(2), 1, -1},
	".DC.W":  {genericDCHandler(2), 1, -1},
	"DC.L":   {genericDCHandler(4), 1, -1},
	".DC.L":  {genericDCHandler(4), 1, -1},
	"DS.B":   {genericDSHandler(1), 1, 1},
	".DS.B":  {genericDSHandler(1), 1, 1},
	"DS.W":   {genericDSHandler(2), 1, 1},
	".DS.W":  {genericDSHandler(2), 1, 1},
	"DS.L":   {genericDSHandler(4), 1, 1},
	".DS.L":  {genericDSHandler(4), 1, 1},
}

// --- Central dispatcher with operand size check ---

type PseudoDirectiveHandler struct {
	Verbose bool
}

func (PseudoDirectiveHandler) Handle(instr parser.Instruction, currentAddress *int, output *[]byte, symtab map[string]int, verbose bool) error {
	key := strings.ToUpper(instr.Mnemonic)
	info, ok := PseudoDirectiveTable[key]
	if !ok {
		return fmt.Errorf("unsupported pseudo-op: %s", instr.Mnemonic)
	}
	numOperands := len(instr.Operands)
	if numOperands < info.MinOperands || (info.MaxOperands != -1 && numOperands > info.MaxOperands) {
		if info.MinOperands == info.MaxOperands {
			return fmt.Errorf("%s: expected %d operand(s), got %d", key, info.MinOperands, numOperands)
		} else if info.MaxOperands == -1 {
			return fmt.Errorf("%s: expected at least %d operand(s), got %d", key, info.MinOperands, numOperands)
		} else {
			return fmt.Errorf("%s: expected between %d and %d operand(s), got %d", key, info.MinOperands, info.MaxOperands, numOperands)
		}
	}
	return info.Handler(instr, currentAddress, output, symtab, verbose)
}

// --- Generic DC/DS handlers ---

func handleDCB(instr parser.Instruction, currentAddress *int, output *[]byte, symtab map[string]int, verbose bool) error {
	for _, op := range instr.Operands {
		valStr := strings.TrimSpace(op.Value)
		if isDoubleQuotedString(valStr) {
			bs, err := decodeGoStringLiteral(valStr)
			if err != nil {
				return fmt.Errorf("%s: invalid string literal %q: %w", instr.Mnemonic, valStr, err)
			}
			*output = append(*output, bs...)
			*currentAddress += len(bs)
			if verbose {
				fmt.Printf("    [DC.B] Emitted bytes for string %q: %v\n", valStr, bs)
			}
		} else if isSingleQuotedChar(valStr) {
			val, err := parseCharLiteral(valStr)
			if err != nil {
				return fmt.Errorf("%s: invalid char literal %q: %w", instr.Mnemonic, valStr, err)
			}
			*output = append(*output, byte(val))
			*currentAddress++
			if verbose {
				fmt.Printf("    [DC.B] Emitted byte for char %q: %d\n", valStr, val)
			}
		} else {
			val, err := parser.EvaluateExpr(valStr, parser.EvalContext{Symbols: symtab})
			if err != nil {
				return fmt.Errorf("%s: invalid operand %q: %w", instr.Mnemonic, valStr, err)
			}
			*output = append(*output, byte(val))
			*currentAddress++
			if verbose {
				fmt.Printf("    [DC.B] Emitted byte for expr %q: %d\n", valStr, val)
			}
		}
	}
	return nil
}

func genericDCHandler(size int) DirectiveHandler {
	return func(instr parser.Instruction, currentAddress *int, output *[]byte, symtab map[string]int, verbose bool) error {
		for _, op := range instr.Operands {
			valStr := strings.TrimSpace(op.Value)
			val, err := parser.EvaluateExpr(valStr, parser.EvalContext{Symbols: symtab})
			if err != nil {
				return fmt.Errorf("%s: invalid operand %q: %w", instr.Mnemonic, valStr, err)
			}
			switch size {
			case 2:
				*output = append(*output, byte(val>>8), byte(val))
				if verbose {
					fmt.Printf("    [DC.W] Emitted word for expr %q: %04X\n", valStr, val&0xFFFF)
				}
			case 4:
				*output = append(*output, byte(val>>24), byte(val>>16), byte(val>>8), byte(val))
				if verbose {
					fmt.Printf("    [DC.L] Emitted long for expr %q: %08X\n", valStr, uint32(val))
				}
			default:
				return fmt.Errorf("unsupported DC size %d", size)
			}
			*currentAddress += size
		}
		return nil
	}
}

func genericDSHandler(size int) DirectiveHandler {
	return func(instr parser.Instruction, currentAddress *int, output *[]byte, symtab map[string]int, verbose bool) error {
		if len(instr.Operands) == 0 {
			return fmt.Errorf("%s requires operand", instr.Mnemonic)
		}
		countStr := instr.Operands[0].Value
		count, err := parser.EvaluateExpr(countStr, parser.EvalContext{Symbols: symtab})
		if err != nil {
			return fmt.Errorf("%s: invalid count %q: %w", instr.Mnemonic, countStr, err)
		}
		total := count * size
		for i := 0; i < total; i++ {
			*output = append(*output, 0x00)
		}
		*currentAddress += total
		if verbose {
			fmt.Printf("    [DS.%c] Reserved %d bytes\n", "BWL"[size/2], total)
		}
		return nil
	}
}

// --- Standard Handlers: ORG, ALIGN, EVEN ---

func handleORG(instr parser.Instruction, currentAddress *int, output *[]byte, symtab map[string]int, verbose bool) error {
	if len(instr.Operands) == 0 {
		return fmt.Errorf("ORG requires operand")
	}
	addrStr := instr.Operands[0].Value
	addr, err := parser.EvaluateExpr(addrStr, parser.EvalContext{Symbols: symtab})
	if err != nil {
		return fmt.Errorf("ORG: invalid address '%s': %w", addrStr, err)
	}
	if addr < *currentAddress {
		return fmt.Errorf("ORG: address 0x%X < current position 0x%X", addr, *currentAddress)
	}
	gap := addr - *currentAddress
	for i := 0; i < gap; i++ {
		*output = append(*output, 0x00)
	}
	if verbose && gap > 0 {
		fmt.Printf("    [ORG] Pad %d bytes to 0x%X\n", gap, addr)
	}
	*currentAddress = addr
	return nil
}

func handleALIGN(instr parser.Instruction, currentAddress *int, output *[]byte, symtab map[string]int, verbose bool) error {
	if len(instr.Operands) == 0 {
		return fmt.Errorf("ALIGN requires operand")
	}
	nStr := instr.Operands[0].Value
	n, err := parser.EvaluateExpr(nStr, parser.EvalContext{Symbols: symtab})
	if err != nil {
		return fmt.Errorf("ALIGN: invalid argument '%s': %w", nStr, err)
	}
	if n <= 0 || (n&(n-1)) != 0 {
		return fmt.Errorf("ALIGN: argument must be power of 2, got %d", n)
	}
	misalignment := *currentAddress % n
	if misalignment != 0 {
		padding := n - misalignment
		for i := 0; i < padding; i++ {
			*output = append(*output, 0x00)
		}
		*currentAddress += padding
		if verbose {
			fmt.Printf("    [ALIGN] Pad %d bytes to alignment %d\n", padding, n)
		}
	}
	return nil
}

func handleEVEN(instr parser.Instruction, currentAddress *int, output *[]byte, symtab map[string]int, verbose bool) error {
	if *currentAddress%2 != 0 {
		*output = append(*output, 0x00)
		*currentAddress++
		if verbose {
			fmt.Printf("    [EVEN] Pad 1 byte for even alignment\n")
		}
	}
	return nil
}

// --- Utilities ---

func isDoubleQuotedString(s string) bool {
	return len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"'
}
func isSingleQuotedChar(s string) bool {
	return len(s) >= 3 && s[0] == '\'' && s[len(s)-1] == '\''
}
func parseCharLiteral(s string) (int, error) {
	s = strings.TrimSpace(s)
	if len(s) >= 3 && s[0] == '\'' && s[len(s)-1] == '\'' {
		runes := []rune(s[1 : len(s)-1])
		if len(runes) == 1 {
			return int(runes[0]), nil
		}
	}
	return 0, fmt.Errorf("not a single-quoted char: %q", s)
}
func decodeGoStringLiteral(lit string) ([]byte, error) {
	if len(lit) < 2 || lit[0] != '"' || lit[len(lit)-1] != '"' {
		return nil, fmt.Errorf("not a double-quoted string")
	}
	return strconv.UnquoteBytes(lit)
}

// For dry-run address calculation in first pass
func (PseudoDirectiveHandler) DryRun(instr parser.Instruction, currentAddress *int, symtab map[string]int) error {
	dummyVerbose := false
	key := strings.ToUpper(instr.Mnemonic)
	info, ok := PseudoDirectiveTable[key]
	if !ok {
		return nil
	}
	numOperands := len(instr.Operands)
	if numOperands < info.MinOperands || (info.MaxOperands != -1 && numOperands > info.MaxOperands) {
		return nil // silently skip address update on operand count mismatch in dry run
	}
	return info.Handler(instr, currentAddress, nil, symtab, dummyVerbose)
}