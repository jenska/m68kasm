package asm

import (
	"encoding/binary"
)

const (
	elfHeaderSize     = 52
	programHeaderSize = 32
)

// AssembleELF assembles the given program and wraps the output bytes in a
// minimal ELF32 header suitable for m68k emulators and loaders.
func AssembleELF(p *Program) ([]byte, error) {
	bytes, _, err := assemble(nil, p, false)
	if err != nil {
		return nil, err
	}

	return FormatELF(bytes, p.Origin), nil
}

// FormatELF wraps the provided machine code bytes into a 32-bit big-endian ELF
// executable image for the Motorola 68000 (EM_68K). The entry point and
// segment addresses are set to origin.
func FormatELF(code []byte, origin uint32) []byte {
	dataOffset := elfHeaderSize + programHeaderSize
	out := make([]byte, dataOffset+len(code))

	// e_ident
	out[0] = 0x7F
	out[1] = 'E'
	out[2] = 'L'
	out[3] = 'F'
	out[4] = 1 // ELFCLASS32
	out[5] = 2 // ELFDATA2MSB (big-endian)
	out[6] = 1 // EV_CURRENT

	binary.BigEndian.PutUint16(out[16:], 2) // e_type = ET_EXEC
	binary.BigEndian.PutUint16(out[18:], 4) // e_machine = EM_68K
	binary.BigEndian.PutUint32(out[20:], 1) // e_version = EV_CURRENT
	binary.BigEndian.PutUint32(out[24:], origin)
	binary.BigEndian.PutUint32(out[28:], elfHeaderSize) // e_phoff
	// e_shoff left zeroed because we omit section headers
	// e_flags remains zero
	binary.BigEndian.PutUint16(out[40:], elfHeaderSize)     // e_ehsize
	binary.BigEndian.PutUint16(out[42:], programHeaderSize) // e_phentsize
	binary.BigEndian.PutUint16(out[44:], 1)                 // e_phnum
	// section header fields remain zero because they are omitted

	// Program header
	ph := out[elfHeaderSize:]
	binary.BigEndian.PutUint32(ph[0:], 1) // p_type = PT_LOAD
	binary.BigEndian.PutUint32(ph[4:], uint32(dataOffset))
	binary.BigEndian.PutUint32(ph[8:], origin)
	binary.BigEndian.PutUint32(ph[12:], origin)
	binary.BigEndian.PutUint32(ph[16:], uint32(len(code)))
	binary.BigEndian.PutUint32(ph[20:], uint32(len(code)))
	binary.BigEndian.PutUint32(ph[24:], 0x5) // p_flags = PF_R | PF_X
	binary.BigEndian.PutUint32(ph[28:], 4)   // p_align

	copy(out[dataOffset:], code)
	return out
}
