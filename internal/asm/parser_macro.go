package asm

import (
	"fmt"
)

func (p *Parser) consumeUntilEOL() []Token {
	tokens := p.tokenScratch[:0]
	if cap(tokens) == 0 {
		tokens = make([]Token, 0, 8)
	}
	for {
		t := p.peek()
		if t.Kind == NEWLINE || t.Kind == EOF {
			break
		}
		tokens = append(tokens, p.next())
	}
	return tokens
}

func (p *Parser) releaseTokens(tokens []Token) {
	p.tokenScratch = tokens[:0]
}

func (p *Parser) prependTokens(tokens []Token) {
	if len(tokens) == 0 {
		return
	}
	buf := make([]Token, len(tokens)+len(p.buf))
	copy(buf, tokens)
	copy(buf[len(tokens):], p.buf)
	p.buf = buf
}

func (p *Parser) invokeMacro(def macroDef) (bool, error) {
	nameTok := p.next()
	rawArgs := p.consumeUntilEOL()
	defer p.releaseTokens(rawArgs)

	args, err := splitMacroArgs(rawArgs)
	if err != nil {
		return false, errorAtLine(nameTok.Line, err)
	}
	if len(args) != len(def.params) {
		return false, errorAtLine(nameTok.Line, fmt.Errorf("macro %s expects %d args, got %d", nameTok.Text, len(def.params), len(args)))
	}

	if p.macroDepth > 64 {
		return false, errorAtLine(nameTok.Line, fmt.Errorf("macro expansion depth exceeded"))
	}
	p.macroDepth++
	defer func() { p.macroDepth-- }()

	expanded := expandMacroBody(def, args, nameTok)
	p.prependTokens(expanded)
	return true, nil
}

func expandMacroBody(def macroDef, args [][]Token, origin Token) []Token {
	expanded := make([]Token, 0, len(def.body)+len(args))
	for _, t := range def.body {
		if repl, ok := macroArgumentTokens(def.params, args, t.Text); ok && t.Kind == IDENT {
			for _, at := range repl {
				expanded = append(expanded, relocatedToken(at, origin))
			}
			continue
		}
		expanded = append(expanded, relocatedToken(t, origin))
	}
	return expanded
}

func macroArgumentTokens(params []string, args [][]Token, name string) ([]Token, bool) {
	for i, param := range params {
		if name == param {
			return args[i], true
		}
	}
	return nil, false
}

func relocatedToken(tok Token, origin Token) Token {
	tok.Line = origin.Line
	tok.Col = origin.Col
	return tok
}

func splitMacroArgs(tokens []Token) ([][]Token, error) {
	if len(tokens) == 0 {
		return [][]Token{}, nil
	}
	args := [][]Token{}
	current := []Token{}
	depth := 0
	for _, t := range tokens {
		switch t.Kind {
		case LPAREN:
			depth++
		case RPAREN:
			if depth == 0 {
				return nil, fmt.Errorf("unmatched ')'")
			}
			depth--
		case COMMA:
			if depth == 0 {
				args = append(args, append([]Token{}, current...))
				current = current[:0]
				continue
			}
		}
		current = append(current, t)
	}
	if depth != 0 {
		return nil, fmt.Errorf("unmatched '('")
	}
	args = append(args, append([]Token{}, current...))
	return args, nil
}

func withSliceLexer(tokens []Token, line int, scratch []Token) (*sliceLexer, []Token) {
	tmp := append(scratch[:0], tokens...)
	tmp = append(tmp, Token{Kind: EOF, Line: line})
	return &sliceLexer{tokens: tmp}, tmp
}
