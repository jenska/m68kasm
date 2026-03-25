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
	prog.SourceLines = append([]string(nil), lines...)
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

	definedLabels := append([]DefinedLabel(nil), p.definedLabels...)
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

func parserError(t Token, msg string) error {
	return errorAtToken(t, fmt.Errorf("%s", msg))
}

// parseLabelDefinition checks for and consumes a label definition (IDENT: or NUMBER:).
// Returns true if a label was parsed.
func (p *Parser) parseLabelDefinition() (bool, error) {
	t := p.peek()
	if (t.Kind == IDENT || t.Kind == NUMBER) && p.peekN(2).Kind == COLON {
		lbl := p.next()
		p.next() // consume ':'
		if lbl.Kind == IDENT {
			p.labels[lbl.Text] = p.pc
			p.recordDefinedLabel(lbl.Text, p.pc)
		} else {
			if err := p.defineLocalLabel(lbl); err != nil {
				return true, err
			}
		}
		return true, nil
	}
	return false, nil
}

func (p *Parser) recordDefinedLabel(name string, addr uint32) {
	if idx, ok := p.definedLabelPos[name]; ok {
		p.definedLabels[idx].Addr = addr
		p.definedLabels[idx].Section = p.section
		return
	}
	p.definedLabelPos[name] = len(p.definedLabels)
	p.definedLabels = append(p.definedLabels, DefinedLabel{Name: name, Addr: addr, Section: p.section})
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

func (p *Parser) parseStmt() (bool, error) {
	t := p.peek()

	if t.Kind == IDENT {
		if def, ok := p.macros[t.Text]; ok {
			return p.invokeMacro(def)
		}
		if p.isConstDefinitionStart() {
			return false, p.parseConstDefinition(t)
		}
		base, suffix := splitMnemonic(t.Text)
		if instrDef := p.instrs.Lookup(base); instrDef != nil {
			return false, p.parseInstruction(instrDef)
		}

		if base == "DC" && suffix != "" {
			_ = p.next()
			return false, parseDC(p, suffix)
		}

		if pseudo, ok := lookupPseudo(t.Text); ok {
			_ = p.next()
			return false, pseudo(p)
		}
		return false, parserError(t, "unknown mnemonic")
	}

	if t.Kind == DOT {
		p.next()
		id, err := p.want(IDENT)
		if err != nil {
			return false, err
		}
		name := "." + strings.ToUpper(id.Text)
		if pseudo, ok := pseudoMap[name]; ok {
			return false, pseudo(p)
		}
		return false, parserError(t, "unknown pseudo op")
	}
	return false, parserError(t, "unexpected token")
}

func (p *Parser) isConstDefinitionStart() bool {
	if p.peekN(2).Kind == EQUAL {
		return true
	}
	return p.peekN(2).Kind == DOT && isEquToken(p.peekN(3))
}

func splitMnemonic(text string) (base, suffix string) {
	upper := strings.ToUpper(text)
	if idx := strings.IndexByte(upper, '.'); idx >= 0 {
		return upper[:idx], upper[idx+1:]
	}
	return upper, ""
}

func lookupPseudo(text string) (func(*Parser) error, bool) {
	pseudo, ok := pseudoMap["."+strings.ToUpper(text)]
	return pseudo, ok
}

func isEquToken(tok Token) bool {
	return tok.Kind == IDENT && strings.EqualFold(tok.Text, "equ")
}

func (p *Parser) parseConstDefinition(nameTok Token) error {
	_ = p.next() // consume name

	if p.peek().Kind == DOT {
		p.next()
		eqTok, err := p.want(IDENT)
		if err != nil {
			return err
		}
		if !isEquToken(eqTok) {
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
	if p.section == SectionBSS {
		return errorAtLine(p.line, fmt.Errorf("instructions are not allowed in %s", p.section.Name()))
	}

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
		ins := &Instr{Def: instrDef, Args: args, PC: p.pc, Line: mn.Line, Col: mn.Col, Section: p.section}
		p.items = append(p.items, ins)
		words, err := instructionWords(&form, args)
		if err != nil {
			return err
		}
		p.pc += uint32(words * 2)
		return nil
	}

	if lastErr != nil {
		return contextualizeAt(mn.Line, mn.Col, lastErr)
	}
	return &Error{Line: mn.Line, Col: mn.Col, Err: fmt.Errorf("no form matches operands")}
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

func (p *Parser) prependTokens(tokens []Token) {
	if len(tokens) == 0 {
		return
	}
	buf := make([]Token, len(tokens)+len(p.buf))
	copy(buf, tokens)
	copy(buf[len(tokens):], p.buf)
	p.buf = buf
}

func (p *Parser) invokeMacro(def macroDef) (bool, error) {
	nameTok := p.next()
	rawArgs := p.consumeUntilEOL()
	defer p.releaseTokens(rawArgs)

	args, err := splitMacroArgs(rawArgs)
	if err != nil {
		return false, errorAtLine(nameTok.Line, err)
	}
	if len(args) != len(def.params) {
		return false, errorAtLine(nameTok.Line, fmt.Errorf("macro %s expects %d args, got %d", nameTok.Text, len(def.params), len(args)))
	}

	if p.macroDepth > 64 {
		return false, errorAtLine(nameTok.Line, fmt.Errorf("macro expansion depth exceeded"))
	}
	p.macroDepth++
	defer func() { p.macroDepth-- }()

	expanded := expandMacroBody(def, args, nameTok)
	p.prependTokens(expanded)
	return true, nil
}

func expandMacroBody(def macroDef, args [][]Token, origin Token) []Token {
	expanded := make([]Token, 0, len(def.body)+len(args))
	for _, t := range def.body {
		if repl, ok := macroArgumentTokens(def.params, args, t.Text); ok && t.Kind == IDENT {
			for _, at := range repl {
				expanded = append(expanded, relocatedToken(at, origin))
			}
			continue
		}
		expanded = append(expanded, relocatedToken(t, origin))
	}
	return expanded
}

func macroArgumentTokens(params []string, args [][]Token, name string) ([]Token, bool) {
	for i, param := range params {
		if name == param {
			return args[i], true
		}
	}
	return nil, false
}

func relocatedToken(tok Token, origin Token) Token {
	tok.Line = origin.Line
	tok.Col = origin.Col
	return tok
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

func withSliceLexer(tokens []Token, line int, scratch []Token) (*sliceLexer, []Token) {
	tmp := append(scratch[:0], tokens...)
	tmp = append(tmp, Token{Kind: EOF, Line: line})
	return &sliceLexer{tokens: tmp}, tmp
}

func (p *Parser) tryParseForm(mn Token, form *instructions.FormDef, tokens []Token) (instructions.Args, error) {
	args := instructions.Args{}
	origLX, origBuf, origLine, origCol := p.lx, p.buf, p.line, p.col
	defer func() {
		p.lx, p.buf, p.line, p.col = origLX, origBuf, origLine, origCol
		p.formScratch = p.formScratch[:0]
	}()

	// isolate parsing to the captured tokens
	var lx *sliceLexer
	lx, p.formScratch = withSliceLexer(tokens, mn.Line, p.formScratch)
	p.lx = lx
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
		special, err := p.parseExpectedSpecialRegister("SR", instructions.EAkSR)
		if err != nil {
			return eaExpr, err
		}
		eaExpr = special

	case instructions.OpkCCR:
		special, err := p.parseExpectedSpecialRegister("CCR", instructions.EAkCCR)
		if err != nil {
			return eaExpr, err
		}
		eaExpr = special

	case instructions.OpkUSP:
		special, err := p.parseExpectedSpecialRegister("USP", instructions.EAkUSP)
		if err != nil {
			return eaExpr, err
		}
		eaExpr = special

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
	p.items = append(p.items, &DataBytes{Bytes: buf, PC: p.pc, Line: p.line, Col: p.col, Section: p.section})
	p.pc += count
	return nil
}

func (p *Parser) setSection(section SectionKind) error {
	if section < p.section {
		return errorAtLine(p.line, fmt.Errorf("sections must stay in .text -> .data -> .bss order"))
	}
	p.section = section
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
	// DBcc instructions (like DBRA) always use a word-sized displacement,
	// but assemblers don't require a ".W" suffix. To ensure the instruction
	// size is calculated correctly, we explicitly set the size to WordSize.
	if len(mn.Text) >= 2 && strings.ToUpper(mn.Text[:2]) == "DB" {
		return instructions.WordSize, nil
	}

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

func parseDirectEAFromIdent(tok Token) (instructions.EAExpr, bool) {
	if ok, dn := isRegDn(tok.Text); ok {
		return instructions.EAExpr{Kind: instructions.EAkDn, Reg: dn}, true
	}
	if ok, an := isRegAn(tok.Text); ok {
		return instructions.EAExpr{Kind: instructions.EAkAn, Reg: an}, true
	}
	return parseSpecialRegisterEA(tok)
}

// parseEA parses an effective address operand. It acts as a dispatcher,
// delegating to more specific parsing functions based on the initial token.
func (p *Parser) parseEA() (instructions.EAExpr, error) {
	t := p.peek()

	// -(An)
	if t.Kind == MINUS && p.peekN(2).Kind == LPAREN {
		return p.parseEAPreDecrement()
	}

	// #imm
	if t.Kind == HASH {
		return p.parseEAImmediate()
	}

	// (An), (An)+, (d,An), (d,PC), etc.
	if t.Kind == LPAREN {
		return p.parseEAIndirect()
	}

	// Dn, An, SR, CCR, USP
	if t.Kind == IDENT {
		if ea, ok := parseDirectEAFromIdent(t); ok {
			p.next()
			return ea, nil
		}
	}

	// If we're here, it must be one of the forms that can start with an
	// expression (a label or a number):
	// - d(An) / d(PC) / d(An,ix) / d(PC,ix)
	// - addr.W / addr.L
	return p.parseEADisplacementOrAbsolute()
}

func parseSpecialRegisterEA(tok Token) (instructions.EAExpr, bool) {
	switch strings.ToUpper(tok.Text) {
	case "SR":
		return instructions.EAExpr{Kind: instructions.EAkSR}, true
	case "CCR":
		return instructions.EAExpr{Kind: instructions.EAkCCR}, true
	case "USP":
		return instructions.EAExpr{Kind: instructions.EAkUSP}, true
	default:
		return instructions.EAExpr{}, false
	}
}

func (p *Parser) parseExpectedSpecialRegister(name string, kind instructions.EAExprKind) (instructions.EAExpr, error) {
	tok, err := p.want(IDENT)
	if err != nil {
		return instructions.EAExpr{}, err
	}
	if !strings.EqualFold(tok.Text, name) {
		return instructions.EAExpr{}, errorAtToken(tok, fmt.Errorf("expected %s", name))
	}
	return instructions.EAExpr{Kind: kind}, nil
}

func parseAbsoluteEA(kind instructions.EAExprKind, value int64) instructions.EAExpr {
	if kind == instructions.EAkAbsW {
		return instructions.EAExpr{Kind: kind, Abs16: uint16(value)}
	}
	return instructions.EAExpr{Kind: kind, Abs32: uint32(value)}
}

func (p *Parser) parseAbsoluteSuffix(defaultKind instructions.EAExprKind, invalidMsg string) (instructions.EAExprKind, error) {
	kind := defaultKind
	if !p.accept(DOT) {
		return kind, nil
	}
	suf, err := p.want(IDENT)
	if err != nil {
		return 0, err
	}
	switch strings.ToUpper(suf.Text) {
	case "W":
		return instructions.EAkAbsW, nil
	case "L":
		return instructions.EAkAbsL, nil
	default:
		return 0, errorAtToken(suf, fmt.Errorf(invalidMsg, suf.Text))
	}
}

func parseEABaseRegister(text string) (reg int, pcRelative bool, ok bool) {
	if ok, an := isRegAn(text); ok {
		return an, false, true
	}
	if isPC(text) {
		return 0, true, true
	}
	return 0, false, false
}

func displacementEA(base Token, disp int64) (instructions.EAExpr, error) {
	if reg, isPC, ok := parseEABaseRegister(base.Text); ok {
		if isPC {
			return instructions.EAExpr{Kind: instructions.EAkPCDisp16, Disp16: int32(disp)}, nil
		}
		return instructions.EAExpr{Kind: instructions.EAkAddrDisp16, Reg: reg, Disp16: int32(disp)}, nil
	}
	return instructions.EAExpr{}, parserError(base, "base must be An or PC for displacement addressing")
}

func indexedEA(base Token, disp int64, ix instructions.EAIndex) (instructions.EAExpr, error) {
	ix.Disp8 = int8(disp)
	if reg, isPC, ok := parseEABaseRegister(base.Text); ok {
		if isPC {
			return instructions.EAExpr{Kind: instructions.EAkIdxPCBrief, Index: ix}, nil
		}
		return instructions.EAExpr{Kind: instructions.EAkIdxAnBrief, Reg: reg, Index: ix}, nil
	}
	return instructions.EAExpr{}, parserError(base, "base must be An or PC for indexed addressing")
}

// ---------- EA parsing helpers ----------

func (p *Parser) parseEAPreDecrement() (instructions.EAExpr, error) {
	p.next() // '-'
	p.next() // '('
	areg, err := p.want(IDENT)
	if err != nil {
		return instructions.EAExpr{}, err
	}
	ok, an := isRegAn(areg.Text)
	if !ok {
		return instructions.EAExpr{}, parserError(areg, "expected address register")
	}
	if _, err := p.want(RPAREN); err != nil {
		return instructions.EAExpr{}, err
	}
	return instructions.EAExpr{Kind: instructions.EAkAddrPredec, Reg: an}, nil
}

func (p *Parser) parseEAImmediate() (instructions.EAExpr, error) {
	p.next() // '#'
	v, err := p.parseExpr()
	if err != nil {
		return instructions.EAExpr{}, err
	}
	return instructions.EAExpr{Kind: instructions.EAkImm, Imm: v}, nil
}

func (p *Parser) parseEADisplacementOrAbsolute() (instructions.EAExpr, error) {
	// This handles two cases that can start with an expression:
	// 1. Displacement modes: d(An), d(PC), d(An,ix), d(PC,ix)
	// 2. Absolute modes: addr.W, addr.L
	expr, err := p.parseExprUntil(LPAREN, DOT, COMMA, NEWLINE, EOF)
	if err != nil {
		return instructions.EAExpr{}, err
	}

	// Case 1: Displacement modes, identified by a following '('.
	if p.accept(LPAREN) {
		return p.parseEADisplacementBody(expr)
	}

	kind, err := p.parseAbsoluteSuffix(instructions.EAkAbsL, "unknown size suffix .%s")
	if err != nil {
		return instructions.EAExpr{}, err
	}
	return parseAbsoluteEA(kind, expr), nil
}

func (p *Parser) parseEAIndirect() (instructions.EAExpr, error) {
	p.next() // consume '('

	// Case 1: (An) or (An)+
	if id := p.peek(); id.Kind == IDENT && p.peekN(2).Kind == RPAREN {
		if ok, an := isRegAn(id.Text); ok {
			p.next() // id
			p.next() // ')'
			if p.accept(PLUS) {
				return instructions.EAExpr{Kind: instructions.EAkAddrPostinc, Reg: an}, nil
			}
			return instructions.EAExpr{Kind: instructions.EAkAddrInd, Reg: an}, nil
		} else if ok, _ := isRegDn(id.Text); ok {
			return instructions.EAExpr{}, parserError(id, "data register not allowed in indirect addressing (expected An)")
		}
	}

	// Case 2: (An, ix) or (PC, ix) -- no outer displacement
	if base := p.peek(); base.Kind == IDENT && p.peekN(2).Kind == COMMA {
		if reg, isPC, ok := parseEABaseRegister(base.Text); ok {
			p.next() // base
			p.next() // ','
			ix, err := p.parseEAIndex()
			if err != nil {
				return instructions.EAExpr{}, err
			}
			if _, err := p.want(RPAREN); err != nil {
				return instructions.EAExpr{}, err
			}
			if isPC {
				return instructions.EAExpr{Kind: instructions.EAkIdxPCBrief, Index: ix}, nil
			}
			return instructions.EAExpr{Kind: instructions.EAkIdxAnBrief, Reg: reg, Index: ix}, nil
		}
	}

	// Case 3: (disp, ...), (abs).W, or (abs).L
	expr, err := p.parseExprUntil(COMMA, RPAREN)
	if err != nil {
		return instructions.EAExpr{}, err
	}

	// Subcase 3a: (disp, An/PC) or (disp, An/PC, ix)
	if p.accept(COMMA) {
		return p.parseEADisplacementBody(expr)
	}

	// Subcase 3b: (abs).W or (abs).L
	if _, err := p.want(RPAREN); err != nil {
		return instructions.EAExpr{}, err
	}
	kind, err := p.parseAbsoluteSuffix(0, "expected .W or .L after (absolute address)")
	if err == nil && kind != 0 {
		return parseAbsoluteEA(kind, expr), nil
	}
	return instructions.EAExpr{}, errorAtLine(p.line, fmt.Errorf("invalid effective address form, expected (abs).W or (abs).L"))
}

func (p *Parser) parseEADisplacementBody(disp int64) (instructions.EAExpr, error) {
	// We are inside the parentheses of d(...) or (d,...)
	base, err := p.want(IDENT)
	if err != nil {
		return instructions.EAExpr{}, err
	}

	// Is it an indexed mode, d(An,ix) or d(PC,ix)?
	if p.accept(COMMA) {
		ix, err := p.parseEAIndex()
		if err != nil {
			return instructions.EAExpr{}, err
		}
		if _, err := p.want(RPAREN); err != nil {
			return instructions.EAExpr{}, err
		}
		return indexedEA(base, disp, ix)
	}

	// It's a simple displacement mode: d(An) or d(PC)
	if _, err := p.want(RPAREN); err != nil {
		return instructions.EAExpr{}, err
	}
	return displacementEA(base, disp)
}

func (p *Parser) parseEAIndex() (instructions.EAIndex, error) {
	idxTok, err := p.want(IDENT)
	if err != nil {
		return instructions.EAIndex{}, err
	}

	// Handle embedded size suffix (e.g. D0.L) which the lexer consumes as one IDENT
	name := idxTok.Text
	var suffix string
	if idx := strings.IndexByte(name, '.'); idx >= 0 {
		suffix = name[idx+1:]
		name = name[:idx]
	}

	ix, err := parseIndexRegister(name)
	if err != nil {
		return ix, parserError(idxTok, err.Error())
	}

	if suffix != "" {
		long, err := parseIndexSizeSuffix(suffix)
		if err != nil {
			return ix, parserError(idxTok, err.Error())
		}
		ix.Long = long
	} else if p.accept(DOT) {
		szTok, err := p.want(IDENT)
		if err != nil {
			return ix, err
		}
		long, err := parseIndexSizeSuffix(szTok.Text)
		if err != nil {
			return ix, parserError(szTok, err.Error())
		}
		ix.Long = long
	}

	// Optional scale factor *1, *2, *4, *8
	if p.accept(STAR) {
		sc, err := p.parseExprUntil(COMMA, RPAREN)
		if err != nil {
			return ix, err
		}
		switch sc {
		case 1, 2, 4, 8:
			ix.Scale = uint8(sc)
		default:
			return ix, fmt.Errorf("invalid scale factor: %d", sc)
		}
	} else {
		ix.Scale = 1 // Default scale
	}

	return ix, nil
}

func parseIndexRegister(name string) (instructions.EAIndex, error) {
	var ix instructions.EAIndex
	if ok, dn := isRegDn(name); ok {
		ix.Reg = dn
		return ix, nil
	}
	if ok, an := isRegAn(name); ok {
		ix.Reg = an
		ix.IsA = true
		return ix, nil
	}
	return ix, fmt.Errorf("expected Dn or An as index register")
}

func parseIndexSizeSuffix(suffix string) (bool, error) {
	switch strings.ToUpper(suffix) {
	case "W":
		return false, nil
	case "L":
		return true, nil
	default:
		return false, fmt.Errorf("expected .W or .L for index register size")
	}
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
