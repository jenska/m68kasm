package asm

import (
	"fmt"
	"strings"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

func parseDirectEAFromIdent(tok Token) (instructions.EAExpr, bool) {
	if ok, dn := isRegDn(tok.Text); ok {
		return instructions.EAExpr{Kind: instructions.EAkDn, Reg: dn}, true
	}
	if ok, an := isRegAn(tok.Text); ok {
		return instructions.EAExpr{Kind: instructions.EAkAn, Reg: an}, true
	}
	return parseSpecialRegisterEA(tok)
}

// parseEA parses an effective address operand, then an optional 68020
// bit-field specifier suffix ("{offset:width}") on it — valid on any EA
// (including Dn), so it's checked once here rather than in every one of
// parseEABase's sub-paths.
func (p *Parser) parseEA() (instructions.EAExpr, error) {
	ea, err := p.parseEABase()
	if err != nil {
		return ea, err
	}
	if p.peek().Kind == LBRACE {
		return p.parseBitFieldSuffix(ea)
	}
	return ea, nil
}

// parseEABase is a dispatcher, delegating to more specific parsing
// functions based on the initial token.
func (p *Parser) parseEABase() (instructions.EAExpr, error) {
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

// parseBitFieldSuffix parses "{offset:width}" and attaches it to base,
// which parseEA has already parsed as the bit field's own addressing
// mode (a plain register or memory location — this suffix only adds
// which bits of it are the field, not a new EA kind).
func (p *Parser) parseBitFieldSuffix(base instructions.EAExpr) (instructions.EAExpr, error) {
	p.next() // consume '{'

	offIsReg, offVal, err := p.parseBitFieldSpec()
	if err != nil {
		return base, err
	}
	if !offIsReg && (offVal < 0 || offVal > 31) {
		return base, fmt.Errorf("bit-field offset out of range: %d (expected 0-31 or a data register)", offVal)
	}
	if _, err := p.want(COLON); err != nil {
		return base, err
	}
	widIsReg, widVal, err := p.parseBitFieldSpec()
	if err != nil {
		return base, err
	}
	if !widIsReg && (widVal < 1 || widVal > 32) {
		return base, fmt.Errorf("bit-field width out of range: %d (expected 1-32 or a data register)", widVal)
	}
	if _, err := p.want(RBRACE); err != nil {
		return base, err
	}

	base.HasBitField = true
	base.BFOffsetIsReg, base.BFOffsetVal = offIsReg, offVal
	base.BFWidthIsReg, base.BFWidthVal = widIsReg, widVal
	return base, nil
}

// parseBitFieldSpec parses one half (offset or width) of a bit-field
// specifier: a bare Dn register, or a plain expression — no leading '#'
// unlike every other immediate in this codebase's syntax, matching the
// standard 68020 assembler convention for this specific construct.
func (p *Parser) parseBitFieldSpec() (isReg bool, val int32, err error) {
	if t := p.peek(); t.Kind == IDENT {
		if ok, dn := isRegDn(t.Text); ok {
			p.next()
			return true, int32(dn), nil
		}
	}
	v, err := p.parseExprUntil(COLON, RBRACE)
	if err != nil {
		return false, 0, err
	}
	return false, int32(v), nil
}

// controlRegisterKind maps a MOVEC control-register name to its
// EAExprKind. Only the registers introduced by the 68010 (SFC, DFC, USP,
// VBR) are recognized here; 68020+ added several more that belong to a
// later milestone (see docs/design/cpu-family-support.md).
func controlRegisterKind(name string) (instructions.EAExprKind, bool) {
	switch strings.ToUpper(name) {
	case "SFC":
		return instructions.EAkSFC, true
	case "DFC":
		return instructions.EAkDFC, true
	case "USP":
		return instructions.EAkUSP, true
	case "VBR":
		return instructions.EAkVBR, true
	default:
		return 0, false
	}
}

// parseFPRegister recognizes an FPU data register name, FP0-FP7.
func parseFPRegister(s string) (int, bool) {
	if len(s) == 3 && (s[0] == 'F' || s[0] == 'f') && (s[1] == 'P' || s[1] == 'p') && s[2] >= '0' && s[2] <= '7' {
		return int(s[2] - '0'), true
	}
	return 0, false
}

// parseRegPair parses DIVSL/DIVUL's "Dr:Dq" (or bare "Dq", the
// 32-bit-dividend shorthand — Dr defaults to the same register). Real
// 68020 assembler syntax for this construct has no leading '#', and
// this codebase's own convention is the colon order matching Motorola's
// documented bit layout: Dr (remainder, or the high word of a 64-bit
// dividend) is written first, Dq (quotient, or the low word) second.
func (p *Parser) parseRegPair() (instructions.EAExpr, error) {
	firstTok, err := p.want(IDENT)
	if err != nil {
		return instructions.EAExpr{}, err
	}
	ok, first := isRegDn(firstTok.Text)
	if !ok {
		return instructions.EAExpr{}, errorAtToken(firstTok, fmt.Errorf("expected a data register, got %s", firstTok.Text))
	}

	e := instructions.EAExpr{Kind: instructions.EAkRegPair, Reg: first, Reg2: first}
	if p.accept(COLON) {
		secondTok, err := p.want(IDENT)
		if err != nil {
			return e, err
		}
		ok, second := isRegDn(secondTok.Text)
		if !ok {
			return e, errorAtToken(secondTok, fmt.Errorf("expected a data register, got %s", secondTok.Text))
		}
		e.Reg2 = second
		e.RegPairWide = true
	}
	return e, nil
}

// parseAnIndPair parses CAS2's "(Rn1):(Rn2)" memory-pointer pair: two
// parenthesized address registers joined by a colon, both mandatory —
// unlike parseRegPair, there is no meaningful single-pointer shorthand
// for a dual compare-and-swap.
func (p *Parser) parseAnIndPair() (instructions.EAExpr, error) {
	first, err := p.parseAnIndirectRegister()
	if err != nil {
		return instructions.EAExpr{}, err
	}
	if _, err := p.want(COLON); err != nil {
		return instructions.EAExpr{}, err
	}
	second, err := p.parseAnIndirectRegister()
	if err != nil {
		return instructions.EAExpr{}, err
	}
	return instructions.EAExpr{Kind: instructions.EAkAnIndPair, Reg: first, Reg2: second}, nil
}

func (p *Parser) parseAnIndirectRegister() (int, error) {
	if _, err := p.want(LPAREN); err != nil {
		return 0, err
	}
	tok, err := p.want(IDENT)
	if err != nil {
		return 0, err
	}
	ok, an := isRegAn(tok.Text)
	if !ok {
		return 0, errorAtToken(tok, fmt.Errorf("expected An, got %s", tok.Text))
	}
	if _, err := p.want(RPAREN); err != nil {
		return 0, err
	}
	return an, nil
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
	if disp < -128 || disp > 127 {
		return instructions.EAExpr{}, parserError(base, fmt.Sprintf("displacement out of range for indexed addressing: %d (brief indexed addressing is limited to a signed byte on every 68k CPU)", disp))
	}
	ix.Disp8 = int8(disp)
	if reg, isPC, ok := parseEABaseRegister(base.Text); ok {
		if isPC {
			return instructions.EAExpr{Kind: instructions.EAkIdxPCBrief, Index: ix}, nil
		}
		return instructions.EAExpr{Kind: instructions.EAkIdxAnBrief, Reg: reg, Index: ix}, nil
	}
	return instructions.EAExpr{}, parserError(base, "base must be An or PC for indexed addressing")
}

// splitSizedSymbol splits a bare label token's text on a trailing ".w"
// or ".l" (case-insensitive), e.g. "mylabel.l" -> ("mylabel", "l", true).
// The lexer's identifier scanner treats '.' as a normal identifier
// character (the same reason stripRegSizeSuffix exists for registers
// like "D0.L"), so by the time a token reaches the parser, a suffix
// written directly against a label name is already part of one merged
// IDENT token rather than a separate DOT token — this recovers it.
func splitSizedSymbol(s string) (name, size string, ok bool) {
	idx := strings.LastIndexByte(s, '.')
	if idx < 0 {
		return s, "", false
	}
	suf := strings.ToLower(s[idx+1:])
	if suf != "w" && suf != "l" {
		return s, "", false
	}
	return s[:idx], suf, true
}

// parseSizedDisp parses a 68020+ base/outer displacement (stopping at
// any of stops) and returns its value together with the DispSize to
// encode it at. A symbol-referencing expression MUST carry an explicit
// .w/.l suffix: a symbol's resolved value can differ between the
// assembler's two passes (it may be a forward reference), and the
// encoded length of an operand can never be allowed to depend on a value
// that might still change. Only a bare "label.w"/"label.l" is supported
// for this — splitSizedSymbol recovers the merged suffix directly,
// without going through the general expression evaluator at all, so a
// symbol used in a larger compound expression (e.g. "label+4") is not
// supported here and is rejected the same as an unsuffixed one. A pure
// constant defaults to DispNull when it is exactly 0, or DispLong
// otherwise (mirroring how absolute addressing already defaults to .L
// when unsuffixed); an explicit suffix on a constant is always honored
// as written, and range-checked for .w.
func (p *Parser) parseSizedDisp(stops ...Kind) (int32, instructions.DispSize, error) {
	if t := p.peek(); t.Kind == IDENT {
		if name, sz, ok := splitSizedSymbol(t.Text); ok {
			p.next()
			addr, hasAddr := p.labels[name]
			if !hasAddr {
				if !p.allowForwardRefs {
					return 0, 0, fmt.Errorf("undefined label in expression: %s", name)
				}
				addr = 0
			}
			if sz == "w" {
				return int32(addr), instructions.DispWord, nil
			}
			return int32(addr), instructions.DispLong, nil
		}
	}

	expr, err := p.parseExprInfoUntil(stops...)
	if err != nil {
		return 0, 0, err
	}
	if expr.HasSymbol {
		return 0, 0, fmt.Errorf("displacement referencing a symbol requires an explicit .w or .l size (e.g. label.l)")
	}

	if p.accept(DOT) {
		szTok, err := p.want(IDENT)
		if err != nil {
			return 0, 0, err
		}
		switch strings.ToUpper(szTok.Text) {
		case "W":
			if expr.Value < -32768 || expr.Value > 32767 {
				return 0, 0, fmt.Errorf("displacement out of range for .w: %d", expr.Value)
			}
			return int32(expr.Value), instructions.DispWord, nil
		case "L":
			return int32(expr.Value), instructions.DispLong, nil
		default:
			return 0, 0, fmt.Errorf("expected .w or .l, got .%s", szTok.Text)
		}
	}

	if expr.Value == 0 {
		return 0, instructions.DispNull, nil
	}
	return int32(expr.Value), instructions.DispLong, nil
}

func (p *Parser) pcRelativeDisp(expr exprInfo, min, max int64) (int64, error) {
	disp := expr.Value
	if expr.HasSymbol {
		// 68000 PC-relative addressing uses the extension-word address as the base.
		disp -= int64(p.pc) + 2
	}
	if disp < min || disp > max {
		return 0, errorAtLine(p.line, fmt.Errorf("PC-relative displacement out of range: %d", disp))
	}
	return disp, nil
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
	expr, err := p.parseExprInfoUntil(LPAREN, DOT, COMMA, NEWLINE, EOF)
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
	return parseAbsoluteEA(kind, expr.Value), nil
}

func (p *Parser) parseEAIndirect() (instructions.EAExpr, error) {
	p.next() // consume '('

	// Case 0: 68020+ memory-indirect addressing, ([bd,An],Xn,od) etc.
	// Deliberately gated on CPU68020 exactly, not CPU32: GNU binutils'
	// gas/config/tc-m68k.c explicitly rejects this addressing mode on
	// cpu32 ("needs 68020 or higher"), unlike scale factors below.
	if p.peek().Kind == LBRACKET {
		if !p.target.Supports(instructions.Target{CPU: instructions.CPU68020}) {
			return instructions.EAExpr{}, fmt.Errorf("memory-indirect addressing requires 68020 or later (target is %s)", p.target.CPU)
		}
		return p.parseMemIndirectEA()
	}

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
	expr, err := p.parseExprInfoUntil(COMMA, RPAREN)
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
		return parseAbsoluteEA(kind, expr.Value), nil
	}
	return instructions.EAExpr{}, errorAtLine(p.line, fmt.Errorf("invalid effective address form, expected (abs).W or (abs).L"))
}

// looksLikeIndexRegister reports whether tok could start an index
// register (Dn/An, optionally with an inline .W/.L suffix like "D0.L"),
// used to disambiguate a pre-indexed memory-indirect EA's optional
// ",Xn" from its optional ",od" — both start with a comma, and only the
// token's own text distinguishes them (od is itself a general
// expression, which can start with any label name).
func looksLikeIndexRegister(tok Token) bool {
	if tok.Kind != IDENT {
		return false
	}
	if ok, _ := isRegDn(tok.Text); ok {
		return true
	}
	ok, _ := isRegAn(tok.Text)
	return ok
}

// parseMemIndirectEA parses 68020+ memory-indirect addressing, entered
// once parseEAIndirect sees '[' right after the EA's opening '(':
//
//	([bd,An],Xn,od)  / ([bd,An],od)  / ([bd,An])   -- pre-indexed, Xn optional
//	([bd,An,Xn],od)  / ([bd,An,Xn])                -- post-indexed, Xn required
//
// and their PC-relative equivalents (An replaced by PC). bd must always
// be written explicitly (0 for "no base displacement"); od may be
// omitted entirely, meaning 0/null. Gating on the target CPU is the
// caller's responsibility.
func (p *Parser) parseMemIndirectEA() (instructions.EAExpr, error) {
	p.next() // consume '['

	bd, bdSize, err := p.parseSizedDisp(COMMA)
	if err != nil {
		return instructions.EAExpr{}, err
	}
	if _, err := p.want(COMMA); err != nil {
		return instructions.EAExpr{}, err
	}

	baseTok, err := p.want(IDENT)
	if err != nil {
		return instructions.EAExpr{}, err
	}
	reg, isPC, ok := parseEABaseRegister(baseTok.Text)
	if !ok {
		return instructions.EAExpr{}, parserError(baseTok, "expected An or PC as the base register")
	}

	var ix instructions.EAIndex
	indexPresent := false
	postIndexed := false

	if p.accept(COMMA) {
		// Post-indexed: [bd,An,Xn]
		postIndexed = true
		indexPresent = true
		ix, err = p.parseEAIndex()
		if err != nil {
			return instructions.EAExpr{}, err
		}
	}
	if _, err := p.want(RBRACKET); err != nil {
		return instructions.EAExpr{}, err
	}

	if !postIndexed && p.peek().Kind == COMMA && looksLikeIndexRegister(p.peekN(2)) {
		p.next() // ','
		indexPresent = true
		ix, err = p.parseEAIndex()
		if err != nil {
			return instructions.EAExpr{}, err
		}
	}

	od := int32(0)
	odSize := instructions.DispNull
	if p.accept(COMMA) {
		od, odSize, err = p.parseSizedDisp(RPAREN)
		if err != nil {
			return instructions.EAExpr{}, err
		}
	}
	if _, err := p.want(RPAREN); err != nil {
		return instructions.EAExpr{}, err
	}

	e := instructions.EAExpr{
		IndexPresent:  indexPresent,
		Index:         ix,
		BaseDisp:      bd,
		BaseDispSize:  bdSize,
		OuterDisp:     od,
		OuterDispSize: odSize,
	}
	switch {
	case isPC && postIndexed:
		e.Kind = instructions.EAkMemPostPC
	case isPC:
		e.Kind = instructions.EAkMemPrePC
	case postIndexed:
		e.Kind, e.Reg = instructions.EAkMemPostAn, reg
	default:
		e.Kind, e.Reg = instructions.EAkMemPreAn, reg
	}
	return e, nil
}

func (p *Parser) parseEADisplacementBody(expr exprInfo) (instructions.EAExpr, error) {
	// We are inside the parentheses of d(...) or (d,...)
	base, err := p.want(IDENT)
	if err != nil {
		return instructions.EAExpr{}, err
	}
	disp := expr.Value
	if isPC(base.Text) {
		if p.accept(COMMA) {
			ix, err := p.parseEAIndex()
			if err != nil {
				return instructions.EAExpr{}, err
			}
			if _, err := p.want(RPAREN); err != nil {
				return instructions.EAExpr{}, err
			}
			disp, err = p.pcRelativeDisp(expr, -128, 127)
			if err != nil {
				return instructions.EAExpr{}, err
			}
			return indexedEA(base, disp, ix)
		}
		if _, err := p.want(RPAREN); err != nil {
			return instructions.EAExpr{}, err
		}
		disp, err = p.pcRelativeDisp(expr, -32768, 32767)
		if err != nil {
			return instructions.EAExpr{}, err
		}
		return displacementEA(base, disp)
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

	// Optional scale factor *1, *2, *4, *8. Scales other than 1 are a
	// 68020+ addressing-mode extension: a real 68000 has no scale field
	// in its brief extension word, so accepting them unconditionally
	// would let the assembler emit an opcode the target CPU can't run.
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
		// CPU32 also has scale factors (confirmed against GNU binutils'
		// gas/config/tc-m68k.c, whose own error message reads "needs
		// cpu32 or 68020 or higher"), so this gates on CPU32 rather than
		// CPU68020 — Supports' plain CPU-floor comparison then grants it
		// to CPU32 and every 68020+ tier from that one tag.
		if ix.Scale != 1 && !p.target.Supports(instructions.Target{CPU: instructions.CPU32}) {
			return ix, fmt.Errorf("scale factor *%d requires 68020 or later (target is %s)", ix.Scale, p.target.CPU)
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
