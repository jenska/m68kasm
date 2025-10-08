package asm

import (
	"fmt"
	"os"
	"strings"
)

type Parser struct {
	lx     *Lexer
	labels map[string]uint32
	pc     uint32
	items  []any
	line   int
	col    int

	buf []Token // N-Token Lookahead
}

func ParseFile(path string) (*Program, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	p := &Parser{lx: New(f), labels: map[string]uint32{}}

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

		// Zeilenende verbrauchen
		for {
			t := p.peek()
			if t.Kind == NEWLINE || t.Kind == EOF {
				break
			}
			p.next()
		}
		if p.peek().Kind == NEWLINE {
			p.next()
		}
	}

	return &Program{Items: p.items, Labels: p.labels, Origin: 0}, nil
}

func (p *Parser) parseStmt() error {
	t := p.peek()
	if t.Kind == IDENT {
		switch kwOf(t.Text) {
		case KW_MOVEQ:
			return p.parseMOVEQ()
		case KW_LEA:
			return p.parseLEA()
		case KW_BRA:
			return p.parseBRA()
		case KW_NONE:
			return fmt.Errorf("Zeile %d: unbekannter Mnemonic: %s", t.Line, t.Text)
		}
	}
	if t.Kind == DOT || (t.Kind == IDENT && strings.HasPrefix(t.Text, ".")) {
		name := t.Text
		if t.Kind == DOT {
			p.next()
			id, err := p.want(IDENT)
			if err != nil {
				return err
			}
			name = "." + id.Text
		} else {
			p.next()
		}
		switch kwOf(name) {
		case KW_ORG:
			return p.parseORG()
		case KW_BYTE:
			return p.parseBYTE()
		default:
			return fmt.Errorf("Zeile %d: unbekannte Pseudoop: %s", t.Line, name)
		}
	}
	return fmt.Errorf("Zeile %d: unerwartetes Token: %v", t.Line, t.Text)
}

func (p *Parser) parseORG() error {
	val, err := p.parseExpr()
	if err != nil {
		return err
	}
	p.pc = uint32(val)
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

func (p *Parser) parseMOVEQ() error {
	mn, err := p.want(IDENT)
	if err != nil {
		return err
	}
	if !strings.EqualFold(mn.Text, "MOVEQ") {
		return fmt.Errorf("Zeile %d: MOVEQ erwartet", mn.Line)
	}
	if _, err := p.want(HASH); err != nil {
		return err
	}
	imm, err := p.parseExpr()
	if err != nil {
		return err
	}
	if _, err := p.want(COMMA); err != nil {
		return err
	}
	dst, err := p.want(IDENT)
	if err != nil {
		return err
	}
	ok, dn := isRegDn(dst.Text)
	if !ok {
		return fmt.Errorf("Zeile %d: erwartetes Dn, got %s", dst.Line, dst.Text)
	}
	ins := &Instr{Op: OP_MOVEQ, Mnemonic: "MOVEQ", Size: SZ_L, Args: Args{HasImm: true, Imm: int64(int8(imm)), Dn: dn}, PC: p.pc, Line: mn.Line}
	p.items = append(p.items, ins)
	p.pc += 2
	return nil
}

func (p *Parser) parseBRA() error {
	mn, err := p.want(IDENT)
	if err != nil {
		return err
	}
	if !strings.EqualFold(mn.Text, "BRA") {
		return fmt.Errorf("Zeile %d: BRA erwartet", mn.Line)
	}
	lbl, err := p.want(IDENT)
	if err != nil {
		return err
	}
	ins := &Instr{Op: OP_BCC, Mnemonic: "BRA", Size: SZ_B, Args: Args{Target: lbl.Text, Cond: CondT}, PC: p.pc, Line: mn.Line}
	p.items = append(p.items, ins)
	p.pc += 2
	return nil
}

func (p *Parser) parseLEA() error {
	mn, err := p.want(IDENT)
	if err != nil {
		return err
	}
	if !strings.EqualFold(mn.Text, "LEA") {
		return fmt.Errorf("Zeile %d: LEA erwartet", mn.Line)
	}
	src, err := p.parseEA()
	if err != nil {
		return err
	}
	if _, err := p.want(COMMA); err != nil {
		return err
	}
	dst, err := p.want(IDENT)
	if err != nil {
		return err
	}
	ok, an := isRegAn(dst.Text)
	if !ok {
		return fmt.Errorf("Zeile %d: erwartetes An, got %s", dst.Line, dst.Text)
	}
	ins := &Instr{Op: OP_LEA, Mnemonic: "LEA", Size: SZ_L, Args: Args{Src: src, An: an}, PC: p.pc, Line: mn.Line}
	p.items = append(p.items, ins)
	p.pc += 4
	return nil
}

// ---------- EA parsing ----------
func (p *Parser) parseEA() (EAExpr, error) {
	t := p.peek()
	if t.Kind == HASH {
		p.next()
		v, err := p.parseExpr()
		if err != nil {
			return EAExpr{}, err
		}
		return EAExpr{Kind: EAkImm, Imm: v}, nil
	}
	if t.Kind == IDENT {
		if ok, dn := isRegDn(t.Text); ok {
			p.next()
			return EAExpr{Kind: EAkDn, Reg: dn}, nil
		}
		if ok, an := isRegAn(t.Text); ok {
			p.next()
			return EAExpr{Kind: EAkAn, Reg: an}, nil
		}
	}
	if p.accept(LPAREN) {
		// (An)
		if p.peek().Kind == IDENT {
			id := p.next()
			if ok, an := isRegAn(id.Text); ok {
				if _, err := p.want(RPAREN); err != nil {
					return EAExpr{}, err
				}
				return EAExpr{Kind: EAkAddrInd, Reg: an}, nil
			}
			return EAExpr{}, fmt.Errorf("Zeile %d: unerwartete EA, erwartete (An) oder (disp,An/PC)", id.Line)
		}
		first, err := p.parseExprUntil(COMMA, RPAREN)
		if err != nil {
			return EAExpr{}, err
		}
		if p.accept(COMMA) {
			base, err := p.want(IDENT)
			if err != nil {
				return EAExpr{}, err
			}
			if p.accept(COMMA) {
				idxTok, err := p.want(IDENT)
				if err != nil {
					return EAExpr{}, err
				}
				var ix EAIndex
				if ok, dn := isRegDn(idxTok.Text); ok {
					ix.Reg = dn
					ix.IsA = false
				} else if ok, an := isRegAn(idxTok.Text); ok {
					ix.Reg = an
					ix.IsA = true
				} else {
					return EAExpr{}, fmt.Errorf("Zeile %d: erwarteter Indexregister Dn/An", idxTok.Line)
				}
				ix.Long = false
				ix.Scale = 1
				if p.accept(STAR) {
					sc, err := p.parseExprUntil(COMMA, RPAREN)
					if err != nil {
						return EAExpr{}, err
					}
					switch sc {
					case 1, 2, 4, 8:
						ix.Scale = uint8(sc)
					default:
						return EAExpr{}, fmt.Errorf("ungültiger Scale-Faktor: %d", sc)
					}
				}
				ix.Disp8 = int8(first)
				if _, err := p.want(RPAREN); err != nil {
					return EAExpr{}, err
				}
				if ok, an := isRegAn(base.Text); ok {
					return EAExpr{Kind: EAkIdxAnBrief, Reg: an, Index: ix}, nil
				}
				if isPC(base.Text) {
					return EAExpr{Kind: EAkIdxPCBrief, Index: ix}, nil
				}
				return EAExpr{}, fmt.Errorf("Zeile %d: Basis muss An oder PC sein", base.Line)
			}
			if _, err := p.want(RPAREN); err != nil {
				return EAExpr{}, err
			}
			if ok, an := isRegAn(base.Text); ok {
				return EAExpr{Kind: EAkAddrDisp16, Reg: an, Disp16: int32(first)}, nil
			}
			if isPC(base.Text) {
				return EAExpr{Kind: EAkPCDisp16, Disp16: int32(first)}, nil
			}
			return EAExpr{}, fmt.Errorf("Zeile %d: Basis muss An oder PC sein", base.Line)
		}
		if _, err := p.want(RPAREN); err != nil {
			return EAExpr{}, err
		}
		if _, err := p.want(DOT); err != nil {
			return EAExpr{}, err
		}
		suf, err := p.want(IDENT)
		if err != nil {
			return EAExpr{}, err
		}
		if strings.EqualFold(suf.Text, "W") {
			return EAExpr{Kind: EAkAbsW, Abs16: uint16(first)}, nil
		}
		return EAExpr{Kind: EAkAbsL, Abs32: uint32(first)}, nil
	}
	return EAExpr{}, fmt.Errorf("Zeile %d: unerwartete EA", t.Line)
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
func (p *Parser) peek() Token       { p.fill(1); return p.buf[0] }
func (p *Parser) peekN(n int) Token { p.fill(n); return p.buf[n-1] }
func (p *Parser) want(k Kind) (Token, error) {
	t := p.next()
	if t.Kind != k {
		return t, fmt.Errorf("Zeile %d Spalte %d: erwartete %v, bekam %v (%q)", t.Line, t.Col, k, t.Kind, t.Text)
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
