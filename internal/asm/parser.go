package asm

import (
	"fmt"
	"io"
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
		lx     lexer
		labels map[string]uint32
		pc     uint32
		origin uint32
		hasOrg bool
		items  []any
		line   int
		col    int

		buf []Token // N-Token Lookahead
	}

	sliceLexer struct {
		tokens []Token
		pos    int
	}
)

func (s *sliceLexer) Next() Token {
	if s.pos >= len(s.tokens) {
		return Token{Kind: EOF}
	}
	t := s.tokens[s.pos]
	s.pos++
	return t
}

func Parse(r io.Reader) (*Program, error) {
	p := &Parser{lx: NewLexer(r), labels: map[string]uint32{}}
	for {
		t := p.peek()
		if t.Kind == EOF {
			break
		}
		if t.Kind == NEWLINE {
			p.next()
			continue
		}

		// Label? IDENT ':'
		if t.Kind == IDENT && p.peekN(2).Kind == COLON {
			lbl := p.next() // IDENT
			p.next()        // ':'
			p.labels[lbl.Text] = p.pc
			nt := p.peek()
			if nt.Kind == NEWLINE || nt.Kind == EOF {
				continue
			}
		}

		if err := p.parseStmt(); err != nil {
			return nil, err
		}

		if t := p.peek(); t.Kind != NEWLINE && t.Kind != EOF {
			return nil, fmt.Errorf("line %d: unexpected token: %s", t.Line, t.Text)
		}
		if p.peek().Kind == NEWLINE {
			p.next()
		}
	}

	origin := p.origin
	if !p.hasOrg {
		origin = 0
	}

	return &Program{Items: p.items, Labels: p.labels, Origin: origin}, nil
}

func ParseFile(path string) (*Program, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return Parse(f)
}

func parserError(t Token, msg string) error {
	return fmt.Errorf("%s: %s", t.String(), msg)
}

func (p *Parser) parseStmt() error {
	t := p.peek()

	if t.Kind == IDENT {
		s := strings.ToUpper(t.Text)
		if idx := strings.IndexRune(s, '.'); idx > 0 {
			s = s[:idx]
		}
		if instrDef, ok := instructions.Instructions[s]; ok {
			return p.parseInstruction(instrDef)
		}
		if kw := kwOf(t.Text); kw != KW_NONE {
			p.next()
			return p.parsePseudo(kw)
		}
		return parserError(t, "unknown mnemonic")
	}

	if t.Kind == DOT {
		p.next()
		id, err := p.want(IDENT)
		if err != nil {
			return err
		}
		name := "." + id.Text
		if kw := kwOf(name); kw != KW_NONE {
			return p.parsePseudo(kw)
		}
		return parserError(t, "unknown pseudo op")
	}
	return parserError(t, "unexpected token")
}

func (p *Parser) parsePseudo(kw kw) error {
	switch kw {
	case KW_ORG:
		return p.parseORG()
	case KW_BYTE:
		return p.parseBYTE()
	case KW_WORD:
		return p.parseWORD()
	case KW_LONG:
		return p.parseLONG()
	case KW_ALIGN:
		return p.parseALIGN()
	default:
		return fmt.Errorf("line %d: unknown pseudo op", p.line)
	}
}

func (p *Parser) parseORG() error {
	val, err := p.parseExpr()
	if err != nil {
		return err
	}
	newPC := uint32(val)

	if newPC > maxProgramSize {
		return fmt.Errorf("line %d: .org would exceed maximum program size of %d bytes", p.line, maxProgramSize)
	}

	if !p.hasOrg && p.pc == 0 {
		p.origin = newPC
		p.hasOrg = true
		p.pc = newPC
		return nil
	}
	if !p.hasOrg {
		p.origin = 0
		p.hasOrg = true
	}

	if newPC < p.pc {
		return fmt.Errorf("line %d: .org cannot move backwards (pc=%d -> %d)", p.line, p.pc, newPC)
	}

	if newPC > p.pc {
		if err := p.emitPaddingBytes(newPC-p.pc, 0x00); err != nil {
			return err
		}
		return nil
	}

	return nil
}

func (p *Parser) parseBYTE() error {
	val, err := p.parseExpr()
	if err != nil {
		return err
	}
	var bytes []byte
	bytes = append(bytes, byte(val))
	for p.accept(COMMA) {
		v, err := p.parseExpr()
		if err != nil {
			return err
		}
		bytes = append(bytes, byte(v))
	}
	p.items = append(p.items, &DataBytes{Bytes: bytes, PC: p.pc, Line: p.line})
	p.pc += uint32(len(bytes))
	return nil
}

// .word <expr>[, <expr>]...
func (p *Parser) parseWORD() error {
	// first value
	v, err := p.parseExpr()
	if err != nil {
		return err
	}

	out := make([]byte, 0, 2*4)
	if v < -0x8000 || v > 0xFFFF {
		return fmt.Errorf("(%d, %d): .word value out of 16-bit range: %d", p.line, p.col, v)
	}
	w := uint16(int16(v))
	out = append(out, byte(w>>8), byte(w))

	// subsequent values
	for p.accept(COMMA) { // use lex.COMMA if you still prefix tokens
		v, err := p.parseExpr()
		if err != nil {
			return err
		}
		if v < -0x8000 || v > 0xFFFF {
			return fmt.Errorf("(%d, %d): .word value out of 16-bit range: %d", p.line, p.col, v)
		}
		w := uint16(int16(v))
		out = append(out, byte(w>>8), byte(w))
	}

	p.items = append(p.items, &DataBytes{Bytes: out, PC: p.pc, Line: p.line})
	p.pc += uint32(len(out))
	return nil
}

// .long <expr>[, <expr>]...
func (p *Parser) parseLONG() error {
	v, err := p.parseExpr()
	if err != nil {
		return err
	}

	out := make([]byte, 0, 4*4)
	if v < -0x80000000 || v > 0xFFFFFFFF {
		return fmt.Errorf("(%d, %d): .long value out of range: %d", p.line, p.col, v)
	}
	u := uint32(v) // two's complement when v is negative
	out = append(out, byte(u>>24), byte(u>>16), byte(u>>8), byte(u))

	for p.accept(COMMA) {
		v, err := p.parseExpr()
		if err != nil {
			return err
		}
		if v < -0x80000000 || v > 0xFFFFFFFF {
			return fmt.Errorf("(%d, %d): .long value out of range: %d", p.line, p.col, v)
		}
		u := uint32(v)
		out = append(out, byte(u>>24), byte(u>>16), byte(u>>8), byte(u))
	}

	p.items = append(p.items, &DataBytes{Bytes: out, PC: p.pc, Line: p.line})
	p.pc += uint32(len(out))
	return nil
}

func (p *Parser) parseInstruction(instrDef *instructions.InstrDef) error {
	mn, err := p.want(IDENT)
	if err != nil {
		return err
	}

	operandTokens := p.consumeUntilEOL()
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
		return lastErr
	}
	return fmt.Errorf("line %d: no form matches operands", mn.Line)
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
			case instructions.T_SrcEAExt:
				words += len(srcEA.Ext)
			case instructions.T_DstEAExt:
				words += len(dstEA.Ext)
			case instructions.T_ImmSized:
				words++
			case instructions.T_SrcImm:
				if args.Src.Kind == instructions.EAkImm {
					switch args.Size {
					case instructions.SZ_L:
						words += 2
					default:
						words++
					}
				}
			case instructions.T_BranchWordIfNeeded:
				if args.Size == instructions.SZ_W {
					words++
				}
			case instructions.T_SrcRegMask, instructions.T_DstRegMask:
				words++
			}
		}
	}

	return words, nil
}

func (p *Parser) consumeUntilEOL() []Token {
	tokens := make([]Token, 0, len(p.buf))
	for {
		t := p.peek()
		if t.Kind == NEWLINE || t.Kind == EOF {
			break
		}
		tokens = append(tokens, p.next())
	}
	return tokens
}

func (p *Parser) tryParseForm(mn Token, form *instructions.FormDef, tokens []Token) (instructions.Args, error) {
	args := instructions.Args{}
	origLX, origBuf, origLine, origCol := p.lx, p.buf, p.line, p.col
	defer func() {
		p.lx, p.buf, p.line, p.col = origLX, origBuf, origLine, origCol
	}()

	// isolate parsing to the captured tokens
	tmp := make([]Token, 0, len(tokens)+1)
	tmp = append(tmp, tokens...)
	tmp = append(tmp, Token{Kind: EOF, Line: mn.Line})
	p.lx = &sliceLexer{tokens: tmp}
	p.buf = nil

	sz, err := p.parseSizeSpec(mn, form.DefaultSize, form.Sizes)
	if err != nil {
		return args, err
	}
	args.Size = sz

	for i, operandKind := range form.OperKinds {
		var eaExpr instructions.EAExpr
		if i == 1 {
			if _, err := p.want(COMMA); err != nil {
				return args, err
			}
		}

		switch operandKind {
		case instructions.OPK_Imm:
			if _, err := p.want(HASH); err != nil {
				return args, err
			}
			imm, err := p.parseExpr()
			if err != nil {
				return args, err
			}
			eaExpr.Kind = instructions.EAkImm
			eaExpr.Imm = imm

		case instructions.OPK_ImmQuick:
			if _, err := p.want(HASH); err != nil {
				return args, err
			}
			imm, err := p.parseExpr()
			if err != nil {
				return args, err
			}
			eaExpr.Kind = instructions.EAkNone
			eaExpr.Imm = imm
			args.HasImmQuick = true

		case instructions.OPK_Dn:
			dreg, err := p.want(IDENT)
			if err != nil {
				return args, err
			}
			ok, dn := isRegDn(dreg.Text)
			if !ok {
				return args, fmt.Errorf("line %d: expected Dn, got %s", dreg.Line, dreg.Text)
			}
			eaExpr.Kind = instructions.EAkDn
			eaExpr.Reg = dn

		case instructions.OPK_An:
			areg, err := p.want(IDENT)
			if err != nil {
				return args, err
			}
			ok, an := isRegAn(areg.Text)
			if !ok {
				return args, fmt.Errorf("line %d: expected Dn, got %s", areg.Line, areg.Text)
			}
			eaExpr.Kind = instructions.EAkAn
			eaExpr.Reg = an

		case instructions.OPK_EA:
			ea, err := p.parseEA()
			if err != nil {
				return args, err
			}
			eaExpr = ea
		case instructions.OPK_PredecAn:
			ea, err := p.parseEA()
			if err != nil {
				return args, err
			}
			if ea.Kind != instructions.EAkAddrPredec {
				return args, fmt.Errorf("line %d: expected -(An)", mn.Line)
			}
			eaExpr = ea
		case instructions.OPK_RegList:
			mask, err := p.parseRegList()
			if err != nil {
				return args, err
			}
			eaExpr.Kind = instructions.EAkNone
			if i == 0 {
				args.RegMaskSrc = mask
			} else {
				args.RegMaskDst = mask
			}

		case instructions.OPK_DispRel:
			lbl, err := p.want(IDENT)
			if err != nil {
				return args, err
			}
			args.Target = lbl.Text

		default:
			return args, fmt.Errorf("line %d: unknown identifier %s", mn.Line, mn.Text)
		}
		if i == 0 {
			args.Src = eaExpr
		} else {
			args.Dst = eaExpr
		}
	}

	if trailing := p.peek(); trailing.Kind != EOF {
		return args, fmt.Errorf("line %d: unexpected token %s", trailing.Line, trailing.Text)
	}

	return args, nil
}

// emitPaddingBytes centralizes emitting zero/filled padding while keeping the
// generated output within a safe upper bound. Without this guard, a crafted
// .org/.align could request gigabytes of padding and exhaust memory.
func (p *Parser) emitPaddingBytes(count uint32, fill byte) error {
	if count == 0 {
		return nil
	}
	if count > maxProgramSize || p.pc > maxProgramSize-count {
		return fmt.Errorf("line %d: padding would exceed maximum program size of %d bytes", p.line, maxProgramSize)
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
		return instructions.SZ_B, true
	case "s":
		return instructions.SZ_B, true
	case "w":
		return instructions.SZ_W, true
	case "l":
		return instructions.SZ_L, true
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
				return 0, fmt.Errorf("line %d: register ranges must stay within D or A registers", toTok.Line)
			}
			if endReg < reg {
				return 0, fmt.Errorf("line %d: descending ranges are not allowed", toTok.Line)
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
	return false, 0, fmt.Errorf("line %d: expected register in list", tok.Line)
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
			return instructions.EAExpr{}, fmt.Errorf("line %d: expected address register", areg.Line)
		}
		if _, err := p.want(RPAREN); err != nil {
			return instructions.EAExpr{}, err
		}
		ok, an := isRegAn(areg.Text)
		if !ok {
			return instructions.EAExpr{}, fmt.Errorf("line %d: expected address register", areg.Line)
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
	if t.Kind == IDENT {
		if ok, dn := isRegDn(t.Text); ok {
			p.next()
			return instructions.EAExpr{Kind: instructions.EAkDn, Reg: dn}, nil
		}
		if ok, an := isRegAn(t.Text); ok {
			p.next()
			return instructions.EAExpr{Kind: instructions.EAkAn, Reg: an}, nil
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
				return instructions.EAExpr{}, fmt.Errorf("line %d: unknown size suffix .%s", suf.Line, suf.Text)
			}
		}
		if kind == instructions.EAkAbsW {
			return instructions.EAExpr{Kind: kind, Abs16: uint16(v)}, nil
		}
		return instructions.EAExpr{Kind: kind, Abs32: uint32(v)}, nil
	}
	if p.accept(LPAREN) {
		// (An)
		if p.peek().Kind == IDENT {
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

// .align <expr>[, <fill>]
func (p *Parser) parseALIGN() error {
	// alignment value
	val, err := p.parseExpr()
	if err != nil {
		return err
	}
	if val < 1 {
		return fmt.Errorf("line %d: .align expects value >= 1, got %d", p.line, val)
	}
	align := uint32(val)

	// optional fill
	fill := byte(0x00)
	if p.accept(COMMA) { // if you still use prefixed tokens, use lex.COMMA
		fv, err := p.parseExpr()
		if err != nil {
			return err
		}
		fill = byte(fv) // truncate to 8-bit
	}

	// compute padding
	m := p.pc % align
	if m == 0 {
		return nil // already aligned, emit nothing
	}
	pad := align - m

	// emit pad bytes of 'fill'
	return p.emitPaddingBytes(pad, fill)
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
		return t, fmt.Errorf("line %d col %d: expected %v, got %v (%q)", t.Line, t.Col, k, t.Kind, t.Text)
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
