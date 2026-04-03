package asm

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFileAndLexerCoverage(t *testing.T) {
	path := writeAsmTempFile(t, ".byte 1\n")
	prog, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	out, err := Assemble(prog)
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if got, want := out, []byte{0x01}; !bytes.Equal(got, want) {
		t.Fatalf("unexpected bytes: got %x want %x", got, want)
	}

	path = writeAsmTempFile(t, ".word FOO\n")
	prog, err = ParseFileWithOptions(path, ParseOptions{Symbols: map[string]uint32{"FOO": 0x1234}})
	if err != nil {
		t.Fatalf("ParseFileWithOptions failed: %v", err)
	}
	out, err = Assemble(prog)
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if got, want := out, []byte{0x12, 0x34}; !bytes.Equal(got, want) {
		t.Fatalf("unexpected bytes: got %x want %x", got, want)
	}

	lx := NewLexer(strings.NewReader("\"line\\n\"\n'A'\n'\\t'\n"))
	if tok := lx.Peek(); tok.Kind != STRING || tok.Text != "line\n" {
		t.Fatalf("unexpected peek token: %+v", tok)
	}
	if tok := lx.Next(); tok.Kind != STRING || tok.Text != "line\n" {
		t.Fatalf("unexpected string token: %+v", tok)
	}
	if got := STRING.String(); got != "string" {
		t.Fatalf("unexpected kind string: %q", got)
	}
	if tok := lx.Next(); tok.Kind != NEWLINE {
		t.Fatalf("expected newline, got %+v", tok)
	}
	if tok := lx.Next(); tok.Kind != NUMBER || tok.Val != 'A' || !strings.Contains(tok.String(), "'A'") {
		t.Fatalf("unexpected char token: %+v", tok)
	}
	_ = lx.Next()
	if tok := lx.Next(); tok.Kind != NUMBER || tok.Val != '\t' {
		t.Fatalf("unexpected escaped char token: %+v", tok)
	}
	if tok := NewLexer(strings.NewReader("\"unterminated")).Next(); tok.Kind != EOF || !strings.Contains(tok.Text, "unterminated string") {
		t.Fatalf("expected unterminated string error token, got %+v", tok)
	}

	if tok := NewLexer(strings.NewReader("101")).scanNumber('%'); tok.Kind != NUMBER || tok.Val != 5 {
		t.Fatalf("unexpected binary token: %+v", tok)
	}
	if tok := NewLexer(strings.NewReader("77")).scanNumber('@'); tok.Kind != NUMBER || tok.Val != 63 {
		t.Fatalf("unexpected octal token: %+v", tok)
	}
}

func writeAsmTempFile(t *testing.T, src string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.s")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write temp source: %v", err)
	}
	return path
}
