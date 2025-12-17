package asm

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
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
		Bytes []byte
		PC    uint32
		Line  int
	}

	Parser struct {
		lx               lexer
		labels           map[string]uint32
		locals           map[int]int
		localForwards    map[int]int
		allowForwardRefs bool
		macros           map[string]macroDef
		instrs           *instructions.Table
		pc               uint32
		origin           uint32
		hasOrg           bool
		items            []any
		line             int
		col              int

		macroDepth    int
		macroExpanded bool

		buf          []Token // N-Token Lookahead
		tokenScratch []Token // reusable buffer for operand collection
		formScratch  []Token // reusable buffer for form parsing
	}

	sliceLexer struct {
		tokens []Token
		pos    int
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
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func localLabelName(num, idx int) string {
	return fmt.Sprintf("__local_%d_%d", num, idx)
}

func (p *Parser) defineLocalLabel(tok Token) error {
	num := int(tok.Val)
	if num < 0 {
		return parserError(tok, "local label must be non-negative")
	}
	idx := p.locals[num] + 1
	name := localLabelName(num, idx)
	p.locals[num] = idx
	p.labels[name] = p.pc
	return nil
}

func (p *Parser) resolveLocalLabel(num int, forward bool) (string, error) {
	if num < 0 {
		return "", fmt.Errorf("local label must be non-negative")
	}
	if forward {
		idx := p.locals[num] + 1
		if p.localForwards[num] < idx {
			p.localForwards[num] = idx
		}
		return localLabelName(num, idx), nil
	}
	if p.locals[num] == 0 {
		return "", fmt.Errorf("no previous local label %d", num)
	}
	return localLabelName(num, p.locals[num]), nil
}

func (p *Parser) ensureLocalForwardsResolved() error {
	for num, needed := range p.localForwards {
		if p.locals[num] < needed {
			return fmt.Errorf("no forward definition for local label %d", num)
		}
	}
	return nil
}

func (s *sliceLexer) Next() Token {
	if s.pos >= len(s.tokens) {
		return Token{Kind: EOF}
	}
	t := s.tokens[s.pos]
	s.pos++
	return t
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

	table := opts.InstrTable
	if table == nil {
		table = instructions.DefaultTable()
	}

	firstPass, err := parseWithLexer(NewLexer(bytes.NewReader(src)), table, opts.Symbols, true)
	if err != nil {
		return nil, err
	}

	return parseWithLexer(NewLexer(bytes.NewReader(src)), table, firstPass.Labels, false)
}

func parseWithLexer(lx lexer, table *instructions.Table, symbols map[string]uint32, allowForward bool) (*Program, error) {
	p := &Parser{
		lx:               lx,
		labels:           copySymbols(symbols),
		locals:           map[int]int{},
		localForwards:    map[int]int{},
		allowForwardRefs: allowForward,
		macros:           map[string]macroDef{},
		instrs:           table,
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

		// Label? IDENT ':' or NUMBER ':' for local labels
		if (t.Kind == IDENT || t.Kind == NUMBER) && p.peekN(2).Kind == COLON {
			lbl := p.next()
			p.next() // ':'
			if lbl.Kind == IDENT {
				p.labels[lbl.Text] = p.pc
			} else {
				if err := p.defineLocalLabel(lbl); err != nil {
					return nil, err
				}
			}
			nt := p.peek()
			if nt.Kind == NEWLINE || nt.Kind == EOF {
				continue
			}
		}

		if err := p.parseStmt(); err != nil {
			return nil, err
		}

		if p.macroExpanded {
			p.macroExpanded = false
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

	return &Program{Items: p.items, Labels: p.labels, Origin: origin}, nil
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

func parserError(t Token, msg string) error {
	return errorAtToken(t, fmt.Errorf("%s", msg))
}

func (p *Parser) consumeLocalLabelRef() (string, bool, error) {
	numTok := p.peek()
	dirTok := p.peekN(2)
	if numTok.Kind != NUMBER || dirTok.Kind != IDENT {
		return "", false, nil
	}
	dir := strings.ToLower(dirTok.Text)
	if dir != "f" && dir != "b" {
		return "", false, nil
	}
	p.next() // number
	p.next() // direction
	name, err := p.resolveLocalLabel(int(numTok.Val), dir == "f")
	return name, true, err
}

func (p *Parser) parseLabelReference() (string, error) {
	if p.peek().Kind == IDENT {
		return p.next().Text, nil
	}
	if name, ok, err := p.consumeLocalLabelRef(); ok {
		return name, err
	}
	t := p.next()
	return "", parserError(t, "expected label")
}

func (p *Parser) parseStmt() error {
	t := p.peek()

	if t.Kind == IDENT {
		if def, ok := p.macros[t.Text]; ok {
			return p.invokeMacro(def)
		}
		if p.peekN(2).Kind == EQUAL {
			return p.parseConstDefinition(t)
		}
		if p.peekN(2).Kind == DOT {
			if eq := p.peekN(3); eq.Kind == IDENT && strings.EqualFold(eq.Text, "equ") {
				return p.parseConstDefinition(t)
			}
		}

		s := strings.ToUpper(t.Text)
		if idx := strings.IndexRune(s, '.'); idx > 0 {
			s = s[:idx]
		}
		if instrDef := p.instrs.Lookup(s); instrDef != nil {
			return p.parseInstruction(instrDef)
		}
		if strings.HasPrefix(strings.ToUpper(t.Text), "DC.") {
			_ = p.next()
			suf := ""
			if idx := strings.IndexRune(t.Text, '.'); idx >= 0 && idx < len(t.Text)-1 {
				suf = t.Text[idx+1:]
			}
			return parseDC(p, suf)
		}
		if pseudo, ok := pseudoMap["."+strings.ToUpper(t.Text)]; ok {
			_ = p.next()
			return pseudo(p)
		}
		return parserError(t, "unknown mnemonic")
	}

	if t.Kind == DOT {
		p.next()
		id, err := p.want(IDENT)
		if err != nil {
			return err
		}
		name := "." + strings.ToUpper(id.Text)
		if pseudo, ok := pseudoMap[name]; ok {
			return pseudo(p)
		}
		return parserError(t, "unknown pseudo op")
	}
	return parserError(t, "unexpected token")
}

func (p *Parser) parseConstDefinition(nameTok Token) error {
	_ = p.next() // consume name

	if p.peek().Kind == DOT {
		p.next()
		eqTok, err := p.want(IDENT)
		if err != nil {
			return err
		}
		if !strings.EqualFold(eqTok.Text, "equ") {
			return parserError(eqTok, "expected EQU")
		}
	} else {
		if _, err := p.want(EQUAL); err != nil {
			return err
		}
	}

	val, err := p.parseExpr()
	if err != nil {
		return err
	}
	if val < 0 || val > math.MaxUint32 {
		return errorAtLine(nameTok.Line, fmt.Errorf("constant out of 32-bit range: %d", val))
	}

	p.labels[nameTok.Text] = uint32(val)
	return nil
}

func (p *Parser) parseInstruction(instrDef *instructions.InstrDef) error {
	mn, err := p.want(IDENT)
	if err != nil {
		return err
	}

	operandTokens := p.consumeUntilEOL()
	defer p.releaseTokens(operandTokens)
	var lastErr error
	for _, form := range instrDef.Forms {
		args, err := p.tryParseForm(mn, &form, operandTokens)
		if err != nil {
			lastErr = err
			continue
		}
		ins := &Instr{Def: instrDef, Args: args, PC: p.pc, Line: mn.Line}
		p.items = append(p.items, ins)
		words, err := instructionWords(&form, args)
		if err != nil {
			return err
		}
		p.pc += uint32(words * 2)
		return nil
	}

	if lastErr != nil {
		return contextualize(mn.Line, lastErr)
	}
	return errorAtLine(mn.Line, fmt.Errorf("no form matches operands"))
}

func instructionWords(form *instructions.FormDef, args instructions.Args) (int, error) {
	words := 0

	var srcEA, dstEA instructions.EAEncoded
	var err error

	if args.Src.Kind != instructions.EAkNone {
		srcEA, err = instructions.EncodeEA(args.Src)
		if err != nil {
			return 0, err
		}
	}
	if args.Dst.Kind != instructions.EAkNone {
		dstEA, err = instructions.EncodeEA(args.Dst)
		if err != nil {
			return 0, err
		}
	}

	for _, step := range form.Steps {
		haveWord := (step.WordBits != 0) || (len(step.Fields) > 0)
		if haveWord {
			words++
		}
		for _, tr := range step.Trailer {
			switch tr {
			case instructions.TSrcEAExt:
				words += len(srcEA.Ext)
			case instructions.TDstEAExt:
				words += len(dstEA.Ext)
			case instructions.TImmSized:
				words++
			case instructions.TSrcImm:
				if args.Src.Kind == instructions.EAkImm {
					switch args.Size {
					case instructions.LongSize:
						words += 2
					default:
						words++
					}
				}
			case instructions.TBranchWordIfNeeded:
				if args.Size == instructions.WordSize {
					words++
				}
			case instructions.TSrcRegMask, instructions.TDstRegMask:
				words++
			}
		}
	}

	return words, nil
}

func (p *Parser) consumeUntilEOL() []Token {
	tokens := p.tokenScratch[:0]
	if cap(tokens) == 0 {
		tokens = make([]Token, 0, 8)
	}
	for {
		t := p.peek()
		if t.Kind == NEWLINE || t.Kind == EOF {
			break
		}
		tokens = append(tokens, p.next())
	}
	return tokens
}

func (p *Parser) releaseTokens(tokens []Token) {
	p.tokenScratch = tokens[:0]
}

func (p *Parser) unshiftTokens(tokens []Token) {
	if len(tokens) == 0 {
		return
	}
	buf := make([]Token, len(tokens)+len(p.buf))
	copy(buf, tokens)
	copy(buf[len(tokens):], p.buf)
	p.buf = buf
}

func (p *Parser) invokeMacro(def macroDef) error {
	nameTok := p.next()
	rawArgs := p.consumeUntilEOL()
	argTokens := append([]Token(nil), rawArgs...)
	p.releaseTokens(rawArgs)

	args, err := splitMacroArgs(argTokens)
	if err != nil {
		return errorAtLine(nameTok.Line, err)
	}
	if len(args) != len(def.params) {
		return errorAtLine(nameTok.Line, fmt.Errorf("macro %s expects %d args, got %d", nameTok.Text, len(def.params), len(args)))
	}

	if p.macroDepth > 64 {
		return errorAtLine(nameTok.Line, fmt.Errorf("macro expansion depth exceeded"))
	}
	p.macroDepth++
	defer func() { p.macroDepth-- }()

	expanded := make([]Token, 0, len(def.body)+len(argTokens))
	for _, t := range def.body {
		replaced := false
		if t.Kind == IDENT {
			for i, param := range def.params {
				if t.Text == param {
					for _, at := range args[i] {
						cloned := at
						cloned.Line = nameTok.Line
						cloned.Col = nameTok.Col
						expanded = append(expanded, cloned)
					}
					replaced = true
					break
				}
			}
		}
		if replaced {
			continue
		}
		clone := t
		clone.Line = nameTok.Line
		clone.Col = nameTok.Col
		expanded = append(expanded, clone)
	}

	p.macroExpanded = true
	p.unshiftTokens(expanded)
	return nil
}

func splitMacroArgs(tokens []Token) ([][]Token, error) {
	if len(tokens) == 0 {
		return [][]Token{}, nil
	}
	args := [][]Token{}
	current := []Token{}
	depth := 0
	for _, t := range tokens {
		switch t.Kind {
		case LPAREN:
			depth++
		case RPAREN:
			if depth == 0 {
				return nil, fmt.Errorf("unmatched ')'")
			}
			depth--
		case COMMA:
			if depth == 0 {
				args = append(args, append([]Token{}, current...))
				current = current[:0]
				continue
			}
		}
		current = append(current, t)
	}
	if depth != 0 {
		return nil, fmt.Errorf("unmatched '('")
	}
	args = append(args, append([]Token{}, current...))
	return args, nil
}

func (p *Parser) tryParseForm(mn Token, form *instructions.FormDef, tokens []Token) (instructions.Args, error) {
	args := instructions.Args{}
	origLX, origBuf, origLine, origCol := p.lx, p.buf, p.line, p.col
	defer func() {
		p.lx, p.buf, p.line, p.col = origLX, origBuf, origLine, origCol
		p.formScratch = p.formScratch[:0]
	}()

	// isolate parsing to the captured tokens
	tmp := append(p.formScratch[:0], tokens...)
	tmp = append(tmp, Token{Kind: EOF, Line: mn.Line})
	p.formScratch = tmp
	p.lx = &sliceLexer{tokens: tmp}
	p.buf = nil

	sz, err := p.parseSizeSpec(mn, form.DefaultSize, form.Sizes)
	if err != nil {
		return args, err
	}
	args.Size = sz

	for i, operandKind := range form.OperKinds {
		if i == 1 {
			if _, err := p.want(COMMA); err != nil {
				return args, err
			}
		}

		eaExpr, err := p.parseOperand(operandKind, mn, &args, i)
		if err != nil {
			return args, err
		}

		if i == 0 {
			args.Src = eaExpr
		} else {
			args.Dst = eaExpr
		}
	}

	if trailing := p.peek(); trailing.Kind != EOF {
		return args, errorAtToken(trailing, fmt.Errorf("unexpected token %s", trailing.Text))
	}

	return args, nil
}

func (p *Parser) parseOperand(kind instructions.OperandKind, mn Token, args *instructions.Args, position int) (instructions.EAExpr, error) {
	var eaExpr instructions.EAExpr

	switch kind {
	case instructions.OpkImm:
		if _, err := p.want(HASH); err != nil {
			return eaExpr, err
		}
		imm, err := p.parseExpr()
		if err != nil {
			return eaExpr, err
		}
		eaExpr.Kind = instructions.EAkImm
		eaExpr.Imm = imm

	case instructions.OpkImmQuick:
		if _, err := p.want(HASH); err != nil {
			return eaExpr, err
		}
		imm, err := p.parseExpr()
		if err != nil {
			return eaExpr, err
		}
		eaExpr.Kind = instructions.EAkNone
		eaExpr.Imm = imm
		args.HasImmQuick = true

	case instructions.OpkDn:
		dreg, err := p.want(IDENT)
		if err != nil {
			return eaExpr, err
		}
		ok, dn := isRegDn(dreg.Text)
		if !ok {
			return eaExpr, errorAtToken(dreg, fmt.Errorf("expected Dn, got %s", dreg.Text))
		}
		eaExpr.Kind = instructions.EAkDn
		eaExpr.Reg = dn

	case instructions.OpkAn:
		areg, err := p.want(IDENT)
		if err != nil {
			return eaExpr, err
		}
		ok, an := isRegAn(areg.Text)
		if !ok {
			return eaExpr, errorAtToken(areg, fmt.Errorf("expected An, got %s", areg.Text))
		}
		eaExpr.Kind = instructions.EAkAn
		eaExpr.Reg = an

	case instructions.OpkSR:
		tok, err := p.want(IDENT)
		if err != nil {
			return eaExpr, err
		}
		if !strings.EqualFold(tok.Text, "SR") {
			return eaExpr, errorAtToken(tok, fmt.Errorf("expected SR"))
		}
		eaExpr.Kind = instructions.EAkSR

	case instructions.OpkCCR:
		tok, err := p.want(IDENT)
		if err != nil {
			return eaExpr, err
		}
		if !strings.EqualFold(tok.Text, "CCR") {
			return eaExpr, errorAtToken(tok, fmt.Errorf("expected CCR"))
		}
		eaExpr.Kind = instructions.EAkCCR

	case instructions.OpkUSP:
		tok, err := p.want(IDENT)
		if err != nil {
			return eaExpr, err
		}
		if !strings.EqualFold(tok.Text, "USP") {
			return eaExpr, errorAtToken(tok, fmt.Errorf("expected USP"))
		}
		eaExpr.Kind = instructions.EAkUSP

	case instructions.OpkEA:
		ea, err := p.parseEA()
		if err != nil {
			return eaExpr, err
		}
		eaExpr = ea

	case instructions.OpkPredecAn:
		ea, err := p.parseEA()
		if err != nil {
			return eaExpr, err
		}
		if ea.Kind != instructions.EAkAddrPredec {
			return eaExpr, errorAtLine(mn.Line, fmt.Errorf("expected -(An)"))
		}
		eaExpr = ea

	case instructions.OpkRegList:
		mask, err := p.parseRegList()
		if err != nil {
			return eaExpr, err
		}
		eaExpr.Kind = instructions.EAkNone
		if position == 0 {
			args.RegMaskSrc = mask
		} else {
			args.RegMaskDst = mask
		}

	case instructions.OpkDispRel:
		name, err := p.parseLabelReference()
		if err != nil {
			return eaExpr, err
		}
		args.Target = name

	default:
		return eaExpr, errorAtLine(mn.Line, fmt.Errorf("unknown identifier %s", mn.Text))
	}

	return eaExpr, nil
}

// emitPaddingBytes centralizes emitting zero/filled padding while keeping the
// generated output within a safe upper bound. Without this guard, a crafted
// .org/.align could request gigabytes of padding and exhaust memory.
func (p *Parser) emitPaddingBytes(count uint32, fill byte) error {
	if count == 0 {
		return nil
	}
	if count > maxProgramSize || p.pc > maxProgramSize-count {
		return errorAtLine(p.line, fmt.Errorf("padding would exceed maximum program size of %d bytes", maxProgramSize))
	}

	buf := make([]byte, int(count))
	if fill != 0 {
		for i := range buf {
			buf[i] = fill
		}
	}
	p.items = append(p.items, &DataBytes{Bytes: buf, PC: p.pc, Line: p.line})
	p.pc += count
	return nil
}

func sizeAllowedList(sz instructions.Size, allowed []instructions.Size) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if a == sz {
			return true
		}
	}
	return false
}

func (p *Parser) parseSizeSpec(mn Token, def instructions.Size, allowed []instructions.Size) (instructions.Size, error) {
	if idx := strings.IndexRune(mn.Text, '.'); idx > 0 {
		suf := mn.Text[idx+1:]
		if suf == "" {
			return 0, parserError(mn, "unknown size suffix")
		}
		sz, ok := sizeFromIdent(suf)
		if !ok {
			return 0, parserError(mn, "unknown size suffix "+suf)
		}
		if !sizeAllowedList(sz, allowed) {
			return 0, parserError(mn, "illegal size for instruction")
		}
		return sz, nil
	}
	sz, err := p.parseSizeSuffix(def, allowed)
	if err != nil {
		return 0, err
	}
	return sz, nil
}

func (p *Parser) parseSizeSuffix(def instructions.Size, allowed []instructions.Size) (instructions.Size, error) {
	sz := def
	if p.accept(DOT) {
		id, err := p.want(IDENT)
		if err != nil {
			return 0, err
		}
		val, ok := sizeFromIdent(id.Text)
		if !ok {
			return 0, parserError(id, "unknown size suffix")
		}
		sz = val
	}
	if !sizeAllowedList(sz, allowed) {
		return 0, fmt.Errorf("(%d, %d): illegal size for instruction", p.line, p.col)
	}
	return sz, nil
}

func sizeFromIdent(s string) (instructions.Size, bool) {
	switch strings.ToLower(s) {
	case "b":
		return instructions.ByteSize, true
	case "s":
		return instructions.ByteSize, true
	case "w":
		return instructions.WordSize, true
	case "l":
		return instructions.LongSize, true
	default:
		return 0, false
	}
}

func (p *Parser) parseRegList() (uint16, error) {
	mask := uint16(0)
	for {
		regTok, err := p.want(IDENT)
		if err != nil {
			return 0, err
		}
		isA, reg, err := parseRegName(regTok)
		if err != nil {
			return 0, err
		}
		endIsA, endReg := isA, reg
		if p.accept(MINUS) {
			toTok, err := p.want(IDENT)
			if err != nil {
				return 0, err
			}
			endIsA, endReg, err = parseRegName(toTok)
			if err != nil {
				return 0, err
			}
			if endIsA != isA {
				return 0, errorAtToken(toTok, fmt.Errorf("register ranges must stay within D or A registers"))
			}
			if endReg < reg {
				return 0, errorAtToken(toTok, fmt.Errorf("descending ranges are not allowed"))
			}
		}
		for r := reg; r <= endReg; r++ {
			bit := uint16(1 << r)
			if isA {
				bit <<= 8
			}
			mask |= bit
		}
		if p.peek().Kind == SLASH {
			p.next()
			continue
		}
		if p.peek().Kind == COMMA {
			nxt := p.peekN(2)
			if nxt.Kind == IDENT {
				if ok, _ := isRegDn(nxt.Text); ok {
					p.next()
					continue
				}
				if ok, _ := isRegAn(nxt.Text); ok {
					p.next()
					continue
				}
			}
			return mask, nil
		}
		break
	}
	return mask, nil
}

func parseRegName(tok Token) (bool, int, error) {
	if ok, dn := isRegDn(tok.Text); ok {
		return false, dn, nil
	}
	if ok, an := isRegAn(tok.Text); ok {
		return true, an, nil
	}
	return false, 0, errorAtToken(tok, fmt.Errorf("expected register in list"))
}

// ---------- EA parsing ----------
func (p *Parser) parseEA() (instructions.EAExpr, error) {
	t := p.peek()
	if t.Kind == MINUS && p.peekN(2).Kind == LPAREN {
		p.next() // '-'
		p.next() // '('
		areg, err := p.want(IDENT)
		if err != nil {
			return instructions.EAExpr{}, err
		}
		if !strings.HasPrefix(strings.ToUpper(areg.Text), "A") {
			return instructions.EAExpr{}, errorAtToken(areg, fmt.Errorf("expected address register"))
		}
		if _, err := p.want(RPAREN); err != nil {
			return instructions.EAExpr{}, err
		}
		ok, an := isRegAn(areg.Text)
		if !ok {
			return instructions.EAExpr{}, errorAtToken(areg, fmt.Errorf("expected address register"))
		}
		return instructions.EAExpr{Kind: instructions.EAkAddrPredec, Reg: an}, nil
	}
	if t.Kind == HASH {
		p.next()
		v, err := p.parseExpr()
		if err != nil {
			return instructions.EAExpr{}, err
		}
		return instructions.EAExpr{Kind: instructions.EAkImm, Imm: v}, nil
	}
	if t.Kind == IDENT || t.Kind == NUMBER {
		if p.peekN(2).Kind == LPAREN {
			disp, err := p.parseExprUntil(LPAREN)
			if err != nil {
				return instructions.EAExpr{}, err
			}
			if _, err := p.want(LPAREN); err != nil {
				return instructions.EAExpr{}, err
			}

			base, err := p.want(IDENT)
			if err != nil {
				return instructions.EAExpr{}, err
			}
			if p.accept(COMMA) {
				idxTok, err := p.want(IDENT)
				if err != nil {
					return instructions.EAExpr{}, err
				}
				ix := instructions.EAIndex{}
				if ok, dn := isRegDn(idxTok.Text); ok {
					ix.Reg = dn
				} else if ok, an := isRegAn(idxTok.Text); ok {
					ix.Reg = an
					ix.IsA = true
				} else {
					return instructions.EAExpr{}, errorAtToken(idxTok, fmt.Errorf("expected Dn or An"))
				}
				if p.accept(DOT) {
					szTok, err := p.want(IDENT)
					if err != nil {
						return instructions.EAExpr{}, err
					}
					sz := strings.ToUpper(szTok.Text)
					if sz == "W" {
						ix.Long = false
					} else if sz == "L" {
						ix.Long = true
					}
				}
				if p.accept(STAR) {
					sc, err := p.parseExprUntil(COMMA, RPAREN)
					if err != nil {
						return instructions.EAExpr{}, err
					}
					switch sc {
					case 1:
						// default
					case 2:
						ix.Scale = 2
					case 4:
						ix.Scale = 4
					case 8:
						ix.Scale = 8
					default:
						return instructions.EAExpr{}, fmt.Errorf("invalid scale factor: %d", sc)
					}
				}
				ix.Disp8 = int8(disp)
				if _, err := p.want(RPAREN); err != nil {
					return instructions.EAExpr{}, err
				}
				if ok, an := isRegAn(base.Text); ok {
					return instructions.EAExpr{Kind: instructions.EAkIdxAnBrief, Reg: an, Index: ix}, nil
				}
				if isPC(base.Text) {
					return instructions.EAExpr{Kind: instructions.EAkIdxPCBrief, Index: ix}, nil
				}
				return instructions.EAExpr{}, parserError(base, "base must be An or PC")
			}

			if _, err := p.want(RPAREN); err != nil {
				return instructions.EAExpr{}, err
			}
			if ok, an := isRegAn(base.Text); ok {
				return instructions.EAExpr{Kind: instructions.EAkAddrDisp16, Reg: an, Disp16: int32(disp)}, nil
			}
			if isPC(base.Text) {
				return instructions.EAExpr{Kind: instructions.EAkPCDisp16, Disp16: int32(disp)}, nil
			}
			return instructions.EAExpr{}, parserError(base, "base must be An or PC")
		}
		if ok, dn := isRegDn(t.Text); ok {
			p.next()
			return instructions.EAExpr{Kind: instructions.EAkDn, Reg: dn}, nil
		}
		if ok, an := isRegAn(t.Text); ok {
			p.next()
			return instructions.EAExpr{Kind: instructions.EAkAn, Reg: an}, nil
		}
		if strings.EqualFold(t.Text, "SR") {
			p.next()
			return instructions.EAExpr{Kind: instructions.EAkSR}, nil
		}
		if strings.EqualFold(t.Text, "CCR") {
			p.next()
			return instructions.EAExpr{Kind: instructions.EAkCCR}, nil
		}
		if strings.EqualFold(t.Text, "USP") {
			p.next()
			return instructions.EAExpr{Kind: instructions.EAkUSP}, nil
		}
		// treat bare identifiers as absolute addresses (default long)
		v, err := p.parseExprUntil(DOT, COMMA, NEWLINE, EOF)
		if err != nil {
			return instructions.EAExpr{}, err
		}
		kind := instructions.EAkAbsL
		if p.accept(DOT) {
			suf, err := p.want(IDENT)
			if err != nil {
				return instructions.EAExpr{}, err
			}
			switch strings.ToUpper(suf.Text) {
			case "W":
				kind = instructions.EAkAbsW
			case "L":
				kind = instructions.EAkAbsL
			default:
				return instructions.EAExpr{}, errorAtToken(suf, fmt.Errorf("unknown size suffix .%s", suf.Text))
			}
		}
		if kind == instructions.EAkAbsW {
			return instructions.EAExpr{Kind: kind, Abs16: uint16(v)}, nil
		}
		return instructions.EAExpr{Kind: kind, Abs32: uint32(v)}, nil
	}
	if p.accept(LPAREN) {
		// (An) or (An)+
		if p.peek().Kind == IDENT && p.peekN(2).Kind == RPAREN {
			id := p.next()
			if ok, an := isRegAn(id.Text); ok {
				if _, err := p.want(RPAREN); err != nil {
					return instructions.EAExpr{}, err
				}
				if p.accept(PLUS) {
					return instructions.EAExpr{Kind: instructions.EAkAddrPostinc, Reg: an}, nil
				}
				return instructions.EAExpr{Kind: instructions.EAkAddrInd, Reg: an}, nil
			}
			return instructions.EAExpr{}, parserError(id, "unexpected EA, expected (An) or (disp,An/PC)")
		}
		// (An, Xn...) or (PC, Xn...)
		if p.peek().Kind == IDENT && p.peekN(2).Kind == COMMA {
			base := p.next()
			if !p.accept(COMMA) {
				return instructions.EAExpr{}, parserError(base, "expected ',' after base register")
			}
			idxTok, err := p.want(IDENT)
			if err != nil {
				return instructions.EAExpr{}, err
			}
			var ix instructions.EAIndex
			ix.Disp8 = 0
			if ok, dn := isRegDn(idxTok.Text); ok {
				ix.Reg = dn
			} else if ok, an := isRegAn(idxTok.Text); ok {
				ix.Reg = an
				ix.IsA = true
			} else {
				return instructions.EAExpr{}, parserError(idxTok, "expected Dn or An")
			}
			if p.accept(DOT) {
				szTok, err := p.want(IDENT)
				if err != nil {
					return instructions.EAExpr{}, err
				}
				sz := strings.ToUpper(szTok.Text)
				if sz == "W" {
					ix.Long = false
				} else if sz == "L" {
					ix.Long = true
				}
			}
			if p.accept(STAR) {
				sc, err := p.parseExprUntil(COMMA, RPAREN)
				if err != nil {
					return instructions.EAExpr{}, err
				}
				switch sc {
				case 1:
				case 2:
					ix.Scale = 2
				case 4:
					ix.Scale = 4
				case 8:
					ix.Scale = 8
				default:
					return instructions.EAExpr{}, fmt.Errorf("invalid scale factor: %d", sc)
				}
			}
			if _, err := p.want(RPAREN); err != nil {
				return instructions.EAExpr{}, err
			}
			if ok, an := isRegAn(base.Text); ok {
				return instructions.EAExpr{Kind: instructions.EAkIdxAnBrief, Reg: an, Index: ix}, nil
			}
			if isPC(base.Text) {
				return instructions.EAExpr{Kind: instructions.EAkIdxPCBrief, Index: ix}, nil
			}
			return instructions.EAExpr{}, parserError(base, "base must be An or PC")
		}
		first, err := p.parseExprUntil(COMMA, RPAREN)
		if err != nil {
			return instructions.EAExpr{}, err
		}
		if p.accept(COMMA) {
			base, err := p.want(IDENT)
			if err != nil {
				return instructions.EAExpr{}, err
			}
			if p.accept(COMMA) {
				idxTok, err := p.want(IDENT)
				if err != nil {
					return instructions.EAExpr{}, err
				}
				var ix instructions.EAIndex
				if ok, dn := isRegDn(idxTok.Text); ok {
					ix.Reg = dn
					ix.IsA = false
				} else if ok, an := isRegAn(idxTok.Text); ok {
					ix.Reg = an
					ix.IsA = true
				} else {
					return instructions.EAExpr{}, parserError(idxTok, "lexpected index register Dn/An")
				}
				ix.Long = false
				ix.Scale = 1
				if p.accept(STAR) {
					sc, err := p.parseExprUntil(COMMA, RPAREN)
					if err != nil {
						return instructions.EAExpr{}, err
					}
					switch sc {
					case 1, 2, 4, 8:
						ix.Scale = uint8(sc)
					default:
						// TODO impprove error message
						return instructions.EAExpr{}, fmt.Errorf("invalid scale factor: %d", sc)
					}
				}
				ix.Disp8 = int8(first)
				if _, err := p.want(RPAREN); err != nil {
					return instructions.EAExpr{}, err
				}
				if ok, an := isRegAn(base.Text); ok {
					return instructions.EAExpr{Kind: instructions.EAkIdxAnBrief, Reg: an, Index: ix}, nil
				}
				if isPC(base.Text) {
					return instructions.EAExpr{Kind: instructions.EAkIdxPCBrief, Index: ix}, nil
				}
				return instructions.EAExpr{}, parserError(base, "base must be An or PC")
			}
			if _, err := p.want(RPAREN); err != nil {
				return instructions.EAExpr{}, err
			}
			if ok, an := isRegAn(base.Text); ok {
				return instructions.EAExpr{Kind: instructions.EAkAddrDisp16, Reg: an, Disp16: int32(first)}, nil
			}
			if isPC(base.Text) {
				return instructions.EAExpr{Kind: instructions.EAkPCDisp16, Disp16: int32(first)}, nil
			}
			return instructions.EAExpr{}, parserError(base, "base must be An or PC")
		}
		if _, err := p.want(RPAREN); err != nil {
			return instructions.EAExpr{}, err
		}
		if _, err := p.want(DOT); err != nil {
			return instructions.EAExpr{}, err
		}
		suf, err := p.want(IDENT)
		if err != nil {
			return instructions.EAExpr{}, err
		}
		if strings.EqualFold(suf.Text, "W") {
			return instructions.EAExpr{Kind: instructions.EAkAbsW, Abs16: uint16(first)}, nil
		}
		return instructions.EAExpr{Kind: instructions.EAkAbsL, Abs32: uint32(first)}, nil
	}
	return instructions.EAExpr{}, parserError(t, "unexpected EA")
}

// Lookahead helpers
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
