package asm

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestFormatELF_HeaderFields(t *testing.T) {
	code := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	origin := uint32(0x1001)

	elf := FormatELF(code, origin)

	if string(elf[:4]) != "\x7fELF" {
		t.Fatalf("invalid magic: %q", elf[:4])
	}
	if got := binary.BigEndian.Uint16(elf[16:18]); got != elfTypeExec {
		t.Fatalf("unexpected e_type: %d", got)
	}
	if got := binary.BigEndian.Uint16(elf[18:20]); got != elfMachine68K {
		t.Fatalf("unexpected e_machine: %d", got)
	}
	if got := binary.BigEndian.Uint32(elf[24:28]); got != origin {
		t.Fatalf("unexpected entry point: 0x%X", got)
	}

	phoff := int(binary.BigEndian.Uint32(elf[28:32]))
	if phoff != elfHeaderSize {
		t.Fatalf("unexpected program header offset: %d", phoff)
	}
	shoff := int(binary.BigEndian.Uint32(elf[32:36]))
	if shoff <= phoff {
		t.Fatalf("section header table should follow program header: phoff=%d shoff=%d", phoff, shoff)
	}
	if got := binary.BigEndian.Uint16(elf[46:48]); got != sectionHeaderSize {
		t.Fatalf("unexpected section header size: %d", got)
	}
	if got := binary.BigEndian.Uint16(elf[48:50]); got != 7 {
		t.Fatalf("unexpected section count: %d", got)
	}
	if got := binary.BigEndian.Uint16(elf[50:52]); got != elfSectionShstrtab {
		t.Fatalf("unexpected shstrtab index: %d", got)
	}

	ph := elf[phoff:]
	if got := binary.BigEndian.Uint32(ph[0:4]); got != elfPhTypeLoad {
		t.Fatalf("unexpected p_type: %d", got)
	}
	if got := binary.BigEndian.Uint32(ph[4:8]); got != elfHeaderSize+programHeaderSize {
		t.Fatalf("unexpected segment offset: %d", got)
	}
	if got := binary.BigEndian.Uint32(ph[8:12]); got != origin {
		t.Fatalf("unexpected virtual address: 0x%X", got)
	}
	if got := binary.BigEndian.Uint32(ph[16:20]); got != uint32(len(code)) {
		t.Fatalf("unexpected file size: %d", got)
	}
	if got := binary.BigEndian.Uint32(ph[24:28]); got != elfPfR|elfPfX {
		t.Fatalf("unexpected flags: 0x%X", got)
	}
	if got := binary.BigEndian.Uint32(ph[28:32]); got != 1 {
		t.Fatalf("unexpected segment alignment: %d", got)
	}

	sections := readSectionHeaders(t, elf)
	shstrtab := sections[elfSectionShstrtab]
	sectionNames := elf[shstrtab.offset : shstrtab.offset+shstrtab.size]
	if got := readString(sectionNames, sections[elfSectionText].name); got != ".text" {
		t.Fatalf("unexpected .text section name: %q", got)
	}
	if got := readString(sectionNames, sections[elfSectionData].name); got != ".data" {
		t.Fatalf("unexpected .data section name: %q", got)
	}
	if got := readString(sectionNames, sections[elfSectionBSS].name); got != ".bss" {
		t.Fatalf("unexpected .bss section name: %q", got)
	}
	if got := readString(sectionNames, sections[elfSectionSymtab].name); got != ".symtab" {
		t.Fatalf("unexpected .symtab section name: %q", got)
	}
	if got := readString(sectionNames, sections[elfSectionStrtab].name); got != ".strtab" {
		t.Fatalf("unexpected .strtab section name: %q", got)
	}
	if got := readString(sectionNames, sections[elfSectionShstrtab].name); got != ".shstrtab" {
		t.Fatalf("unexpected .shstrtab section name: %q", got)
	}

	text := sections[elfSectionText]
	if text.addr != origin {
		t.Fatalf("unexpected .text addr: 0x%X", text.addr)
	}
	if text.size != uint32(len(code)) {
		t.Fatalf("unexpected .text size: %d", text.size)
	}
	payload := elf[text.offset : text.offset+text.size]
	if got := string(payload); got != string(code) {
		t.Fatalf("payload mismatch: %x vs %x", payload, code)
	}
}

func TestFormatELFWithLabels_SymbolTable(t *testing.T) {
	code := []byte{0x70, 0x01, 0x4E, 0x75}
	origin := uint32(0x2000)
	labels := []DefinedLabel{
		{Name: "start", Addr: origin},
		{Name: "done", Addr: origin + 2},
	}

	elf := FormatELFWithLabels(code, origin, labels)
	sections := readSectionHeaders(t, elf)

	strtab := sections[elfSectionStrtab]
	strtabBytes := elf[strtab.offset : strtab.offset+strtab.size]
	symtab := sections[elfSectionSymtab]
	if symtab.link != elfSectionStrtab {
		t.Fatalf("symtab link=%d want %d", symtab.link, elfSectionStrtab)
	}
	if symtab.info != 4 {
		t.Fatalf("symtab info=%d want 4", symtab.info)
	}
	if symtab.entsize != elfSymbolSize {
		t.Fatalf("symtab entsize=%d want %d", symtab.entsize, elfSymbolSize)
	}

	symbols := readSymbols(t, elf[symtab.offset:symtab.offset+symtab.size])
	if len(symbols) != 4 {
		t.Fatalf("unexpected symbol count: %d", len(symbols))
	}
	if symbols[1].info != elfStInfo(elfStbLocal, elfSttSection) {
		t.Fatalf("unexpected section symbol info: 0x%X", symbols[1].info)
	}
	if symbols[1].shndx != elfSectionText {
		t.Fatalf("section symbol should point at .text, got %d", symbols[1].shndx)
	}

	if got := readString(strtabBytes, symbols[2].name); got != "start" {
		t.Fatalf("unexpected first label name: %q", got)
	}
	if symbols[2].value != origin {
		t.Fatalf("unexpected start value: 0x%X", symbols[2].value)
	}
	if symbols[2].shndx != elfSectionText {
		t.Fatalf("start should point at .text, got %d", symbols[2].shndx)
	}

	if got := readString(strtabBytes, symbols[3].name); got != "done" {
		t.Fatalf("unexpected second label name: %q", got)
	}
	if symbols[3].value != origin+2 {
		t.Fatalf("unexpected done value: 0x%X", symbols[3].value)
	}
}

func TestAssembleELF_FromProgram(t *testing.T) {
	asmSrc := ".org 0x10\nstart:\n.byte 0x01,0x02\nend:\n"
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

	sections := readSectionHeaders(t, elf)
	text := sections[elfSectionText]
	if text.offset >= uint32(len(elf)) {
		t.Fatalf(".text offset beyond file size: %d >= %d", text.offset, len(elf))
	}
	if elf[text.offset] != 0x01 || elf[text.offset+1] != 0x02 {
		t.Fatalf("assembled bytes missing at offset %d", text.offset)
	}

	symtab := sections[elfSectionSymtab]
	strtab := sections[elfSectionStrtab]
	symbols := readSymbols(t, elf[symtab.offset:symtab.offset+symtab.size])
	strtabBytes := elf[strtab.offset : strtab.offset+strtab.size]

	var names []string
	for _, sym := range symbols[2:] {
		names = append(names, readString(strtabBytes, sym.name))
	}
	if got := strings.Join(names, ","); got != "start,end" {
		t.Fatalf("unexpected symbol names: %s", got)
	}
}

func TestAssembleELF_WithSectionDirectives(t *testing.T) {
	asmSrc := ".org 0x1000\n.text\nstart:\n.byte 0xAA\n.data\nvalue:\n.word 0x1234\n.bss\nzeroes:\n.byte 0,0\n"
	prog, err := Parse(strings.NewReader(asmSrc))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	elf, err := AssembleELF(prog)
	if err != nil {
		t.Fatalf("assemble error: %v", err)
	}

	sections := readSectionHeaders(t, elf)
	text := sections[elfSectionText]
	if text.addr != 0x1000 || text.size != 1 {
		t.Fatalf("unexpected .text addr/size: addr=0x%X size=%d", text.addr, text.size)
	}
	if got := elf[text.offset]; got != 0xAA {
		t.Fatalf("unexpected .text payload byte: 0x%X", got)
	}

	data := sections[elfSectionData]
	if data.addr != 0x1001 || data.size != 2 {
		t.Fatalf("unexpected .data addr/size: addr=0x%X size=%d", data.addr, data.size)
	}
	if got := elf[data.offset : data.offset+data.size]; string(got) != string([]byte{0x12, 0x34}) {
		t.Fatalf("unexpected .data payload: %x", got)
	}

	bss := sections[elfSectionBSS]
	if bss.addr != 0x1003 || bss.size != 2 {
		t.Fatalf("unexpected .bss addr/size: addr=0x%X size=%d", bss.addr, bss.size)
	}
	if got := binary.BigEndian.Uint32(elf[elfHeaderSize+24 : elfHeaderSize+28]); got != elfPfR|elfPfX|elfPfW {
		t.Fatalf("unexpected program header flags: 0x%X", got)
	}
	if got := binary.BigEndian.Uint32(elf[elfHeaderSize+16 : elfHeaderSize+20]); got != 3 {
		t.Fatalf("unexpected program file size: %d", got)
	}
	if got := binary.BigEndian.Uint32(elf[elfHeaderSize+20 : elfHeaderSize+24]); got != 5 {
		t.Fatalf("unexpected program memory size: %d", got)
	}

	symtab := sections[elfSectionSymtab]
	strtab := sections[elfSectionStrtab]
	symbols := readSymbols(t, elf[symtab.offset:symtab.offset+symtab.size])
	strtabBytes := elf[strtab.offset : strtab.offset+strtab.size]

	checks := map[string]uint16{
		"start":  elfSectionText,
		"value":  elfSectionData,
		"zeroes": elfSectionBSS,
	}
	for name, wantSection := range checks {
		sym, ok := findSymbolByName(symbols, strtabBytes, name)
		if !ok {
			t.Fatalf("missing symbol %q", name)
		}
		if sym.shndx != wantSection {
			t.Fatalf("symbol %q in section %d, want %d", name, sym.shndx, wantSection)
		}
	}
}

func TestParseRejectsBackwardSectionSwitch(t *testing.T) {
	_, err := Parse(strings.NewReader(".data\n.byte 1\n.text\n.byte 2\n"))
	if err == nil {
		t.Fatalf("expected parse error for backward section switch")
	}
	if !strings.Contains(err.Error(), ".text -> .data -> .bss order") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseRejectsNonZeroBSSData(t *testing.T) {
	_, err := Parse(strings.NewReader(".bss\n.byte 1\n"))
	if err == nil {
		t.Fatalf("expected parse error for initialized bss data")
	}
	if !strings.Contains(err.Error(), "must be zero-initialized") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func readSectionHeaders(t *testing.T, elf []byte) []elfSectionHeader {
	t.Helper()

	shoff := int(binary.BigEndian.Uint32(elf[32:36]))
	shnum := int(binary.BigEndian.Uint16(elf[48:50]))
	headers := make([]elfSectionHeader, shnum)
	for i := 0; i < shnum; i++ {
		base := shoff + i*sectionHeaderSize
		headers[i] = elfSectionHeader{
			name:      binary.BigEndian.Uint32(elf[base:]),
			typ:       binary.BigEndian.Uint32(elf[base+4:]),
			flags:     binary.BigEndian.Uint32(elf[base+8:]),
			addr:      binary.BigEndian.Uint32(elf[base+12:]),
			offset:    binary.BigEndian.Uint32(elf[base+16:]),
			size:      binary.BigEndian.Uint32(elf[base+20:]),
			link:      binary.BigEndian.Uint32(elf[base+24:]),
			info:      binary.BigEndian.Uint32(elf[base+28:]),
			addralign: binary.BigEndian.Uint32(elf[base+32:]),
			entsize:   binary.BigEndian.Uint32(elf[base+36:]),
		}
	}
	return headers
}

func readSymbols(t *testing.T, data []byte) []elfSymbol {
	t.Helper()

	if len(data)%elfSymbolSize != 0 {
		t.Fatalf("symbol table size %d is not a multiple of %d", len(data), elfSymbolSize)
	}

	symbols := make([]elfSymbol, 0, len(data)/elfSymbolSize)
	for base := 0; base < len(data); base += elfSymbolSize {
		symbols = append(symbols, elfSymbol{
			name:  binary.BigEndian.Uint32(data[base:]),
			value: binary.BigEndian.Uint32(data[base+4:]),
			size:  binary.BigEndian.Uint32(data[base+8:]),
			info:  data[base+12],
			other: data[base+13],
			shndx: binary.BigEndian.Uint16(data[base+14:]),
		})
	}
	return symbols
}

func findSymbolByName(symbols []elfSymbol, strtab []byte, name string) (elfSymbol, bool) {
	for _, sym := range symbols {
		if readString(strtab, sym.name) == name {
			return sym, true
		}
	}
	return elfSymbol{}, false
}

func readString(table []byte, off uint32) string {
	if off >= uint32(len(table)) {
		return ""
	}
	end := off
	for end < uint32(len(table)) && table[end] != 0 {
		end++
	}
	return string(table[off:end])
}
