package asm

import (
	"strings"
	"testing"
)

func BenchmarkParseAndAssemble(b *testing.B) {
	src := buildBenchmarkSource(512)
	b.ReportAllocs()
	b.SetBytes(int64(len(src)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prog, err := Parse(strings.NewReader(src))
		if err != nil {
			b.Fatalf("parse error: %v", err)
		}
		if _, err := Assemble(prog); err != nil {
			b.Fatalf("assemble error: %v", err)
		}
	}
}

func BenchmarkParseOnly(b *testing.B) {
	src := buildBenchmarkSource(512)
	b.ReportAllocs()
	b.SetBytes(int64(len(src)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Parse(strings.NewReader(src)); err != nil {
			b.Fatalf("parse error: %v", err)
		}
	}
}

func BenchmarkAssembleOnly(b *testing.B) {
	src := buildBenchmarkSource(512)
	prog, err := Parse(strings.NewReader(src))
	if err != nil {
		b.Fatalf("parse error: %v", err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(src)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Assemble(prog); err != nil {
			b.Fatalf("assemble error: %v", err)
		}
	}
}

func buildBenchmarkSource(lines int) string {
	var sb strings.Builder
	sb.Grow(lines * 32)
	sb.WriteString("start:\n")
	for i := 0; i < lines; i++ {
		sb.WriteString("        MOVEQ #1,D0\n")
		sb.WriteString("        ADD.L D0,D1\n")
		sb.WriteString("        SUB.L D1,D2\n")
	}
	sb.WriteString("        RTS\n")
	return sb.String()
}
