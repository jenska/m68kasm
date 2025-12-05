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

func TestAssembleFileVariants(t *testing.T) {
	path := filepath.Join("tests", "testdata", "api_sample.s")

	bytesOut, err := AssembleFile(path)
	if err != nil {
		t.Fatalf("assemble file failed: %v", err)
	}

	if want := []byte{0x12, 0x34, 0x70, 0x01}; !bytes.Equal(bytesOut, want) {
		t.Fatalf("unexpected encoding: got %x want %x", bytesOut, want)
	}

	withListing, listing, err := AssembleFileWithListing(path)
	if err != nil {
		t.Fatalf("assemble file with listing failed: %v", err)
	}

	if !bytes.Equal(withListing, bytesOut) {
		t.Fatalf("expected bytes with listing to match AssembleFile output")
	}

	if len(listing) != 2 {
		t.Fatalf("expected listing entries for two lines, got %d", len(listing))
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
