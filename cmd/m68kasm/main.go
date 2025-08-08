package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenska/m68kasm/internal/codegen"
	"github.com/jenska/m68kasm/internal/parser"
)

type LabelTable map[string]int

var verbose bool

func vprintln(args ...interface{}) {
	if verbose {
		fmt.Println(args...)
	}
}

func main() {
	flag.BoolVar(&verbose, "v", false, "Enable verbose output")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Printf("Usage: %s [-v] <input.asm>\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	infile := flag.Arg(0)
	f, err := os.Open(infile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open %s: %v\n", infile, err)
		os.Exit(2)
	}
	defer f.Close()

	lines := []string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
		os.Exit(5)
	}

	address := 0
	labelTable := make(LabelTable)
	instructions := []parser.Instruction{}
	handler := codegen.PseudoDirectiveHandler{Verbose: verbose}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			continue
		}
		ins, err := parser.ParseInstruction(address, trimmed, i+1, verbose)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Parse error at line %d: %v\n", i+1, err)
			os.Exit(3)
		}
		if ins.Label != "" {
			if _, exists := labelTable[ins.Label]; exists {
				fmt.Fprintf(os.Stderr, "Duplicate label '%s' at line %d\n", ins.Label, i+1)
				os.Exit(4)
			}
			labelTable[ins.Label] = address
			vprintln("Label:", ins.Label, "->", fmt.Sprintf("0x%X", address))
		}
		if ins.Mnemonic == "" {
			continue
		}
		instructions = append(instructions, ins)
		key := strings.ToUpper(ins.Mnemonic)
		if _, isPseudo := codegen.PseudoDirectiveTable[key]; isPseudo {
			_ = handler.DryRun(ins, &address, labelTable)
		} else {
			address += 2
		}
	}

	address = 0
	output := []byte{}
	for _, ins := range instructions {
		key := strings.ToUpper(ins.Mnemonic)
		if handlerFn, isPseudo := codegen.PseudoDirectiveTable[key]; isPseudo {
			if verbose {
				fmt.Printf("[0x%04X] %-8s %v\n", address, ins.Mnemonic, ins.Operands)
			}
			if err := handlerFn(ins, &address, &output, labelTable, verbose); err != nil {
				fmt.Fprintf(os.Stderr, "Directive error at line %d: %v\n", ins.LineIndex, err)
				os.Exit(7)
			}
		} else {
			if verbose {
				fmt.Printf("[0x%04X] %-8s %v\n", address, ins.Mnemonic, ins.Operands)
			} else {
				fmt.Printf("0x%04X: %+v\n", address, ins)
			}
			address += 2
		}
	}
}