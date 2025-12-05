package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jenska/m68kasm"
	"github.com/jenska/m68kasm/internal/asm"
)

func main() {
	in := flag.String("i", "", "input assembly file")
	out := flag.String("o", "out.bin", "output binary file")
	list := flag.String("list", "", "write listing output (use '-' for stdout)")
	format := flag.String("format", "bin", "output format: bin, srec, or elf")
	showVersion := flag.Bool("version", false, "print assembler version and exit")
	var includePaths multiFlag
	defines := make(defineFlag)

	flag.Var(&includePaths, "I", "add include search path")
	flag.Var(&defines, "D", "define symbol (NAME or NAME=VALUE)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("m68kasm v%s\n", m68kasm.Version)
		return
	}

	fmtFormat := strings.ToLower(*format)
	if fmtFormat != "bin" && fmtFormat != "srec" && fmtFormat != "elf" {
		fmt.Println("unknown format:", *format)
		os.Exit(1)
	}
	if *in == "" {
		fmt.Println("Usage: m68kasm -i input.s [-o out.bin] [--list out.lst] [--format bin|srec|elf] [-I path] [-D name[=val]]")
		os.Exit(1)
	}
	srcPath, err := resolveInputPath(*in, includePaths)
	if err != nil {
		fmt.Println("input error:", err)
		os.Exit(1)
	}

	prog, err := asm.ParseFileWithOptions(srcPath, asm.ParseOptions{Symbols: defines})
	if err != nil {
		fmt.Println("assemble error:", err)
		os.Exit(2)
	}
	var (
		listing []asm.ListingEntry
		bytes   []byte
	)
	wantListing := *list != "" || fmtFormat == "srec"
	if wantListing {
		bytes, listing, err = asm.AssembleWithListing(prog)
	} else {
		bytes, err = asm.Assemble(prog)
	}
	if err != nil {
		fmt.Println("assemble error:", err)
		os.Exit(3)
	}
	switch fmtFormat {

	case "srec":
		header := fmt.Sprintf("m68kasm v%s", m68kasm.Version)
		srec := asm.FormatSRecords(listing, prog.Origin, header)
		if err := os.WriteFile(*out, srec, 0644); err != nil {
			fmt.Println("write error:", err)
			os.Exit(4)
		}
		fmt.Printf("assembled %d bytes into S-record %s\n", len(bytes), *out)
	case "elf":
		elfBytes := asm.FormatELF(bytes, prog.Origin)
		if err := os.WriteFile(*out, elfBytes, 0644); err != nil {
			fmt.Println("write error:", err)
			os.Exit(4)
		}
		fmt.Printf("assembled %d bytes into ELF %s\n", len(bytes), *out)
	default:
		if err := os.WriteFile(*out, bytes, 0644); err != nil {
			fmt.Println("write error:", err)
			os.Exit(4)
		}
		fmt.Printf("wrote %d bytes to %s\n", len(bytes), *out)
	}
	if *list != "" {
		if err := writeListing(*list, listing, srcPath); err != nil {
			fmt.Println("listing error:", err)
			os.Exit(5)
		}
	}
}

func writeListing(path string, entries []asm.ListingEntry, srcPath string) error {
	lines, err := readLines(srcPath)
	if err != nil {
		return err
	}

	w, closeFn, err := listingWriter(path)
	if err != nil {
		return err
	}
	if closeFn != nil {
		defer closeFn()
	}

	fmt.Fprintln(w, "Line  Address    Bytes                     Source")
	fmt.Fprintln(w, "----- -------- -------------------------------- ------------------------------")
	for _, e := range entries {
		lineText := ""
		if idx := e.Line - 1; idx >= 0 && idx < len(lines) {
			lineText = lines[idx]
		}
		fmt.Fprintf(w, "%5d  0x%08X  %-32s %s\n", e.Line, e.PC, formatBytes(e.Bytes), lineText)
	}
	return nil
}

func listingWriter(path string) (io.Writer, func(), error) {
	if path == "-" {
		return os.Stdout, nil, nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { _ = f.Close() }, nil
}

func readLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.TrimRight(text, "\n")
	return strings.Split(text, "\n"), nil
}

func formatBytes(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, v := range b {
		if i > 0 {
			sb.WriteByte(' ')
		}
		fmt.Fprintf(&sb, "%02X", v)
	}
	return sb.String()
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, string(os.PathListSeparator))
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

type defineFlag map[string]uint32

func (d defineFlag) String() string {
	parts := make([]string, 0, len(d))
	for k, v := range d {
		parts = append(parts, fmt.Sprintf("%s=%d", k, v))
	}
	return strings.Join(parts, ",")
}

func (d defineFlag) Set(value string) error {
	name, val, err := parseDefine(value)
	if err != nil {
		return err
	}
	d[name] = val
	return nil
}

func parseDefine(val string) (string, uint32, error) {
	parts := strings.SplitN(val, "=", 2)
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return "", 0, fmt.Errorf("invalid symbol name in -D: %q", val)
	}

	v := uint64(1)
	if len(parts) == 2 {
		parsed, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 0, 32)
		if err != nil {
			return "", 0, fmt.Errorf("invalid value for -D %s: %w", name, err)
		}
		v = parsed
	}
	return name, uint32(v), nil
}

func resolveInputPath(path string, includePaths []string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("no input file provided")
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	for _, dir := range includePaths {
		candidate := filepath.Join(dir, path)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("cannot find %s (searched: %s)", path, strings.Join(includePaths, string(os.PathListSeparator)))
}
