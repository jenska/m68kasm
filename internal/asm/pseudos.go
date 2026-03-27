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

func parseBYTE(p *Parser) error {
	col := p.col
	val, err := p.parseExpr()
	if err != nil {
		return err
	}
	if err := ensureBSSValue(p, val, ".byte"); err != nil {
		return err
	}
	var bytes []byte
	bytes = append(bytes, byte(val))
	for p.accept(COMMA) {
		v, err := p.parseExpr()
		if err != nil {
			return err
		}
		if err := ensureBSSValue(p, v, ".byte"); err != nil {
			return err
		}
		bytes = append(bytes, byte(v))
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
