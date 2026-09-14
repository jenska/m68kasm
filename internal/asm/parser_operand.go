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
		if i > 0 {
			if _, err := p.want(COMMA); err != nil {
				return args, err
			}
		}

		eaExpr, err := p.parseOperand(operandKind, mn, &args, i)
		if err != nil {
			return args, err
		}

		switch i {
		case 0:
			args.Src = eaExpr
		case 1:
			args.Dst = eaExpr
		case 2:
			// A third operand (CAS's <ea>, PACK/UNPK's #adjustment,
			// PFLUSH's optional <ea>, PTEST's #level).
			args.Aux = eaExpr
		default:
			// A fourth operand — only PTEST's optional trailing An
			// result register reaches this; every other instruction in
			// this codebase has at most three.
			args.Aux2 = eaExpr
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
		imm, err := p.parseImmExpr()
		if err != nil {
			return eaExpr, err
		}
		eaExpr = imm

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

	case instructions.OpkTC:
		special, err := p.parseExpectedSpecialRegister("TC", instructions.EAkTC)
		if err != nil {
			return eaExpr, err
		}
		eaExpr = special

	case instructions.OpkCRP:
		special, err := p.parseExpectedSpecialRegister("CRP", instructions.EAkCRP)
		if err != nil {
			return eaExpr, err
		}
		eaExpr = special

	case instructions.OpkSRP:
		special, err := p.parseExpectedSpecialRegister("SRP", instructions.EAkSRP)
		if err != nil {
			return eaExpr, err
		}
		eaExpr = special

	case instructions.OpkTT0:
		special, err := p.parseExpectedSpecialRegister("TT0", instructions.EAkTT0)
		if err != nil {
			return eaExpr, err
		}
		eaExpr = special

	case instructions.OpkTT1:
		special, err := p.parseExpectedSpecialRegister("TT1", instructions.EAkTT1)
		if err != nil {
			return eaExpr, err
		}
		eaExpr = special

	case instructions.OpkMMUSR:
		// "PSR" is the 68851's own name for this exact register and
		// encoding; "MMUSR" is the 68030's name for it. Real hardware,
		// not two different things — see cpu030_pmmu3.go.
		tok, err := p.want(IDENT)
		if err != nil {
			return eaExpr, err
		}
		if !strings.EqualFold(tok.Text, "MMUSR") && !strings.EqualFold(tok.Text, "PSR") {
			return eaExpr, errorAtToken(tok, fmt.Errorf("expected MMUSR or PSR"))
		}
		eaExpr = instructions.EAExpr{Kind: instructions.EAkMMUSR}

	case instructions.OpkDRP:
		special, err := p.parseExpectedSpecialRegister("DRP", instructions.EAkDRP)
		if err != nil {
			return eaExpr, err
		}
		eaExpr = special

	case instructions.OpkCAL:
		special, err := p.parseExpectedSpecialRegister("CAL", instructions.EAkCAL)
		if err != nil {
			return eaExpr, err
		}
		eaExpr = special

	case instructions.OpkVAL:
		special, err := p.parseExpectedSpecialRegister("VAL", instructions.EAkVAL)
		if err != nil {
			return eaExpr, err
		}
		eaExpr = special

	case instructions.OpkSCC:
		special, err := p.parseExpectedSpecialRegister("SCC", instructions.EAkSCC)
		if err != nil {
			return eaExpr, err
		}
		eaExpr = special

	case instructions.OpkAC:
		special, err := p.parseExpectedSpecialRegister("AC", instructions.EAkAC)
		if err != nil {
			return eaExpr, err
		}
		eaExpr = special

	case instructions.OpkPCSR:
		special, err := p.parseExpectedSpecialRegister("PCSR", instructions.EAkPCSR)
		if err != nil {
			return eaExpr, err
		}
		eaExpr = special

	case instructions.OpkBAD:
		tok, err := p.want(IDENT)
		if err != nil {
			return eaExpr, err
		}
		n, ok := parsePmmuNumberedRegister(tok.Text, "BAD")
		if !ok {
			return eaExpr, errorAtToken(tok, fmt.Errorf("expected BAD0-BAD7, got %s", tok.Text))
		}
		eaExpr = instructions.EAExpr{Kind: instructions.EAkBAD, Reg: n}

	case instructions.OpkBAC:
		tok, err := p.want(IDENT)
		if err != nil {
			return eaExpr, err
		}
		n, ok := parsePmmuNumberedRegister(tok.Text, "BAC")
		if !ok {
			return eaExpr, errorAtToken(tok, fmt.Errorf("expected BAC0-BAC7, got %s", tok.Text))
		}
		eaExpr = instructions.EAExpr{Kind: instructions.EAkBAC, Reg: n}

	case instructions.OpkFCSpec:
		spec, err := p.parseFCSpec()
		if err != nil {
			return eaExpr, err
		}
		eaExpr = spec

	case instructions.OpkCtrlReg:
		tok, err := p.want(IDENT)
		if err != nil {
			return eaExpr, err
		}
		ctrlKind, ok := controlRegisterKind(tok.Text)
		if !ok {
			return eaExpr, errorAtToken(tok, fmt.Errorf("expected a control register (SFC, DFC, USP, or VBR), got %s", tok.Text))
		}
		eaExpr.Kind = ctrlKind

	case instructions.OpkFPn:
		tok, err := p.want(IDENT)
		if err != nil {
			return eaExpr, err
		}
		n, ok := parseFPRegister(tok.Text)
		if !ok {
			return eaExpr, errorAtToken(tok, fmt.Errorf("expected an FPU register (FP0-FP7), got %s", tok.Text))
		}
		eaExpr.Kind = instructions.EAkFPn
		eaExpr.Reg = n

	case instructions.OpkRegPair:
		pair, err := p.parseRegPair()
		if err != nil {
			return eaExpr, err
		}
		eaExpr = pair

	case instructions.OpkAnIndPair:
		pair, err := p.parseAnIndPair()
		if err != nil {
			return eaExpr, err
		}
		eaExpr = pair

	case instructions.OpkFPRegPair:
		pair, err := p.parseFPRegPair()
		if err != nil {
			return eaExpr, err
		}
		eaExpr = pair

	case instructions.OpkEAKFactor:
		ea, err := p.parseEAKFactor()
		if err != nil {
			return eaExpr, err
		}
		eaExpr = ea

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

	case instructions.OpkPostincAn:
		ea, err := p.parseEA()
		if err != nil {
			return eaExpr, err
		}
		if ea.Kind != instructions.EAkAddrPostinc {
			return eaExpr, errorAtLine(mn.Line, fmt.Errorf("expected (An)+"))
		}
		eaExpr = ea

	case instructions.OpkCacheSel:
		tok, err := p.want(IDENT)
		if err != nil {
			return eaExpr, err
		}
		sel, ok := cacheSelectorKind(tok.Text)
		if !ok {
			return eaExpr, errorAtToken(tok, fmt.Errorf("expected a cache selector (NC, DC, IC, or BC), got %s", tok.Text))
		}
		eaExpr.Kind = instructions.EAkCacheSel
		eaExpr.Reg = sel

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

	case instructions.OpkFPRegList:
		mask, err := p.parseFPRegList()
		if err != nil {
			return eaExpr, err
		}
		eaExpr.Kind = instructions.EAkNone
		if position == 0 {
			args.FPRegMaskSrc = mask
		} else {
			args.FPRegMaskDst = mask
		}

	case instructions.OpkFPCtrlRegList:
		mask, err := p.parseFPCtrlRegList()
		if err != nil {
			return eaExpr, err
		}
		eaExpr.Kind = instructions.EAkNone
		if position == 0 {
			args.FPCtrlMaskSrc = mask
		} else {
			args.FPCtrlMaskDst = mask
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

	// FPU mnemonics ("F..." — FADD, FMOVE, etc.) use .s/.d/.x for single/
	// double/extended precision, which collides with the plain ".s" =
	// byte-branch alias every other mnemonic uses: the same letter means
	// something different depending on which instruction family it
	// follows, so the lookup itself must be mnemonic-aware.
	parseSuffix := sizeFromIdent
	if len(mn.Text) > 0 && (mn.Text[0] == 'F' || mn.Text[0] == 'f') {
		parseSuffix = fpSizeFromIdent
	}

	if idx := strings.IndexRune(mn.Text, '.'); idx > 0 {
		suf := mn.Text[idx+1:]
		if suf == "" {
			return 0, parserError(mn, "unknown size suffix")
		}
		sz, ok := parseSuffix(suf)
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

// fpSizeFromIdent is sizeFromIdent for FPU mnemonics: same integer sizes
// (.b/.w/.l, for FPU instructions that convert to/from an integer EA),
// plus .s/.d/.x for the IEEE single/double and Motorola extended
// floating-point formats. ".s" means byte-sized-branch for every other
// mnemonic (sizeFromIdent) but single-precision here — see
// parseSizeSpec's dispatch and Size's doc comment on SingleSize.
func fpSizeFromIdent(s string) (instructions.Size, bool) {
	switch strings.ToLower(s) {
	case "b":
		return instructions.ByteSize, true
	case "w":
		return instructions.WordSize, true
	case "l":
		return instructions.LongSize, true
	case "s":
		return instructions.SingleSize, true
	case "d":
		return instructions.DoubleSize, true
	case "x":
		return instructions.ExtendedSize, true
	case "p":
		return instructions.PackedSize, true
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

// parseFPRegList is parseRegList for FMOVEM's FPn register-list operand
// ("FP0-FP3", "FP1/FP4/FP6", or a bare "FP0") — same range/slash/comma
// syntax, but over the single FP0-FP7 namespace (no A/D distinction),
// producing an 8-bit mask (bit N = FPn) rather than RegList's 16-bit
// Dn/An mask.
func (p *Parser) parseFPRegList() (uint16, error) {
	mask := uint16(0)
	for {
		regTok, err := p.want(IDENT)
		if err != nil {
			return 0, err
		}
		reg, ok := parseFPRegister(regTok.Text)
		if !ok {
			return 0, errorAtToken(regTok, fmt.Errorf("expected FP0-FP7 in register list, got %s", regTok.Text))
		}
		endReg := reg
		if p.accept(MINUS) {
			toTok, err := p.want(IDENT)
			if err != nil {
				return 0, err
			}
			endReg, ok = parseFPRegister(toTok.Text)
			if !ok {
				return 0, errorAtToken(toTok, fmt.Errorf("expected FP0-FP7 in register list, got %s", toTok.Text))
			}
			if endReg < reg {
				return 0, errorAtToken(toTok, fmt.Errorf("descending ranges are not allowed"))
			}
		}
		for r := reg; r <= endReg; r++ {
			mask |= uint16(1 << r)
		}
		if p.peek().Kind == SLASH {
			p.next()
			continue
		}
		if p.peek().Kind == COMMA {
			nxt := p.peekN(2)
			if nxt.Kind == IDENT {
				if _, ok := parseFPRegister(nxt.Text); ok {
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

// parseFPCtrlRegList parses FMOVEM's FPCR/FPSR/FPIAR control-register
// list operand — a bare name ("FPIAR") or a slash-separated combination
// ("FPCR/FPSR"). No range syntax: unlike FP0-FP7 or D0-D7, there's no
// numeric ordering among these three registers to range over.
func (p *Parser) parseFPCtrlRegList() (uint16, error) {
	mask := uint16(0)
	for {
		regTok, err := p.want(IDENT)
		if err != nil {
			return 0, err
		}
		bit, ok := fpCtrlRegisterBit(regTok.Text)
		if !ok {
			return 0, errorAtToken(regTok, fmt.Errorf("expected FPIAR, FPSR, or FPCR, got %s", regTok.Text))
		}
		mask |= bit
		if p.peek().Kind == SLASH {
			p.next()
			continue
		}
		break
	}
	return mask, nil
}

// fpCtrlRegisterBit maps one of FMOVEM's control-register names to its
// selector bit (GAS's own assignment: FPIAR=bit0, FPSR=bit1, FPCR=bit2
// — gas/config/tc-m68k.c's "case 'l': case 'L':" REGLST conversion for
// CONTROL-mode FPI/FPS/FPC, reduced from its own 1<<24/1<<25/1<<26
// internal encoding).
func fpCtrlRegisterBit(name string) (uint16, bool) {
	switch strings.ToUpper(name) {
	case "FPIAR":
		return 1 << 0, true
	case "FPSR":
		return 1 << 1, true
	case "FPCR":
		return 1 << 2, true
	default:
		return 0, false
	}
}
