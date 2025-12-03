package m68kasm

import (
	"bytes"
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
