package instructions

import (
	"maps"
	"sync"
)

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

// ForTarget returns a table containing only the forms legal for target,
// dropping any mnemonic left with no forms at all. The receiver is left
// unmodified. Calling ForTarget with the zero-value Target (a bare
// 68000) is a no-op today, since no built-in form sets Requires.
func (t *Table) ForTarget(target Target) *Table {
	out := make(map[string]*InstrDef, len(t.defs))
	for mnemonic, def := range t.defs {
		forms := make([]FormDef, 0, len(def.Forms))
		for _, form := range def.Forms {
			if target.Supports(form.Requires) {
				forms = append(forms, form)
			}
		}
		if len(forms) == 0 {
			continue
		}
		filtered := *def
		filtered.Forms = forms
		out[mnemonic] = &filtered
	}
	return &Table{defs: out}
}

func cloneDefs(src map[string]*InstrDef) map[string]*InstrDef {
	dst := make(map[string]*InstrDef, len(src))
	maps.Copy(dst, src)
	return dst
}
