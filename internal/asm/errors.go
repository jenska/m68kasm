package asm

import "fmt"

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
		return fmt.Sprintf("%s: %v\n    %s", loc, e.Err, e.LineText)
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
	if err == nil {
		return nil
	}
	if _, ok := err.(*Error); ok {
		return err
	}
	return errorAtLine(line, err)
}
