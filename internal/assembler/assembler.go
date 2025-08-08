package assembler

import (
	"fmt"
	"os"

	"github.com/jenska/m68kasm/internal/lexer"
	"github.com/jenska/m68kasm/internal/parser"
	"github.com/jenska/m68kasm/internal/codegen"
)

func AssembleFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	tokens, err := lexer.Tokenize(string(data))
	if err != nil {
		return fmt.Errorf("lexing failed: %w", err)
	}

	instructions, symbols, err := parser.Parse(tokens)
	if err != nil {
		return fmt.Errorf("parsing failed: %w", err)
	}

	fmt.Println("Assembly listing with resolved label references:")
	err = codegen.Emit(instructions, symbols)
	if err != nil {
		return fmt.Errorf("codegen failed: %w", err)
	}

	return nil
}