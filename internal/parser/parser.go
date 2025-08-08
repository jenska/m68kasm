package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// --- Expression Evaluator (simple, supports +, -, *, /, parenthesis, numbers, $hex, labels) ---

type EvalContext struct {
	Symbols map[string]int // label table, or nil
	Verbose bool
}

// EvaluateExpr evaluates an (arithmetic) expression string, resolving labels from the context if present.
// Supports: +, -, *, /, (, ), numbers (decimal, $hex), and label identifiers.
func EvaluateExpr(expr string, ctx EvalContext) (int, error) {
	toks, err := tokenizeExpr(expr)
	if err != nil {
		return 0, err
	}
	val, err := parseExpr(toks, ctx)
	if ctx.Verbose {
		fmt.Printf("    [EXPR] Eval %q => %d (err=%v)\n", expr, val, err)
	}
	return val, err
}

type exprToken struct {
	typ  string // "num", "op", "paren", "ident"
	val  string
}

func tokenizeExpr(s string) ([]exprToken, error) {
	var toks []exprToken
	s = strings.TrimSpace(s)
	for i := 0; i < len(s); {
		switch {
		case s[i] == ' ' || s[i] == '\t':
			i++
		case s[i] == '(' || s[i] == ')':
			toks = append(toks, exprToken{"paren", string(s[i])})
			i++
		case s[i] == '+' || s[i] == '-' || s[i] == '*' || s[i] == '/':
			toks = append(toks, exprToken{"op", string(s[i])})
			i++
		case s[i] == '$':
			start := i
			i++
			for i < len(s) && isHexDigit(s[i]) {
				i++
			}
			toks = append(toks, exprToken{"num", s[start:i]})
		case isDigit(s[i]):
			start := i
			for i < len(s) && isDigit(s[i]) {
				i++
			}
			toks = append(toks, exprToken{"num", s[start:i]})
		case isIdentStart(s[i]):
			start := i
			for i < len(s) && isIdentPart(s[i]) {
				i++
			}
			toks = append(toks, exprToken{"ident", s[start:i]})
		default:
			return nil, fmt.Errorf("invalid char in expression: '%c'", s[i])
		}
	}
	return toks, nil
}

func isDigit(b byte) bool      { return b >= '0' && b <= '9' }
func isHexDigit(b byte) bool   { return (b >= '0' && b <= '9') || (b >= 'A' && b <= 'F') || (b >= 'a' && b <= 'f') }
func isIdentStart(b byte) bool { return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '_' || b == '.' }
func isIdentPart(b byte) bool  { return isIdentStart(b) || isDigit(b) }

// Recursive descent, operator precedence:  +,- lowest; *,/ higher; parenthesis.
func parseExpr(tokens []exprToken, ctx EvalContext) (int, error) {
	var idx int
	var expr func() (int, error)

	// Forward declaration to allow recursion
	var factor func() (int, error)
	var term func() (int, error)

	factor = func() (int, error) {
		if idx >= len(tokens) {
			return 0, fmt.Errorf("unexpected end of expression")
		}
		tok := tokens[idx]
		if tok.typ == "num" {
			idx++
			if strings.HasPrefix(tok.val, "$") {
				v, err := strconv.ParseInt(tok.val[1:], 16, 64)
				return int(v), err
			}
			v, err := strconv.Atoi(tok.val)
			return v, err
		}
		if tok.typ == "ident" {
			idx++
			if ctx.Symbols != nil {
				if v, ok := ctx.Symbols[tok.val]; ok {
					return v, nil
				}
			}
			return 0, fmt.Errorf("unknown label '%s'", tok.val)
		}
		if tok.typ == "op" && tok.val == "-" {
			idx++
			val, err := factor()
			return -val, err
		}
		if tok.typ == "paren" && tok.val == "(" {
			idx++
			val, err := expr()
			if err != nil {
				return 0, err
			}
			if idx >= len(tokens) || tokens[idx].typ != "paren" || tokens[idx].val != ")" {
				return 0, fmt.Errorf("missing closing parenthesis")
			}
			idx++
			return val, nil
		}
		return 0, fmt.Errorf("unexpected token '%s' in expression", tok.val)
	}
	term = func() (int, error) {
		val, err := factor()
		if err != nil {
			return 0, err
		}
		for idx < len(tokens) && tokens[idx].typ == "op" && (tokens[idx].val == "*" || tokens[idx].val == "/") {
			op := tokens[idx].val
			idx++
			nextVal, err := factor()
			if err != nil {
				return 0, err
			}
			if op == "*" {
				val *= nextVal
			} else {
				if nextVal == 0 {
					return 0, fmt.Errorf("division by zero")
				}
				val /= nextVal
			}
		}
		return val, nil
	}
	expr = func() (int, error) {
		val, err := term()
		if err != nil {
			return 0, err
		}
		for idx < len(tokens) && tokens[idx].typ == "op" && (tokens[idx].val == "+" || tokens[idx].val == "-") {
			op := tokens[idx].val
			idx++
			nextVal, err := term()
			if err != nil {
				return 0, err
			}
			if op == "+" {
				val += nextVal
			} else {
				val -= nextVal
			}
		}
		return val, nil
	}
	result, err := expr()
	if err != nil {
		return 0, err
	}
	if idx < len(tokens) {
		return 0, fmt.Errorf("unexpected token at end of expression: %s", tokens[idx].val)
	}
	return result, nil
}

// --- END Expression Evaluator ---

type OperandType int

const (
	UnknownOperand OperandType = iota
	Register
	Immediate
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
	RegisterList
	Directive
)

type OperandInfo struct {
	Type  OperandType
	Value string
}

type Instruction struct {
	Address   int
	Mnemonic  string
	Size      OperandSize
	Operands  []OperandInfo
	Label     string // Label at this line (if any, else empty)
	RawLine   string // For error messages/debugging
	LineIndex int    // For error reporting
}

type OperandSize int

const (
	SizeUnknown OperandSize = iota
	SizeByte
	SizeWord
	SizeLong
	SizeShort
)

var regListRe = regexp.MustCompile(`^([Dd][0-7](-[Dd][0-7])?(/[Dd][0-7](-[Dd][0-7])?)*)?(/[Aa][0-7](-[Aa][0-7])?(/[Aa][0-7](-[Aa][0-7])?)*)*$|^[Dd][0-7](-[Dd][0-7])?(/[Aa][0-7](-[Aa][0-7])?)*$|^[Aa][0-7](-[Aa][0-7])?(/[Dd][0-7](-[Dd][0-7])?)*$`)

func IsRegisterList(s string) bool {
	s = strings.ReplaceAll(s, " ", "")
	if len(s) == 0 {
		return false
	}
	if regListRe.MatchString(s) {
		return true
	}
	parts := strings.Split(s, "/")
	for _, part := range parts {
		part = strings.ToUpper(part)
		if strings.HasPrefix(part, "D") || strings.HasPrefix(part, "A") {
			continue
		}
		return false
	}
	return true
}

func ParseOperand(s string) OperandInfo {
	s = strings.TrimSpace(s)
	if s == "" {
		return OperandInfo{Type: UnknownOperand, Value: ""}
	}
	// Immediate
	if strings.HasPrefix(s, "#") {
		return OperandInfo{Type: Immediate, Value: s}
	}
	// RegisterList
	if IsRegisterList(s) {
		return OperandInfo{Type: RegisterList, Value: s}
	}
	// Directive
	if isDirectiveOperand(s) {
		return OperandInfo{Type: Directive, Value: s}
	}
	// Register direct
	if reg, ok := parseRegister(s); ok {
		return OperandInfo{Type: Register, Value: reg}
	}
	// Addressing
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		inner := s[1 : len(s)-1]
		if strings.HasPrefix(inner, "-") && strings.HasSuffix(inner, "A") {
			return OperandInfo{Type: AddressRegisterIndirectPreDec, Value: s}
		}
		if strings.HasSuffix(s, ")+") {
			return OperandInfo{Type: AddressRegisterIndirectPostInc, Value: s}
		}
		if strings.Contains(inner, ",") && strings.Contains(inner, "X") {
			return OperandInfo{Type: AddressRegisterIndirectIndex, Value: s}
		}
		if strings.Contains(inner, ",") {
			return OperandInfo{Type: AddressRegisterIndirectDispl, Value: s}
		}
		return OperandInfo{Type: AddressRegisterIndirect, Value: s}
	}
	if strings.HasPrefix(s, "-(") && strings.HasSuffix(s, ")") {
		return OperandInfo{Type: AddressRegisterIndirectPreDec, Value: s}
	}
	if strings.HasSuffix(s, ")+") && strings.HasPrefix(s, "(") {
		return OperandInfo{Type: AddressRegisterIndirectPostInc, Value: s}
	}
	if strings.HasPrefix(s, "$") {
		if len(s) > 2 {
			return OperandInfo{Type: AbsoluteLong, Value: s}
		}
		return OperandInfo{Type: AbsoluteShort, Value: s}
	}
	if isLabel(s) {
		return OperandInfo{Type: Label, Value: s}
	}
	return OperandInfo{Type: UnknownOperand, Value: s}
}

func isDirectiveOperand(s string) bool {
	s = strings.ToUpper(s)
	if s == ".EVEN" || s == "EVEN" {
		return true
	}
	if s == ".ALIGN" || s == "ALIGN" {
		return true
	}
	if s == ".ORG" || s == "ORG" {
		return true
	}
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return true // string literal for dc.b
	}
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		return true // char literal for dc.b
	}
	// Accept expressions as directive operands (defer actual parsing to codegen)
	return true
}

func parseRegister(s string) (string, bool) {
	s = strings.ToUpper(s)
	if (strings.HasPrefix(s, "D") || strings.HasPrefix(s, "A")) && len(s) == 2 {
		n, err := strconv.Atoi(s[1:])
		if err == nil && n >= 0 && n <= 7 {
			return s, true
		}
	}
	return "", false
}

func isLabel(s string) bool {
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "#") || strings.HasPrefix(s, "(") || strings.HasPrefix(s, "D") || strings.HasPrefix(s, "A") || strings.HasPrefix(s, "$") {
		return false
	}
	return true
}

// --- Pseudo-directive operand counts (for parser-level arity check) ---

type DirectiveArity struct {
	MinOperands int
	MaxOperands int // -1 for unlimited
}

var PseudoDirectiveArity = map[string]DirectiveArity{
	"ORG":    {1, 1},
	".ORG":   {1, 1},
	"ALIGN":  {1, 1},
	".ALIGN": {1, 1},
	"EVEN":   {0, 0},
	".EVEN":  {0, 0},
	"DC.B":   {1, -1},
	".DC.B":  {1, -1},
	"DC.W":   {1, -1},
	".DC.W":  {1, -1},
	"DC.L":   {1, -1},
	".DC.L":  {1, -1},
	"DS.B":   {1, 1},
	".DS.B":  {1, 1},
	"DS.W":   {1, 1},
	".DS.W":  {1, 1},
	"DS.L":   {1, 1},
	".DS.L":  {1, 1},
}

// ParseInstruction parses a single assembly line (with possible label at start)
// and checks operand count for pseudo-directives
func ParseInstruction(addr int, s string, lineno int, verbose bool) (Instruction, error) {
	// Remove comments
	s = strings.SplitN(s, ";", 2)[0]
	s = strings.TrimSpace(s)
	if s == "" {
		return Instruction{}, fmt.Errorf("empty instruction")
	}

	// Check for label at start: must be at beginning, ends with ':'
	label := ""
	rest := s
	colonIdx := strings.Index(rest, ":")
	if colonIdx >= 0 {
		possibleLabel := strings.TrimSpace(rest[:colonIdx])
		if isValidLabelName(possibleLabel) {
			label = possibleLabel
			rest = strings.TrimSpace(rest[colonIdx+1:])
		}
	}

	if rest == "" {
		// Label-only line
		return Instruction{
			Address:   addr,
			Mnemonic:  "",
			Operands:  nil,
			Label:     label,
			RawLine:   s,
			LineIndex: lineno,
		}, nil
	}

	// Now parse mnemonic and operands as before
	parts := strings.Fields(rest)
	if len(parts) == 0 {
		// Only a label on this line
		return Instruction{
			Address:   addr,
			Mnemonic:  "",
			Operands:  nil,
			Label:     label,
			RawLine:   s,
			LineIndex: lineno,
		}, nil
	}
	mnemonic := strings.ToUpper(parts[0])
	restOps := strings.TrimSpace(rest[len(parts[0]):])
	ops := splitOperandsQuoteAware(restOps)
	var infos []OperandInfo
	for _, op := range ops {
		if op != "" {
			infos = append(infos, ParseOperand(op))
		}
	}
	// --- Pseudo-directive operand count check ---
	if arity, ok := PseudoDirectiveArity[mnemonic]; ok {
		numOps := len(infos)
		if numOps < arity.MinOperands || (arity.MaxOperands != -1 && numOps > arity.MaxOperands) {
			if arity.MinOperands == arity.MaxOperands {
				return Instruction{}, fmt.Errorf("%s: expected %d operand(s), got %d", mnemonic, arity.MinOperands, numOps)
			} else if arity.MaxOperands == -1 {
				return Instruction{}, fmt.Errorf("%s: expected at least %d operand(s), got %d", mnemonic, arity.MinOperands, numOps)
			} else {
				return Instruction{}, fmt.Errorf("%s: expected between %d and %d operand(s), got %d", mnemonic, arity.MinOperands, arity.MaxOperands, numOps)
			}
		}
	}
	if verbose {
		fmt.Printf("[PARSE] 0x%04X: label=%q, mnemonic=%s, operands=%v\n", addr, label, mnemonic, infos)
	}
	return Instruction{
		Address:   addr,
		Mnemonic:  mnemonic,
		Size:      SizeUnknown,
		Operands:  infos,
		Label:     label,
		RawLine:   s,
		LineIndex: lineno,
	}, nil
}

func isValidLabelName(s string) bool {
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "#") || strings.HasPrefix(s, "(") || strings.HasPrefix(s, "D") || strings.HasPrefix(s, "A") || strings.HasPrefix(s, "$") {
		return false
	}
	for _, c := range s {
		if !(c == '_' || c == '.' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// splitOperandsQuoteAware splits operands at commas, but not inside (), "" or ''
func splitOperandsQuoteAware(rest string) []string {
	var ops []string
	var cur strings.Builder
	level := 0
	inDoubleQuotes := false
	inSingleQuotes := false
	for i := 0; i < len(rest); i++ {
		c := rest[i]
		switch c {
		case '"':
			cur.WriteByte(c)
			inDoubleQuotes = !inDoubleQuotes
		case '\'':
			cur.WriteByte(c)
			inSingleQuotes = !inSingleQuotes
		case '(':
			if !inDoubleQuotes && !inSingleQuotes {
				level++
			}
			cur.WriteByte(c)
		case ')':
			if !inDoubleQuotes && !inSingleQuotes {
				level--
			}
			cur.WriteByte(c)
		case ',':
			if level == 0 && !inDoubleQuotes && !inSingleQuotes {
				ops = append(ops, strings.TrimSpace(cur.String()))
				cur.Reset()
			} else {
				cur.WriteByte(c)
			}
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 {
		ops = append(ops, strings.TrimSpace(cur.String()))
	}
	return ops
}