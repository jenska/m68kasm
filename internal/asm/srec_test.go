package asm

import (
	"fmt"
	"strings"
	"testing"
)

func TestFormatSRecords_OriginAndChecksums(t *testing.T) {
	src := ".org 0x1000\n.byte 0x11,0x22,0x33\n"
	prog, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	_, listing, err := AssembleWithListing(prog)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	srec := string(FormatSRecords(listing, prog.Origin, ""))
	want := "S00A00006D36386B61736D6E\nS3080000100011223381\nS70500001000EA\n"
	if srec != want {
		t.Fatalf("unexpected S-record output:\n%s\nwant:\n%s", srec, want)
	}
}

func TestFormatSRecords_SplitsIntoChunks(t *testing.T) {
	values := make([]string, 20)
	for i := range values {
		values[i] = fmt.Sprintf("%d", i)
	}
	src := ".org 0\n.byte " + strings.Join(values, ",") + "\n"

	prog, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	_, listing, err := AssembleWithListing(prog)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	srec := string(FormatSRecords(listing, prog.Origin, ""))
	want := strings.Join([]string{
		"S00A00006D36386B61736D6E",
		"S31500000000000102030405060708090A0B0C0D0E0F72",
		"S3090000001010111213A0",
		"S70500000000FA",
		"",
	}, "\n")

	if srec != want {
		t.Fatalf("unexpected multi-record output:\n%s\nwant:\n%s", srec, want)
	}
}
