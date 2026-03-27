package m68kasm

import (
	"bytes"
	"errors"
	"io"
	"os"

	internal "github.com/jenska/m68kasm/internal/asm"
)

// DefinedLabel captures a named label defined in source.
type DefinedLabel = internal.DefinedLabel

// InstructionMetadata describes a single assembled instruction.
type InstructionMetadata struct {
	Line      int
	PC        uint32
	Bytes     []byte
	Size      int
	Words     int
	Canonical string
}

// AssemblyResult captures both encoded bytes and test-friendly metadata.
type AssemblyResult struct {
	Bytes         []byte
	Origin        uint32
	Labels        map[string]uint32
	DefinedLabels []DefinedLabel
	LineAddresses map[int]uint32
	Listing       []ListingEntry
	Instructions  []InstructionMetadata
}

// AddressOf resolves a named source label to its assembled address.
func (r *AssemblyResult) AddressOf(name string) (uint32, bool) {
	if r == nil {
		return 0, false
	}
	addr, ok := r.Labels[name]
	return addr, ok
}

// AddressForLine returns the first assembled address associated with a source line.
func (r *AssemblyResult) AddressForLine(line int) (uint32, bool) {
	if r == nil {
		return 0, false
	}
	addr, ok := r.LineAddresses[line]
	return addr, ok
}

// NormalizeError strips source-location prefixes from assembler errors so tests
// can assert the stable message independently of line and column numbers.
func NormalizeError(err error) string {
	if err == nil {
		return ""
	}
	var asmErr *Error
	if errors.As(err, &asmErr) {
		return asmErr.Message()
	}
	return err.Error()
}

// AssembleDetailed assembles source and returns bytes plus label and line metadata.
func AssembleDetailed(r io.Reader) (*AssemblyResult, error) {
	return AssembleDetailedWithOptions(r, ParseOptions{})
}

// AssembleDetailedWithOptions assembles source with parser options and returns
// bytes plus test-friendly metadata.
func AssembleDetailedWithOptions(r io.Reader, opts ParseOptions) (*AssemblyResult, error) {
	prog, err := internal.ParseWithOptions(r, internal.ParseOptions(opts))
	if err != nil {
		return nil, err
	}
	return buildAssemblyResult(prog)
}

// AssembleBytesDetailed assembles byte-slice source and returns detailed metadata.
func AssembleBytesDetailed(src []byte) (*AssemblyResult, error) {
	return AssembleDetailed(bytes.NewReader(src))
}

// AssembleBytesDetailedWithOptions assembles byte-slice source and returns detailed metadata.
func AssembleBytesDetailedWithOptions(src []byte, opts ParseOptions) (*AssemblyResult, error) {
	return AssembleDetailedWithOptions(bytes.NewReader(src), opts)
}

// AssembleStringDetailed assembles string source and returns detailed metadata.
func AssembleStringDetailed(src string) (*AssemblyResult, error) {
	return AssembleDetailed(bytes.NewBufferString(src))
}

// AssembleStringDetailedWithOptions assembles string source and returns detailed metadata.
func AssembleStringDetailedWithOptions(src string, opts ParseOptions) (*AssemblyResult, error) {
	return AssembleDetailedWithOptions(bytes.NewBufferString(src), opts)
}

// AssembleFileDetailed assembles a file and returns detailed metadata.
func AssembleFileDetailed(path string) (*AssemblyResult, error) {
	return AssembleFileDetailedWithOptions(path, ParseOptions{})
}

// AssembleFileDetailedWithOptions assembles a file and returns detailed metadata.
func AssembleFileDetailedWithOptions(path string, opts ParseOptions) (*AssemblyResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return AssembleDetailedWithOptions(f, opts)
}

func buildAssemblyResult(prog *internal.Program) (*AssemblyResult, error) {
	out, listing, err := internal.AssembleWithListing(prog)
	if err != nil {
		return nil, err
	}

	result := &AssemblyResult{
		Bytes:         out,
		Origin:        prog.Origin,
		Labels:        make(map[string]uint32, len(prog.DefinedLabels)),
		DefinedLabels: append([]DefinedLabel(nil), prog.DefinedLabels...),
		Listing:       cloneListing(listing),
		LineAddresses: make(map[int]uint32, len(listing)+len(prog.DefinedLabels)),
	}

	for _, label := range prog.DefinedLabels {
		result.Labels[label.Name] = label.Addr
		if _, exists := result.LineAddresses[label.Line]; !exists && label.Line > 0 {
			result.LineAddresses[label.Line] = label.Addr
		}
	}

	for _, entry := range result.Listing {
		if _, exists := result.LineAddresses[entry.Line]; !exists && entry.Line > 0 {
			result.LineAddresses[entry.Line] = entry.PC
		}
	}

	result.Instructions = collectInstructionMetadata(prog.Items, result.Listing)
	return result, nil
}

func cloneListing(listing []ListingEntry) []ListingEntry {
	if len(listing) == 0 {
		return nil
	}
	out := make([]ListingEntry, len(listing))
	for i, entry := range listing {
		out[i] = ListingEntry{
			Line:  entry.Line,
			PC:    entry.PC,
			Bytes: append([]byte(nil), entry.Bytes...),
		}
	}
	return out
}

func collectInstructionMetadata(items []any, listing []ListingEntry) []InstructionMetadata {
	if len(items) == 0 || len(listing) == 0 {
		return nil
	}

	metadata := make([]InstructionMetadata, 0, len(items))
	listingIdx := 0
	for _, item := range items {
		if listingIdx >= len(listing) {
			break
		}
		entry := listing[listingIdx]
		listingIdx++

		ins, ok := item.(*internal.Instr)
		if !ok {
			continue
		}

		md := InstructionMetadata{
			Line:      ins.Line,
			PC:        ins.PC,
			Bytes:     append([]byte(nil), entry.Bytes...),
			Size:      len(entry.Bytes),
			Words:     len(entry.Bytes) / 2,
			Canonical: canonicalInstruction(ins),
		}
		metadata = append(metadata, md)
	}

	if len(metadata) == 0 {
		return nil
	}
	return metadata
}
