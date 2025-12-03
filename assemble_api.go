package m68kasm

import (
	"bytes"
	"io"

	internal "github.com/jenska/m68kasm/internal/asm"
)

// Assemble parses Motorola 68k assembly source from r and returns the encoded
// machine code bytes. The program origin, if specified via directives, is
// accounted for in the parser but the returned slice contains only the
// assembled bytes.
func Assemble(r io.Reader) ([]byte, error) {
	return AssembleInto(nil, r)
}

// AssembleInto parses Motorola 68k assembly source from r and appends the
// encoded machine code bytes to dst.
func AssembleInto(dst []byte, r io.Reader) ([]byte, error) {
	prog, err := internal.Parse(r)
	if err != nil {
		return nil, err
	}

	return internal.AssembleInto(dst, prog)
}

// AssembleBytes assembles Motorola 68k source provided as a byte slice.
func AssembleBytes(src []byte) ([]byte, error) {
	return Assemble(bytes.NewReader(src))
}

// AssembleBytesInto assembles Motorola 68k source provided as a byte slice and
// appends the encoded bytes to dst.
func AssembleBytesInto(dst, src []byte) ([]byte, error) {
	return AssembleInto(dst, bytes.NewReader(src))
}

// AssembleString assembles Motorola 68k source provided as a string.
func AssembleString(src string) ([]byte, error) {
	return Assemble(bytes.NewBufferString(src))
}

// AssembleStringInto assembles Motorola 68k source provided as a string and
// appends the encoded bytes to dst.
func AssembleStringInto(dst []byte, src string) ([]byte, error) {
	return AssembleInto(dst, bytes.NewBufferString(src))
}

// AssembleFile assembles a Motorola 68k source file specified by path.
func AssembleFile(path string) ([]byte, error) {
	return AssembleFileInto(nil, path)
}

// AssembleFileInto assembles a Motorola 68k source file specified by path and
// appends the encoded bytes to dst.
func AssembleFileInto(dst []byte, path string) ([]byte, error) {
	prog, err := internal.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return internal.AssembleInto(dst, prog)
}
