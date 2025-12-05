package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/jenska/m68kasm"
)

const source = `; Macro-heavy example showing parameter substitution and nested macros
.org 0x1000

; Save and restore helpers so the main loop leaves the stack balanced
.macro PUSH regs
    MOVEM.L regs,-(A7)
.endmacro

.macro POP regs
    MOVEM.L (A7)+,regs
.endmacro

; Sum a block of bytes using a loop label provided by the caller so the macro
; can be reused multiple times without label collisions.
.macro SUM_BYTES base, length, loop_label
    LEA base,A0
    MOVE.W #length-1,D0
    CLR.L D1
loop_label:
    ADD.B (A0)+,D1
    DBF D0, loop_label
.endmacro

; Simple data-building macro.
.macro BYTEPAIR first, second
    .byte first, second
.endmacro

; Nested macro: uses BYTEPAIR twice to build a four-byte row derived from two
; arguments.
.macro TABLE_ROW x, y
    BYTEPAIR x, y
    BYTEPAIR x+y, x-y
.endmacro

start:
    PUSH D3/D4

    ; First data set
    SUM_BYTES samples, sample_count, sum_loop_a
    MOVE.L D1,total_first

    ; Second data set (shows macro reuse with a different loop label)
    SUM_BYTES more_samples, more_count, sum_loop_b
    MOVE.L D1,total_second

    POP D3/D4
    BRA start

; Data built via nested macros
samples:
    TABLE_ROW 1,2
    TABLE_ROW 3,4
    TABLE_ROW 5,6
samples_end:
sample_count = samples_end - samples

more_samples:
    TABLE_ROW 10,20
    TABLE_ROW 30,25
more_samples_end:
more_count = more_samples_end - more_samples

total_first:
    .long 0
total_second:
    .long 0
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
