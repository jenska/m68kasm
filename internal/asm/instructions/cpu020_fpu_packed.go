package instructions

import "fmt"

// This file adds packed BCD (.p), the last of the FPU's seven data
// formats — chosen by the maintainer from the open items list after
// milestone 28's urgent R/M-bit fix. It closes out the FPU data-format
// story entirely (byte/word/long/single/double/extended/packed all
// supported).
//
// Cross-checked against GNU binutils' GAS m68k opcode table
// (opcodes/m68k-opc.c): packed BCD as a *source* format needs nothing
// new at all — every <ea>-sourced FPU form already reads its format
// code from FFPFormat (fpFormatCode(sz)<<10, plus the R/M bit fixed in
// milestone 28), and format code 3 (packed) slots in exactly like every
// other size once PackedSize exists (fpLoadSizes, cpu020_fpu.go).
// Confirmed directly against "fmovep"'s own load row,
// `two(0xF000,0x4C00), "Ii;pF7"` — identical shape to fmovex's own
// `0x4800`/fadds' `0x4400`/etc., differing only in the format bits.
//
// The *store* direction is genuinely different, and GAS spells it as
// its own mnemonic, "fmovep", not a `.p`-suffixed row of the generic
// "fmovex" family:
//
//	{"fmovep", two(0xF000, 0x6C00), two(0xF1C0, 0xFC00), "IiF7~pkC", mfloat },
//	{"fmovep", two(0xF000, 0x7C00), two(0xF1C0, 0xFC0F), "IiF7~pDk", mfloat },
//
// Real hardware needs a k-factor (a twos complement integer, -64 to
// +17, specifying how many mantissa digits to generate) to convert an
// extended-precision register value down to a lossy packed-decimal
// string — there's no sensible default, so (unlike every other size)
// FMOVE.P's store direction has no implicit "just use fewer digits"
// fallback. GAS's own assembler syntax (MC68881/MC68882 User's Manual
// §4.5.2, and its own "FMOVE.P FP3,BUFFER{#-5}" example) attaches the
// k-factor as a "{...}" suffix on the destination <ea>, mirroring a
// 68020 bit-field spec's own "{offset:width}" syntax closely enough
// that OpkEAKFactor's parser (parser_ea.go's parseEAKFactor) is modeled
// directly on parseBitFieldSuffix — but deliberately doesn't reuse it,
// since a k-factor is one value, not an offset:width pair, and the two
// need genuinely different grammars ("#k" is mandatory for a static
// k-factor; a bit-field's own equivalent value is written bare, no '#').
//
// The two rows above differ only in whether the k-factor is static (a
// literal 7-bit value at bits 6-0, word2 base 0x6C00) or dynamic (bit
// 12 set, the holding Dn register number at bits 6-4 instead, word2
// base 0x7C00) — a single bit apart, so FKFactor (encode.go) is one
// FieldRef with an internal switch, and one Form (not two) covers both,
// with 0x6C00 as its base literal (FKFactor itself ORs in the extra
// 0x1000 for the dynamic case).
//
// Neither row's mask leaves any room for a format field the way every
// other FPU instruction's word2 does (bits 13-10 here are the k-factor/
// opcode-identity bits instead — the destination format is always
// packed, implied by the mnemonic itself, so there is nothing to
// select) — confirmed by checking that 0x6C00's own bits 13-10 (0110)
// don't decode to format 3 under the usual fpFormatCode convention,
// unlike every load-direction row's own bits. This form's Steps
// therefore do not use FFPFormat at all.
func init() {
	defFMOVE.Forms = append(defFMOVE.Forms, FormDef{
		// FPn,<ea>{k}
		DefaultSize: PackedSize,
		Sizes:       []Size{PackedSize},
		OperKinds:   []OperandKind{OpkFPn, OpkEAKFactor},
		Validate:    validateFMOVEPStore,
		Requires:    requireFPU,
		Steps: []EmitStep{
			{WordBits: fpuWord1Base, Fields: []FieldRef{FDstEA}},
			{WordBits: 0x6C00, Fields: []FieldRef{FFPSrcReg7, FKFactor}},
			{Trailer: []TrailerItem{TDstEAExt}},
		},
	})
}

func validateFMOVEPStore(a *Args) error {
	if !a.Dst.HasKFactor {
		return fmt.Errorf("FMOVE.P requires a k-factor destination suffix, \"<ea>{#k}\" or \"<ea>{Dn}\"")
	}
	if !memoryAlterableEA[a.Dst.Kind] {
		return fmt.Errorf("FMOVE.P destination must be memory-alterable")
	}
	return nil
}
