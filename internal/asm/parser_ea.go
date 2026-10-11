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
// EAExprKind and, for the registers added after the 68010 (EAkCtrlReg),
// the MOVEC selector to store in the EAExpr's Reg. Which CPU has which
// register is validateMOVEC's business, not the parser's.
func controlRegisterKind(name string) (instructions.EAExprKind, int, bool) {
	if sel, ok := instructions.LaterControlRegisterSelector(name); ok {
		return instructions.EAkCtrlReg, int(sel), true
	}
	kind, ok := baseControlRegisterKind(name)
	return kind, 0, ok
}

func baseControlRegisterKind(name string) (instructions.EAExprKind, bool) {
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

// cacheSelectorKind recognizes CINV/CPUSH's cache-selector pseudo-
// registers (NC/DC/IC/BC — none/data/instruction/both), returning GAS's
// own numeric encoding (gas/config/tc-m68k.c's install_operand case
// 'c': NC=0, DC=1, IC=2, BC=3).
func cacheSelectorKind(name string) (int, bool) {
	switch strings.ToUpper(name) {
	case "NC":
		return 0, true
	case "DC":
		return 1, true
	case "IC":
		return 2, true
	case "BC":
		return 3, true
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

// parsePmmuNumberedRegister recognizes PMOVE's numbered breakpoint
// registers, "BAD0"-"BAD7" or "BAC0"-"BAC7" (prefix is "BAD" or "BAC"),
// the same "fixed prefix plus a single 0-7 digit" shape parseFPRegister
// uses for FP0-FP7.
func parsePmmuNumberedRegister(s, prefix string) (int, bool) {
	if len(s) == len(prefix)+1 && strings.EqualFold(s[:len(prefix)], prefix) && s[len(prefix)] >= '0' && s[len(prefix)] <= '7' {
		return int(s[len(prefix)] - '0'), true
	}
	return 0, false
}

// parseFCSpec parses PFLUSH/PLOAD/PTEST's function-code specifier
// operand, which GAS itself accepts in three alternate spellings: the
// named registers SFC/DFC, a plain Dn (holding the function code at
// runtime), or "#<imm>" (a literal function code, 0-7 — real 68k
// function codes are always a 3-bit CPU signal). See EAExpr.FCMode's
// doc comment for how the result is tagged.
func (p *Parser) parseFCSpec() (instructions.EAExpr, error) {
	if p.peek().Kind == HASH {
		p.next()
		imm, err := p.parseExpr()
		if err != nil {
			return instructions.EAExpr{}, err
		}
		if imm < 0 || imm > 7 {
			return instructions.EAExpr{}, fmt.Errorf("function code immediate out of range 0-7: %d", imm)
		}
		return instructions.EAExpr{Kind: instructions.EAkFCSpec, FCMode: 2, Imm: imm}, nil
	}
	tok, err := p.want(IDENT)
	if err != nil {
		return instructions.EAExpr{}, err
	}
	switch {
	case strings.EqualFold(tok.Text, "SFC"):
		return instructions.EAExpr{Kind: instructions.EAkFCSpec, FCMode: 0, Reg: 0}, nil
	case strings.EqualFold(tok.Text, "DFC"):
		return instructions.EAExpr{Kind: instructions.EAkFCSpec, FCMode: 0, Reg: 1}, nil
	}
	if ok, dn := isRegDn(tok.Text); ok {
		return instructions.EAExpr{Kind: instructions.EAkFCSpec, FCMode: 1, Reg: dn}, nil
	}
	return instructions.EAExpr{}, errorAtToken(tok, fmt.Errorf("expected SFC, DFC, a data register, or #<imm> (function code specifier), got %s", tok.Text))
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

// parseFPRegPair parses FSINCOS's "FPc:FPs" dual-destination operand:
// FPc (cosine result) first, FPs (sine result) second — Motorola's own
// documented mnemonic order, matching GAS's own bit layout (cosine in
// the low field, sine sharing every other FPU instruction's usual
// single-destination field). Unlike parseRegPair's "Dr:Dq" shorthand,
// there is no bare single-register form: sine and cosine can never
// share one destination register, so the colon is always mandatory.
func (p *Parser) parseFPRegPair() (instructions.EAExpr, error) {
	firstTok, err := p.want(IDENT)
	if err != nil {
		return instructions.EAExpr{}, err
	}
	first, ok := parseFPRegister(firstTok.Text)
	if !ok {
		return instructions.EAExpr{}, errorAtToken(firstTok, fmt.Errorf("expected an FPU register (FP0-FP7), got %s", firstTok.Text))
	}
	if _, err := p.want(COLON); err != nil {
		return instructions.EAExpr{}, err
	}
	secondTok, err := p.want(IDENT)
	if err != nil {
		return instructions.EAExpr{}, err
	}
	second, ok := parseFPRegister(secondTok.Text)
	if !ok {
		return instructions.EAExpr{}, errorAtToken(secondTok, fmt.Errorf("expected an FPU register (FP0-FP7), got %s", secondTok.Text))
	}
	return instructions.EAExpr{Kind: instructions.EAkFPRegPair, Reg: first, Reg2: second}, nil
}

// parseEAKFactor parses FMOVE.P's packed-BCD store destination,
// "<ea>{#k}" (static) or "<ea>{Dn}" (dynamic) — the k-factor suffix is
// mandatory, unlike a bit-field spec's optional "{offset:width}".
// Deliberately calls parseEABase, not parseEA: parseEA would try to
// consume the same "{...}" as a bit-field suffix instead (expecting a
// colon-separated "offset:width" pair, which a k-factor never has).
func (p *Parser) parseEAKFactor() (instructions.EAExpr, error) {
	base, err := p.parseEABase()
	if err != nil {
		return base, err
	}
	if _, err := p.want(LBRACE); err != nil {
		return base, err
	}
	isReg, val, err := p.parseKFactor()
	if err != nil {
		return base, err
	}
	if !isReg && (val < -64 || val > 17) {
		return base, fmt.Errorf("k-factor out of range: %d (expected -64 to 17, or a data register)", val)
	}
	if _, err := p.want(RBRACE); err != nil {
		return base, err
	}
	base.HasKFactor = true
	base.KFactorIsReg = isReg
	base.KFactorVal = val
	return base, nil
}

// parseKFactor parses one k-factor value: a bare Dn (dynamic — the
// register holds the k-factor at runtime) or "#<signed integer>"
// (static — unlike every other immediate in this codebase, which
// forwards straight to the general integer expression evaluator, this
// one stays a single literal since the manual's own range, -64 to +17,
// makes a "#<expr>" arithmetic combination meaningless here).
func (p *Parser) parseKFactor() (isReg bool, val int32, err error) {
	if t := p.peek(); t.Kind == IDENT {
		if ok, dn := isRegDn(t.Text); ok {
			p.next()
			return true, int32(dn), nil
		}
	}
	if _, err := p.want(HASH); err != nil {
		return false, 0, err
	}
	v, err := p.parseExprUntil(RBRACE)
	if err != nil {
		return false, 0, err
	}
	return false, int32(v), nil
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

// parseAnIndirectRegister parses one "(Rn)" of CAS2's pointer pair. Rn may
// be a data or an address register; the result is the register number with
// bit 3 set for An, which is how CAS2's extension words encode it (D/A in
// bit 15, Rn in bits 14-12).
func (p *Parser) parseAnIndirectRegister() (int, error) {
	if _, err := p.want(LPAREN); err != nil {
		return 0, err
	}
	tok, err := p.want(IDENT)
	if err != nil {
		return 0, err
	}
	var reg int
	if ok, an := isRegAn(tok.Text); ok {
		reg = 8 | an
	} else if ok, dn := isRegDn(tok.Text); ok {
		reg = dn
	} else {
		return 0, errorAtToken(tok, fmt.Errorf("expected Dn or An, got %s", tok.Text))
	}
	if _, err := p.want(RPAREN); err != nil {
		return 0, err
	}
	return reg, nil
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

// parseEABaseRegister parses the base register of an indexed or memory
// indirect operand: An, PC, or their suppressed forms ZAn and ZPC (68020+
// full extension word format).
func parseEABaseRegister(text string) (reg int, pcRelative, suppressed, ok bool) {
	if len(text) > 1 && (text[0] == 'z' || text[0] == 'Z') {
		if r, pc, _, ok := parseEABaseRegister(text[1:]); ok {
			return r, pc, true, true
		}
		return 0, false, false, false
	}
	if ok, an := isRegAn(text); ok {
		return an, false, false, true
	}
	if isPC(text) {
		return 0, true, false, true
	}
	return 0, false, false, false
}

func displacementEA(base Token, disp int64) (instructions.EAExpr, error) {
	if reg, isPC, suppressed, ok := parseEABaseRegister(base.Text); ok && !suppressed {
		if isPC {
			return instructions.EAExpr{Kind: instructions.EAkPCDisp16, Disp16: int32(disp)}, nil
		}
		return instructions.EAExpr{Kind: instructions.EAkAddrDisp16, Reg: reg, Disp16: int32(disp)}, nil
	}
	return instructions.EAExpr{}, parserError(base, "base must be An or PC for displacement addressing")
}

func indexedEA(base Token, disp int64, ix instructions.EAIndex) (instructions.EAExpr, error) {
	if disp < -128 || disp > 127 {
		return instructions.EAExpr{}, parserError(base, fmt.Sprintf("displacement out of range for brief indexed addressing: %d (a signed byte; the 68020 full format takes a constant or an explicit .w/.l size)", disp))
	}
	ix.Disp8 = int8(disp)
	if reg, isPC, suppressed, ok := parseEABaseRegister(base.Text); ok && !suppressed {
		if isPC {
			return instructions.EAExpr{Kind: instructions.EAkIdxPCBrief, Index: ix}, nil
		}
		return instructions.EAExpr{Kind: instructions.EAkIdxAnBrief, Reg: reg, Index: ix}, nil
	}
	return instructions.EAExpr{}, parserError(base, "base must be An or PC for indexed addressing")
}

// baseDisp is a base displacement as written: its value, whether it uses a
// symbol, and its size if given explicitly with .w or .l.
type baseDisp struct {
	value     int64
	hasSymbol bool
	size      instructions.DispSize
	explicit  bool
}

// needsFullFormat reports whether an (d,An/PC[,Xn]) operand must use the
// 68020 full extension word: a suppressed base or index, an explicit
// displacement size, or a constant displacement too large for the brief
// format (limit is the largest brief displacement: a byte with an index,
// a word without).
func needsFullFormat(d baseDisp, baseSuppressed, indexSuppressed bool, limit int64) bool {
	if baseSuppressed || indexSuppressed || d.explicit {
		return true
	}
	return !d.hasSymbol && (d.value < -limit-1 || d.value > limit)
}

// fullIndexedEA builds a 68020+ full-format indexed operand without memory
// indirection: (bd,An,Xn), (bd,PC,Xn) and their forms with a suppressed
// base or index. A constant base displacement gets the smallest size.
func (p *Parser) fullIndexedEA(base Token, d baseDisp, ix instructions.EAIndex, indexPresent bool) (instructions.EAExpr, error) {
	if !p.target.Supports(instructions.Target{CPU: instructions.CPU68020}) {
		return instructions.EAExpr{}, parserError(base, fmt.Sprintf("the full extension word format requires 68020 or later (target is %s)", p.target.CPU))
	}
	reg, isPC, suppressed, ok := parseEABaseRegister(base.Text)
	if !ok {
		return instructions.EAExpr{}, parserError(base, "base must be An, PC, ZAn or ZPC")
	}
	size := d.size
	if !d.explicit {
		switch {
		case d.value == 0:
			size = instructions.DispNull
		case d.value >= -32768 && d.value <= 32767:
			size = instructions.DispWord
		default:
			size = instructions.DispLong
		}
	}
	e := instructions.EAExpr{
		Kind:           instructions.EAkIdxAnBrief,
		Reg:            reg,
		Full:           true,
		Index:          ix,
		IndexPresent:   indexPresent && !ix.Suppressed,
		BaseSuppressed: suppressed,
		BaseDisp:       int32(d.value),
		BaseDispSize:   size,
	}
	if isPC {
		e.Kind, e.Reg = instructions.EAkIdxPCBrief, 0
	}
	return e, nil
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
	return p.parseImmExpr()
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
		return p.parseEADisplacementBody(expr, instructions.DispNull, false)
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

	// Case 2: (An, ix) or (PC, ix) -- no displacement
	if base := p.peek(); base.Kind == IDENT && p.peekN(2).Kind == COMMA {
		if reg, isPC, suppressed, ok := parseEABaseRegister(base.Text); ok {
			p.next() // base
			p.next() // ','
			ix, err := p.parseEAIndex()
			if err != nil {
				return instructions.EAExpr{}, err
			}
			if _, err := p.want(RPAREN); err != nil {
				return instructions.EAExpr{}, err
			}
			if suppressed || ix.Suppressed {
				return p.fullIndexedEA(base, baseDisp{}, ix, true)
			}
			if isPC {
				return instructions.EAExpr{Kind: instructions.EAkIdxPCBrief, Index: ix}, nil
			}
			return instructions.EAExpr{Kind: instructions.EAkIdxAnBrief, Reg: reg, Index: ix}, nil
		}
	}

	// Case 3: (disp, ...), (abs).W, or (abs).L. The displacement may
	// carry a .w or .l size (68020 full format), on a number or a label.
	if t := p.peek(); t.Kind == IDENT && p.peekN(2).Kind == COMMA {
		if _, _, sized := splitSizedSymbol(t.Text); sized {
			value, size, err := p.parseSizedDisp(COMMA)
			if err != nil {
				return instructions.EAExpr{}, err
			}
			p.next() // ','
			return p.parseEADisplacementBody(exprInfo{Value: int64(value), HasSymbol: true}, size, true)
		}
	}
	expr, err := p.parseExprInfoUntil(COMMA, RPAREN, DOT)
	if err != nil {
		return instructions.EAExpr{}, err
	}
	size, explicit := instructions.DispNull, false
	if p.peek().Kind == DOT && p.peekN(3).Kind == COMMA {
		p.next() // '.'
		szTok, err := p.want(IDENT)
		if err != nil {
			return instructions.EAExpr{}, err
		}
		switch strings.ToUpper(szTok.Text) {
		case "W":
			if expr.Value < -32768 || expr.Value > 32767 {
				return instructions.EAExpr{}, parserError(szTok, fmt.Sprintf("displacement out of range for .w: %d", expr.Value))
			}
			size, explicit = instructions.DispWord, true
		case "L":
			size, explicit = instructions.DispLong, true
		default:
			return instructions.EAExpr{}, parserError(szTok, "expected .w or .l for the displacement size")
		}
	}

	// Subcase 3a: (disp, An/PC) or (disp, An/PC, ix)
	if p.accept(COMMA) {
		return p.parseEADisplacementBody(expr, size, explicit)
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
	if _, err := parseIndexRegister(strings.SplitN(tok.Text, ".", 2)[0]); err == nil {
		return true
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
//	([bd,An,Xn],od)  / ([bd,An,Xn])                -- preindexed: index inside
//	([bd,An],Xn,od)  / ([bd,An],od)  / ([bd,An])   -- postindexed, Xn optional
//
// Preindexed adds the index before reading the address from memory,
// postindexed after (I/IS bit 2 set), as in Motorola's manuals and GNU
// as (tc-m68k.c sets bit 2 for its POST mode, the index outside the
// brackets).
//
// and their PC-relative equivalents (An replaced by PC). bd must always
// be written explicitly (0 for "no base displacement"); od may be
// omitted entirely, meaning 0/null. Gating on the target CPU is the
// caller's responsibility.
func (p *Parser) parseMemIndirectEA() (instructions.EAExpr, error) {
	p.next() // consume '['

	// Inside the brackets: an optional base displacement, an optional base
	// register (An, PC, ZAn, ZPC; none suppresses the base) and, for the
	// preindexed form, an index register.
	e := instructions.EAExpr{BaseDispSize: instructions.DispNull}
	var baseTok Token
	haveBase, isPC, preIndexed := false, false, false
	var ix instructions.EAIndex
	first := true
	for p.peek().Kind != RBRACKET {
		if !first {
			if _, err := p.want(COMMA); err != nil {
				return instructions.EAExpr{}, err
			}
		}
		t := p.peek()
		reg, pc, suppressed, isBase := parseEABaseRegister(t.Text)
		switch {
		case t.Kind == IDENT && isBase && !haveBase && !preIndexed && !hasIndexSyntax(p, t):
			p.next()
			baseTok, haveBase, isPC = t, true, pc
			e.Reg, e.BaseSuppressed = reg, suppressed
		case looksLikeIndexRegister(t):
			if preIndexed {
				return instructions.EAExpr{}, parserError(t, "only one index register allowed")
			}
			var err error
			if ix, err = p.parseEAIndex(); err != nil {
				return instructions.EAExpr{}, err
			}
			preIndexed = true
		case first:
			bd, size, err := p.parseSizedDisp(COMMA, RBRACKET)
			if err != nil {
				return instructions.EAExpr{}, err
			}
			e.BaseDisp, e.BaseDispSize = bd, size
		default:
			return instructions.EAExpr{}, parserError(t, "expected a base or index register")
		}
		first = false
	}
	p.next() // ']'
	if !haveBase {
		e.BaseSuppressed = true
	}
	_ = baseTok

	indexPresent := preIndexed
	if !preIndexed && p.peek().Kind == COMMA && looksLikeIndexRegister(p.peekN(2)) {
		// Postindexed: [bd,An],Xn, the index is added after the
		// indirection.
		p.next() // ','
		var err error
		if ix, err = p.parseEAIndex(); err != nil {
			return instructions.EAExpr{}, err
		}
		indexPresent = true
	}

	e.OuterDispSize = instructions.DispNull
	if p.accept(COMMA) {
		od, odSize, err := p.parseSizedDisp(RPAREN)
		if err != nil {
			return instructions.EAExpr{}, err
		}
		e.OuterDisp, e.OuterDispSize = od, odSize
	}
	if _, err := p.want(RPAREN); err != nil {
		return instructions.EAExpr{}, err
	}

	e.Index = ix
	e.IndexPresent = indexPresent && !ix.Suppressed
	postIndexed := indexPresent && !preIndexed
	switch {
	case isPC && postIndexed:
		e.Kind, e.Reg = instructions.EAkMemPostPC, 0
	case isPC:
		e.Kind, e.Reg = instructions.EAkMemPrePC, 0
	case postIndexed:
		e.Kind = instructions.EAkMemPostAn
	default:
		e.Kind = instructions.EAkMemPreAn
	}
	return e, nil
}

// hasIndexSyntax reports whether the register token t carries an index size
// or scale (D0.L, A1*4), which makes an An an index rather than the base.
func hasIndexSyntax(p *Parser, t Token) bool {
	if strings.ContainsRune(t.Text, '.') {
		return true
	}
	next := p.peekN(2).Kind
	return next == STAR || next == DOT
}

func (p *Parser) parseEADisplacementBody(expr exprInfo, size instructions.DispSize, explicit bool) (instructions.EAExpr, error) {
	// We are inside the parentheses of d(...) or (d,...)
	base, err := p.want(IDENT)
	if err != nil {
		return instructions.EAExpr{}, err
	}
	_, pcBase, suppressed, ok := parseEABaseRegister(base.Text)
	if !ok {
		return instructions.EAExpr{}, parserError(base, "base must be An, PC, ZAn or ZPC")
	}
	d := baseDisp{value: expr.Value, hasSymbol: expr.HasSymbol, size: size, explicit: explicit}
	if pcBase && !suppressed && expr.HasSymbol {
		// PC-relative addressing uses the extension word's address as the
		// base.
		d.value -= int64(p.pc) + 2
	}

	// Indexed: d(An,ix) or d(PC,ix)
	if p.accept(COMMA) {
		ix, err := p.parseEAIndex()
		if err != nil {
			return instructions.EAExpr{}, err
		}
		if _, err := p.want(RPAREN); err != nil {
			return instructions.EAExpr{}, err
		}
		if needsFullFormat(d, suppressed, ix.Suppressed, 127) && p.target.Supports(instructions.Target{CPU: instructions.CPU68020}) {
			return p.fullIndexedEA(base, d, ix, true)
		}
		if pcBase && (d.value < -128 || d.value > 127) {
			return instructions.EAExpr{}, errorAtLine(p.line, fmt.Errorf("PC-relative displacement out of range: %d", d.value))
		}
		return indexedEA(base, d.value, ix)
	}

	// Simple displacement: d(An) or d(PC)
	if _, err := p.want(RPAREN); err != nil {
		return instructions.EAExpr{}, err
	}
	if needsFullFormat(d, suppressed, false, 32767) && p.target.Supports(instructions.Target{CPU: instructions.CPU68020}) {
		return p.fullIndexedEA(base, d, instructions.EAIndex{}, false)
	}
	if d.value < -32768 || d.value > 32767 {
		if pcBase {
			return instructions.EAExpr{}, errorAtLine(p.line, fmt.Errorf("PC-relative displacement out of range: %d", d.value))
		}
		return instructions.EAExpr{}, errorAtLine(p.line, fmt.Errorf("displacement out of range: %d", d.value))
	}
	return displacementEA(base, d.value)
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
	if len(name) == 3 && (name[0] == 'z' || name[0] == 'Z') {
		ix, err := parseIndexRegister(name[1:])
		ix.Suppressed = true
		return ix, err
	}
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
