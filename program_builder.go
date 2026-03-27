package m68kasm

import (
	"fmt"
	"sort"
	"strings"
)

// ProgramBuilder is a lightweight helper for test-oriented source generation.
type ProgramBuilder struct {
	lines []string
}

// NewProgramBuilder creates a new assembly source builder.
func NewProgramBuilder() *ProgramBuilder {
	return &ProgramBuilder{}
}

// Line appends a raw source line.
func (b *ProgramBuilder) Line(line string) *ProgramBuilder {
	b.lines = append(b.lines, line)
	return b
}

// Instruction appends an instruction line.
func (b *ProgramBuilder) Instruction(line string) *ProgramBuilder {
	return b.Line(line)
}

// Label appends a global label definition.
func (b *ProgramBuilder) Label(name string) *ProgramBuilder {
	return b.Line(name + ":")
}

// Origin appends an .org directive.
func (b *ProgramBuilder) Origin(addr uint32) *ProgramBuilder {
	return b.Line(".org " + formatUint32Hex(addr, 0))
}

// Text switches subsequent output to the text section.
func (b *ProgramBuilder) Text() *ProgramBuilder {
	return b.Line(".text")
}

// Data switches subsequent output to the data section.
func (b *ProgramBuilder) Data() *ProgramBuilder {
	return b.Line(".data")
}

// BSS switches subsequent output to the bss section.
func (b *ProgramBuilder) BSS() *ProgramBuilder {
	return b.Line(".bss")
}

// Section appends a .section directive.
func (b *ProgramBuilder) Section(name string) *ProgramBuilder {
	if strings.HasPrefix(name, ".") {
		return b.Line(".section " + name)
	}
	return b.Line(`.section "` + name + `"`)
}

// Byte appends a .byte directive with numeric values.
func (b *ProgramBuilder) Byte(values ...uint8) *ProgramBuilder {
	exprs := make([]string, len(values))
	for i, value := range values {
		exprs[i] = formatUint64Hex(uint64(value), 2)
	}
	return b.ByteExpr(exprs...)
}

// ByteExpr appends a .byte directive with arbitrary expressions.
func (b *ProgramBuilder) ByteExpr(exprs ...string) *ProgramBuilder {
	return b.directive(".byte", exprs...)
}

// Word appends a .word directive with numeric values.
func (b *ProgramBuilder) Word(values ...uint16) *ProgramBuilder {
	exprs := make([]string, len(values))
	for i, value := range values {
		exprs[i] = formatUint64Hex(uint64(value), 4)
	}
	return b.WordExpr(exprs...)
}

// WordExpr appends a .word directive with arbitrary expressions.
func (b *ProgramBuilder) WordExpr(exprs ...string) *ProgramBuilder {
	return b.directive(".word", exprs...)
}

// Long appends a .long directive with numeric values.
func (b *ProgramBuilder) Long(values ...uint32) *ProgramBuilder {
	exprs := make([]string, len(values))
	for i, value := range values {
		exprs[i] = formatUint32Hex(value, 8)
	}
	return b.LongExpr(exprs...)
}

// LongExpr appends a .long directive with arbitrary expressions.
func (b *ProgramBuilder) LongExpr(exprs ...string) *ProgramBuilder {
	return b.directive(".long", exprs...)
}

// Align appends an .align directive with optional fill byte.
func (b *ProgramBuilder) Align(boundary uint32, fill ...byte) *ProgramBuilder {
	line := ".align " + fmt.Sprintf("%d", boundary)
	if len(fill) > 0 {
		line += "," + formatUint64Hex(uint64(fill[0]), 2)
	}
	return b.Line(line)
}

// Even appends an .even directive.
func (b *ProgramBuilder) Even() *ProgramBuilder {
	return b.Line(".even")
}

// VectorTable emits a zero-filled vector table up to the highest supplied vector index.
func (b *ProgramBuilder) VectorTable(entries map[uint32]string) *ProgramBuilder {
	if len(entries) == 0 {
		return b
	}

	keys := make([]int, 0, len(entries))
	for idx := range entries {
		keys = append(keys, int(idx))
	}
	sort.Ints(keys)

	values := make([]string, keys[len(keys)-1]+1)
	for i := range values {
		values[i] = "0"
	}
	for _, idx := range keys {
		values[idx] = strings.TrimSpace(entries[uint32(idx)])
		if values[idx] == "" {
			values[idx] = "0"
		}
	}
	return b.LongExpr(values...)
}

// String returns the assembled source text.
func (b *ProgramBuilder) String() string {
	if len(b.lines) == 0 {
		return ""
	}
	return strings.Join(b.lines, "\n") + "\n"
}

// Assemble assembles the generated source and returns detailed metadata.
func (b *ProgramBuilder) Assemble() (*AssemblyResult, error) {
	return AssembleStringDetailed(b.String())
}

// AssembleWithOptions assembles the generated source with parser options.
func (b *ProgramBuilder) AssembleWithOptions(opts ParseOptions) (*AssemblyResult, error) {
	return AssembleStringDetailedWithOptions(b.String(), opts)
}

func (b *ProgramBuilder) directive(name string, exprs ...string) *ProgramBuilder {
	if len(exprs) == 0 {
		return b
	}
	values := make([]string, 0, len(exprs))
	for _, expr := range exprs {
		expr = strings.TrimSpace(expr)
		if expr == "" {
			continue
		}
		values = append(values, expr)
	}
	if len(values) == 0 {
		return b
	}
	return b.Line(name + " " + strings.Join(values, ","))
}
