package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jenska/m68kasm/internal/asm"
)

func main() {
	in := flag.String("i", "", "input assembly file")
	out := flag.String("o", "out.bin", "output binary file")
	list := flag.String("list", "", "write listing output (use '-' for stdout)")
	flag.Parse()
	if *in == "" {
		fmt.Println("Usage: m68kasm -i input.s [-o out.bin] [--list out.lst]")
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
	if *list != "" {
		bytes, listing, err = asm.AssembleWithListing(prog)
	} else {
		bytes, err = asm.Assemble(prog)
	}
	if err != nil {
		fmt.Println("assemble error:", err)
		os.Exit(3)
	}
	if err := os.WriteFile(*out, bytes, 0644); err != nil {
		fmt.Println("write error:", err)
		os.Exit(4)
	}
	if *list != "" {
		if err := writeListing(*list, listing, *in); err != nil {
			fmt.Println("listing error:", err)
			os.Exit(5)
		}
	}
	fmt.Printf("wrote %d bytes to %s\n", len(bytes), *out)
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
