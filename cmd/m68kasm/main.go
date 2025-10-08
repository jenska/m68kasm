package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jenska/m68kasm/internal/asm"
)

func main() {
	in := flag.String("i", "", "input assembly file")
	out := flag.String("o", "out.bin", "output binary file")
	flag.Parse()
	if *in == "" {
		fmt.Println("Usage: m68kasm -i input.s [-o out.bin]")
		os.Exit(1)
	}
	prog, err := asm.ParseFile(*in)
	if err != nil {
		fmt.Println("parse error:", err)
		os.Exit(2)
	}
	bytes, err := asm.Assemble(prog)
	if err != nil {
		fmt.Println("assemble error:", err)
		os.Exit(3)
	}
	if err := os.WriteFile(*out, bytes, 0644); err != nil {
		fmt.Println("write error:", err)
		os.Exit(4)
	}
	fmt.Printf("wrote %d bytes to %s\n", len(bytes), *out)
}
