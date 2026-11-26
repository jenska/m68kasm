package asm

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// maxIncludeDepth bounds nested INCLUDEs, which also stops a file that
// includes itself.
const maxIncludeDepth = 32

// includeLexer reads the tokens of an included file. It reports every token
// at the position of the INCLUDE statement, the way a macro expansion does,
// so errors and listing entries point into the including source (the only
// one the program keeps source lines for). The first lexer error, which the
// lexer reports as an EOF token carrying the message, is kept in err.
type includeLexer struct {
	inner     *Lexer
	line, col int
	err       error
}

func (l *includeLexer) Next() Token {
	t := l.inner.Next()
	if t.Kind == EOF && t.Text != "" && l.err == nil {
		l.err = fmt.Errorf("line %d: %s", t.Line, t.Text)
	}
	t.Line, t.Col = l.line, l.col
	return t
}

// resolveIncludePath returns the file a directive's string operand names:
// relative paths are relative to the directory of the including source.
func (p *Parser) resolveIncludePath(directive string, name Token) (string, error) {
	if name.Text == "" {
		return "", errorAtToken(name, fmt.Errorf("%s: empty file name", directive))
	}
	path := filepath.FromSlash(name.Text)
	if filepath.IsAbs(path) {
		return path, nil
	}
	if p.baseDir == "" {
		return "", errorAtToken(name, fmt.Errorf("%s %q: the source has no directory to resolve it against (assemble a file, or set ParseOptions.BaseDir)", directive, name.Text))
	}
	return filepath.Join(p.baseDir, path), nil
}

// readIncludeFile reads the file for an INCLUDE or INCBIN operand.
func (p *Parser) readIncludeFile(directive string) (Token, string, []byte, error) {
	name, err := p.want(STRING)
	if err != nil {
		return name, "", nil, err
	}
	path, err := p.resolveIncludePath(directive, name)
	if err != nil {
		return name, "", nil, err
	}
	if t := p.peek(); t.Kind != NEWLINE && t.Kind != EOF {
		return name, "", nil, errorAtToken(t, fmt.Errorf("unexpected token: %s", t.Text))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return name, "", nil, errorAtToken(name, fmt.Errorf("%s: %v", directive, err))
	}
	if uint64(len(data)) > uint64(maxProgramSize) {
		return name, "", nil, errorAtToken(name, fmt.Errorf("%s %q would exceed maximum program size of %d bytes", directive, name.Text, maxProgramSize))
	}
	return name, path, data, nil
}

// parseINCLUDE parses INCLUDE "file": the statements of the file are
// assembled at this point, sharing symbols, macros and the location counter
// with the including source. Relative names are relative to the including
// file; END in an included file ends the whole assembly.
func parseINCLUDE(p *Parser) error {
	name, path, data, err := p.readIncludeFile("INCLUDE")
	if err != nil {
		return err
	}
	if p.includeDepth >= maxIncludeDepth {
		return errorAtToken(name, fmt.Errorf("INCLUDE %q: nested more than %d deep (does the file include itself?)", name.Text, maxIncludeDepth))
	}

	lx, buf, baseDir, line, col := p.lx, p.buf, p.baseDir, p.line, p.col
	inc := &includeLexer{inner: NewLexer(bytes.NewReader(data)), line: name.Line, col: name.Col}
	p.lx, p.buf, p.baseDir = inc, nil, filepath.Dir(path)
	p.includeDepth++
	err = p.parseStatements()
	p.includeDepth--
	p.lx, p.buf, p.baseDir, p.line, p.col = lx, buf, baseDir, line, col

	if err == nil && !p.ended {
		err = inc.err
	}
	if err != nil {
		var perr *Error
		msg := err.Error()
		if errors.As(err, &perr) {
			msg = perr.Message()
		}
		return &Error{Line: name.Line, Col: name.Col, Err: fmt.Errorf("in included file %q: %s", name.Text, msg)}
	}
	return nil
}

// parseINCBIN parses INCBIN "file": the contents of the file are emitted as
// data bytes.
func parseINCBIN(p *Parser) error {
	name, _, data, err := p.readIncludeFile("INCBIN")
	if err != nil {
		return err
	}
	col := p.col
	if p.section == SectionBSS {
		for _, b := range data {
			if b != 0 {
				return errorAtToken(name, fmt.Errorf("INCBIN in %s must be zero-initialized", p.section.Name()))
			}
		}
	}
	if len(data) == 0 {
		return nil
	}
	n := uint32(len(data))
	if p.pc > maxProgramSize-n {
		return errorAtToken(name, fmt.Errorf("INCBIN would exceed maximum program size of %d bytes", maxProgramSize))
	}
	p.items = append(p.items, &DataBytes{Bytes: data, PC: p.pc, Line: p.line, Col: col, Section: p.section})
	p.pc += n
	return nil
}
