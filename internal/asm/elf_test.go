package asm

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestFormatELF_HeaderFields(t *testing.T) {
	code := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	origin := uint32(0x1000)

	elf := FormatELF(code, origin)

	if string(elf[:4]) != "\x7fELF" {
		t.Fatalf("invalid magic: %q", elf[:4])
	}
	if got := binary.BigEndian.Uint16(elf[16:18]); got != 2 {
		t.Fatalf("unexpected e_type: %d", got)
	}
	if got := binary.BigEndian.Uint16(elf[18:20]); got != 4 {
		t.Fatalf("unexpected e_machine: %d", got)
	}
	if got := binary.BigEndian.Uint32(elf[24:28]); got != origin {
		t.Fatalf("unexpected entry point: 0x%X", got)
	}
	phoff := binary.BigEndian.Uint32(elf[28:32])
	if phoff != elfHeaderSize {
		t.Fatalf("unexpected program header offset: %d", phoff)
	}

	ph := elf[phoff:]
	if got := binary.BigEndian.Uint32(ph[4:8]); got != elfHeaderSize+programHeaderSize {
		t.Fatalf("unexpected segment offset: %d", got)
	}
	if got := binary.BigEndian.Uint32(ph[16:20]); got != uint32(len(code)) {
		t.Fatalf("unexpected file size: %d", got)
	}
	if got := binary.BigEndian.Uint32(ph[24:28]); got != 0x5 {
		t.Fatalf("unexpected flags: 0x%X", got)
	}

	payload := elf[elfHeaderSize+programHeaderSize:]
	if got := string(payload); got != string(code) {
		t.Fatalf("payload mismatch: %x vs %x", payload, code)
	}
}

func TestAssembleELF_FromProgram(t *testing.T) {
	asmSrc := ".org 0x10\n.byte 0x01,0x02\n"
	prog, err := Parse(strings.NewReader(asmSrc))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	elf, err := AssembleELF(prog)
	if err != nil {
		t.Fatalf("assemble error: %v", err)
	}

	entry := binary.BigEndian.Uint32(elf[24:28])
	if entry != 0x10 {
		t.Fatalf("unexpected entry point: 0x%X", entry)
	}
	ph := elf[elfHeaderSize:]
	dataOffset := binary.BigEndian.Uint32(ph[4:8])
	if dataOffset >= uint32(len(elf)) {
		t.Fatalf("data offset beyond file size: %d >= %d", dataOffset, len(elf))
	}
	if elf[dataOffset] != 0x01 || elf[dataOffset+1] != 0x02 {
		t.Fatalf("assembled bytes missing at offset %d", dataOffset)
	}
}
