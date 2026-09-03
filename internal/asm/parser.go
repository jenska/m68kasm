package asm

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// maxProgramSize limits the amount of padding bytes we emit to protect against
// malicious inputs that would otherwise force enormous allocations (e.g. via
// huge .org/.align directives).
const maxProgramSize uint32 = 64 * 1024 * 1024 // 64 MiB

type lexer interface {
	Next() Token
}

type (
	DataBytes struct {
		Bytes   []byte
		PC      uint32
		Line    int
		Col     int
		Section SectionKind
	}

	Parser struct {
		lx               lexer
		labels           map[string]uint32
		definedLabelPos  map[string]int
		definedLabels    []DefinedLabel
		locals           map[int]int
		localForwards    map[int]int
		allowForwardRefs bool
		macros           map[string]macroDef
		instrs           *instructions.Table
		pc               uint32
		origin           uint32
		hasOrg           bool
		section          SectionKind
		items            []any
		line             int
		col              int

		macroDepth int

		buf          []Token // N-Token Lookahead
		tokenScratch []Token // reusable buffer for operand collection
		formScratch  []Token // reusable buffer for form parsing
	}

	macroDef struct {
		params []string
		body   []Token
	}
)

func copySymbols(src map[string]uint32) map[string]uint32 {
	if len(src) == 0 {
		return map[string]uint32{}
	}
	dst := make(map[string]uint32, len(src))
	maps.Copy(dst, src)
	return dst
}

type ParseOptions struct {
	Symbols    map[string]uint32
	InstrTable *instructions.Table
}

func Parse(r io.Reader) (*Program, error) {
	return ParseWithOptions(r, ParseOptions{})
}

func ParseWithOptions(r io.Reader, opts ParseOptions) (*Program, error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	lines := splitSourceLines(src)

	table := opts.InstrTable
	if table == nil {
		table = instructions.DefaultTable()
	}

	firstPass, err := parseWithLexer(NewLexer(bytes.NewReader(src)), table, opts.Symbols, true)
	if err != nil {
		return nil, withSourceLines(err, lines)
	}

	prog, err := parseWithLexer(NewLexer(bytes.NewReader(src)), table, firstPass.Labels, false)
	if err != nil {
		return nil, withSourceLines(err, lines)
	}
	prog.SourceLines = slices.Clone(lines)
	return prog, nil
}

func splitSourceLines(src []byte) []string {
	text := strings.ReplaceAll(string(src), "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func parseWithLexer(lx lexer, table *instructions.Table, symbols map[string]uint32, allowForward bool) (*Program, error) {
	p := &Parser{
		lx:               lx,
		labels:           copySymbols(symbols),
		definedLabelPos:  map[string]int{},
		locals:           map[int]int{},
		localForwards:    map[int]int{},
		allowForwardRefs: allowForward,
		macros:           map[string]macroDef{},
		instrs:           table,
		section:          SectionText,
	}
	for {
		t := p.peek()
		if t.Kind == EOF {
			break
		}
		if t.Kind == NEWLINE {
			p.next()
			continue
		}

		// Try parsing a label definition
		didLabel, err := p.parseLabelDefinition()
		if err != nil {
			return nil, err
		}
		if didLabel && (p.peek().Kind == NEWLINE || p.peek().Kind == EOF) {
			continue
		}

		expanded, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		if expanded {
			continue
		}

		if t := p.peek(); t.Kind != NEWLINE && t.Kind != EOF {
			return nil, errorAtToken(t, fmt.Errorf("unexpected token: %s", t.Text))
		}
		if p.peek().Kind == NEWLINE {
			p.next()
		}
	}

	if err := p.ensureLocalForwardsResolved(); err != nil {
		return nil, err
	}

	origin := p.origin
	if !p.hasOrg {
		origin = 0
	}

	definedLabels := slices.Clone(p.definedLabels)
	return &Program{Items: p.items, Labels: p.labels, DefinedLabels: definedLabels, Origin: origin}, nil
}

func ParseFile(path string) (*Program, error) {
	return ParseFileWithOptions(path, ParseOptions{})
}

func ParseFileWithOptions(path string, opts ParseOptions) (*Program, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ParseWithOptions(f, opts)
}

// -----------------------------------------------------------------------------
// Token Reader / Lexer Integration
func (p *Parser) fill(n int) {
	for len(p.buf) < n {
		p.buf = append(p.buf, p.lx.Next())
	}
}
func (p *Parser) next() Token {
	p.fill(1)
	t := p.buf[0]
	p.buf = p.buf[1:]
	p.line, p.col = t.Line, t.Col
	return t
}
func (p *Parser) peek() Token {
	p.fill(1)
	return p.buf[0]
}

func (p *Parser) peekN(n int) Token {
	p.fill(n)
	return p.buf[n-1]
}

func (p *Parser) want(k Kind) (Token, error) {
	t := p.next()
	if t.Kind != k {
		return t, errorAtToken(t, fmt.Errorf("expected %v, got %v (%q)", k, t.Kind, t.Text))
	}
	return t, nil
}

func (p *Parser) accept(k Kind) bool {
	if p.peek().Kind == k {
		_ = p.next()
		return true
	}
	return false
}

// -----------------------------------------------------------------------------
// Register Helpers
func isRegDn(s string) (bool, int) {
	if trimmed, ok := stripRegSizeSuffix(s); ok {
		s = trimmed
	}
	if len(s) == 2 && (s[0] == 'd' || s[0] == 'D') {
		r := int(s[1] - '0')
		if 0 <= r && r <= 7 {
			return true, r
		}
	}
	return false, 0
}

func isRegAn(s string) (bool, int) {
	if trimmed, ok := stripRegSizeSuffix(s); ok {
		s = trimmed
	}
	if len(s) == 2 && (s[0] == 'a' || s[0] == 'A') {
		r := int(s[1] - '0')
		if 0 <= r && r <= 7 {
			return true, r
		}
	}
	if strings.EqualFold(s, "SP") || strings.EqualFold(s, "SSP") {
		return true, 7
	}
	return false, 0
}

func isPC(s string) bool {
	return strings.EqualFold(s, "PC")
}

func stripRegSizeSuffix(s string) (string, bool) {
	if len(s) >= 3 && s[2] == '.' {
		suf := strings.ToLower(s[3:])
		if suf == "b" || suf == "w" || suf == "l" || suf == "s" {
			return s[:2], true
		}
	}
	return s, false
}
