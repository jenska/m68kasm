package asm

import (
	"encoding/binary"
)

const (
	elfHeaderSize     = 52
	programHeaderSize = 32
	sectionHeaderSize = 40
	elfSymbolSize     = 16
)

const (
	elfTypeExec    = 2
	elfMachine68K  = 4
	elfVersion     = 1
	elfClass32     = 1
	elfDataMSB     = 2
	elfPhTypeLoad  = 1
	elfShTypeNull  = 0
	elfShTypeBits  = 1
	elfShTypeSym   = 2
	elfShTypeStr   = 3
	elfShTypeNoBit = 8
	elfPfX         = 0x1
	elfPfW         = 0x2
	elfPfR         = 0x4
	elfShfWrite    = 0x1
	elfShfAlloc    = 0x2
	elfShfExec     = 0x4
	elfStbLocal    = 0
	elfSttNotype   = 0
	elfSttSection  = 3
)

const (
	elfSectionNull = iota
	elfSectionText
	elfSectionData
	elfSectionBSS
	elfSectionSymtab
	elfSectionStrtab
	elfSectionShstrtab
)

type elfSectionHeader struct {
	name      uint32
	typ       uint32
	flags     uint32
	addr      uint32
	offset    uint32
	size      uint32
	link      uint32
	info      uint32
	addralign uint32
	entsize   uint32
}

type elfSymbol struct {
	name  uint32
	value uint32
	size  uint32
	info  byte
	other byte
	shndx uint16
}

// AssembleELF assembles the given program and wraps the output bytes in a
// standard ELF32 executable image suitable for m68k emulators and loaders.
func AssembleELF(p *Program) ([]byte, error) {
	bytes, _, _, err := assemble(nil, nil, p, false)
	if err != nil {
		return nil, err
	}

	return FormatELFWithLabels(bytes, p.Origin, p.DefinedLabels), nil
}

// FormatELF wraps the provided machine code bytes into a 32-bit big-endian ELF
// executable image for the Motorola 68000 (EM_68K). The entry point and
// segment addresses are set to origin.
func FormatELF(code []byte, origin uint32) []byte {
	return FormatELFWithLabels(code, origin, nil)
}

// FormatELFWithLabels wraps the provided machine code bytes into a 32-bit
// big-endian ELF executable image for the Motorola 68000 (EM_68K). The output
// keeps the existing flat load segment model while also emitting section and
// symbol tables for better compatibility with ELF-aware tooling.
func FormatELFWithLabels(code []byte, origin uint32, labels []DefinedLabel) []byte {
	textOffset := elfHeaderSize + programHeaderSize
	dataEnd := textOffset + len(code)

	strtab := newELFStringTable()
	for _, label := range labels {
		strtab.add(label.Name)
	}
	strtabBytes := strtab.bytes()

	symbols := make([]elfSymbol, 0, 2+len(labels))
	symbols = append(symbols, elfSymbol{})
	symbols = append(symbols, elfSymbol{
		info:  elfStInfo(elfStbLocal, elfSttSection),
		shndx: elfSectionText,
	})
	for _, label := range labels {
		symbols = append(symbols, elfSymbol{
			name:  strtab.add(label.Name),
			value: label.Addr,
			info:  elfStInfo(elfStbLocal, elfSttNotype),
			shndx: elfSectionText,
		})
	}
	symtabBytes := encodeELFSymbols(symbols)

	shstrtab := newELFStringTable()
	textName := shstrtab.add(".text")
	dataName := shstrtab.add(".data")
	bssName := shstrtab.add(".bss")
	symtabName := shstrtab.add(".symtab")
	strtabName := shstrtab.add(".strtab")
	shstrtabName := shstrtab.add(".shstrtab")
	shstrtabBytes := shstrtab.bytes()

	strtabOffset := dataEnd
	symtabOffset := alignOffset(strtabOffset+len(strtabBytes), 4)
	shstrtabOffset := symtabOffset + len(symtabBytes)
	sectionOffset := alignOffset(shstrtabOffset+len(shstrtabBytes), 4)

	sections := []elfSectionHeader{
		{},
		{
			name:      textName,
			typ:       elfShTypeBits,
			flags:     elfShfAlloc | elfShfExec,
			addr:      origin,
			offset:    uint32(textOffset),
			size:      uint32(len(code)),
			addralign: 1,
		},
		{
			name:      dataName,
			typ:       elfShTypeBits,
			flags:     elfShfAlloc | elfShfWrite,
			addr:      origin + uint32(len(code)),
			offset:    uint32(dataEnd),
			addralign: 1,
		},
		{
			name:      bssName,
			typ:       elfShTypeNoBit,
			flags:     elfShfAlloc | elfShfWrite,
			addr:      origin + uint32(len(code)),
			offset:    uint32(dataEnd),
			addralign: 1,
		},
		{
			name:      symtabName,
			typ:       elfShTypeSym,
			offset:    uint32(symtabOffset),
			size:      uint32(len(symtabBytes)),
			link:      elfSectionStrtab,
			info:      uint32(len(symbols)),
			addralign: 4,
			entsize:   elfSymbolSize,
		},
		{
			name:      strtabName,
			typ:       elfShTypeStr,
			offset:    uint32(strtabOffset),
			size:      uint32(len(strtabBytes)),
			addralign: 1,
		},
		{
			name:      shstrtabName,
			typ:       elfShTypeStr,
			offset:    uint32(shstrtabOffset),
			size:      uint32(len(shstrtabBytes)),
			addralign: 1,
		},
	}

	out := make([]byte, sectionOffset+sectionHeaderSize*len(sections))

	// e_ident
	out[0] = 0x7F
	out[1] = 'E'
	out[2] = 'L'
	out[3] = 'F'
	out[4] = elfClass32
	out[5] = elfDataMSB
	out[6] = elfVersion

	binary.BigEndian.PutUint16(out[16:], elfTypeExec)
	binary.BigEndian.PutUint16(out[18:], elfMachine68K)
	binary.BigEndian.PutUint32(out[20:], elfVersion)
	binary.BigEndian.PutUint32(out[24:], origin)
	binary.BigEndian.PutUint32(out[28:], elfHeaderSize)
	binary.BigEndian.PutUint32(out[32:], uint32(sectionOffset))
	binary.BigEndian.PutUint32(out[36:], 0)
	binary.BigEndian.PutUint16(out[40:], elfHeaderSize)
	binary.BigEndian.PutUint16(out[42:], programHeaderSize)
	binary.BigEndian.PutUint16(out[44:], 1)
	binary.BigEndian.PutUint16(out[46:], sectionHeaderSize)
	binary.BigEndian.PutUint16(out[48:], uint16(len(sections)))
	binary.BigEndian.PutUint16(out[50:], elfSectionShstrtab)

	// Program header
	ph := out[elfHeaderSize:]
	binary.BigEndian.PutUint32(ph[0:], elfPhTypeLoad)
	binary.BigEndian.PutUint32(ph[4:], uint32(textOffset))
	binary.BigEndian.PutUint32(ph[8:], origin)
	binary.BigEndian.PutUint32(ph[12:], origin)
	binary.BigEndian.PutUint32(ph[16:], uint32(len(code)))
	binary.BigEndian.PutUint32(ph[20:], uint32(len(code)))
	binary.BigEndian.PutUint32(ph[24:], elfPfR|elfPfX)
	binary.BigEndian.PutUint32(ph[28:], 1)

	copy(out[textOffset:], code)
	copy(out[strtabOffset:], strtabBytes)
	copy(out[symtabOffset:], symtabBytes)
	copy(out[shstrtabOffset:], shstrtabBytes)
	encodeSectionHeaders(out[sectionOffset:], sections)
	return out
}

type elfStringTable struct {
	buf   []byte
	index map[string]uint32
}

func newELFStringTable() *elfStringTable {
	return &elfStringTable{
		buf:   []byte{0},
		index: map[string]uint32{"": 0},
	}
}

func (t *elfStringTable) add(name string) uint32 {
	if off, ok := t.index[name]; ok {
		return off
	}
	off := uint32(len(t.buf))
	t.buf = append(t.buf, name...)
	t.buf = append(t.buf, 0)
	t.index[name] = off
	return off
}

func (t *elfStringTable) bytes() []byte {
	return append([]byte(nil), t.buf...)
}

func elfStInfo(bind, typ byte) byte {
	return (bind << 4) | (typ & 0x0F)
}

func encodeELFSymbols(symbols []elfSymbol) []byte {
	out := make([]byte, len(symbols)*elfSymbolSize)
	for i, sym := range symbols {
		base := i * elfSymbolSize
		binary.BigEndian.PutUint32(out[base:], sym.name)
		binary.BigEndian.PutUint32(out[base+4:], sym.value)
		binary.BigEndian.PutUint32(out[base+8:], sym.size)
		out[base+12] = sym.info
		out[base+13] = sym.other
		binary.BigEndian.PutUint16(out[base+14:], sym.shndx)
	}
	return out
}

func encodeSectionHeaders(dst []byte, headers []elfSectionHeader) {
	for i, sh := range headers {
		base := i * sectionHeaderSize
		binary.BigEndian.PutUint32(dst[base:], sh.name)
		binary.BigEndian.PutUint32(dst[base+4:], sh.typ)
		binary.BigEndian.PutUint32(dst[base+8:], sh.flags)
		binary.BigEndian.PutUint32(dst[base+12:], sh.addr)
		binary.BigEndian.PutUint32(dst[base+16:], sh.offset)
		binary.BigEndian.PutUint32(dst[base+20:], sh.size)
		binary.BigEndian.PutUint32(dst[base+24:], sh.link)
		binary.BigEndian.PutUint32(dst[base+28:], sh.info)
		binary.BigEndian.PutUint32(dst[base+32:], sh.addralign)
		binary.BigEndian.PutUint32(dst[base+36:], sh.entsize)
	}
}

func alignOffset(v, align int) int {
	if align <= 1 {
		return v
	}
	mask := align - 1
	return (v + mask) &^ mask
}
