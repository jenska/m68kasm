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
