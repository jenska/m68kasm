package asm

import (
	"fmt"
	"math"
	"strings"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

func (p *Parser) parseStmt() (bool, error) {
	t := p.peek()

	if t.Kind == IDENT {
		if def, ok := p.macros[t.Text]; ok {
			return p.invokeMacro(def)
		}
		if p.isConstDefinitionStart() {
			return false, p.parseConstDefinition(t)
		}
		base, suffix := splitMnemonic(t.Text)
		if instrDef := p.instrs.Lookup(base); instrDef != nil {
			return false, p.parseInstruction(instrDef)
		}

		if base == "DC" && suffix != "" {
			_ = p.next()
			return false, parseDC(p, suffix)
		}

		if pseudo, ok := lookupPseudo(t.Text); ok {
			_ = p.next()
			return false, pseudo(p)
		}
		return false, parserError(t, "unknown mnemonic")
	}

	if t.Kind == DOT {
		p.next()
		id, err := p.want(IDENT)
		if err != nil {
			return false, err
		}
		name := "." + strings.ToUpper(id.Text)
		if pseudo, ok := pseudoMap[name]; ok {
			return false, pseudo(p)
		}
		return false, parserError(t, "unknown pseudo op")
	}
	return false, parserError(t, "unexpected token")
}

func (p *Parser) isConstDefinitionStart() bool {
	if p.peekN(2).Kind == EQUAL {
		return true
	}
	return p.peekN(2).Kind == DOT && isEquToken(p.peekN(3))
}

func splitMnemonic(text string) (base, suffix string) {
	upper := strings.ToUpper(text)
	if before, after, ok := strings.Cut(upper, "."); ok {
		return before, after
	}
	return upper, ""
}

func lookupPseudo(text string) (func(*Parser) error, bool) {
	pseudo, ok := pseudoMap["."+strings.ToUpper(text)]
	return pseudo, ok
}

func isEquToken(tok Token) bool {
	return tok.Kind == IDENT && strings.EqualFold(tok.Text, "equ")
}

func (p *Parser) parseConstDefinition(nameTok Token) error {
	_ = p.next() // consume name

	if p.peek().Kind == DOT {
		p.next()
		eqTok, err := p.want(IDENT)
		if err != nil {
			return err
		}
		if !isEquToken(eqTok) {
			return parserError(eqTok, "expected EQU")
		}
	} else {
		if _, err := p.want(EQUAL); err != nil {
			return err
		}
	}

	val, err := p.parseExpr()
	if err != nil {
		return err
	}
	if val < 0 || val > math.MaxUint32 {
		return errorAtLine(nameTok.Line, fmt.Errorf("constant out of 32-bit range: %d", val))
	}

	p.labels[nameTok.Text] = uint32(val)
	return nil
}

func (p *Parser) parseInstruction(instrDef *instructions.InstrDef) error {
	if p.section == SectionBSS {
		return errorAtLine(p.line, fmt.Errorf("instructions are not allowed in %s", p.section.Name()))
	}

	mn, err := p.want(IDENT)
	if err != nil {
		return err
	}

	operandTokens := p.consumeUntilEOL()
	defer p.releaseTokens(operandTokens)
	var lastErr error
	for i := range instrDef.Forms {
		form := &instrDef.Forms[i]
		args, err := p.tryParseForm(mn, form, operandTokens)
		if err != nil {
			lastErr = err
			continue
		}
		ins := &Instr{Def: instrDef, Form: form, Args: args, PC: p.pc, Line: mn.Line, Col: mn.Col, Section: p.section}
		p.items = append(p.items, ins)
		words, err := instructionWords(form, args)
		if err != nil {
			return err
		}
		p.pc += uint32(words * 2)
		return nil
	}

	if lastErr != nil {
		return contextualizeAt(mn.Line, mn.Col, lastErr)
	}
	return &Error{Line: mn.Line, Col: mn.Col, Err: fmt.Errorf("no form matches operands")}
}

func instructionWords(form *instructions.FormDef, args instructions.Args) (int, error) {
	words := 0

	var srcEA, dstEA instructions.EAEncoded
	var err error

	if args.Src.Kind != instructions.EAkNone {
		srcEA, err = instructions.EncodeEA(args.Src, 0)
		if err != nil {
			return 0, err
		}
	}
	if args.Dst.Kind != instructions.EAkNone {
		dstEA, err = instructions.EncodeEA(args.Dst, 0)
		if err != nil {
			return 0, err
		}
	}

	for _, step := range form.Steps {
		haveWord := (step.WordBits != 0) || (len(step.Fields) > 0)
		if haveWord {
			words++
		}
		for _, tr := range step.Trailer {
			switch tr {
			case instructions.TSrcEAExt:
				words += len(srcEA.Ext)
			case instructions.TDstEAExt:
				words += len(dstEA.Ext)
			case instructions.TImmSized:
				words++
			case instructions.TSrcImm:
				if args.Src.Kind == instructions.EAkImm {
					switch args.Size {
					case instructions.LongSize:
						words += 2
					default:
						words++
					}
				}
			case instructions.TBranchWordIfNeeded:
				if args.Size == instructions.WordSize {
					words++
				}
			case instructions.TSrcRegMask, instructions.TDstRegMask:
				words++
			}
		}
	}

	return words, nil
}

// emitPaddingBytes centralizes emitting zero/filled padding while keeping the
// generated output within a safe upper bound. Without this guard, a crafted
// .org/.align could request gigabytes of padding and exhaust memory.
func (p *Parser) emitPaddingBytes(count uint32, fill byte) error {
	if count == 0 {
		return nil
	}
	if count > maxProgramSize || p.pc > maxProgramSize-count {
		return errorAtLine(p.line, fmt.Errorf("padding would exceed maximum program size of %d bytes", maxProgramSize))
	}

	buf := make([]byte, int(count))
	if fill != 0 {
		for i := range buf {
			buf[i] = fill
		}
	}
	p.items = append(p.items, &DataBytes{Bytes: buf, PC: p.pc, Line: p.line, Col: p.col, Section: p.section})
	p.pc += count
	return nil
}

func (p *Parser) setSection(section SectionKind) error {
	if section < p.section {
		return errorAtLine(p.line, fmt.Errorf("sections must stay in .text -> .data -> .bss order"))
	}
	p.section = section
	return nil
}
