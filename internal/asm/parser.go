package asm

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

var branchCondMap = map[string]instructions.Cond{
	"BRA": instructions.CondT,
	"BSR": instructions.CondBSR,
	"BHI": instructions.CondHI,
	"BLS": instructions.CondLS,
	"BCC": instructions.CondCC,
	"BHS": instructions.CondCC,
	"BCS": instructions.CondCS,
	"BLO": instructions.CondCS,
	"BNE": instructions.CondNE,
	"BEQ": instructions.CondEQ,
	"BVC": instructions.CondVC,
	"BVS": instructions.CondVS,
	"BPL": instructions.CondPL,
	"BMI": instructions.CondMI,
	"BGE": instructions.CondGE,
	"BLT": instructions.CondLT,
	"BGT": instructions.CondGT,
	"BLE": instructions.CondLE,
}

type (
	DataBytes struct {
		Bytes []byte
		PC    uint32
		Line  int
	}

	Parser struct {
		lx     *Lexer
		labels map[string]uint32
		pc     uint32
		items  []any
		line   int
		col    int

		buf []Token // N-Token Lookahead
	}
)

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

func branchMnemonicInfo(text string) (string, instructions.Cond, bool) {
	name := strings.ToUpper(text)
	if idx := strings.IndexRune(name, '.'); idx > 0 {
		name = name[:idx]
	}
	cond, ok := branchCondMap[name]
	if !ok {
		return "", 0, false
	}
	return name, cond, true
}

func branchDefaultSize(name string) instructions.Size {
	if name == "BSR" {
		return instructions.SZ_W
	}
	return instructions.SZ_B
}

func ParseFile(path string) (*Program, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return Parse(f)
}

func (p *Parser) parseStmt() error {
	t := p.peek()
	if t.Kind == IDENT {
		if name, cond, ok := branchMnemonicInfo(t.Text); ok {
			return p.parseBranch(name, cond)
		}
		switch kwOf(t.Text) {
		case KW_MOVEQ:
			return p.parseMOVEQ()
		case KW_MOVE:
			return p.parseMOVE()
		case KW_ADD:
			return p.parseADD()
		case KW_SUB:
			return p.parseSUB()
		case KW_MULTI:
			return p.parseMULTI()
		case KW_DIV:
			return p.parseDIV()
		case KW_CMP:
			return p.parseCMP()
		case KW_LEA:
			return p.parseLEA()
		case KW_NONE:
			return fmt.Errorf("line %d: unknown mnemonic: %s", t.Line, t.Text)
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
		case KW_WORD:
			return p.parseWORD()
		case KW_LONG:
			return p.parseLONG()
		case KW_ALIGN:
			return p.parseALIGN()
		default:
			return fmt.Errorf("line %d: unknown pseudo op: %s", t.Line, name)
		}
	}
	return fmt.Errorf("line %d: unexpected token: %v", t.Line, t.Text)
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

// .word <expr>[, <expr>]...
func (p *Parser) parseWORD() error {
	// first value
	v, err := p.parseExpr()
	if err != nil {
		return err
	}

	out := make([]byte, 0, 2*4)
	if v < -0x8000 || v > 0xFFFF {
		return fmt.Errorf("line %d: .word value out of 16-bit range: %d", p.line, v)
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
			return fmt.Errorf("line %d: .word value out of 16-bit range: %d", p.line, v)
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
		return fmt.Errorf("line %d: .long value out of range: %d", p.line, v)
	}
	u := uint32(v) // two's complement when v is negative
	out = append(out, byte(u>>24), byte(u>>16), byte(u>>8), byte(u))

	for p.accept(COMMA) {
		v, err := p.parseExpr()
		if err != nil {
			return err
		}
		if v < -0x80000000 || v > 0xFFFFFFFF {
			return fmt.Errorf("line %d: .long value out of range: %d", p.line, v)
		}
		u := uint32(v)
		out = append(out, byte(u>>24), byte(u>>16), byte(u>>8), byte(u))
	}

	p.items = append(p.items, &DataBytes{Bytes: out, PC: p.pc, Line: p.line})
	p.pc += uint32(len(out))
	return nil
}

func (p *Parser) parseMOVEQ() error {
	mn, err := p.want(IDENT)
	if err != nil {
		return err
	}
	if !strings.EqualFold(mn.Text, "MOVEQ") {
		return fmt.Errorf("line %d: MOVEQ expected", mn.Line)
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
		return fmt.Errorf("line %d: expected Dn, got %s", dst.Line, dst.Text)
	}
	ins := &Instr{Op: instructions.OP_MOVEQ, Mnemonic: "MOVEQ", Size: instructions.SZ_L, Args: instructions.Args{HasImm: true, Imm: int64(int8(imm)), Dn: dn, Src: instructions.EAExpr{Kind: instructions.EAkImm, Imm: int64(int8(imm))}, Dst: instructions.EAExpr{Kind: instructions.EAkDn, Reg: dn}, Size: instructions.SZ_L}, PC: p.pc, Line: mn.Line}
	p.items = append(p.items, ins)
	p.pc += 2
	return nil
}

func (p *Parser) parseMOVE() error {
	mn, err := p.want(IDENT)
	if err != nil {
		return err
	}
	sz, err := p.parseSizeSpec(mn, instructions.SZ_W, []instructions.Size{instructions.SZ_B, instructions.SZ_W, instructions.SZ_L})
	if err != nil {
		return err
	}
	src, err := p.parseEA()
	if err != nil {
		return err
	}
	if _, err := p.want(COMMA); err != nil {
		return err
	}
	dst, err := p.parseEA()
	if err != nil {
		return err
	}
	args := instructions.Args{Src: src, Dst: dst, Size: sz}
	if src.Kind == instructions.EAkImm {
		args.Imm = src.Imm
	}
	if dst.Kind == instructions.EAkDn {
		args.Dn = dst.Reg
	} else if dst.Kind == instructions.EAkAn {
		args.An = dst.Reg
	}
	ins := &Instr{Op: instructions.OP_MOVE, Mnemonic: "MOVE", Size: sz, Args: args, PC: p.pc, Line: mn.Line}
	p.items = append(p.items, ins)
	words := 1 + eaExtraWords(src, sz, true) + eaExtraWords(dst, sz, false)
	p.pc += uint32(words * 2)
	return nil
}

func (p *Parser) parseADD() error {
	mn, err := p.want(IDENT)
	if err != nil {
		return err
	}
	sz, err := p.parseSizeSpec(mn, instructions.SZ_W, []instructions.Size{instructions.SZ_B, instructions.SZ_W, instructions.SZ_L})
	if err != nil {
		return err
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
	ok, dn := isRegDn(dst.Text)
	if !ok {
		return fmt.Errorf("line %d: expected Dn, got %s", dst.Line, dst.Text)
	}
	args := instructions.Args{Src: src, Dst: instructions.EAExpr{Kind: instructions.EAkDn, Reg: dn}, Dn: dn, Size: sz}
	if src.Kind == instructions.EAkImm {
		args.Imm = src.Imm
	}
	ins := &Instr{Op: instructions.OP_ADD, Mnemonic: "ADD", Size: sz, Args: args, PC: p.pc, Line: mn.Line}
	p.items = append(p.items, ins)
	words := 1 + eaExtraWords(src, sz, true)
	p.pc += uint32(words * 2)
	return nil
}

func (p *Parser) parseSUB() error {
	mn, err := p.want(IDENT)
	if err != nil {
		return err
	}
	sz, err := p.parseSizeSpec(mn, instructions.SZ_W, []instructions.Size{instructions.SZ_B, instructions.SZ_W, instructions.SZ_L})
	if err != nil {
		return err
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
	ok, dn := isRegDn(dst.Text)
	if !ok {
		return fmt.Errorf("line %d: expected Dn, got %s", dst.Line, dst.Text)
	}
	args := instructions.Args{Src: src, Dst: instructions.EAExpr{Kind: instructions.EAkDn, Reg: dn}, Dn: dn, Size: sz}
	if src.Kind == instructions.EAkImm {
		args.Imm = src.Imm
	}
	ins := &Instr{Op: instructions.OP_SUB, Mnemonic: "SUB", Size: sz, Args: args, PC: p.pc, Line: mn.Line}
	p.items = append(p.items, ins)
	words := 1 + eaExtraWords(src, sz, true)
	p.pc += uint32(words * 2)
	return nil
}

func (p *Parser) parseMULTI() error {
	mn, err := p.want(IDENT)
	if err != nil {
		return err
	}
	sz, err := p.parseSizeSpec(mn, instructions.SZ_W, []instructions.Size{instructions.SZ_W})
	if err != nil {
		return err
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
	ok, dn := isRegDn(dst.Text)
	if !ok {
		return fmt.Errorf("line %d: expected Dn, got %s", dst.Line, dst.Text)
	}
	args := instructions.Args{Src: src, Dst: instructions.EAExpr{Kind: instructions.EAkDn, Reg: dn}, Dn: dn, Size: sz}
	if src.Kind == instructions.EAkImm {
		args.Imm = src.Imm
	}
	ins := &Instr{Op: instructions.OP_MULTI, Mnemonic: "MULTI", Size: sz, Args: args, PC: p.pc, Line: mn.Line}
	p.items = append(p.items, ins)
	words := 1 + eaExtraWords(src, sz, true)
	p.pc += uint32(words * 2)
	return nil
}

func (p *Parser) parseDIV() error {
	mn, err := p.want(IDENT)
	if err != nil {
		return err
	}
	sz, err := p.parseSizeSpec(mn, instructions.SZ_W, []instructions.Size{instructions.SZ_W})
	if err != nil {
		return err
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
	ok, dn := isRegDn(dst.Text)
	if !ok {
		return fmt.Errorf("line %d: expected Dn, got %s", dst.Line, dst.Text)
	}
	args := instructions.Args{Src: src, Dst: instructions.EAExpr{Kind: instructions.EAkDn, Reg: dn}, Dn: dn, Size: sz}
	if src.Kind == instructions.EAkImm {
		args.Imm = src.Imm
	}
	ins := &Instr{Op: instructions.OP_DIV, Mnemonic: "DIV", Size: sz, Args: args, PC: p.pc, Line: mn.Line}
	p.items = append(p.items, ins)
	words := 1 + eaExtraWords(src, sz, true)
	p.pc += uint32(words * 2)
	return nil
}

func (p *Parser) parseCMP() error {
	mn, err := p.want(IDENT)
	if err != nil {
		return err
	}
	sz, err := p.parseSizeSpec(mn, instructions.SZ_W, []instructions.Size{instructions.SZ_B, instructions.SZ_W, instructions.SZ_L})
	if err != nil {
		return err
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
	ok, dn := isRegDn(dst.Text)
	if !ok {
		return fmt.Errorf("line %d: expected Dn, got %s", dst.Line, dst.Text)
	}
	args := instructions.Args{Src: src, Dst: instructions.EAExpr{Kind: instructions.EAkDn, Reg: dn}, Dn: dn, Size: sz}
	if src.Kind == instructions.EAkImm {
		args.Imm = src.Imm
	}
	ins := &Instr{Op: instructions.OP_CMP, Mnemonic: "CMP", Size: sz, Args: args, PC: p.pc, Line: mn.Line}
	p.items = append(p.items, ins)
	words := 1 + eaExtraWords(src, sz, true)
	p.pc += uint32(words * 2)
	return nil
}

func (p *Parser) parseBranch(name string, cond instructions.Cond) error {
	mn, err := p.want(IDENT)
	if err != nil {
		return err
	}
	actualName, _, ok := branchMnemonicInfo(mn.Text)
	if !ok || actualName != name {
		return fmt.Errorf("line %d: %s expected", mn.Line, name)
	}
	sz, err := p.parseSizeSpec(mn, branchDefaultSize(name), []instructions.Size{instructions.SZ_B, instructions.SZ_W})
	if err != nil {
		return err
	}
	lbl, err := p.want(IDENT)
	if err != nil {
		return err
	}
	args := instructions.Args{Target: lbl.Text, Cond: cond, Size: sz}
	ins := &Instr{Op: instructions.OP_BCC, Mnemonic: name, Size: sz, Args: args, PC: p.pc, Line: mn.Line}
	p.items = append(p.items, ins)
	words := 1
	if sz == instructions.SZ_W {
		words = 2
	}
	p.pc += uint32(words * 2)
	return nil
}

func (p *Parser) parseLEA() error {
	mn, err := p.want(IDENT)
	if err != nil {
		return err
	}
	if !strings.EqualFold(mn.Text, "LEA") {
		return fmt.Errorf("line %d: LEA expected", mn.Line)
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
		return fmt.Errorf("line %d: expected An, got %s", dst.Line, dst.Text)
	}
	ins := &Instr{Op: instructions.OP_LEA, Mnemonic: "LEA", Size: instructions.SZ_L, Args: instructions.Args{Src: src, An: an, Dst: instructions.EAExpr{Kind: instructions.EAkAn, Reg: an}, Size: instructions.SZ_L}, PC: p.pc, Line: mn.Line}
	p.items = append(p.items, ins)
	words := 1 + eaExtraWords(src, instructions.SZ_L, true)
	p.pc += uint32(words * 2)
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
			return 0, fmt.Errorf("line %d: unknown size suffix", mn.Line)
		}
		sz, ok := sizeFromIdent(suf)
		if !ok {
			return 0, fmt.Errorf("line %d: unknown size suffix .%s", mn.Line, suf)
		}
		if !sizeAllowedList(sz, allowed) {
			return 0, fmt.Errorf("line %d: illegal size for instruction", mn.Line)
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
			return 0, fmt.Errorf("line %d: unknown size suffix .%s", id.Line, id.Text)
		}
		sz = val
	}
	if !sizeAllowedList(sz, allowed) {
		return 0, fmt.Errorf("line %d: illegal size for instruction", p.line)
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

func eaExtraWords(e instructions.EAExpr, sz instructions.Size, source bool) int {
	switch e.Kind {
	case instructions.EAkAddrDisp16, instructions.EAkPCDisp16, instructions.EAkIdxAnBrief, instructions.EAkIdxPCBrief, instructions.EAkAbsW:
		return 1
	case instructions.EAkAbsL:
		return 2
	case instructions.EAkImm:
		if !source {
			return 0
		}
		if sz == instructions.SZ_L {
			return 2
		}
		return 1
	default:
		return 0
	}
}

// ---------- EA parsing ----------
func (p *Parser) parseEA() (instructions.EAExpr, error) {
	t := p.peek()
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
	}
	if p.accept(LPAREN) {
		// (An)
		if p.peek().Kind == IDENT {
			id := p.next()
			if ok, an := isRegAn(id.Text); ok {
				if _, err := p.want(RPAREN); err != nil {
					return instructions.EAExpr{}, err
				}
				return instructions.EAExpr{Kind: instructions.EAkAddrInd, Reg: an}, nil
			}
			return instructions.EAExpr{}, fmt.Errorf("line %d: unexpected EA, expected (An) or (disp,An/PC)", id.Line)
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
					return instructions.EAExpr{}, fmt.Errorf("line %d: expected index register Dn/An", idxTok.Line)
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
				return instructions.EAExpr{}, fmt.Errorf("line %d: base must be An or PC", base.Line)
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
			return instructions.EAExpr{}, fmt.Errorf("line %d: base must be An or PC", base.Line)
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
	return instructions.EAExpr{}, fmt.Errorf("line %d: unexpected EA", t.Line)
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
	pad := int(align - m)

	// emit pad bytes of 'fill'
	buf := make([]byte, pad)
	if fill != 0 {
		for i := range buf {
			buf[i] = fill
		}
	}
	p.items = append(p.items, &DataBytes{Bytes: buf, PC: p.pc, Line: p.line})
	p.pc += uint32(pad)
	return nil
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
