package asm

import (
	"fmt"
	"strings"
)

var pseudoMap = map[string]func(*Parser) error{
	".ORG":   parseORG,
	".BYTE":  parseBYTE,
	".WORD":  parseWORD,
	".LONG":  parseLONG,
	".ALIGN": parseALIGN,
	".EVEN":  parseEVEN,
	".MACRO": parseMACRO,
}

func parseORG(p *Parser) error {
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

func parseBYTE(p *Parser) error {
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
func parseWORD(p *Parser) error {
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
func parseLONG(p *Parser) error {
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

// .align <expr>[, <fill>]
func parseALIGN(p *Parser) error {
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
			return fmt.Errorf("line %d: unexpected EOF inside macro", nameTok.Line)
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
