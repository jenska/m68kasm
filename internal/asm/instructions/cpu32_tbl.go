package instructions

import "fmt"

// This file adds CPU32's table-lookup-and-interpolate instructions:
// TBLS/TBLSN (signed, with/without rounding) and TBLU/TBLUN (unsigned,
// with/without rounding), each in two forms — "<ea>,Dn" (read two
// adjacent table entries from memory) and "Dn1:Dn2,Dn3" (interpolate
// directly between two register values).
//
// IMPORTANT CAVEAT, unlike every other milestone in this codebase: this
// encoding could NOT be cross-checked against a working GAS reference,
// because GAS doesn't actually have one. opcodes/m68k-opc.c has the
// four TBL(...) table-entry macros (real bit-layout data, presumably
// accurate to actual CPU32 silicon), but their argument-string type
// code ('`', GAS's own notation for TBL's <ea> operand) has no
// corresponding case anywhere in gas/config/tc-m68k.c's parser, and
// m68k-dis.c's disassembler has no TBL-printing logic either — these
// appear to be dormant opcode-table entries binutils never finished
// wiring into either direction of real tool support. Every other
// instruction in this codebase was verified against GAS's actual
// parsing/installation logic; this one instead rests on:
//   - the raw opcode/mask literals in the TBL1 macro (real hardware
//     bit positions: signed at bit 11, "not rounded" at bit 10, size at
//     bits 7-6, a "reads from memory" flag at bit 8, Dn-destination at
//     bits 14-12 in both forms, and — inferred from the generic
//     'D'/'s'/'1'/'3' install codes this codebase has already
//     cross-checked working for every other instruction that uses
//     them — Dn1 in word1's low 3 bits and Dn2 in word2's low 3 bits
//     for the register-to-register form);
//   - a judgment call for the memory form's <ea> restriction —
//     controlAlterableEA (genuine memory reference only: no Dn/An,
//     predecrement/postincrement, or immediate), matching CHK2/CMP2's
//     own restriction for a similar "read from a well-defined memory
//     location" operand. This isn't just the closest available map: an
//     earlier draft used readableDataEA (which includes Dn) and it
//     created a real, observable bug — "TBLS.B D0,D2" (meant to hit the
//     register-pair form's mandatory-colon rejection) instead silently
//     matched the *memory* form first, parsing D0 as a Dn-based <ea>,
//     since Forms are tried in order and the memory form is listed
//     first. Excluding Dn here fixes that ambiguity as a side effect of
//     what's also the more defensible semantic choice — no working
//     reference confirms the exact addressing-mode subset real CPU32
//     silicon accepts, so this is flagged explicitly rather than
//     guessed silently;
//   - this codebase's own established .B/.W/.L-suffix convention
//     (TBLS/TBLSN/TBLU/TBLUN, not GAS's six-mnemonics-per-family
//     "tblsb"/"tblsw"/"tblsl"/... spelling), the same choice already
//     made for BRA.L/BSR.L (milestone 3), FBcc.L (milestone 13), and
//     the 64-bit DIVSL/DIVUL colon syntax (milestone 10).
//
// Gated requireCPU32Only (not require68020orCPU32): GAS's own arch tag
// is plain `cpu32`, with no `m68020up` union — TBLS/TBLU are CPU32-
// exclusive, matching BGND (cpu020_misc.go) rather than EXTB/CHK2/CMP2/
// TRAPcc.

var tblSizeAllowed = []Size{ByteSize, WordSize, LongSize}

func init() {
	registerInstrDef(newTblDef("TBLS", 0x0800))
	registerInstrDef(newTblDef("TBLSN", 0x0C00))
	registerInstrDef(newTblDef("TBLU", 0x0000))
	registerInstrDef(newTblDef("TBLUN", 0x0400))
}

// newTblDef builds one TBL mnemonic. signRoundBits is (signed<<11) |
// (notRounded<<10) — the two bits that vary by mnemonic identity, fixed
// regardless of which form or size is chosen. Bit 8 (0x100, set only on
// the memory form) and bits 7-6 (the size, via the existing FSizeBits
// field every instruction's Args.Size already populates) are added on
// top of this base per Form/Step.
func newTblDef(name string, signRoundBits uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				// Dn1:Dn2,Dn3 — interpolate directly between two
				// register values (the colon is mandatory: unlike
				// DIVSL/DIVUL's "Dr:Dq"/"Dq" shorthand, there's no
				// sensible single-register form here).
				//
				// Tried before the memory form below: the parser
				// accepts the first Form whose shape matches the token
				// stream (see parseInstruction, parser_stmt.go), and a
				// bare "D0,D2" parses against BOTH this form's OpkRegPair
				// (parseRegPair happily accepts a colonless single Dn)
				// and the memory form's OpkEA (Dn is a valid EA on its
				// own). Trying this form first means that input hits
				// this form's mandatory-colon Validate — a clear,
				// specific error — rather than the memory form's more
				// generic "not a memory addressing mode" rejection.
				DefaultSize: WordSize,
				Sizes:       tblSizeAllowed,
				OperKinds:   []OperandKind{OpkRegPair, OpkDn},
				Validate:    validateTblRegPair(name),
				Requires:    requireCPU32Only,
				Steps: []EmitStep{
					{WordBits: 0xF800, Fields: []FieldRef{FSrcDnReg}},
					{WordBits: signRoundBits, Fields: []FieldRef{FSizeBits, FSrcReg2Low, FMove16Reg2_12}},
				},
			},
			{
				// <ea>,Dn — read two adjacent table entries from
				// memory, interpolate using Dn's fractional low byte,
				// write the result back into Dn.
				DefaultSize: WordSize,
				Sizes:       tblSizeAllowed,
				OperKinds:   []OperandKind{OpkEA, OpkDn},
				Validate:    validateTblMemory(name),
				Requires:    requireCPU32Only,
				Steps: []EmitStep{
					{WordBits: 0xF800, Fields: []FieldRef{FSrcEA}},
					{WordBits: signRoundBits | 0x100, Fields: []FieldRef{FSizeBits, FMove16Reg2_12}},
					{Trailer: []TrailerItem{TSrcEAExt}},
				},
			},
		},
	}
}

func validateTblMemory(name string) func(*Args) error {
	return func(a *Args) error {
		if a.Src.Kind == EAkNone {
			return fmt.Errorf("%s requires a source", name)
		}
		if !controlAlterableEA[a.Src.Kind] {
			return fmt.Errorf("%s source must be a memory addressing mode", name)
		}
		return nil
	}
}

func validateTblRegPair(name string) func(*Args) error {
	return func(a *Args) error {
		if !a.Src.RegPairWide {
			return fmt.Errorf("%s requires \"Dn1:Dn2\" (the colon is mandatory)", name)
		}
		return nil
	}
}
