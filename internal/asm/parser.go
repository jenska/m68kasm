package asm

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
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
		target           instructions.Target
		pc               uint32
		origin           uint32
		hasOrg           bool
		section          SectionKind
		items            []any
		line             int
		col              int

		macroDepth int

		// baseDir is the directory INCLUDE and INCBIN resolve relative
		// paths against: the directory of the including source file.
		// Empty when the source did not come from a file.
		baseDir      string
		includeDepth int
		// ended is set by END; the statement loops stop when it is true.
		ended bool
		// symKinds records, for the current pass, how each EQU/SET/=
		// symbol was defined, so EQU cannot be redefined.
		symKinds map[string]symKind

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

// ParseOptions controls parser customization for all public assembly
// helpers. Symbols predefines label values, InstrTable lets advanced
// callers supply an alternate instruction table, and Target selects the
// CPU tier (and, in later milestones, coprocessor features) to assemble
// for. The zero value targets a bare 68000, matching this assembler's
// behavior before Target existed.
type ParseOptions struct {
	Symbols    map[string]uint32
	InstrTable *instructions.Table
	Target     instructions.Target

	// BaseDir is the directory INCLUDE and INCBIN resolve relative paths
	// against. ParseFile and the AssembleFile helpers set it to the
	// directory of the source file; for a reader or string it is empty and
	// INCLUDE/INCBIN are errors unless the caller sets it.
	BaseDir string
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
	table = table.ForTarget(opts.Target)

	firstPass, err := parseWithLexer(NewLexer(bytes.NewReader(src)), table, opts.Symbols, true, opts.Target, opts.BaseDir)
	if err != nil {
		return nil, withSourceLines(err, lines)
	}

	prog, err := parseWithLexer(NewLexer(bytes.NewReader(src)), table, firstPass.Labels, false, opts.Target, opts.BaseDir)
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

func parseWithLexer(lx lexer, table *instructions.Table, symbols map[string]uint32, allowForward bool, target instructions.Target, baseDir string) (*Program, error) {
	p := &Parser{
		lx:               lx,
		labels:           copySymbols(symbols),
		definedLabelPos:  map[string]int{},
		locals:           map[int]int{},
		localForwards:    map[int]int{},
		allowForwardRefs: allowForward,
		macros:           map[string]macroDef{},
		instrs:           table,
		target:           target,
		section:          SectionText,
		baseDir:          baseDir,
		symKinds:         map[string]symKind{},
	}
	if err := p.parseStatements(); err != nil {
		return nil, err
	}

	if err := p.ensureLocalForwardsResolved(); err != nil {
		return nil, err
	}

	origin := p.origin
	if !p.hasOrg {
		origin = 0
	}

	definedLabels := slices.Clone(p.definedLabels)
	return &Program{Items: p.items, Labels: p.labels, DefinedLabels: definedLabels, Origin: origin, Target: target}, nil
}

// parseStatements parses statements from the current lexer until it runs
// out of tokens or END is seen.
func (p *Parser) parseStatements() error {
	for !p.ended {
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
			return err
		}
		if didLabel && (p.peek().Kind == NEWLINE || p.peek().Kind == EOF) {
			continue
		}

		expanded, err := p.parseStmt()
		if err != nil {
			return err
		}
		if expanded || p.ended {
			continue
		}

		if t := p.peek(); t.Kind != NEWLINE && t.Kind != EOF {
			return errorAtToken(t, fmt.Errorf("unexpected token: %s", t.Text))
		}
		if p.peek().Kind == NEWLINE {
			p.next()
		}
	}
	return nil
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

	if opts.BaseDir == "" {
		opts.BaseDir = filepath.Dir(path)
	}
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
