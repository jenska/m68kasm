package instructions

import "fmt"

// This file adds DIVSL and DIVUL, the 68020 64-bit-dividend divide
// forms. Encodings were cross-checked against GNU binutils' GAS m68k
// opcode table (opcodes/m68k-opc.c) for the opcode words, and against
// its install_operand/print_insn_arg logic (gas/config/tc-m68k.c,
// opcodes/m68k-dis.c) to resolve which physical bit position (14-12 vs
// 2-0) is the quotient register (Dq) versus the remainder register
// (Dr) — a detail the opcode table's mask alone doesn't settle, and
// easy to get backwards by guessing from the manual's bit diagram
// alone. See docs/design/cpu-family-support.md.
//
// Like milestone 7's Bcc.L/CHK2/EXTB/TRAPcc, DIVSL/DIVUL are tagged
// m68020up|cpu32 in GAS — CPU32 has them too — so they're gated on
// Target{CPU: CPU32}, not Target{CPU: CPU68020}.
var requireDiv = Target{CPU: CPU32}

func init() {
	registerInstrDef(newDivDef("DIVSL", 0x0800))
	registerInstrDef(newDivDef("DIVUL", 0x0000))
}

// newDivDef builds "DIVSL.L <ea>,Dr:Dq" / "DIVUL.L <ea>,Dq". signBit is
// 0x0800 for DIVSL, 0 for DIVUL — the only difference between the two
// besides the mnemonic itself.
func newDivDef(name string, signBit uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: LongSize,
				Sizes:       []Size{LongSize},
				OperKinds:   []OperandKind{OpkEA, OpkRegPair},
				Validate:    func(a *Args) error { return validateDiv(name, a) },
				Requires:    requireDiv,
				Steps: []EmitStep{
					{WordBits: 0x4C40, Fields: []FieldRef{FSrcEA}},
					{WordBits: signBit, Fields: []FieldRef{FDivWide, FDivRegQ, FDivRegR}},
					{Trailer: []TrailerItem{TSrcEAExt}},
				},
			},
		},
	}
}

func validateDiv(name string, a *Args) error {
	if a.Src.Kind == EAkAn {
		return fmt.Errorf("%s does not support address register operands", name)
	}
	if !readableDataEA[a.Src.Kind] {
		if a.Src.Kind == EAkNone {
			return fmt.Errorf("%s requires a source operand", name)
		}
		return fmt.Errorf("%s source must be a readable addressing mode", name)
	}
	return nil
}
