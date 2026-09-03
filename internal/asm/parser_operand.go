package asm

import (
	"fmt"
	"strings"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

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
		if name, ok, err := p.consumeLocalLabelRef(); err != nil {
			return eaExpr, err
		} else if ok {
			args.Target = name
			return eaExpr, nil
		}
		if p.peek().Kind == IDENT && (p.peekN(2).Kind == EOF || p.peekN(2).Kind == NEWLINE) {
			args.Target = p.next().Text
			return eaExpr, nil
		}
		target, err := p.parseExpr()
		if err != nil {
			return eaExpr, err
		}
		args.TargetAddr = target
		args.HasTargetAddr = true

	default:
		return eaExpr, errorAtLine(mn.Line, fmt.Errorf("unknown identifier %s", mn.Text))
	}

	return eaExpr, nil
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
		if !sizeAllowed(allowed, sz) {
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
	if !sizeAllowed(allowed, sz) {
		return 0, contextualizeAt(p.line, p.col, fmt.Errorf("illegal size for instruction"))
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
