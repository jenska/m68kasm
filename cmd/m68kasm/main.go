package main

import (
	"flag"
	"fmt"
	"io"
	"os"
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
		fmt.Println("Usage: m68kasm -i input.s [-o out.bin] [--list out.lst] [--format bin|srec|elf]")
		os.Exit(1)
	}
	prog, err := asm.ParseFile(*in)
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
	if fmtFormat == "srec" {
		header := fmt.Sprintf("m68kasm v%s", m68kasm.Version)
		srec := asm.FormatSRecords(listing, prog.Origin, header)
		if err := os.WriteFile(*out, srec, 0644); err != nil {
			fmt.Println("write error:", err)
			os.Exit(4)
		}
		fmt.Printf("assembled %d bytes into S-record %s\n", len(bytes), *out)
	} else if fmtFormat == "elf" {
		elfBytes := asm.FormatELF(bytes, prog.Origin)
		if err := os.WriteFile(*out, elfBytes, 0644); err != nil {
			fmt.Println("write error:", err)
			os.Exit(4)
		}
		fmt.Printf("assembled %d bytes into ELF %s\n", len(bytes), *out)
	} else {
		if err := os.WriteFile(*out, bytes, 0644); err != nil {
			fmt.Println("write error:", err)
			os.Exit(4)
		}
		fmt.Printf("wrote %d bytes to %s\n", len(bytes), *out)
	}
	if *list != "" {
		if err := writeListing(*list, listing, *in); err != nil {
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
