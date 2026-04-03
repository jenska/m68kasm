package m68kasm

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssembleStreamWithListingWrappers(t *testing.T) {
	t.Run("without options", func(t *testing.T) {
		var buf bytes.Buffer
		written, listing, err := AssembleStreamWithListing(&buf, strings.NewReader(".byte 0xAA\nMOVEQ #1,D0\n"))
		if err != nil {
			t.Fatalf("assemble failed: %v", err)
		}
		if written != int64(buf.Len()) {
			t.Fatalf("unexpected written count: got %d want %d", written, buf.Len())
		}
		if got, want := buf.Bytes(), []byte{0xAA, 0x70, 0x01}; !bytes.Equal(got, want) {
			t.Fatalf("unexpected bytes: got %x want %x", got, want)
		}
		if len(listing) != 2 {
			t.Fatalf("expected 2 listing entries, got %d", len(listing))
		}
	})

	t.Run("with options", func(t *testing.T) {
		var buf bytes.Buffer
		opts := ParseOptions{Symbols: map[string]uint32{"FOO": 0xAA}}
		written, listing, err := AssembleStreamWithListingWithOptions(&buf, strings.NewReader(".byte FOO\nMOVEQ #1,D0\n"), opts)
		if err != nil {
			t.Fatalf("assemble failed: %v", err)
		}
		if written != int64(buf.Len()) {
			t.Fatalf("unexpected written count: got %d want %d", written, buf.Len())
		}
		if got, want := buf.Bytes(), []byte{0xAA, 0x70, 0x01}; !bytes.Equal(got, want) {
			t.Fatalf("unexpected bytes: got %x want %x", got, want)
		}
		if len(listing) != 2 || listing[1].PC != 1 {
			t.Fatalf("unexpected listing: %+v", listing)
		}
	})
}

func TestAdditionalSliceAndStringWrappers(t *testing.T) {
	srcBytes := []byte(".byte FOO\nMOVEQ #1,D0\n")
	srcString := string(srcBytes)
	opts := ParseOptions{Symbols: map[string]uint32{"FOO": 0xAA}}
	want := []byte{0xAA, 0x70, 0x01}

	got, err := AssembleBytesWithOptions(srcBytes, opts)
	if err != nil {
		t.Fatalf("AssembleBytesWithOptions failed: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected bytes: got %x want %x", got, want)
	}

	got, listing, err := AssembleBytesWithListingWithOptions(srcBytes, opts)
	if err != nil {
		t.Fatalf("AssembleBytesWithListingWithOptions failed: %v", err)
	}
	if !bytes.Equal(got, want) || len(listing) != 2 {
		t.Fatalf("unexpected result: bytes=%x listing=%+v", got, listing)
	}

	prefixed := []byte{0xFF}
	got, err = AssembleBytesIntoWithOptions(prefixed, srcBytes, opts)
	if err != nil {
		t.Fatalf("AssembleBytesIntoWithOptions failed: %v", err)
	}
	if !bytes.Equal(got, append([]byte{0xFF}, want...)) {
		t.Fatalf("unexpected prefixed bytes: got %x", got)
	}

	prefixed = []byte{0xEE}
	got, listing, err = AssembleBytesWithListingIntoWithOptions(prefixed, srcBytes, opts)
	if err != nil {
		t.Fatalf("AssembleBytesWithListingIntoWithOptions failed: %v", err)
	}
	if !bytes.Equal(got, append([]byte{0xEE}, want...)) || len(listing) != 2 {
		t.Fatalf("unexpected result: bytes=%x listing=%+v", got, listing)
	}

	got, listing, err = AssembleStringWithListingWithOptions(srcString, opts)
	if err != nil {
		t.Fatalf("AssembleStringWithListingWithOptions failed: %v", err)
	}
	if !bytes.Equal(got, want) || len(listing) != 2 {
		t.Fatalf("unexpected result: bytes=%x listing=%+v", got, listing)
	}

	prefixed = []byte{0xDD}
	got, err = AssembleStringIntoWithOptions(prefixed, srcString, opts)
	if err != nil {
		t.Fatalf("AssembleStringIntoWithOptions failed: %v", err)
	}
	if !bytes.Equal(got, append([]byte{0xDD}, want...)) {
		t.Fatalf("unexpected prefixed bytes: got %x", got)
	}

	prefixed = []byte{0xCC}
	got, listing, err = AssembleStringWithListingIntoWithOptions(prefixed, srcString, opts)
	if err != nil {
		t.Fatalf("AssembleStringWithListingIntoWithOptions failed: %v", err)
	}
	if !bytes.Equal(got, append([]byte{0xCC}, want...)) || len(listing) != 2 {
		t.Fatalf("unexpected result: bytes=%x listing=%+v", got, listing)
	}
}

func TestFileAndFormatWrappers(t *testing.T) {
	path := writeTempSource(t, ".org 0x1000\nstart:\n.byte FOO\nMOVEQ #1,D0\n")
	opts := ParseOptions{Symbols: map[string]uint32{"FOO": 0xAA}}

	got, err := AssembleFileWithOptions(path, opts)
	if err != nil {
		t.Fatalf("AssembleFileWithOptions failed: %v", err)
	}
	if !bytes.Equal(got, []byte{0xAA, 0x70, 0x01}) {
		t.Fatalf("unexpected file bytes: %x", got)
	}

	got, listing, err := AssembleFileWithListingWithOptions(path, opts)
	if err != nil {
		t.Fatalf("AssembleFileWithListingWithOptions failed: %v", err)
	}
	if !bytes.Equal(got, []byte{0xAA, 0x70, 0x01}) || len(listing) != 2 {
		t.Fatalf("unexpected file result: bytes=%x listing=%+v", got, listing)
	}

	elfSrc := ".org 0x1000\nstart:\n.byte 0xAA\nMOVEQ #1,D0\n"
	for name, fn := range map[string]func() ([]byte, error){
		"AssembleELF":                func() ([]byte, error) { return AssembleELF(strings.NewReader(elfSrc)) },
		"AssembleBytesELF":           func() ([]byte, error) { return AssembleBytesELF([]byte(elfSrc)) },
		"AssembleBytesELFWithOptions": func() ([]byte, error) {
			return AssembleBytesELFWithOptions([]byte(".org 0x1000\n.byte FOO\n"), opts)
		},
		"AssembleStringELF":       func() ([]byte, error) { return AssembleStringELF(elfSrc) },
		"AssembleFileELF":         func() ([]byte, error) { return AssembleFileELF(writeTempSource(t, elfSrc)) },
		"AssembleFileELFWithOptions": func() ([]byte, error) {
			return AssembleFileELFWithOptions(writeTempSource(t, ".org 0x1000\n.byte FOO\n"), opts)
		},
	} {
		t.Run(name, func(t *testing.T) {
			out, err := fn()
			if err != nil {
				t.Fatalf("%s failed: %v", name, err)
			}
			assertELFImage(t, out)
		})
	}

	for name, fn := range map[string]func() ([]byte, error){
		"AssembleBytesSRecordWithOptions": func() ([]byte, error) {
			return AssembleBytesSRecordWithOptions([]byte(".org 0x1000\n.byte FOO\n"), opts)
		},
		"AssembleFileSRecord": func() ([]byte, error) {
			return AssembleFileSRecord(writeTempSource(t, ".org 0x1000\n.byte 0xAA\n"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			out, err := fn()
			if err != nil {
				t.Fatalf("%s failed: %v", name, err)
			}
			assertSRecordText(t, out)
		})
	}
}

func TestDetailedHelpersAndProgramBuilder(t *testing.T) {
	var nilResult *AssemblyResult
	if _, ok := nilResult.AddressOf("start"); ok {
		t.Fatalf("nil result should not resolve labels")
	}
	if _, ok := nilResult.AddressForLine(1); ok {
		t.Fatalf("nil result should not resolve line addresses")
	}

	const src = ".org 0x1000\nstart:\n.byte FOO\nMOVEQ #1,D0\n"
	opts := ParseOptions{Symbols: map[string]uint32{"FOO": 0xAA}}

	result, err := AssembleBytesDetailed([]byte(".org 0x1000\nstart:\n.byte 0xAA\nMOVEQ #1,D0\n"))
	if err != nil {
		t.Fatalf("AssembleBytesDetailed failed: %v", err)
	}
	assertDetailedResult(t, result)

	result, err = AssembleBytesDetailedWithOptions([]byte(src), opts)
	if err != nil {
		t.Fatalf("AssembleBytesDetailedWithOptions failed: %v", err)
	}
	assertDetailedResult(t, result)

	path := writeTempSource(t, src)
	result, err = AssembleFileDetailed(writeTempSource(t, ".org 0x1000\nstart:\n.byte 0xAA\nMOVEQ #1,D0\n"))
	if err != nil {
		t.Fatalf("AssembleFileDetailed failed: %v", err)
	}
	assertDetailedResult(t, result)

	result, err = AssembleFileDetailedWithOptions(path, opts)
	if err != nil {
		t.Fatalf("AssembleFileDetailedWithOptions failed: %v", err)
	}
	assertDetailedResult(t, result)

	builder := NewProgramBuilder().
		Origin(0x1000).
		Text().
		Label("start").
		Instruction("MOVEQ #1,D0").
		Data().
		Byte(0x11, 0x22).
		ByteExpr("VALUE").
		Word(0x3344).
		WordExpr("VALUE").
		Long(0x55667788).
		LongExpr("VALUE").
		Align(4, 0xCC).
		Even().
		BSS().
		Byte(0)

	if srcText := builder.String(); !strings.Contains(srcText, ".text") || !strings.Contains(srcText, ".bss") {
		t.Fatalf("builder source missing expected sections:\n%s", srcText)
	}

	sectionBuilder := NewProgramBuilder().Section(".data").Section("bss")
	if got := sectionBuilder.String(); got != ".section .data\n.section \"bss\"\n" {
		t.Fatalf("unexpected section builder output:\n%s", got)
	}

	result, err = builder.AssembleWithOptions(ParseOptions{Symbols: map[string]uint32{"VALUE": 3}})
	if err != nil {
		t.Fatalf("builder assemble with options failed: %v", err)
	}
	if len(result.Bytes) == 0 {
		t.Fatalf("expected builder bytes")
	}

	result, err = NewProgramBuilder().Instruction("NOP").Assemble()
	if err != nil {
		t.Fatalf("builder assemble failed: %v", err)
	}
	if got, want := result.Bytes, []byte{0x4E, 0x71}; !bytes.Equal(got, want) {
		t.Fatalf("unexpected builder bytes: got %x want %x", got, want)
	}
}

func writeTempSource(t *testing.T, src string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.s")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write temp source: %v", err)
	}
	return path
}

func assertELFImage(t *testing.T, data []byte) {
	t.Helper()
	if len(data) < 4 || string(data[:4]) != "\x7fELF" {
		t.Fatalf("missing ELF magic: %x", data)
	}
}

func assertSRecordText(t *testing.T, data []byte) {
	t.Helper()
	text := string(data)
	for _, want := range []string{"S0", "S3", "S7"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %s in S-record output:\n%s", want, text)
		}
	}
}

func assertDetailedResult(t *testing.T, result *AssemblyResult) {
	t.Helper()
	if result == nil {
		t.Fatalf("expected result")
	}
	if addr, ok := result.AddressOf("start"); !ok || addr != 0x1000 {
		t.Fatalf("unexpected start address: addr=0x%X ok=%v", addr, ok)
	}
	if addr, ok := result.AddressForLine(3); !ok || addr != 0x1000 {
		t.Fatalf("unexpected line address: addr=0x%X ok=%v", addr, ok)
	}
	if len(result.Listing) == 0 || len(result.Instructions) == 0 {
		t.Fatalf("expected listing and instruction metadata: %+v", result)
	}
}
