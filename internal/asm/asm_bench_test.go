package asm

import (
	"fmt"
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

func BenchmarkAssembleControlAndStack(b *testing.B) {
	src := `.org 0
start:
    LINK A6,#-32
    PEA (A1)
    BTST #5,(A0)
    BTST D2,(A3)
    TST.W (A2)
    NEGX.B D3
    ADDX.W D1,D0
    SUBX.L -(A2),-(A3)
    JSR (8,PC)
    UNLK A6
    RTS
`

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

func buildBenchmarkSource(blocks int) string {
	var sb strings.Builder
	// Roughly 60 bytes per block; helps avoid repeated allocations.
	sb.Grow(blocks * 64)

	sb.WriteString(".org 0\n")
	for i := 0; i < blocks; i++ {
		sb.WriteString(fmt.Sprintf("label%d: moveq #%d,d0\n", i, i%8))
		sb.WriteString("nop\n")

		sb.WriteString("add.l d0,d1\n")
		sb.WriteString("sub.l d1,d0\n")
		sb.WriteString("MOVEM.L D0-D1/A6,-(A7)\n")
		sb.WriteString(fmt.Sprintf("bra label%d\n", i))
		if (i+1)%50 == 0 {
			sb.WriteString(".byte $AA,$BB,$CC,$DD\n")
			sb.WriteString(".align 4,$00\n")
		}
	}

	return sb.String()
}
