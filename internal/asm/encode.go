package asm

import (
	"fmt"
	"math"

	"github.com/jenska/m68kasm/internal/asm/instructions"
)

type Instr struct {
	Def     *instructions.InstrDef
	Form    *instructions.FormDef
	Args    instructions.Args
	PC      uint32
	Line    int
	Col     int
	Section SectionKind
}

func sizeToBits(sz instructions.Size) uint16 {
	switch sz {
	case instructions.ByteSize:
		return 0x0000
	case instructions.WordSize:
		return 0x0040
	case instructions.LongSize:
		return 0x0080
	default:
		return 0
	}
}

func appendWord(out []byte, v uint16) []byte {
	return append(out, byte(v>>8), byte(v))
}

// appendFloatImm encodes f as an FPU immediate in the given size
// (SingleSize/DoubleSize/ExtendedSize only — see TSrcImm) and appends
// the big-endian result. Single and double are plain IEEE-754 binary32/
// binary64; extended is the 68881/68882's own 96-bit format (see
// encodeExtendedReal's own doc comment).
func appendFloatImm(out []byte, sz instructions.Size, f float64) []byte {
	switch sz {
	case instructions.SingleSize:
		bits := math.Float32bits(float32(f))
		out = appendWord(out, uint16(bits>>16))
		out = appendWord(out, uint16(bits))
		return out
	case instructions.DoubleSize:
		bits := math.Float64bits(f)
		for shift := 48; shift >= 0; shift -= 16 {
			out = appendWord(out, uint16(bits>>shift))
		}
		return out
	default: // ExtendedSize
		for _, w := range encodeExtendedReal(f) {
			out = appendWord(out, w)
		}
		return out
	}
}

// encodeExtendedReal converts f to the 68881/68882's 96-bit ("double
// extended") real format, as six big-endian 16-bit words: word0 is a
// 1-bit sign plus a 15-bit biased (bias 16383) exponent, word1 is a
// reserved zero word (for long-word alignment — see the MC68881/MC68882
// User's Manual, §1.3.2's "Extended Precision Real" figure), and words
// 2-5 are the 64-bit mantissa with an EXPLICIT leading integer bit
// (unlike single/double precision's implicit leading one) — bit-for-bit
// the same layout as the well-known x87 80-bit extended format, just
// with that extra reserved word inserted after the exponent.
//
// math.Frexp gives f = frac*2^exp with frac in [0.5,1) (or frac==0),
// which maps onto this format directly: frac*2^64 lands in [2^63,2^64)
// — exactly a 64-bit mantissa with its top (integer) bit already set —
// and the biased extended exponent is (exp-1)+16383, since a mantissa
// normalized to [1,2) (as this format's explicit-integer-bit convention
// expects) is frac*2 = a shift of one less than frac's own exponent.
// Only handles finite values — the lexer's float literals can only ever
// produce those (an out-of-range literal is a strconv.ParseFloat error,
// caught at parse time), so infinities/NaNs are deliberately not
// special-cased here.
func encodeExtendedReal(f float64) [6]uint16 {
	var words [6]uint16
	if f == 0 {
		if math.Signbit(f) {
			words[0] = 0x8000
		}
		return words
	}
	sign := uint16(0)
	if f < 0 {
		sign = 0x8000
		f = -f
	}
	frac, exp := math.Frexp(f)
	mantissa := uint64(math.Round(frac * 18446744073709551616.0)) // frac * 2^64
	exponent := uint16((exp-1)+16383) & 0x7FFF
	words[0] = sign | exponent
	words[1] = 0
	words[2] = uint16(mantissa >> 48)
	words[3] = uint16(mantissa >> 32)
	words[4] = uint16(mantissa >> 16)
	words[5] = uint16(mantissa)
	return words
}

type prepared struct {
	PC       uint32
	SizeBits uint16
	Size     instructions.Size

	Imm        int64
	SrcReg     int
	DstReg     int
	SrcRegMask uint16
	DstRegMask uint16

	SrcFCMode int
	DstImm    int64
	Aux2Reg   int

	SrcImmFloat   float64
	SrcImmIsFloat bool

	DstKFactorIsReg bool
	DstKFactorVal   int32

	FPRegMaskSrc uint16
	FPRegMaskDst uint16

	FPCtrlMaskSrc uint16
	FPCtrlMaskDst uint16

	SrcEA  instructions.EAEncoded
	DstEA  instructions.EAEncoded
	AuxEA  instructions.EAEncoded
	AuxImm int64

	SrcBFOffset, SrcBFWidth uint16
	DstBFOffset, DstBFWidth uint16

	SrcReg2         int
	DstReg2         int
	DstRegPairWide  bool
	AuxReg, AuxReg2 int

	CtrlRegSel uint16

	TargetPC  uint32
	BrUseWord bool
	BrDisp8   int8
	BrDisp16  int16
	BrUseLong bool
	BrDisp32  int32
}

func applyField(wordVal uint16, f instructions.FieldRef, p *prepared) uint16 {
	switch f {
	case instructions.FSizeBits:
		return wordVal | p.SizeBits
	case instructions.FSrcEA:
		return wordVal | (uint16(p.SrcEA.Mode&7) << 3) | uint16(p.SrcEA.Reg&7)
	case instructions.FDstEA:
		return wordVal | (uint16(p.DstEA.Mode&7) << 3) | uint16(p.DstEA.Reg&7)
	case instructions.FDnReg:
		return wordVal | (uint16(p.DstReg&7) << 9)
	case instructions.FAnReg:
		return wordVal | (uint16(p.DstReg&7) << 9)
	case instructions.FImmLow8:
		return wordVal | uint16(uint8(p.Imm))
	case instructions.FBranchLow8:
		if p.BrUseLong {
			return wordVal | 0x00FF
		}
		if !p.BrUseWord {
			return wordVal | uint16(uint8(p.BrDisp8))
		}
		return wordVal
	case instructions.FMoveDestEA:
		return wordVal | (uint16(p.DstEA.Mode&7) << 6) | (uint16(p.DstEA.Reg&7) << 9)
	case instructions.FMoveSize:
		switch p.Size {
		case instructions.ByteSize:
			return wordVal | 0x1000
		case instructions.WordSize:
			return wordVal | 0x3000
		case instructions.LongSize:
			return wordVal | 0x2000
		default:
			return wordVal
		}
	case instructions.FQuickData:
		quick := uint16(p.Imm)
		if quick == 8 {
			quick = 0
		}
		return wordVal | (quick&7)<<9
	case instructions.FSrcDnReg:
		return wordVal | uint16(p.SrcReg&7)
	case instructions.FSrcAnReg:
		return wordVal | uint16(p.SrcReg&7)
	case instructions.FDstRegLow:
		return wordVal | uint16(p.DstReg&7)
	case instructions.FMovemSize:
		// MOVEM's size bit (bit 6) reflects the actual size specification.
		// Bit 6 = 0 for word-sized transfers, 1 for long-word transfers.
		// The previous implementation incorrectly overrode explicit size
		// specifications based on addressing mode, causing identical opcodes
		// for MOVEM.L and MOVEM.W with certain modes. This fix respects
		// the user's explicit size choice.
		if p.Size == instructions.LongSize {
			return wordVal | 0x0040
		}
		return wordVal
	case instructions.FAddaSize:
		if p.Size == instructions.LongSize {
			return wordVal | 0x0100
		}
		return wordVal
	case instructions.FSrcDnRegHi:
		return wordVal | (uint16(p.SrcReg&7) << 9)
	case instructions.FCtrlRegSel:
		return wordVal | p.CtrlRegSel
	case instructions.FExtRegSrc:
		return wordVal | (uint16(p.SrcEA.Mode&1) << 15) | (uint16(p.SrcEA.Reg&7) << 12)
	case instructions.FExtRegDst:
		return wordVal | (uint16(p.DstEA.Mode&1) << 15) | (uint16(p.DstEA.Reg&7) << 12)
	case instructions.FFPFormat:
		return wordVal | fpRMBit | fpFormatCode(p.Size)<<10
	case instructions.FFPDstReg7:
		return wordVal | (uint16(p.DstReg&7) << 7)
	case instructions.FFPSrcReg7:
		return wordVal | (uint16(p.SrcReg&7) << 7)
	case instructions.FFPSrcReg10:
		return wordVal | (uint16(p.SrcReg&7) << 10)
	case instructions.FAuxEA:
		return wordVal | (uint16(p.AuxEA.Mode&7) << 3) | uint16(p.AuxEA.Reg&7)
	case instructions.FDstReg6:
		return wordVal | (uint16(p.DstReg&7) << 6)
	case instructions.FSrcRegNibble0:
		return wordVal | (uint16(p.SrcEA.Mode&1) << 3) | uint16(p.SrcEA.Reg&7)
	case instructions.FBFOffsetSrc:
		return wordVal | (p.SrcBFOffset << 6)
	case instructions.FBFWidthSrc:
		return wordVal | p.SrcBFWidth
	case instructions.FBFOffsetDst:
		return wordVal | (p.DstBFOffset << 6)
	case instructions.FBFWidthDst:
		return wordVal | p.DstBFWidth
	case instructions.FDivRegQ:
		return wordVal | (uint16(p.DstReg2&7) << 12)
	case instructions.FDivRegR:
		return wordVal | uint16(p.DstReg&7)
	case instructions.FDivWide:
		if p.DstRegPairWide {
			return wordVal | 0x0400
		}
		return wordVal
	case instructions.FCas2Word2:
		return wordVal | (uint16(p.AuxReg&7) << 12) | (uint16(p.DstReg&7) << 6) | uint16(p.SrcReg&7)
	case instructions.FCas2Word3:
		return wordVal | (uint16(p.AuxReg2&7) << 12) | (uint16(p.DstReg2&7) << 6) | uint16(p.SrcReg2&7)
	case instructions.FFPRomConst:
		return wordVal | (uint16(p.Imm) & 0x7F)
	case instructions.FFPRegMaskDst:
		return wordVal | (p.FPRegMaskDst & 0xFF)
	case instructions.FFPDynDstReg4:
		return wordVal | (uint16(p.DstReg&7) << 4)
	case instructions.FFPMovemStoreWord2:
		if p.DstEA.Mode == 4 { // -(An): predecrement, list order reversed
			return wordVal | 0xE000 | (reverse16(p.FPRegMaskSrc&0xFF) >> 8)
		}
		return wordVal | 0xF000 | (p.FPRegMaskSrc & 0xFF)
	case instructions.FFPMovemDynStoreWord2:
		if p.DstEA.Mode == 4 { // -(An): predecrement
			return wordVal | 0xE800 | (uint16(p.SrcReg&7) << 4)
		}
		return wordVal | 0xF800 | (uint16(p.SrcReg&7) << 4)
	case instructions.FCacheSel6:
		return wordVal | (uint16(p.SrcReg&3) << 6)
	case instructions.FMove16Reg2_12:
		return wordVal | (uint16(p.DstReg&7) << 12)
	case instructions.FMove16AbsForm:
		if p.SrcEA.Mode == 2 { // (An),ABS.L
			return wordVal | 0xF610 | uint16(p.SrcEA.Reg&7)
		}
		return wordVal | 0xF618 | uint16(p.DstEA.Reg&7) // ABS.L,(An)
	case instructions.FSrcReg2Low:
		return wordVal | uint16(p.SrcReg2&7)
	case instructions.FFPCtrlSelSrc10:
		return wordVal | ((p.FPCtrlMaskSrc & 7) << 10)
	case instructions.FFPCtrlSelDst10:
		return wordVal | ((p.FPCtrlMaskDst & 7) << 10)
	case instructions.FSrcRegShift2:
		return wordVal | (uint16(p.SrcReg&7) << 2)
	case instructions.FDstRegShift2:
		return wordVal | (uint16(p.DstReg&7) << 2)
	case instructions.FFCSpecWord:
		switch p.SrcFCMode {
		case 1: // Dn
			return wordVal | 0x08 | (uint16(p.SrcReg) & 7)
		case 2: // immediate
			return wordVal | 0x10 | (uint16(p.Imm) & 7)
		default: // named SFC(0)/DFC(1)
			return wordVal | (uint16(p.SrcReg) & 7)
		}
	case instructions.FDstImmShift5:
		return wordVal | ((uint16(p.DstImm) & 0x1F) << 5)
	case instructions.FAuxImmShift10:
		return wordVal | ((uint16(p.AuxImm) & 7) << 10)
	case instructions.FSincosRegCos0:
		return wordVal | uint16(p.DstReg&7)
	case instructions.FSincosRegSin7:
		return wordVal | (uint16(p.DstReg2&7) << 7)
	case instructions.FAux2RegShift5:
		return wordVal | (uint16(p.Aux2Reg&7) << 5)
	case instructions.FKFactor:
		if p.DstKFactorIsReg {
			return wordVal | 0x1000 | (uint16(p.DstKFactorVal&7) << 4)
		}
		return wordVal | (uint16(p.DstKFactorVal) & 0x7F)
	case instructions.FSrcRegOnly:
		return wordVal | uint16(p.SrcReg&7)
	default:
		return wordVal
	}
}

// bitFieldSpecCode encodes one half (offset or width) of a 68020
// bit-field specifier into the 6-bit field format shared by both: bit 5
// set plus the register number (0-7) if it's a register, or just the
// literal value (0-31) if not. Cross-checked against GNU binutils'
// gas/config/tc-m68k.c, whose 'O' argument case computes exactly this:
// "(mode == DREG) ? 0x20 + reg : (value & 0x1F)".
func bitFieldSpecCode(isReg bool, val int32) uint16 {
	if isReg {
		return 0x20 | uint16(val&7)
	}
	return uint16(val) & 0x1F
}

// fpRMBit is the FPU "general instruction" word2's R/M bit (bit 14):
// per the MC68881/MC68882 User's Manual (§4.5.2's instruction field
// descriptions), R/M=0 means the source specifier field (bits 13-10)
// holds a source FPm register number, and R/M=1 means it holds a
// memory operand's data-format code instead — i.e. it must be set
// whenever FFPFormat is (see that FieldRef's own doc comment), and
// clear whenever the source is a bare FPm register (FFPSrcReg10, never
// combined with FFPFormat in this codebase).
//
// Missing entirely until found while researching packed BCD's own
// format code (milestone 28): every one of this codebase's <ea>-
// sourced FPU forms had been emitting bit 14 as 0, which isn't merely
// a differently-shaped but still-decodable encoding — a real 68881, or
// any correctly-opcode-table-driven disassembler, matches word2's fixed
// bits against each candidate row in turn, and an R/M=0 word2 matches
// the *register-to-register* row's own fixed-bit pattern instead
// (confirmed by hand: 0x0822, this codebase's old — wrong — encoding
// for "FADD.X (A0),FP0"'s word2, satisfies faddx's register-form row's
// own mask/literal, `two(0xF1C0,0xE07F)`/`0x0022`, exactly). So the bug
// wasn't a cosmetic bit-layout difference: real hardware would have
// read the wrong source entirely (a register, ignoring the <ea> and
// its extension words) rather than erroring.
const fpRMBit = 0x4000

// fpFormatCode returns the 3-bit FPU source/destination format code
// (extension-word bits 12-10) for sz. Cross-checked against GNU
// binutils' GAS m68k opcode table by decoding the per-size literal
// values shared across every FPU instruction (faddl=0x4022,
// fadds=0x4422, faddx=0x4822, faddp=0x4C22, faddw=0x5022, faddd=0x5422,
// faddb=0x5822 — each is fpRMBit | opBase(0x22) | this size's own
// format code<<10, confirming both the per-size table below AND
// fpRMBit's own 0x4000 value at once) — see
// docs/design/cpu-family-support.md.
func fpFormatCode(sz instructions.Size) uint16 {
	switch sz {
	case instructions.LongSize:
		return 0
	case instructions.SingleSize:
		return 1
	case instructions.ExtendedSize:
		return 2
	case instructions.PackedSize:
		return 3
	case instructions.WordSize:
		return 4
	case instructions.DoubleSize:
		return 5
	case instructions.ByteSize:
		return 6
	default:
		return 0
	}
}

func emitTrailer(out []byte, t instructions.TrailerItem, p *prepared) ([]byte, error) {
	switch t {
	case instructions.TSrcEAExt:
		if len(p.SrcEA.Ext) == 0 {
			return out, nil
		}
		for _, w := range p.SrcEA.Ext {
			out = appendWord(out, w)
		}
		return out, nil
	case instructions.TDstEAExt:
		if len(p.DstEA.Ext) == 0 {
			return out, nil
		}
		for _, w := range p.DstEA.Ext {
			out = appendWord(out, w)
		}
		return out, nil
	case instructions.TImmSized:
		return appendWord(out, uint16(int16(p.Imm))), nil
	case instructions.TSrcImm:
		if p.SrcEA.Mode == 7 && p.SrcEA.Reg == 4 {
			switch p.Size {
			case instructions.ByteSize:
				return appendWord(out, uint16(uint8(p.Imm))), nil
			case instructions.WordSize:
				return appendWord(out, uint16(uint16(p.Imm))), nil
			case instructions.LongSize:
				u := uint32(int32(p.Imm))
				out = appendWord(out, uint16(u>>16))
				out = appendWord(out, uint16(u))
				return out, nil
			case instructions.SingleSize, instructions.DoubleSize, instructions.ExtendedSize:
				// A float-format FPU immediate: the value is either a
				// literal float ("#3.14", SrcImmIsFloat) or a plain
				// integer auto-promoted to float ("#5" against a .s/.d/.x
				// size — see validateFPUOperand). Either way it's encoded
				// as an IEEE-754 (single/double) or Motorola 96-bit
				// extended-precision constant, not a raw integer pattern.
				f := p.SrcImmFloat
				if !p.SrcImmIsFloat {
					f = float64(p.Imm)
				}
				return appendFloatImm(out, p.Size, f), nil
			default:
				return appendWord(out, uint16(uint16(p.Imm))), nil
			}
		}
		return out, nil
	case instructions.TBranchWordIfNeeded:
		if p.BrUseWord {
			return appendWord(out, uint16(p.BrDisp16)), nil
		}
		return out, nil
	case instructions.TBranchLongIfNeeded:
		if p.BrUseLong {
			u := uint32(p.BrDisp32)
			out = appendWord(out, uint16(u>>16))
			out = appendWord(out, uint16(u))
		}
		return out, nil
	case instructions.TSrcRegMask:
		mask := p.SrcRegMask
		if p.DstEA.Mode == 4 {
			mask = reverse16(mask)
		}
		return appendWord(out, mask), nil
	case instructions.TDstRegMask:
		return appendWord(out, p.DstRegMask), nil
	case instructions.TAuxEAExt:
		for _, w := range p.AuxEA.Ext {
			out = appendWord(out, w)
		}
		return out, nil
	case instructions.TAuxImmWord:
		return appendWord(out, uint16(p.AuxImm)), nil
	}
	return out, nil
}

func Encode(def *instructions.InstrDef, form *instructions.FormDef, ins *Instr, sym map[string]uint32) ([]byte, error) {
	p := prepared{PC: ins.PC, Size: ins.Args.Size, Imm: ins.Args.Src.Imm, SrcReg: ins.Args.Src.Reg, DstReg: ins.Args.Dst.Reg, SrcRegMask: ins.Args.RegMaskSrc, DstRegMask: ins.Args.RegMaskDst, FPRegMaskSrc: ins.Args.FPRegMaskSrc, FPRegMaskDst: ins.Args.FPRegMaskDst, FPCtrlMaskSrc: ins.Args.FPCtrlMaskSrc, FPCtrlMaskDst: ins.Args.FPCtrlMaskDst, SrcFCMode: ins.Args.Src.FCMode, DstImm: ins.Args.Dst.Imm, Aux2Reg: ins.Args.Aux2.Reg, SrcImmFloat: ins.Args.Src.ImmFloat, SrcImmIsFloat: ins.Args.Src.ImmIsFloat}
	var err error

	if ins.Args.Src.Kind != instructions.EAkNone {
		p.SrcEA, err = instructions.EncodeEA(ins.Args.Src, p.PC)
		if err != nil {
			return nil, err
		}
	}
	if sel, ok := instructions.ControlRegisterSelector(ins.Args.Src.Kind); ok {
		p.CtrlRegSel = sel
	} else if sel, ok := instructions.ControlRegisterSelector(ins.Args.Dst.Kind); ok {
		p.CtrlRegSel = sel
	}

	if ins.Args.Dst.Kind != instructions.EAkNone {
		p.DstEA, err = instructions.EncodeEA(ins.Args.Dst, p.PC)
		if err != nil {
			return nil, err
		}
	}

	p.AuxImm = ins.Args.Aux.Imm
	if ins.Args.Aux.Kind != instructions.EAkNone {
		p.AuxEA, err = instructions.EncodeEA(ins.Args.Aux, p.PC)
		if err != nil {
			return nil, err
		}
	}

	p.SrcBFOffset = bitFieldSpecCode(ins.Args.Src.BFOffsetIsReg, ins.Args.Src.BFOffsetVal)
	p.SrcBFWidth = bitFieldSpecCode(ins.Args.Src.BFWidthIsReg, ins.Args.Src.BFWidthVal)
	p.DstBFOffset = bitFieldSpecCode(ins.Args.Dst.BFOffsetIsReg, ins.Args.Dst.BFOffsetVal)
	p.DstBFWidth = bitFieldSpecCode(ins.Args.Dst.BFWidthIsReg, ins.Args.Dst.BFWidthVal)

	p.SrcReg2 = ins.Args.Src.Reg2
	p.DstReg2 = ins.Args.Dst.Reg2
	p.DstRegPairWide = ins.Args.Dst.RegPairWide
	p.DstKFactorIsReg = ins.Args.Dst.KFactorIsReg
	p.DstKFactorVal = ins.Args.Dst.KFactorVal
	p.AuxReg = ins.Args.Aux.Reg
	p.AuxReg2 = ins.Args.Aux.Reg2

	p.SizeBits = sizeToBits(ins.Args.Size)

	if ins.Args.Target != "" || ins.Args.HasTargetAddr {
		var addr uint32
		if ins.Args.Target != "" {
			resolved, ok := sym[ins.Args.Target]
			if !ok {
				return nil, fmt.Errorf("undefined label: %s", ins.Args.Target)
			}
			addr = resolved
		} else {
			if ins.Args.TargetAddr < 0 || ins.Args.TargetAddr > math.MaxUint32 {
				return nil, fmt.Errorf("branch target out of 32-bit range: %d", ins.Args.TargetAddr)
			}
			addr = uint32(ins.Args.TargetAddr)
		}
		p.TargetPC = addr
		basePC := p.PC + 2
		switch ins.Args.Size {
		case instructions.ByteSize:
			d8 := int32(addr) - int32(basePC)
			if d8 < -128 || d8 > 127 {
				return nil, fmt.Errorf("branch displacement out of range for .S")
			}
			p.BrUseWord = false
			p.BrDisp8 = int8(d8)
		case instructions.WordSize:
			d16 := int32(addr) - int32(basePC)
			// Check if this is a DBcc instruction with target == current PC
			if len(def.Mnemonic) >= 2 && def.Mnemonic[0:2] == "DB" && addr == basePC {
				d16 = -2
			}
			if d16 < -32768 || d16 > 32767 {
				return nil, fmt.Errorf("branch displacement out of range for .W")
			}
			p.BrUseWord = true
			p.BrDisp16 = int16(d16)
		case instructions.LongSize:
			d32 := int64(addr) - int64(basePC)
			if d32 < math.MinInt32 || d32 > math.MaxInt32 {
				return nil, fmt.Errorf("branch displacement out of range for .L")
			}
			p.BrUseLong = true
			p.BrDisp32 = int32(d32)
		default:
			return nil, fmt.Errorf("unsupported branch size")
		}
	}

	out := make([]byte, 0, 12)
	for _, step := range form.Steps {
		// A step emits its word unless it exists purely to carry a
		// Trailer (WordBits==0, no Fields, but a non-empty Trailer) —
		// that's the one case a literal, all-zero word (e.g. FNOP's
		// second word, 0x0000) must still be distinguishable from "no
		// word here." instructionWords (parser_stmt.go) must compute
		// this identically, or PC advancement during parsing would
		// disagree with what Encode actually emits.
		haveWord := step.WordBits != 0 || len(step.Fields) > 0 || len(step.Trailer) == 0
		if haveWord {
			w := step.WordBits
			for _, f := range step.Fields {
				w = applyField(w, f, &p)
			}
			out = append(out, byte(w>>8), byte(w))
		}
		for _, tr := range step.Trailer {
			var err error
			out, err = emitTrailer(out, tr, &p)
			if err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

func reverse16(v uint16) uint16 {
	v = (v >> 8) | (v << 8)
	v = ((v & 0xF0F0) >> 4) | ((v & 0x0F0F) << 4)
	v = ((v & 0xCCCC) >> 2) | ((v & 0x3333) << 2)
	v = ((v & 0xAAAA) >> 1) | ((v & 0x5555) << 1)
	return v
}
