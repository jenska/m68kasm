package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jenska/m68kasm"
)

const source = `.org 0x2000
start:
    MOVEQ #7,D0
    MOVEQ #3,D1
    ADD.L D1,D0
    BRA start
`

func main() {
	bin, err := m68kasm.AssembleString(source)
	if err != nil {
		log.Fatalf("assemble binary: %v", err)
	}
	writeFile("example.bin", bin)

	srec, err := m68kasm.AssembleStringSRecord(source)
	if err != nil {
		log.Fatalf("assemble s-record: %v", err)
	}
	writeFile("example.srec", srec)

	elf, err := m68kasm.AssembleStringELF(source)
	if err != nil {
		log.Fatalf("assemble elf: %v", err)
	}
	writeFile("example.elf", elf)

	fmt.Println("Wrote example.bin, example.srec, and example.elf in the current directory")
}

func writeFile(name string, contents []byte) {
	if err := os.WriteFile(name, contents, 0o644); err != nil {
		log.Fatalf("write %s: %v", name, err)
	}
}
