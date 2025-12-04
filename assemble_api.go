package m68kasm

import (
	"bytes"
	"io"
	"strings"

	internal "github.com/jenska/m68kasm/internal/asm"
)

// ListingEntry describes the assembled bytes for a single source line. It can
// be used to generate human-readable listing output alongside assembled
// binaries.
type ListingEntry = internal.ListingEntry

// Assemble parses Motorola 68k assembly source from r and returns the encoded
// machine code bytes. The program origin, if specified via directives, is
// accounted for in the parser but the returned slice contains only the
// assembled bytes.
func Assemble(r io.Reader) ([]byte, error) {
	return AssembleInto(nil, r)
}

// AssembleWithListing parses Motorola 68k assembly source from r, returns the
// encoded machine code bytes, and also provides listing metadata per source
// line.
func AssembleWithListing(r io.Reader) ([]byte, []ListingEntry, error) {
	return AssembleWithListingInto(nil, r)
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

// AssembleWithListingInto parses Motorola 68k assembly source from r, appends
// the encoded machine code bytes to dst, and returns listing metadata per
// source line.
func AssembleWithListingInto(dst []byte, r io.Reader) ([]byte, []ListingEntry, error) {
	prog, err := internal.Parse(r)
	if err != nil {
		return nil, nil, err
	}

	return internal.AssembleWithListingInto(dst, prog)
}

// AssembleBytes assembles Motorola 68k source provided as a byte slice.
func AssembleBytes(src []byte) ([]byte, error) {
	return Assemble(bytes.NewReader(src))
}

// AssembleBytesWithListing assembles Motorola 68k source provided as a byte
// slice and returns both the encoded bytes and listing metadata.
func AssembleBytesWithListing(src []byte) ([]byte, []ListingEntry, error) {
	return AssembleWithListing(bytes.NewReader(src))
}

// AssembleBytesInto assembles Motorola 68k source provided as a byte slice and
// appends the encoded bytes to dst.
func AssembleBytesInto(dst, src []byte) ([]byte, error) {
	return AssembleInto(dst, bytes.NewReader(src))
}

// AssembleBytesWithListingInto assembles Motorola 68k source provided as a
// byte slice, appends the encoded bytes to dst, and returns listing metadata.
func AssembleBytesWithListingInto(dst, src []byte) ([]byte, []ListingEntry, error) {
	return AssembleWithListingInto(dst, bytes.NewReader(src))
}

// AssembleString assembles Motorola 68k source provided as a string.
func AssembleString(src string) ([]byte, error) {
	return Assemble(bytes.NewBufferString(src))
}

// AssembleStringWithListing assembles Motorola 68k source provided as a
// string and returns listing metadata.
func AssembleStringWithListing(src string) ([]byte, []ListingEntry, error) {
	return AssembleWithListing(bytes.NewBufferString(src))
}

// AssembleStringInto assembles Motorola 68k source provided as a string and
// appends the encoded bytes to dst.
func AssembleStringInto(dst []byte, src string) ([]byte, error) {
	return AssembleInto(dst, bytes.NewBufferString(src))
}

// AssembleStringWithListingInto assembles Motorola 68k source provided as a
// string, appends the encoded bytes to dst, and returns listing metadata.
func AssembleStringWithListingInto(dst []byte, src string) ([]byte, []ListingEntry, error) {
	return AssembleWithListingInto(dst, bytes.NewBufferString(src))
}

// AssembleFile assembles a Motorola 68k source file specified by path.
func AssembleFile(path string) ([]byte, error) {
	return AssembleFileInto(nil, path)
}

// AssembleFileWithListing assembles a Motorola 68k source file specified by
// path and returns listing metadata.
func AssembleFileWithListing(path string) ([]byte, []ListingEntry, error) {
	return AssembleFileWithListingInto(nil, path)
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

// AssembleFileWithListingInto assembles a Motorola 68k source file specified by
// path, appends the encoded bytes to dst, and returns listing metadata.
func AssembleFileWithListingInto(dst []byte, path string) ([]byte, []ListingEntry, error) {
	prog, err := internal.ParseFile(path)
	if err != nil {
		return nil, nil, err
	}
	return internal.AssembleWithListingInto(dst, prog)
}

// AssembleSRecord parses Motorola 68k assembly source from r and returns a
// Motorola S-record representation using the current assembler version as the
// header.
func AssembleSRecord(r io.Reader) ([]byte, error) {
	prog, err := internal.Parse(r)
	if err != nil {
		return nil, err
	}

	return internal.AssembleSRecord(prog, srecHeader())
}

// AssembleBytesSRecord assembles Motorola 68k source provided as a byte slice
// and returns a Motorola S-record representation.
func AssembleBytesSRecord(src []byte) ([]byte, error) {
	return AssembleSRecord(bytes.NewReader(src))
}

// AssembleStringSRecord assembles Motorola 68k source provided as a string and
// returns a Motorola S-record representation.
func AssembleStringSRecord(src string) ([]byte, error) {
	return AssembleSRecord(strings.NewReader(src))
}

// AssembleFileSRecord assembles a Motorola 68k source file specified by path
// and returns a Motorola S-record representation.
func AssembleFileSRecord(path string) ([]byte, error) {
	prog, err := internal.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return internal.AssembleSRecord(prog, srecHeader())
}

func srecHeader() string {
	return "m68kasm v" + Version
}
