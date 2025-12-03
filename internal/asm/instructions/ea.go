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
}

// EncodeEA converts an addressing expression into the mode/reg pair and any extension words.
func EncodeEA(e EAExpr) (EAEncoded, error) {
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

func encodeBriefIndex(ix EAIndex) uint16 {
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
	return (hi << 8) | uint16(uint8(ix.Disp8))
}
