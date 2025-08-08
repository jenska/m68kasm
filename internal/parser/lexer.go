package parser

import (
	"strings"
	"unicode"
)

// TokenKind describes types of tokens emitted by the lexer
type TokenKind int

const (
	TokenEOF TokenKind = iota
	TokenMnemonic
	TokenSize
	TokenOperand
	TokenComma
	TokenUnknown
)

// Token represents a single token
type Token struct {
	Kind  TokenKind
	Value string
}

// Lexer splits an instruction string into tokens.
type Lexer struct {
	input string
	pos   int
}

func NewLexer(s string) *Lexer {
	return &Lexer{input: s}
}

func (lx *Lexer) NextToken() Token {
	// Skip whitespace
	for lx.pos < len(lx.input) && unicode.IsSpace(rune(lx.input[lx.pos])) {
		lx.pos++
	}
	if lx.pos >= len(lx.input) {
		return Token{Kind: TokenEOF}
	}

	start := lx.pos
	ch := lx.input[lx.pos]

	// Mnemonic or size (leading alpha)
	if unicode.IsLetter(rune(ch)) {
		for lx.pos < len(lx.input) && (unicode.IsLetter(rune(lx.input[lx.pos])) || lx.input[lx.pos] == '.') {
			lx.pos++
		}
		val := lx.input[start:lx.pos]
		if strings.HasPrefix(val, ".") {
			return Token{Kind: TokenSize, Value: val}
		}
		return Token{Kind: TokenMnemonic, Value: val}
	}

	// Comma
	if ch == ',' {
		lx.pos++
		return Token{Kind: TokenComma, Value: ","}
	}

	// Operand: parse up to comma at top level (do not split inside parenthesis or register list)
	level := 0
	inDoubleQuotes := false
	inSingleQuotes := false
	for lx.pos < len(lx.input) {
		c := lx.input[lx.pos]
		switch c {
		case '(':
			if !inDoubleQuotes && !inSingleQuotes {
				level++
			}
		case ')':
			if !inDoubleQuotes && !inSingleQuotes {
				level--
			}
		case '"':
			inDoubleQuotes = !inDoubleQuotes
		case '\'':
			inSingleQuotes = !inSingleQuotes
		case ',':
			if level == 0 && !inDoubleQuotes && !inSingleQuotes {
				break
			}
		}
		if c == ',' && level == 0 && !inDoubleQuotes && !inSingleQuotes {
			break
		}
		lx.pos++
	}
	val := strings.TrimSpace(lx.input[start:lx.pos])
	return Token{Kind: TokenOperand, Value: val}
}

// LexInstruction splits an instruction line into mnemonic, size, and operand tokens
func LexInstruction(line string) []Token {
	lx := NewLexer(line)
	var toks []Token
	for {
		tok := lx.NextToken()
		if tok.Kind == TokenEOF {
			break
		}
		toks = append(toks, tok)
	}
	return toks
}