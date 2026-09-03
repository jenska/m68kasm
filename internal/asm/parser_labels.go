package asm

import (
	"fmt"
	"strings"
)

func localLabelName(num, idx int) string {
	return fmt.Sprintf("__local_%d_%d", num, idx)
}

func (p *Parser) defineLocalLabel(tok Token) error {
	num := int(tok.Val)
	if num < 0 {
		return parserError(tok, "local label must be non-negative")
	}
	idx := p.locals[num] + 1
	name := localLabelName(num, idx)
	p.locals[num] = idx
	p.labels[name] = p.pc
	return nil
}

func (p *Parser) resolveLocalLabel(num int, forward bool) (string, error) {
	if num < 0 {
		return "", fmt.Errorf("local label must be non-negative")
	}
	if forward {
		idx := p.locals[num] + 1
		if p.localForwards[num] < idx {
			p.localForwards[num] = idx
		}
		return localLabelName(num, idx), nil
	}
	if p.locals[num] == 0 {
		return "", fmt.Errorf("no previous local label %d", num)
	}
	return localLabelName(num, p.locals[num]), nil
}

func (p *Parser) ensureLocalForwardsResolved() error {
	for num, needed := range p.localForwards {
		if p.locals[num] < needed {
			return fmt.Errorf("no forward definition for local label %d", num)
		}
	}
	return nil
}

func parserError(t Token, msg string) error {
	return errorAtToken(t, fmt.Errorf("%s", msg))
}

// parseLabelDefinition checks for and consumes a label definition (IDENT: or NUMBER:).
// Returns true if a label was parsed.
func (p *Parser) parseLabelDefinition() (bool, error) {
	t := p.peek()
	if (t.Kind == IDENT || t.Kind == NUMBER) && p.peekN(2).Kind == COLON {
		lbl := p.next()
		p.next() // consume ':'
		if lbl.Kind == IDENT {
			p.labels[lbl.Text] = p.pc
			p.recordDefinedLabel(lbl.Text, p.pc, lbl.Line)
		} else {
			if err := p.defineLocalLabel(lbl); err != nil {
				return true, err
			}
		}
		return true, nil
	}
	return false, nil
}

func (p *Parser) recordDefinedLabel(name string, addr uint32, line int) {
	if idx, ok := p.definedLabelPos[name]; ok {
		p.definedLabels[idx].Addr = addr
		p.definedLabels[idx].Line = line
		p.definedLabels[idx].Section = p.section
		return
	}
	p.definedLabelPos[name] = len(p.definedLabels)
	p.definedLabels = append(p.definedLabels, DefinedLabel{Name: name, Addr: addr, Line: line, Section: p.section})
}

func (p *Parser) consumeLocalLabelRef() (string, bool, error) {
	numTok := p.peek()
	dirTok := p.peekN(2)
	if numTok.Kind != NUMBER || dirTok.Kind != IDENT {
		return "", false, nil
	}
	dir := strings.ToLower(dirTok.Text)
	if dir != "f" && dir != "b" {
		return "", false, nil
	}
	p.next() // number
	p.next() // direction
	name, err := p.resolveLocalLabel(int(numTok.Val), dir == "f")
	return name, true, err
}

func (p *Parser) parseLabelReference() (string, error) {
	if p.peek().Kind == IDENT {
		return p.next().Text, nil
	}
	if name, ok, err := p.consumeLocalLabelRef(); ok {
		return name, err
	}
	t := p.next()
	return "", parserError(t, "expected label")
}
