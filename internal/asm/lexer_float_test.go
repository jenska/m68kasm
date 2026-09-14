package asm

import (
	"strings"
	"testing"
)

// TestLexerScansFloatLiterals guards scanNumber's new "<digits>[.<digits>]
// [eE[+-]<digits>]" float detection (milestone 27, FPU immediate
// literals): each of these must come back as a single NUMBER token with
// IsFloat set and the expected value, not split into separate NUMBER/
// DOT/IDENT tokens.
func TestLexerScansFloatLiterals(t *testing.T) {
	tests := []struct {
		src  string
		want float64
	}{
		{"1.5", 1.5},
		{"0.5", 0.5},
		{"3.14159", 3.14159},
		{"1e10", 1e10},
		{"1.5e10", 1.5e10},
		{"1.5e+10", 1.5e10},
		{"1.5e-10", 1.5e-10},
		{"1E5", 1e5},
	}
	for _, tc := range tests {
		t.Run(tc.src, func(t *testing.T) {
			tok := NewLexer(strings.NewReader(tc.src)).Next()
			if tok.Kind != NUMBER || !tok.IsFloat {
				t.Fatalf("Next() = %+v, want a float NUMBER token", tok)
			}
			if tok.FVal != tc.want {
				t.Fatalf("FVal = %v, want %v", tok.FVal, tc.want)
			}
		})
	}
}

// TestLexerBareTrailingDotStaysInteger guards the "don't consume unless
// certain" discipline scanNumber's float detection relies on: a digit
// run followed by '.' with no digit after it (e.g. a size suffix
// immediately after a plain integer, however unlikely) must NOT be
// swallowed into the number — the dot is left for its own DOT token.
func TestLexerBareTrailingDotStaysInteger(t *testing.T) {
	lx := NewLexer(strings.NewReader("5.L"))
	tok := lx.Next()
	if tok.Kind != NUMBER || tok.IsFloat || tok.Val != 5 {
		t.Fatalf("first token = %+v, want plain integer NUMBER 5", tok)
	}
	dot := lx.Next()
	if dot.Kind != DOT {
		t.Fatalf("second token = %+v, want DOT", dot)
	}
}

// TestLexerBareTrailingEStaysInteger is TestLexerBareTrailingDotStaysInteger's
// exponent-marker counterpart: "5end" must not misparse "e" as an
// exponent marker just because it's followed by more identifier
// characters that happen to start with a non-digit.
func TestLexerBareTrailingEStaysInteger(t *testing.T) {
	lx := NewLexer(strings.NewReader("5end"))
	tok := lx.Next()
	if tok.Kind != NUMBER || tok.IsFloat || tok.Val != 5 {
		t.Fatalf("first token = %+v, want plain integer NUMBER 5", tok)
	}
	ident := lx.Next()
	if ident.Kind != IDENT || ident.Text != "end" {
		t.Fatalf("second token = %+v, want IDENT \"end\"", ident)
	}
}

// TestLexerHexBinaryOctalStayInteger guards that float-literal detection
// is decimal-only: $/%/@-prefixed numbers never gain a fractional part,
// matching real-world assembler convention (and avoiding ambiguity with
// a following '.' that means something else, e.g. a size suffix).
func TestLexerHexBinaryOctalStayInteger(t *testing.T) {
	for _, src := range []string{"$FF", "%101", "@17"} {
		t.Run(src, func(t *testing.T) {
			tok := NewLexer(strings.NewReader(src)).Next()
			if tok.Kind != NUMBER || tok.IsFloat {
				t.Fatalf("Next() = %+v, want a plain integer NUMBER token", tok)
			}
		})
	}
}

// TestLexerBinaryLiteralReachableFromNext guards a real, unrelated bug
// found while working on this file: '%' had its own unconditional
// "case '%':" in next()'s main switch returning a bare PERCENT operator
// token, before ever reaching the default branch that would dispatch to
// scanNumber's own (already-correct, but dead from here) '%'-prefix
// binary-literal handling — so docs/syntax.md's documented "%10100110"
// binary syntax could never actually be lexed as a number through the
// real entry point, only through a white-box unit test that calls
// scanNumber directly (TestParseFileAndLexerCoverage, coverage_
// additional_test.go). Fixed the same way '$' already disambiguates
// itself: peek one rune ahead and only take the literal path when it's
// plausible.
func TestLexerBinaryLiteralReachableFromNext(t *testing.T) {
	tok := NewLexer(strings.NewReader("%1010")).Next()
	if tok.Kind != NUMBER || tok.IsFloat || tok.Val != 10 {
		t.Fatalf("Next() = %+v, want integer NUMBER 10", tok)
	}
}

// TestLexerModuloOperatorStillWorks guards the trade-off the fix above
// accepts: '%' immediately followed by a binary digit (0 or 1) is now a
// literal, not the modulo operator — matching '$'/hex's own existing
// precedent. Modulo by anything else is unaffected regardless of
// spacing (already covered by expr_enhanced_test.go's unspaced "5%3");
// a divisor that happens to start with 0 or 1 needs a space *after* the
// '%' to force the operator reading, since disambiguation looks at what
// follows '%', not what precedes it.
func TestLexerModuloOperatorStillWorks(t *testing.T) {
	lx := NewLexer(strings.NewReader("10 % 101"))
	if tok := lx.Next(); tok.Kind != NUMBER || tok.Val != 10 {
		t.Fatalf("first token = %+v, want integer NUMBER 10", tok)
	}
	if tok := lx.Next(); tok.Kind != PERCENT {
		t.Fatalf("second token = %+v, want PERCENT", tok)
	}
}
