package instructions

import "sync"

// Table is a read-only lookup structure for instruction definitions.
//
// It can be shared safely between goroutines once constructed.
type Table struct {
	defs map[string]*InstrDef
}

var (
	defaultTable     *Table
	defaultTableOnce sync.Once
)

// DefaultTable returns a shared, read-only instruction table containing all
// built-in opcodes.
func DefaultTable() *Table {
	defaultTableOnce.Do(func() {
		defaultTable = &Table{defs: cloneDefs(Instructions)}
	})
	return defaultTable
}

// Clone returns a shallow copy of the table that can be augmented without
// affecting the original.
func (t *Table) Clone() *Table {
	return &Table{defs: cloneDefs(t.defs)}
}

// Lookup returns the instruction definition for a mnemonic or nil if none
// exists.
func (t *Table) Lookup(mnemonic string) *InstrDef {
	return t.defs[mnemonic]
}

func cloneDefs(src map[string]*InstrDef) map[string]*InstrDef {
	dst := make(map[string]*InstrDef, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
