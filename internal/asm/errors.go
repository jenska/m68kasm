package asm

import (
	"fmt"
	"strings"
)

// Error wraps an underlying failure with source location information so that
// callers can surface detailed diagnostics.
type Error struct {
	Line     int
	Col      int
	LineText string
	Err      error
}

func (e *Error) Error() string {
	loc := fmt.Sprintf("line %d", e.Line)
	if e.Col > 0 {
		loc += fmt.Sprintf(", col %d", e.Col)
	}
	if e.LineText != "" {
		msg := fmt.Sprintf("%s: %v\n    %s", loc, e.Err, e.LineText)
		if e.Col > 0 {
			msg += fmt.Sprintf("\n    %s^", strings.Repeat(" ", e.Col-1))
		}
		return msg
	}
	return fmt.Sprintf("%s: %v", loc, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

func errorAtToken(t Token, err error) error {
	return &Error{Line: t.Line, Col: t.Col, Err: err}
}

func errorAtLine(line int, err error) error {
	return &Error{Line: line, Err: err}
}

func contextualize(line int, err error) error {
	return contextualizeAt(line, 0, err)
}

func contextualizeAt(line, col int, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*Error); ok {
		return err
	}
	return &Error{Line: line, Col: col, Err: err}
}

func withSourceLines(err error, lines []string) error {
	if err == nil || len(lines) == 0 {
		return err
	}
	e, ok := err.(*Error)
	if !ok {
		return err
	}
	if e.LineText != "" {
		return e
	}
	if line := sourceLine(e.Line, lines); line != "" {
		e.LineText = line
	}
	return e
}

func sourceLine(line int, lines []string) string {
	if line <= 0 || line > len(lines) {
		return ""
	}
	return lines[line-1]
}
