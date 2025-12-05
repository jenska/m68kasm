package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/jenska/m68kasm"
)

const source = `.org 0x1000
start:
    MOVEQ #1,D0
    MOVEQ #0,D1
    BRA start
`

func main() {
	bin, listing, err := m68kasm.AssembleStringWithListing(source)
	if err != nil {
		log.Fatalf("assemble: %v", err)
	}

	fmt.Printf("Assembled %d bytes:\n%s\n", len(bin), hex.Dump(bin))
	fmt.Println("Listing entries:")

	for _, entry := range listing {
		fmt.Printf("line %d @ 0x%04x => % x\n", entry.Line, entry.PC, entry.Bytes)
	}
}
