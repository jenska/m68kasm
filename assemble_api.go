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

// Error provides source-location context for parse and assembly failures.
type Error = internal.Error
type ParseOptions = internal.ParseOptions

// Assemble parses Motorola 68k assembly source from r and returns the encoded
// machine code bytes. The program origin, if specified via directives, is
// accounted for in the parser but the returned slice contains only the
// assembled bytes.
func Assemble(r io.Reader) ([]byte, error) {
	return AssembleWithOptions(nil, r, ParseOptions{})
}

// AssembleStream assembles Motorola 68k source from r and streams the encoded
// bytes to w without buffering the entire program in memory.
func AssembleStream(w io.Writer, r io.Reader) (int64, error) {
	return AssembleStreamWithOptions(w, r, ParseOptions{})
}

// AssembleStreamWithOptions assembles Motorola 68k source from r using the
// supplied parsing options and streams the encoded bytes to w. The number of
// bytes written is returned.
func AssembleStreamWithOptions(w io.Writer, r io.Reader, opts ParseOptions) (int64, error) {
	prog, err := internal.ParseWithOptions(r, internal.ParseOptions(opts))
	if err != nil {
		return 0, err
	}

	return internal.AssembleStream(w, prog)
}

// AssembleStreamWithListing streams assembled bytes to w and also returns
// listing metadata per source line.
func AssembleStreamWithListing(w io.Writer, r io.Reader) (int64, []ListingEntry, error) {
	return AssembleStreamWithListingWithOptions(w, r, ParseOptions{})
}

// AssembleStreamWithListingWithOptions streams assembled bytes to w using the
// supplied parsing options and returns listing metadata per source line.
func AssembleStreamWithListingWithOptions(w io.Writer, r io.Reader, opts ParseOptions) (int64, []ListingEntry, error) {
	prog, err := internal.ParseWithOptions(r, internal.ParseOptions(opts))
	if err != nil {
		return 0, nil, err
	}

	return internal.AssembleStreamWithListing(w, prog)
}

// AssembleWithListing parses Motorola 68k assembly source from r, returns the
// encoded machine code bytes, and also provides listing metadata per source
// line.
func AssembleWithListing(r io.Reader) ([]byte, []ListingEntry, error) {
	return AssembleWithListingWithOptions(nil, r, ParseOptions{})
}

// AssembleInto parses Motorola 68k assembly source from r and appends the
// encoded machine code bytes to dst.
func AssembleInto(dst []byte, r io.Reader) ([]byte, error) {
	return AssembleWithOptions(dst, r, ParseOptions{})
}

// AssembleWithListingInto parses Motorola 68k assembly source from r, appends
// the encoded machine code bytes to dst, and returns listing metadata per
// source line.
func AssembleWithListingInto(dst []byte, r io.Reader) ([]byte, []ListingEntry, error) {
	return AssembleWithListingWithOptions(dst, r, ParseOptions{})
}

// AssembleWithOptions parses Motorola 68k assembly source from r, applies the
// provided parsing options (such as predefined symbols), and appends the
// encoded machine code bytes to dst.
func AssembleWithOptions(dst []byte, r io.Reader, opts ParseOptions) ([]byte, error) {
	prog, err := internal.ParseWithOptions(r, internal.ParseOptions(opts))
	if err != nil {
		return nil, err
	}

	return internal.AssembleInto(dst, prog)
}

// AssembleWithListingWithOptions parses Motorola 68k assembly source from r,
// applies the provided parsing options (such as predefined symbols), appends
// the encoded machine code bytes to dst, and returns listing metadata per
// source line.
func AssembleWithListingWithOptions(dst []byte, r io.Reader, opts ParseOptions) ([]byte, []ListingEntry, error) {
	prog, err := internal.ParseWithOptions(r, internal.ParseOptions(opts))
	if err != nil {
		return nil, nil, err
	}

	return internal.AssembleWithListingInto(dst, prog)
}

// AssembleBytes assembles Motorola 68k source provided as a byte slice.
func AssembleBytes(src []byte) ([]byte, error) {
	return Assemble(bytes.NewReader(src))
}

// AssembleBytesWithOptions assembles Motorola 68k source provided as a byte
// slice using the supplied parsing options.
func AssembleBytesWithOptions(src []byte, opts ParseOptions) ([]byte, error) {
	return AssembleWithOptions(nil, bytes.NewReader(src), opts)
}

// AssembleBytesWithListing assembles Motorola 68k source provided as a byte
// slice and returns both the encoded bytes and listing metadata.
func AssembleBytesWithListing(src []byte) ([]byte, []ListingEntry, error) {
	return AssembleWithListing(bytes.NewReader(src))
}

// AssembleBytesWithListingWithOptions assembles Motorola 68k source provided as
// a byte slice using the supplied parsing options and returns both the encoded
// bytes and listing metadata.
func AssembleBytesWithListingWithOptions(src []byte, opts ParseOptions) ([]byte, []ListingEntry, error) {
	return AssembleWithListingWithOptions(nil, bytes.NewReader(src), opts)
}

// AssembleBytesInto assembles Motorola 68k source provided as a byte slice and
// appends the encoded bytes to dst.
func AssembleBytesInto(dst, src []byte) ([]byte, error) {
	return AssembleInto(dst, bytes.NewReader(src))
}

// AssembleBytesIntoWithOptions assembles Motorola 68k source provided as a byte
// slice, applies the supplied parsing options, and appends the encoded bytes to
// dst.
func AssembleBytesIntoWithOptions(dst, src []byte, opts ParseOptions) ([]byte, error) {
	return AssembleWithOptions(dst, bytes.NewReader(src), opts)
}

// AssembleBytesWithListingInto assembles Motorola 68k source provided as a
// byte slice, appends the encoded bytes to dst, and returns listing metadata.
func AssembleBytesWithListingInto(dst, src []byte) ([]byte, []ListingEntry, error) {
	return AssembleWithListingInto(dst, bytes.NewReader(src))
}

// AssembleBytesWithListingIntoWithOptions assembles Motorola 68k source
// provided as a byte slice, appends the encoded machine code bytes to dst,
// applies the supplied parsing options, and returns listing metadata.
func AssembleBytesWithListingIntoWithOptions(dst, src []byte, opts ParseOptions) ([]byte, []ListingEntry, error) {
	return AssembleWithListingWithOptions(dst, bytes.NewReader(src), opts)
}

// AssembleString assembles Motorola 68k source provided as a string.
func AssembleString(src string) ([]byte, error) {
	return Assemble(bytes.NewBufferString(src))
}

// AssembleStringWithOptions assembles Motorola 68k source provided as a string
// using the supplied parsing options.
func AssembleStringWithOptions(src string, opts ParseOptions) ([]byte, error) {
	return AssembleWithOptions(nil, bytes.NewBufferString(src), opts)
}

// AssembleStringWithListing assembles Motorola 68k source provided as a
// string and returns listing metadata.
func AssembleStringWithListing(src string) ([]byte, []ListingEntry, error) {
	return AssembleWithListing(bytes.NewBufferString(src))
}

// AssembleStringWithListingWithOptions assembles Motorola 68k source provided
// as a string using the supplied parsing options and returns listing metadata.
func AssembleStringWithListingWithOptions(src string, opts ParseOptions) ([]byte, []ListingEntry, error) {
	return AssembleWithListingWithOptions(nil, bytes.NewBufferString(src), opts)
}

// AssembleStringInto assembles Motorola 68k source provided as a string and
// appends the encoded bytes to dst.
func AssembleStringInto(dst []byte, src string) ([]byte, error) {
	return AssembleInto(dst, bytes.NewBufferString(src))
}

// AssembleStringIntoWithOptions assembles Motorola 68k source provided as a
// string, applies the supplied parsing options, and appends the encoded bytes
// to dst.
func AssembleStringIntoWithOptions(dst []byte, src string, opts ParseOptions) ([]byte, error) {
	return AssembleWithOptions(dst, bytes.NewBufferString(src), opts)
}

// AssembleStringWithListingInto assembles Motorola 68k source provided as a
// string, appends the encoded bytes to dst, and returns listing metadata.
func AssembleStringWithListingInto(dst []byte, src string) ([]byte, []ListingEntry, error) {
	return AssembleWithListingInto(dst, bytes.NewBufferString(src))
}

// AssembleStringWithListingIntoWithOptions assembles Motorola 68k source
// provided as a string, applies the supplied parsing options, appends the
// encoded bytes to dst, and returns listing metadata.
func AssembleStringWithListingIntoWithOptions(dst []byte, src string, opts ParseOptions) ([]byte, []ListingEntry, error) {
	return AssembleWithListingWithOptions(dst, bytes.NewBufferString(src), opts)
}

// AssembleFile assembles a Motorola 68k source file specified by path.
func AssembleFile(path string) ([]byte, error) {
	return AssembleFileInto(nil, path)
}

// AssembleFileWithOptions assembles a Motorola 68k source file specified by
// path using the supplied parsing options.
func AssembleFileWithOptions(path string, opts ParseOptions) ([]byte, error) {
	return AssembleFileIntoWithOptions(nil, path, opts)
}

// AssembleFileWithListing assembles a Motorola 68k source file specified by
// path and returns listing metadata.
func AssembleFileWithListing(path string) ([]byte, []ListingEntry, error) {
	return AssembleFileWithListingInto(nil, path)
}

// AssembleFileWithListingWithOptions assembles a Motorola 68k source file
// specified by path using the supplied parsing options and returns listing
// metadata.
func AssembleFileWithListingWithOptions(path string, opts ParseOptions) ([]byte, []ListingEntry, error) {
	return AssembleFileWithListingIntoWithOptions(nil, path, opts)
}

// AssembleFileInto assembles a Motorola 68k source file specified by path and
// appends the encoded bytes to dst.
func AssembleFileInto(dst []byte, path string) ([]byte, error) {
	return AssembleFileIntoWithOptions(dst, path, ParseOptions{})
}

// AssembleFileWithListingInto assembles a Motorola 68k source file specified by
// path, appends the encoded bytes to dst, and returns listing metadata.
func AssembleFileWithListingInto(dst []byte, path string) ([]byte, []ListingEntry, error) {
	return AssembleFileWithListingIntoWithOptions(dst, path, ParseOptions{})
}

// AssembleFileIntoWithOptions assembles a Motorola 68k source file specified by
// path, applies the supplied parsing options, and appends the encoded bytes to
// dst.
func AssembleFileIntoWithOptions(dst []byte, path string, opts ParseOptions) ([]byte, error) {
	prog, err := internal.ParseFileWithOptions(path, internal.ParseOptions(opts))
	if err != nil {
		return nil, err
	}
	return internal.AssembleInto(dst, prog)
}

// AssembleFileWithListingIntoWithOptions assembles a Motorola 68k source file
// specified by path, applies the supplied parsing options, appends the encoded
// bytes to dst, and returns listing metadata.
func AssembleFileWithListingIntoWithOptions(dst []byte, path string, opts ParseOptions) ([]byte, []ListingEntry, error) {
	prog, err := internal.ParseFileWithOptions(path, internal.ParseOptions(opts))
	if err != nil {
		return nil, nil, err
	}
	return internal.AssembleWithListingInto(dst, prog)
}

// AssembleELF parses Motorola 68k assembly source from r and returns an ELF32
// executable image targeting the m68k architecture. The program origin is used
// as the entry point and load address.
func AssembleELF(r io.Reader) ([]byte, error) {
	return AssembleELFWithOptions(r, ParseOptions{})
}

// AssembleELFWithOptions parses Motorola 68k assembly source from r using the
// supplied parsing options and returns an ELF32 executable image targeting the
// m68k architecture.
func AssembleELFWithOptions(r io.Reader, opts ParseOptions) ([]byte, error) {
	prog, err := internal.ParseWithOptions(r, internal.ParseOptions(opts))
	if err != nil {
		return nil, err
	}

	return internal.AssembleELF(prog)
}

// AssembleBytesELF assembles Motorola 68k source provided as a byte slice and
// returns an ELF32 executable image targeting the m68k architecture.
func AssembleBytesELF(src []byte) ([]byte, error) {
	return AssembleELF(bytes.NewReader(src))
}

// AssembleStringELF assembles Motorola 68k source provided as a string and
// returns an ELF32 executable image targeting the m68k architecture.
func AssembleStringELF(src string) ([]byte, error) {
	return AssembleELF(strings.NewReader(src))
}

// AssembleFileELF assembles a Motorola 68k source file specified by path and
// returns an ELF32 executable image targeting the m68k architecture.
func AssembleFileELF(path string) ([]byte, error) {
	prog, err := internal.ParseFileWithOptions(path, internal.ParseOptions{})
	if err != nil {
		return nil, err
	}
	return internal.AssembleELF(prog)
}

// AssembleSRecord parses Motorola 68k assembly source from r and returns a
// Motorola S-record representation using the current assembler version as the
// header.
func AssembleSRecord(r io.Reader) ([]byte, error) {
	return AssembleSRecordWithOptions(r, ParseOptions{})
}

// AssembleSRecordWithOptions parses Motorola 68k assembly source from r using
// the supplied parsing options and returns a Motorola S-record representation
// using the current assembler version as the header.
func AssembleSRecordWithOptions(r io.Reader, opts ParseOptions) ([]byte, error) {
	prog, err := internal.ParseWithOptions(r, internal.ParseOptions(opts))
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
