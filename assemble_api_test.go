package m68kasm

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssembleBytes(t *testing.T) {
	src := []byte("MOVEQ #1,D0\n")

	got, err := AssembleBytes(src)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	want := []byte{0x70, 0x01}
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected encoding: got %x want %x", got, want)
	}
}

func TestAssembleBytesInto(t *testing.T) {
	src := []byte("MOVEQ #1,D0\n")
	dst := make([]byte, 2, 8)
	dst[0], dst[1] = 0xde, 0xad
	base := &dst[0]

	out, err := AssembleBytesInto(dst, src)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	if &out[0] != base {
		t.Fatalf("expected output to reuse destination slice")
	}

	want := []byte{0xde, 0xad, 0x70, 0x01}
	if !bytes.Equal(out, want) {
		t.Fatalf("unexpected encoding: got %x want %x", out, want)
	}
}

func TestAssembleString(t *testing.T) {
	src := "label:\n.WORD 0\nMOVE.B label,D0\n"

	got, err := AssembleString(src)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	want := []byte{0x00, 0x00, 0x10, 0x39, 0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected encoding: got %x want %x", got, want)
	}
}

func TestAssembleStringInto(t *testing.T) {
	src := "label:\n.WORD 0\nMOVE.B label,D0\n"
	dst := make([]byte, 1, 32)
	dst[0] = 0xff

	out, err := AssembleStringInto(dst, src)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	want := []byte{0xff, 0x00, 0x00, 0x10, 0x39, 0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(out, want) {
		t.Fatalf("unexpected encoding: got %x want %x", out, want)
	}
	if &out[0] != &dst[0] {
		t.Fatalf("expected output to reuse destination slice")
	}
}

func TestAssembleStringWithListing(t *testing.T) {
	src := "label:\n.WORD 0\nMOVE.B label,D0\n"

	got, listing, err := AssembleStringWithListing(src)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	wanted := []byte{0x00, 0x00, 0x10, 0x39, 0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(got, wanted) {
		t.Fatalf("unexpected encoding: got %x want %x", got, wanted)
	}

	if len(listing) != 2 {
		t.Fatalf("expected 2 listing entries, got %d", len(listing))
	}

	if listing[0].Line != 2 || listing[0].PC != 0 || !bytes.Equal(listing[0].Bytes, []byte{0x00, 0x00}) {
		t.Fatalf("unexpected first listing entry: %+v", listing[0])
	}

	if listing[1].Line != 3 || listing[1].PC != 2 || !bytes.Equal(listing[1].Bytes, []byte{0x10, 0x39, 0x00, 0x00, 0x00, 0x00}) {
		t.Fatalf("unexpected second listing entry: %+v", listing[1])
	}
}

func TestAssembleBytesWithListing(t *testing.T) {
	src := []byte(".byte 0xAA\nMOVEQ #1,D0\n")

	encoded, listing, err := AssembleBytesWithListing(src)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	want := []byte{0xAA, 0x70, 0x01}
	if !bytes.Equal(encoded, want) {
		t.Fatalf("unexpected encoding: got %x want %x", encoded, want)
	}

	if len(listing) != 2 {
		t.Fatalf("expected listing for 2 lines, got %d", len(listing))
	}

	if listing[0].PC != 0x0 || !bytes.Equal(listing[0].Bytes, []byte{0xAA}) {
		t.Fatalf("unexpected first listing entry: %+v", listing[0])
	}
}

func TestAssembleBytesWithListingInto(t *testing.T) {
	src := []byte(".byte 0xBB\nMOVEQ #2,D0\n")
	dst := make([]byte, 1, 8)
	dst[0] = 0xFF

	encoded, listing, err := AssembleBytesWithListingInto(dst, src)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	want := []byte{0xFF, 0xBB, 0x70, 0x02}
	if !bytes.Equal(encoded, want) {
		t.Fatalf("unexpected encoding: got %x want %x", encoded, want)
	}

	if &encoded[0] != &dst[0] {
		t.Fatalf("expected output to reuse destination slice")
	}

	if len(listing) != 2 || listing[0].PC != 0 || len(listing[1].Bytes) != 2 {
		t.Fatalf("unexpected listing entries: %+v", listing)
	}
}

func TestAssembleStringWithListingInto(t *testing.T) {
	src := ".byte 0xCC\nMOVEQ #3,D0\n"
	dst := make([]byte, 1, 8)
	dst[0] = 0xEE

	encoded, listing, err := AssembleStringWithListingInto(dst, src)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	want := []byte{0xEE, 0xCC, 0x70, 0x03}
	if !bytes.Equal(encoded, want) {
		t.Fatalf("unexpected encoding: got %x want %x", encoded, want)
	}

	if &encoded[0] != &dst[0] {
		t.Fatalf("expected output to reuse destination slice")
	}

	if len(listing) != 2 || listing[1].PC != 1 {
		t.Fatalf("unexpected listing entries: %+v", listing)
	}
}

func TestAssembleStream(t *testing.T) {
	src := strings.Repeat("NOP\n", 8)
	var buf bytes.Buffer

	written, err := AssembleStream(&buf, strings.NewReader(src))
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	if written != int64(buf.Len()) {
		t.Fatalf("expected written count %d to equal buffer length %d", written, buf.Len())
	}

	want, err := AssembleString(src)
	if err != nil {
		t.Fatalf("assemble baseline failed: %v", err)
	}

	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("streamed bytes differ: got %x want %x", buf.Bytes(), want)
	}
}

func TestContextualError(t *testing.T) {
	_, err := AssembleString("NOP\nBADTOKEN\n")
	if err == nil {
		t.Fatalf("expected error")
	}

	var asmErr *Error
	if !errors.As(err, &asmErr) {
		t.Fatalf("expected contextual Error, got %T", err)
	}

	if asmErr.Line != 2 {
		t.Fatalf("expected error on line 2, got %d", asmErr.Line)
	}
}

func TestAssembleWithPredefinedSymbols(t *testing.T) {
	src := ".word FOO\n"
	opts := ParseOptions{Symbols: map[string]uint32{"FOO": 0xBEEF}}

	got, err := AssembleStringWithOptions(src, opts)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	want := []byte{0xBE, 0xEF}
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected encoding: got %x want %x", got, want)
	}
}

func TestAssembleStringSRecord(t *testing.T) {
	src := ".org 0x1000\n.byte 0x11,0x22,0x33\n"

	got, err := AssembleStringSRecord(src)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	header := s0Record(srecHeader())
	want := fmt.Sprintf("%s\nS3080000100011223381\nS70500001000EA\n", header)
	if string(got) != want {
		t.Fatalf("unexpected S-record output:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestAssembleStringSRecordWithOptions(t *testing.T) {
	src := ".org 0x1000\n.byte FOO\n"
	opts := ParseOptions{Symbols: map[string]uint32{"FOO": 0x11}}

	got, err := AssembleStringSRecordWithOptions(src, opts)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	header := s0Record(srecHeader())
	want := fmt.Sprintf("%s\nS3060000100011D8\nS70500001000EA\n", header)
	if string(got) != want {
		t.Fatalf("unexpected S-record output:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestAssembleStringDetailed(t *testing.T) {
	src := ".org $1000\nstart:\nMOVEQ #1,D0\nloop:\nBRA.W $1002\n"

	result, err := AssembleStringDetailed(src)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	if result.Origin != 0x1000 {
		t.Fatalf("unexpected origin: got 0x%X want 0x1000", result.Origin)
	}

	if got, ok := result.AddressOf("start"); !ok || got != 0x1000 {
		t.Fatalf("unexpected start label: got 0x%X ok=%v", got, ok)
	}

	if got, ok := result.AddressOf("loop"); !ok || got != 0x1002 {
		t.Fatalf("unexpected loop label: got 0x%X ok=%v", got, ok)
	}

	if got, ok := result.AddressForLine(2); !ok || got != 0x1000 {
		t.Fatalf("unexpected line 2 address: got 0x%X ok=%v", got, ok)
	}

	if got, ok := result.AddressForLine(4); !ok || got != 0x1002 {
		t.Fatalf("unexpected line 4 address: got 0x%X ok=%v", got, ok)
	}

	wantBytes := []byte{0x70, 0x01, 0x60, 0x00, 0xFF, 0xFE}
	if !bytes.Equal(result.Bytes, wantBytes) {
		t.Fatalf("unexpected bytes: got %x want %x", result.Bytes, wantBytes)
	}

	if len(result.Instructions) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(result.Instructions))
	}

	if got := result.Instructions[0].Canonical; got != "MOVEQ.L #$1,D0" {
		t.Fatalf("unexpected first canonical instruction: %q", got)
	}

	if got := result.Instructions[1].Canonical; got != "BRA.W $1002" {
		t.Fatalf("unexpected second canonical instruction: %q", got)
	}
}

func TestProgramBuilder(t *testing.T) {
	builder := NewProgramBuilder().
		Origin(0x1000).
		VectorTable(map[uint32]string{
			0: "STACK_TOP",
			1: "reset",
			3: "handler",
		}).
		Label("reset").
		Instruction("NOP").
		Label("handler").
		Instruction("RTS")

	result, err := builder.AssembleWithOptions(ParseOptions{
		Symbols: map[string]uint32{"STACK_TOP": 0x2000},
	})
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	wantSource := ".org $1000\n.long STACK_TOP,reset,0,handler\nreset:\nNOP\nhandler:\nRTS\n"
	if got := builder.String(); got != wantSource {
		t.Fatalf("unexpected builder source:\n%s\nwant:\n%s", got, wantSource)
	}

	wantBytes := []byte{
		0x00, 0x00, 0x20, 0x00,
		0x00, 0x00, 0x10, 0x10,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x10, 0x12,
		0x4E, 0x71,
		0x4E, 0x75,
	}
	if !bytes.Equal(result.Bytes, wantBytes) {
		t.Fatalf("unexpected bytes: got %x want %x", result.Bytes, wantBytes)
	}

	if got, ok := result.AddressOf("reset"); !ok || got != 0x1010 {
		t.Fatalf("unexpected reset label: got 0x%X ok=%v", got, ok)
	}

	if got, ok := result.AddressOf("handler"); !ok || got != 0x1012 {
		t.Fatalf("unexpected handler label: got 0x%X ok=%v", got, ok)
	}
}

func TestAssembleInstructionString(t *testing.T) {
	result, err := AssembleInstructionString("MOVEQ #1,D0\n")
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	if want := []byte{0x70, 0x01}; !bytes.Equal(result.Bytes, want) {
		t.Fatalf("unexpected bytes: got %x want %x", result.Bytes, want)
	}

	if result.EncodedSize != 2 || result.Words != 1 {
		t.Fatalf("unexpected size metadata: %+v", result)
	}

	if result.Canonical != "MOVEQ.L #$1,D0" {
		t.Fatalf("unexpected canonical instruction: %q", result.Canonical)
	}
}

func TestAssembleInstructionStringWithAbsoluteBranchTarget(t *testing.T) {
	result, err := AssembleInstructionString(".org $1000\nBRA.W $1004\n")
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	want := []byte{0x60, 0x00, 0x00, 0x02}
	if !bytes.Equal(result.Bytes, want) {
		t.Fatalf("unexpected bytes: got %x want %x", result.Bytes, want)
	}

	if result.PC != 0x1000 {
		t.Fatalf("unexpected pc: got 0x%X want 0x1000", result.PC)
	}

	if result.Canonical != "BRA.W $1004" {
		t.Fatalf("unexpected canonical instruction: %q", result.Canonical)
	}
}

func TestCanonicalizeInstructionString(t *testing.T) {
	got, err := CanonicalizeInstructionString("  move.w  (a0)+ , d1 \n")
	if err != nil {
		t.Fatalf("canonicalize failed: %v", err)
	}

	if got != "MOVE.W (A0)+,D1" {
		t.Fatalf("unexpected canonical form: %q", got)
	}
}

func TestNormalizeError(t *testing.T) {
	_, err := AssembleString(".align 0\n")
	if err == nil {
		t.Fatalf("expected error")
	}

	if got := NormalizeError(err); got != ".align expects value >= 1, got 0" {
		t.Fatalf("unexpected normalized error: %q", got)
	}

	var asmErr *Error
	if !errors.As(err, &asmErr) {
		t.Fatalf("expected structured Error, got %T", err)
	}
	if strings.Contains(asmErr.Message(), "line 1") {
		t.Fatalf("normalized message unexpectedly includes location: %q", asmErr.Message())
	}
	if strings.Count(err.Error(), "line 1") != 1 {
		t.Fatalf("formatted error duplicated location unexpectedly: %q", err.Error())
	}
}

func TestNormalizeErrorForUndefinedExpressionLabel(t *testing.T) {
	_, err := AssembleString(".word missing+1\n")
	if err == nil {
		t.Fatalf("expected error")
	}

	if got := NormalizeError(err); got != "undefined label in expression: missing" {
		t.Fatalf("unexpected normalized error: %q", got)
	}
}

func TestAssembleSRecordVariants(t *testing.T) {
	src := ".org 0x1000\n.byte 0x11,0x22,0x33\n"

	stringSrec, err := AssembleStringSRecord(src)
	if err != nil {
		t.Fatalf("assemble string failed: %v", err)
	}

	bytesSrec, err := AssembleBytesSRecord([]byte(src))
	if err != nil {
		t.Fatalf("assemble bytes failed: %v", err)
	}

	readerSrec, err := AssembleSRecord(strings.NewReader(src))
	if err != nil {
		t.Fatalf("assemble reader failed: %v", err)
	}

	tempFile := filepath.Join(t.TempDir(), "sample.s")
	if err := os.WriteFile(tempFile, []byte(src), 0o644); err != nil {
		t.Fatalf("write temp source: %v", err)
	}

	fileSrec, err := AssembleFileSRecord(tempFile)
	if err != nil {
		t.Fatalf("assemble file failed: %v", err)
	}

	for name, got := range map[string][]byte{
		"bytes":  bytesSrec,
		"reader": readerSrec,
		"file":   fileSrec,
	} {
		if string(got) != string(stringSrec) {
			t.Fatalf("%s s-record mismatch:\n%s\nwant:\n%s", name, string(got), string(stringSrec))
		}
	}
}

func TestAssembleELFWithOptions(t *testing.T) {
	src := ".org ENTRY\nMOVEQ #1,D0\n"
	opts := ParseOptions{Symbols: map[string]uint32{"ENTRY": 0x2000}}

	elf, err := AssembleStringELFWithOptions(src, opts)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	if len(elf) < 4 || string(elf[:4]) != "\x7fELF" {
		t.Fatalf("missing ELF magic: %x", elf[:4])
	}
}

func TestAssembleFileVariants(t *testing.T) {
	path := filepath.Join("tests", "testdata", "api_sample.s")

	bytesOut, err := AssembleFile(path)
	if err != nil {
		t.Fatalf("assemble file failed: %v", err)
	}

	if want := []byte{0x12, 0x34, 0x70, 0x01, 0xc0, 0xfc, 0x81, 0xfc}; !bytes.Equal(bytesOut, want) {
		t.Fatalf("unexpected encoding: got %x want %x", bytesOut, want)
	}

	withListing, listing, err := AssembleFileWithListing(path)
	if err != nil {
		t.Fatalf("assemble file with listing failed: %v", err)
	}

	if !bytes.Equal(withListing, bytesOut) {
		t.Fatalf("expected bytes with listing to match AssembleFile output")
	}

	if listing[0].PC != 0x1000 || listing[1].PC != 0x1002 {
		t.Fatalf("unexpected listing PCs: %+v", listing)
	}
}

func s0Record(header string) string {
	if len(header) > 252 {
		header = header[:252]
	}

	count := byte(2 + len(header) + 1)
	sum := uint32(count)

	var sb strings.Builder
	sb.WriteString("S0")
	fmt.Fprintf(&sb, "%02X0000", count)

	for _, b := range []byte(header) {
		sum += uint32(b)
		fmt.Fprintf(&sb, "%02X", b)
	}

	checksum := byte(^sum)
	fmt.Fprintf(&sb, "%02X", checksum)
	return sb.String()
}
