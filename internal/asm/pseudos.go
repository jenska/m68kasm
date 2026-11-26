package asm

import (
	"fmt"
	"strings"
)

var pseudoMap = map[string]func(*Parser) error{
	".ORG":     parseORG,
	".BYTE":    parseBYTE,
	".WORD":    parseWORD,
	".LONG":    parseLONG,
	".ALIGN":   parseALIGN,
	".EVEN":    parseEVEN,
	".MACRO":   parseMACRO,
	".TEXT":    parseTEXT,
	".DATA":    parseDATA,
	".BSS":     parseBSS,
	".SECTION": parseSECTION,
	".SPACE":   parseSPACE,
	".SKIP":    parseSPACE,
	".ASCII":   func(p *Parser) error { return parseASCII(p, false) },
	".ASCIZ":   func(p *Parser) error { return parseASCII(p, true) },
	".END":     parseEND,
	".INCBIN":  parseINCBIN,
}

func init() {
	// Registered here, not in the literal above: INCLUDE parses statements,
	// which look pseudo ops up in pseudoMap (an initialization cycle).
	pseudoMap[".INCLUDE"] = parseINCLUDE
}

func parseTEXT(p *Parser) error {
	return p.setSection(SectionText)
}

func parseDATA(p *Parser) error {
	return p.setSection(SectionData)
}

func parseBSS(p *Parser) error {
	return p.setSection(SectionBSS)
}

func parseSECTION(p *Parser) error {
	name, err := parseSectionOperand(p)
	if err != nil {
		return err
	}
	section, ok := parseSectionName(name)
	if !ok {
		return contextualizeAt(p.line, p.col, fmt.Errorf("unsupported section %q", name))
	}
	return p.setSection(section)
}

func parseSectionOperand(p *Parser) (string, error) {
	t := p.next()
	switch t.Kind {
	case IDENT, STRING:
		return t.Text, nil
	case DOT:
		nameTok, err := p.want(IDENT)
		if err != nil {
			return "", err
		}
		return "." + nameTok.Text, nil
	default:
		return "", errorAtToken(t, fmt.Errorf("expected section name"))
	}
}

func parseORG(p *Parser) error {
	val, err := p.parseExpr()
	if err != nil {
		return err
	}
	newPC := uint32(val)

	if newPC > maxProgramSize {
		return contextualizeAt(p.line, p.col, fmt.Errorf(".org would exceed maximum program size of %d bytes", maxProgramSize))
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
		return contextualizeAt(p.line, p.col, fmt.Errorf(".org cannot move backwards (pc=%d -> %d)", p.pc, newPC))
	}

	if newPC > p.pc {
		if err := p.emitPaddingBytes(newPC-p.pc, 0x00); err != nil {
			return err
		}
		return nil
	}

	return nil
}

// parseBYTE parses .byte / DC.B items: expressions and strings, in any mix.
// A string contributes one byte per character.
func parseBYTE(p *Parser) error {
	col := p.col
	var bytes []byte
	for {
		if p.peek().Kind == STRING {
			str := p.next().Text
			for i := 0; i < len(str); i++ {
				if err := ensureBSSValue(p, int64(str[i]), ".byte"); err != nil {
					return err
				}
			}
			bytes = append(bytes, str...)
		} else {
			v, err := p.parseExpr()
			if err != nil {
				return err
			}
			if err := ensureBSSValue(p, v, ".byte"); err != nil {
				return err
			}
			bytes = append(bytes, byte(v))
		}
		if !p.accept(COMMA) {
			break
		}
	}
	p.items = append(p.items, &DataBytes{Bytes: bytes, PC: p.pc, Line: p.line, Col: col, Section: p.section})
	p.pc += uint32(len(bytes))
	return nil
}

// .word <expr>[, <expr>]...
func parseWORD(p *Parser) error {
	col := p.col
	// first value
	v, err := p.parseExpr()
	if err != nil {
		return err
	}

	out := make([]byte, 0, 2*4)
	if v < -0x8000 || v > 0xFFFF {
		return contextualizeAt(p.line, p.col, fmt.Errorf(".word value out of 16-bit range: %d", v))
	}
	if err := ensureBSSValue(p, v, ".word"); err != nil {
		return err
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
			return contextualizeAt(p.line, p.col, fmt.Errorf(".word value out of 16-bit range: %d", v))
		}
		if err := ensureBSSValue(p, v, ".word"); err != nil {
			return err
		}
		w := uint16(int16(v))
		out = append(out, byte(w>>8), byte(w))
	}

	p.items = append(p.items, &DataBytes{Bytes: out, PC: p.pc, Line: p.line, Col: col, Section: p.section})
	p.pc += uint32(len(out))
	return nil
}

// .long <expr>[, <expr>]...
func parseLONG(p *Parser) error {
	col := p.col
	v, err := p.parseExpr()
	if err != nil {
		return err
	}

	out := make([]byte, 0, 4*4)
	if v < -0x80000000 || v > 0xFFFFFFFF {
		return contextualizeAt(p.line, p.col, fmt.Errorf(".long value out of range: %d", v))
	}
	if err := ensureBSSValue(p, v, ".long"); err != nil {
		return err
	}
	u := uint32(v) // two's complement when v is negative
	out = append(out, byte(u>>24), byte(u>>16), byte(u>>8), byte(u))

	for p.accept(COMMA) {
		v, err := p.parseExpr()
		if err != nil {
			return err
		}
		if v < -0x80000000 || v > 0xFFFFFFFF {
			return contextualizeAt(p.line, p.col, fmt.Errorf(".long value out of range: %d", v))
		}
		if err := ensureBSSValue(p, v, ".long"); err != nil {
			return err
		}
		u := uint32(v)
		out = append(out, byte(u>>24), byte(u>>16), byte(u>>8), byte(u))
	}

	p.items = append(p.items, &DataBytes{Bytes: out, PC: p.pc, Line: p.line, Col: col, Section: p.section})
	p.pc += uint32(len(out))
	return nil
}

func ensureBSSValue(p *Parser, v int64, directive string) error {
	if p.section != SectionBSS || p.allowForwardRefs {
		return nil
	}
	if v != 0 {
		return contextualizeAt(p.line, p.col, fmt.Errorf("%s in %s must be zero-initialized", directive, p.section.Name()))
	}
	return nil
}

// .align <expr>[, <fill>]
func parseALIGN(p *Parser) error {
	// alignment value
	val, err := p.parseExpr()
	if err != nil {
		return err
	}
	if val < 1 {
		return contextualizeAt(p.line, p.col, fmt.Errorf(".align expects value >= 1, got %d", val))
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

func parseEVEN(p *Parser) error {
	if p.pc%2 == 0 {
		return nil
	}
	return p.emitPaddingBytes(1, 0x00)
}

func parseDC(p *Parser, suffix string) error {
	suf := strings.ToUpper(suffix)
	switch suf {
	case "B":
		return parseBYTE(p)
	case "W":
		return parseWORD(p)
	case "L":
		return parseLONG(p)
	default:
		return fmt.Errorf("unknown DC size .%s", suffix)
	}
}

func parseMACRO(p *Parser) error {
	nameTok, err := p.want(IDENT)
	if err != nil {
		return err
	}

	params := []string{}
	for {
		t := p.peek()
		if t.Kind == NEWLINE || t.Kind == EOF {
			_ = p.next()
			break
		}
		paramTok, err := p.want(IDENT)
		if err != nil {
			return err
		}
		params = append(params, paramTok.Text)
		if !p.accept(COMMA) {
			if p.peek().Kind == NEWLINE {
				_ = p.next()
			}
			break
		}
	}

	body := []Token{}
	for {
		t := p.next()
		if t.Kind == EOF {
			return contextualizeAt(nameTok.Line, nameTok.Col, fmt.Errorf("unexpected EOF inside macro"))
		}
		if t.Kind == DOT {
			nxt := p.peek()
			if nxt.Kind == IDENT && strings.EqualFold(nxt.Text, "ENDMACRO") {
				_ = p.next()
				if p.peek().Kind == NEWLINE {
					_ = p.next()
				}
				break
			}
		}
		body = append(body, t)
	}

	p.macros[nameTok.Text] = macroDef{params: params, body: body}
	return nil
}

// unitSize returns the size in bytes of a DS/DCB .B/.W/.L suffix.
func unitSize(directive, suffix string) (uint32, error) {
	switch strings.ToUpper(suffix) {
	case "B":
		return 1, nil
	case "W":
		return 2, nil
	case "L":
		return 4, nil
	default:
		return 0, fmt.Errorf("unknown %s size .%s", directive, suffix)
	}
}

// parseCount parses a non-negative repeat count or byte count.
func parseCount(p *Parser, directive string) (uint32, error) {
	n, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, contextualizeAt(p.line, p.col, fmt.Errorf("%s count must not be negative: %d", directive, n))
	}
	if n > int64(maxProgramSize) {
		return 0, contextualizeAt(p.line, p.col, fmt.Errorf("%s count %d would exceed maximum program size of %d bytes", directive, n, maxProgramSize))
	}
	return uint32(n), nil
}

// parseDS parses DS.B/.W/.L <count>: reserve count zero-filled units. Like
// DC.W and DC.L, it does not align the location counter.
func parseDS(p *Parser, suffix string) error {
	unit, err := unitSize("DS", suffix)
	if err != nil {
		return err
	}
	count, err := parseCount(p, "DS")
	if err != nil {
		return err
	}
	return p.emitPaddingBytes(count*unit, 0)
}

// parseSPACE parses .space / .skip <n>: n zero bytes.
func parseSPACE(p *Parser) error {
	n, err := parseCount(p, ".space")
	if err != nil {
		return err
	}
	return p.emitPaddingBytes(n, 0)
}

// parseDCB parses DCB.B/.W/.L <count>,<value>: count copies of value.
func parseDCB(p *Parser, suffix string) error {
	unit, err := unitSize("DCB", suffix)
	if err != nil {
		return err
	}
	col := p.col
	count, err := parseCount(p, "DCB")
	if err != nil {
		return err
	}
	if _, err := p.want(COMMA); err != nil {
		return err
	}
	v, err := p.parseExpr()
	if err != nil {
		return err
	}
	var lo, hi int64
	switch unit {
	case 1:
		lo, hi = -0x80, 0xFF
	case 2:
		lo, hi = -0x8000, 0xFFFF
	default:
		lo, hi = -0x80000000, 0xFFFFFFFF
	}
	if v < lo || v > hi {
		return contextualizeAt(p.line, p.col, fmt.Errorf("DCB.%s value out of range: %d", strings.ToUpper(suffix), v))
	}
	if err := ensureBSSValue(p, v, "DCB"); err != nil {
		return err
	}
	total := count * unit
	if total > maxProgramSize || p.pc > maxProgramSize-total {
		return errorAtLine(p.line, fmt.Errorf("DCB would exceed maximum program size of %d bytes", maxProgramSize))
	}
	unitBytes := make([]byte, 0, 4)
	for shift := int(unit-1) * 8; shift >= 0; shift -= 8 {
		unitBytes = append(unitBytes, byte(uint32(v)>>uint(shift)))
	}
	out := make([]byte, 0, total)
	for range count {
		out = append(out, unitBytes...)
	}
	p.items = append(p.items, &DataBytes{Bytes: out, PC: p.pc, Line: p.line, Col: col, Section: p.section})
	p.pc += total
	return nil
}

// parseASCII parses .ascii / .asciz "text"[, "text"]...; .asciz appends a
// zero byte to every string.
func parseASCII(p *Parser, terminate bool) error {
	col := p.col
	var out []byte
	for {
		t, err := p.want(STRING)
		if err != nil {
			return err
		}
		for i := 0; i < len(t.Text); i++ {
			if err := ensureBSSValue(p, int64(t.Text[i]), ".ascii"); err != nil {
				return err
			}
		}
		out = append(out, t.Text...)
		if terminate {
			out = append(out, 0)
		}
		if !p.accept(COMMA) {
			break
		}
	}
	p.items = append(p.items, &DataBytes{Bytes: out, PC: p.pc, Line: p.line, Col: col, Section: p.section})
	p.pc += uint32(len(out))
	return nil
}

// parseEND stops assembling; the rest of the source, including the rest
// of the line, is ignored.
func parseEND(p *Parser) error {
	p.ended = true
	return nil
}
