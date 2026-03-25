package asm

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

type Kind int

const (
	EOF Kind = iota
	IDENT
	NUMBER
	STRING
	COMMA
	COLON
	EQUAL
	EQEQ
	HASH
	LPAREN
	RPAREN
	DOT
	PLUS
	MINUS
	STAR
	SLASH
	PERCENT
	BANG
	LT
	GT
	LTE
	GTE
	NEQ
	ANDAND
	OROR
	LSHIFT // <<
	RSHIFT // >>
	AMP
	PIPE
	CARET
	TILDE
	NEWLINE
)

type (
	Token struct {
		Kind Kind
		Text string
		Val  int64
		Line int
		Col  int
	}

	Lexer struct {
		r    *bufio.Reader
		line int
		col  int
		peek *Token
	}
)

func (t *Token) String() string {
	return fmt.Sprintf("(%d, %d) token '%s'", t.Line, t.Col, t.Text)
}

func (k Kind) String() string {
	switch k {
	case EOF:
		return "EOF"
	case IDENT:
		return "identifier"
	case NUMBER:
		return "number"
	case STRING:
		return "string"
	case COMMA:
		return "comma"
	case COLON:
		return "colon"
	case EQUAL:
		return "equal"
	case EQEQ:
		return "double-equal"
	case HASH:
		return "hash"
	case LPAREN:
		return "left paren"
	case RPAREN:
		return "right paren"
	case DOT:
		return "dot"
	case PLUS:
		return "plus"
	case MINUS:
		return "minus"
	case STAR:
		return "star"
	case SLASH:
		return "slash"
	case PERCENT:
		return "percent"
	case BANG:
		return "bang"
	case LT:
		return "less-than"
	case GT:
		return "greater-than"
	case LTE:
		return "less-than-equal"
	case GTE:
		return "greater-than-equal"
	case NEQ:
		return "not-equal"
	case ANDAND:
		return "double-ampersand"
	case OROR:
		return "double-pipe"
	case LSHIFT:
		return "left-shift"
	case RSHIFT:
		return "right-shift"
	case AMP:
		return "ampersand"
	case PIPE:
		return "pipe"
	case CARET:
		return "caret"
	case TILDE:
		return "tilde"
	case NEWLINE:
		return "newline"
	default:
		return fmt.Sprintf("Kind(%d)", int(k))
	}
}

func NewLexer(r io.Reader) *Lexer { return &Lexer{r: bufio.NewReader(r), line: 1, col: 0} }

func (lx *Lexer) Next() Token {
	if lx.peek != nil {
		t := *lx.peek
		lx.peek = nil
		return t
	}
	return lx.next()
}

func (lx *Lexer) Peek() Token {
	if lx.peek != nil {
		return *lx.peek
	}
	t := lx.next()
	lx.peek = &t
	return t
}

func (lx *Lexer) next() Token {
	for {
		ch := lx.read()
		if ch == eof {
			return lx.tok(EOF, "", 0)
		}
		if ch == '\r' {
			if lx.peekRune() == '\n' {
				lx.read()
			}
			return lx.tok(NEWLINE, "\n", 0)
		}
		if ch == '\n' {
			return lx.tok(NEWLINE, "\n", 0)
		}
		if ch == ';' {
			lx.skipUntilNewline()
			return lx.tok(NEWLINE, "\n", 0)
		}
		if unicode.IsSpace(ch) {
			continue
		}
		switch ch {
		case ',':
			return lx.tok(COMMA, ",", 0)
		case ':':
			return lx.tok(COLON, ":", 0)
		case '=':
			if lx.peekRune() == '=' {
				lx.read()
				return lx.tok(EQEQ, "==", 0)
			}
			return lx.tok(EQUAL, "=", 0)
		case '#':
			return lx.tok(HASH, "#", 0)
		case '(':
			return lx.tok(LPAREN, "(", 0)
		case ')':
			return lx.tok(RPAREN, ")", 0)
		case '.':
			return lx.tok(DOT, ".", 0)
		case '+':
			return lx.tok(PLUS, "+", 0)
		case '-':
			return lx.tok(MINUS, "-", 0)
		case '*':
			return lx.tok(STAR, "*", 0)
		case '/':
			return lx.tok(SLASH, "/", 0)
		case '%':
			return lx.tok(PERCENT, "%", 0)
		case '!':
			if lx.peekRune() == '=' {
				lx.read()
				return lx.tok(NEQ, "!=", 0)
			}
			return lx.tok(BANG, "!", 0)
		case '<':
			if lx.peekRune() == '<' {
				lx.read()
				return lx.tok(LSHIFT, "<<", 0)
			}
			if lx.peekRune() == '=' {
				lx.read()
				return lx.tok(LTE, "<=", 0)
			}
			return lx.tok(LT, "<", 0)
		case '>':
			if lx.peekRune() == '>' {
				lx.read()
				return lx.tok(RSHIFT, ">>", 0)
			}
			if lx.peekRune() == '=' {
				lx.read()
				return lx.tok(GTE, ">=", 0)
			}
			return lx.tok(GT, ">", 0)
		case '&':
			if lx.peekRune() == '&' {
				lx.read()
				return lx.tok(ANDAND, "&&", 0)
			}
			return lx.tok(AMP, "&", 0)
		case '|':
			if lx.peekRune() == '|' {
				lx.read()
				return lx.tok(OROR, "||", 0)
			}
			return lx.tok(PIPE, "|", 0)
		case '^':
			return lx.tok(CARET, "^", 0)
		case '~':
			return lx.tok(TILDE, "~", 0)
		case '"':
			return lx.scanString()
		case '\'':
			return lx.scanChar()
		default:
			if isIdentStart(ch) {
				return lx.scanIdent(ch)
			}
			if unicode.IsDigit(ch) || ch == '$' || ch == '%' || ch == '@' {
				return lx.scanNumber(ch)
			}
			return lx.errToken(fmt.Errorf("unexpected char: %q", ch))
		}
	}
}

func (lx *Lexer) scanIdent(first rune) Token {
	var b strings.Builder
	b.WriteRune(first)
	for {
		ch := lx.peekRune()
		if isIdentContinue(ch) {
			lx.read()
			b.WriteRune(ch)
			continue
		}
		break
	}
	return lx.tok(IDENT, b.String(), 0)
}

func (lx *Lexer) scanNumber(first rune) Token {
	var b strings.Builder
	b.WriteRune(first)

	if first == '$' {
		return lx.finishBaseNumber(&b, 1, 16, isHex)
	}
	if first == '%' {
		return lx.finishBaseNumber(&b, 1, 2, isBinary)
	}
	if first == '@' {
		return lx.finishBaseNumber(&b, 1, 8, isOctal)
	}
	if first == '0' && (lx.peekRune() == 'x' || lx.peekRune() == 'X') {
		lx.read() // x
		b.WriteByte('x')
		return lx.finishBaseNumber(&b, 2, 16, isHex)
	}
	lx.scanWhile(&b, unicode.IsDigit)
	v, err := strconv.ParseInt(b.String(), 10, 64)
	if err != nil {
		return lx.errToken(err)
	}
	return lx.tok(NUMBER, b.String(), v)
}

func (lx *Lexer) scanString() Token {
	var b strings.Builder
	for {
		ch := lx.read()
		if ch == eof || ch == '\n' || ch == '\r' {
			return lx.errToken(fmt.Errorf("unterminated string"))
		}
		if ch == '\\' {
			b.WriteRune(lx.readEscapedRune())
			continue
		}
		if ch == '"' {
			break
		}
		b.WriteRune(ch)
	}
	return lx.tok(STRING, b.String(), 0)
}

func (lx *Lexer) scanChar() Token {
	ch := lx.read()
	var v rune
	if ch == '\\' {
		v = lx.readEscapedRune()
	} else {
		v = ch
	}
	if lx.read() != '\'' {
		return lx.errToken(fmt.Errorf("unterminated char literal"))
	}
	return lx.tok(NUMBER, fmt.Sprintf("'%c'", v), int64(v))
}

func (lx *Lexer) finishBaseNumber(b *strings.Builder, prefixLen int, base int, accept func(rune) bool) Token {
	lx.scanWhile(b, accept)
	text := b.String()
	v, err := strconv.ParseInt(text[prefixLen:], base, 64)
	if err != nil {
		return lx.errToken(err)
	}
	return lx.tok(NUMBER, text, v)
}

func (lx *Lexer) scanWhile(b *strings.Builder, accept func(rune) bool) {
	for {
		ch := lx.peekRune()
		if !accept(ch) {
			return
		}
		lx.read()
		b.WriteRune(ch)
	}
}

func (lx *Lexer) readEscapedRune() rune {
	switch esc := lx.read(); esc {
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	case '\\':
		return '\\'
	case '"':
		return '"'
	case '\'':
		return '\''
	case '0':
		return 0
	default:
		return esc
	}
}

func (lx *Lexer) skipUntilNewline() {
	for {
		ch := lx.peekRune()
		if ch == eof || ch == '\n' || ch == '\r' {
			return
		}
		lx.read()
	}
}

const eof = rune(0)

func (lx *Lexer) read() rune {
	ch, _, err := lx.r.ReadRune()
	if err != nil {
		return eof
	}
	if ch == '\n' {
		lx.line++
		lx.col = 0
	} else {
		lx.col++
	}
	return ch
}

func (lx *Lexer) peekRune() rune {
	ch, _, err := lx.r.ReadRune()
	if err != nil {
		return eof
	}
	_ = lx.r.UnreadRune()
	return ch
}

func (lx *Lexer) tok(k Kind, text string, val int64) Token {
	return Token{Kind: k, Text: text, Val: val, Line: lx.line, Col: lx.col}
}

func (lx *Lexer) errToken(err error) Token {
	return Token{Kind: EOF, Text: err.Error(), Val: 0, Line: lx.line, Col: lx.col}
}

func isIdentStart(ch rune) bool { return unicode.IsLetter(ch) || ch == '_' || ch == '.' }
func isIdentContinue(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '.'
}
func isHex(ch rune) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}
func isBinary(ch rune) bool { return ch == '0' || ch == '1' }
func isOctal(ch rune) bool  { return ch >= '0' && ch <= '7' }

// sliceLexer is a lexer implementation that reads from a pre-filled slice of tokens.
// It is primarily used for macro expansion and re-parsing instruction forms.
type sliceLexer struct {
	tokens []Token
	pos    int
}

func (s *sliceLexer) Next() Token {
	if s.pos >= len(s.tokens) {
		return Token{Kind: EOF}
	}
	t := s.tokens[s.pos]
	s.pos++
	return t
}
