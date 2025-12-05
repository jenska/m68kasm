package m68kasm

import (
	"bytes"
	"strings"
	"testing"
)

func BenchmarkAssembleStream(b *testing.B) {
	src := strings.Repeat("NOP\n", 1024)
	b.ReportAllocs()
	b.SetBytes(int64(len(src)))

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		if _, err := AssembleStream(&buf, strings.NewReader(src)); err != nil {
			b.Fatalf("assemble stream failed: %v", err)
		}
	}
}
