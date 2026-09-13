package instructions

import "fmt"

type eaEntry struct {
	mode        int
	reg         int
	regFromExpr bool
	ext         func(EAExpr) ([]uint16, error)
	valid       bool
}

var eaTable = []eaEntry{
	/* EAkNone */ {},
	/* EAkImm */ {mode: 7, reg: 4, valid: true},
	/* EAkDn */ {mode: 0, regFromExpr: true, valid: true},
	/* EAkAn */ {mode: 1, regFromExpr: true, valid: true},
	/* EAkAddrPredec */ {mode: 4, regFromExpr: true, valid: true},
	/* EAkAddrPostinc */ {mode: 3, regFromExpr: true, valid: true},
	/* EAkAddrInd */ {mode: 2, regFromExpr: true, valid: true},
	/* EAkAddrDisp16 */ {mode: 5, regFromExpr: true, ext: eaExtDisp16, valid: true},
	/* EAkPCDisp16 */ {mode: 7, reg: 2, ext: eaExtDisp16, valid: true},
	/* EAkIdxAnBrief */ {mode: 6, regFromExpr: true, ext: eaExtIndexBrief, valid: true},
	/* EAkIdxPCBrief */ {mode: 7, reg: 3, ext: eaExtIndexBrief, valid: true},
	/* EAkAbsW */ {mode: 7, reg: 0, ext: eaExtAbsW, valid: true},
	/* EAkAbsL */ {mode: 7, reg: 1, ext: eaExtAbsL, valid: true},
	/* EAkSR */ {mode: 0, reg: 0, valid: true},
	/* EAkCCR */ {mode: 0, reg: 0, valid: true},
	/* EAkUSP */ {mode: 0, reg: 0, valid: true},
	/* EAkSFC */ {mode: 0, reg: 0, valid: true},
	/* EAkDFC */ {mode: 0, reg: 0, valid: true},
	/* EAkVBR */ {mode: 0, reg: 0, valid: true},
	/* EAkMemPreAn */ {mode: 6, regFromExpr: true, ext: eaExtFull, valid: true},
	/* EAkMemPrePC */ {mode: 7, reg: 3, ext: eaExtFull, valid: true},
	/* EAkMemPostAn */ {mode: 6, regFromExpr: true, ext: eaExtFull, valid: true},
	/* EAkMemPostPC */ {mode: 7, reg: 3, ext: eaExtFull, valid: true},
	/* EAkFPn */ {mode: 0, reg: 0, valid: true},
	/* EAkRegPair */ {mode: 0, reg: 0, valid: true},
	/* EAkAnIndPair */ {mode: 0, reg: 0, valid: true},
	/* EAkTC */ {mode: 0, reg: 0, valid: true},
	/* EAkCacheSel */ {mode: 0, reg: 0, valid: true},
	/* EAkCRP */ {mode: 0, reg: 0, valid: true},
	/* EAkSRP */ {mode: 0, reg: 0, valid: true},
	/* EAkTT0 */ {mode: 0, reg: 0, valid: true},
	/* EAkTT1 */ {mode: 0, reg: 0, valid: true},
	/* EAkMMUSR */ {mode: 0, reg: 0, valid: true},
	/* EAkDRP */ {mode: 0, reg: 0, valid: true},
	/* EAkCAL */ {mode: 0, reg: 0, valid: true},
	/* EAkVAL */ {mode: 0, reg: 0, valid: true},
	/* EAkSCC */ {mode: 0, reg: 0, valid: true},
	/* EAkAC */ {mode: 0, reg: 0, valid: true},
	/* EAkPCSR */ {mode: 0, reg: 0, valid: true},
	/* EAkBAD */ {mode: 0, reg: 0, valid: true},
	/* EAkBAC */ {mode: 0, reg: 0, valid: true},
}

// EncodeEA converts an addressing expression into the mode/reg pair and any extension words.
func EncodeEA(e EAExpr, pc uint32) (EAEncoded, error) {
	if int(e.Kind) < 0 || int(e.Kind) >= len(eaTable) {
		return EAEncoded{}, fmt.Errorf("unsupported EA kind: %d", e.Kind)
	}

	entry := eaTable[e.Kind]
	if !entry.valid {
		return EAEncoded{}, fmt.Errorf("unsupported EA kind: %d", e.Kind)
	}

	out := EAEncoded{Mode: entry.mode, Reg: entry.reg}
	if entry.regFromExpr {
		out.Reg = e.Reg
	}

	if entry.ext != nil {
		ext, err := entry.ext(e)
		if err != nil {
			return EAEncoded{}, err
		}
		out.Ext = append(out.Ext, ext...)
	}

	return out, nil
}

func eaExtDisp16(e EAExpr) ([]uint16, error) {
	return []uint16{uint16(e.Disp16)}, nil
}

func eaExtIndexBrief(e EAExpr) ([]uint16, error) {
	return []uint16{encodeBriefIndex(e.Index)}, nil
}

func eaExtAbsW(e EAExpr) ([]uint16, error) {
	return []uint16{e.Abs16}, nil
}

func eaExtAbsL(e EAExpr) ([]uint16, error) {
	return []uint16{uint16(e.Abs32 >> 16), uint16(e.Abs32)}, nil
}

// indexSpecifierBits returns bits 15-9 of an index-bearing extension
// word: A/D select (bit 15), register number (bits 14-12), W/L (bit 11),
// and scale (bits 10-9). This layout is shared by the classic brief
// extension word (below) and the 68020+ full extension word (eaExtFull);
// only the low byte's meaning differs between the two formats.
func indexSpecifierBits(ix EAIndex) uint16 {
	hi := uint16(0)
	if ix.IsA {
		hi |= 1 << 7
	}
	hi |= (uint16(ix.Reg&7) << 4)
	if ix.Long {
		hi |= 1 << 3
	}
	switch ix.Scale {
	case 1:
	case 2:
		hi |= 1 << 1
	case 4:
		hi |= 2 << 1
	case 8:
		hi |= 3 << 1
	}
	return hi << 8
}

func encodeBriefIndex(ix EAIndex) uint16 {
	return indexSpecifierBits(ix) | uint16(uint8(ix.Disp8))
}

// bdSizeBits returns the BD SIZE field (bits 5-4) of a 68020+ full
// extension word: 01 = no base-displacement word (value is 0), 10 = one
// word, 11 = two words (long). 00 is reserved by Motorola and never
// produced here.
func bdSizeBits(size DispSize) uint16 {
	switch size {
	case DispNull:
		return 0x10
	case DispWord:
		return 0x20
	case DispLong:
		return 0x30
	default:
		return 0
	}
}

// odFieldBits returns the low 3 bits (I/IS) of a memory-indirect full
// extension word: the outer-displacement size (bits 1-0: 01/10/11 for
// null/word/long) combined with the pre/post-indexed selector (bit 2).
// When the index is suppressed, pre- and post-indexing compute the same
// address (there is no index register to place before or after the
// indirection), so hardware — and this function, to match — canonicalize
// to the pre-indexed bit pattern regardless of which the caller asked for.
func odFieldBits(postIndexed, indexPresent bool, size DispSize) uint16 {
	var bits uint16
	switch size {
	case DispNull:
		bits = 0x1
	case DispWord:
		bits = 0x2
	case DispLong:
		bits = 0x3
	}
	if postIndexed && indexPresent {
		bits |= 0x4
	}
	return bits
}

// dispWords encodes a base/outer displacement per its DispSize: no words
// for DispNull, one sign-extended word for DispWord (range-checked, since
// a symbol-derived value is never checked at parse time — see
// docs/design/cpu-family-support.md), or two words (high, then low) for
// DispLong.
func dispWords(size DispSize, val int32) ([]uint16, error) {
	switch size {
	case DispWord:
		if val < -32768 || val > 32767 {
			return nil, fmt.Errorf("displacement out of range for .W: %d", val)
		}
		return []uint16{uint16(int16(val))}, nil
	case DispLong:
		u := uint32(val)
		return []uint16{uint16(u >> 16), uint16(u)}, nil
	default:
		return nil, nil
	}
}

// eaExtFull encodes the 68020+ full-format extension word(s) for
// memory-indirect addressing (EAkMemPreAn/PrePC/PostAn/PostPC): the
// index/flags word, then the base-displacement words, then the
// outer-displacement words. Cross-checked against GNU binutils'
// gas/config/tc-m68k.c (the POST/PRE/BASE case) — see
// docs/design/cpu-family-support.md.
func eaExtFull(e EAExpr) ([]uint16, error) {
	word := uint16(0x0100) // bit 8: full-format flag
	if e.IndexPresent {
		word |= indexSpecifierBits(e.Index)
	} else {
		word |= 0x0040 // IS: index suppressed
	}
	word |= bdSizeBits(e.BaseDispSize)

	postIndexed := e.Kind == EAkMemPostAn || e.Kind == EAkMemPostPC
	word |= odFieldBits(postIndexed, e.IndexPresent, e.OuterDispSize)

	bdWords, err := dispWords(e.BaseDispSize, e.BaseDisp)
	if err != nil {
		return nil, err
	}
	odWords, err := dispWords(e.OuterDispSize, e.OuterDisp)
	if err != nil {
		return nil, err
	}

	words := make([]uint16, 0, 1+len(bdWords)+len(odWords))
	words = append(words, word)
	words = append(words, bdWords...)
	words = append(words, odWords...)
	return words, nil
}
