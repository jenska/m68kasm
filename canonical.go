package m68kasm

import (
	"fmt"
	"strings"

	internal "github.com/jenska/m68kasm/internal/asm"
	"github.com/jenska/m68kasm/internal/asm/instructions"
)

// InstructionResult captures a narrow single-instruction assembly result.
type InstructionResult struct {
	Bytes       []byte
	PC          uint32
	EncodedSize int
	Words       int
	Canonical   string
}

// AssembleInstructionString assembles a single instruction and returns its bytes and metadata.
func AssembleInstructionString(src string) (InstructionResult, error) {
	return AssembleInstructionStringWithOptions(src, ParseOptions{})
}

// AssembleInstructionStringWithOptions assembles a single instruction with parser options.
func AssembleInstructionStringWithOptions(src string, opts ParseOptions) (InstructionResult, error) {
	prog, ins, err := parseSingleInstruction(src, opts)
	if err != nil {
		return InstructionResult{}, err
	}

	bytes, err := internal.Assemble(prog)
	if err != nil {
		return InstructionResult{}, err
	}

	return InstructionResult{
		Bytes:       bytes,
		PC:          ins.PC,
		EncodedSize: len(bytes),
		Words:       len(bytes) / 2,
		Canonical:   canonicalInstruction(ins),
	}, nil
}

// CanonicalizeInstructionString parses a single instruction and returns a stable canonical spelling.
func CanonicalizeInstructionString(src string) (string, error) {
	return CanonicalizeInstructionStringWithOptions(src, ParseOptions{})
}

// CanonicalizeInstructionStringWithOptions parses a single instruction with parser options
// and returns a stable canonical spelling suitable for disassembly comparisons.
func CanonicalizeInstructionStringWithOptions(src string, opts ParseOptions) (string, error) {
	_, ins, err := parseSingleInstruction(src, opts)
	if err != nil {
		return "", err
	}
	return canonicalInstruction(ins), nil
}

func parseSingleInstruction(src string, opts ParseOptions) (*internal.Program, *internal.Instr, error) {
	prog, err := internal.ParseWithOptions(strings.NewReader(src), internal.ParseOptions(opts))
	if err != nil {
		return nil, nil, err
	}

	if len(prog.Items) != 1 {
		return nil, nil, fmt.Errorf("expected exactly one instruction, got %d emitted items", len(prog.Items))
	}

	ins, ok := prog.Items[0].(*internal.Instr)
	if !ok {
		return nil, nil, fmt.Errorf("expected a single instruction")
	}
	return prog, ins, nil
}

func canonicalInstruction(ins *internal.Instr) string {
	if ins == nil || ins.Def == nil {
		return ""
	}

	mnemonic := strings.ToUpper(ins.Def.Mnemonic)
	if hasExplicitInstructionSize(ins) {
		mnemonic += "." + sizeSuffix(ins.Args.Size)
	}

	operands := canonicalOperands(ins)
	if len(operands) == 0 {
		return mnemonic
	}
	return mnemonic + " " + strings.Join(operands, ",")
}

func hasExplicitInstructionSize(ins *internal.Instr) bool {
	return ins != nil && ins.Form != nil && len(ins.Form.Sizes) > 0
}

func canonicalOperands(ins *internal.Instr) []string {
	if ins == nil || ins.Form == nil {
		return nil
	}

	ops := make([]string, 0, len(ins.Form.OperKinds))
	for i, kind := range ins.Form.OperKinds {
		switch kind {
		case instructions.OpkImm:
			ops = append(ops, "#"+formatSignedHex(ins.Args.Src.Imm))
		case instructions.OpkImmQuick:
			ops = append(ops, "#"+formatSignedHex(ins.Args.Src.Imm))
		case instructions.OpkDn, instructions.OpkAn, instructions.OpkSR, instructions.OpkCCR, instructions.OpkUSP, instructions.OpkEA, instructions.OpkPredecAn:
			if i == 0 {
				ops = append(ops, formatEA(ins.Args.Src))
			} else {
				ops = append(ops, formatEA(ins.Args.Dst))
			}
		case instructions.OpkRegList:
			if i == 0 {
				ops = append(ops, formatRegList(ins.Args.RegMaskSrc))
			} else {
				ops = append(ops, formatRegList(ins.Args.RegMaskDst))
			}
		case instructions.OpkDispRel:
			if ins.Args.Target != "" {
				ops = append(ops, ins.Args.Target)
			} else {
				ops = append(ops, formatSignedHex(ins.Args.TargetAddr))
			}
		default:
			if i == 0 {
				ops = append(ops, formatEA(ins.Args.Src))
			} else {
				ops = append(ops, formatEA(ins.Args.Dst))
			}
		}
	}
	return ops
}

func sizeSuffix(sz instructions.Size) string {
	switch sz {
	case instructions.ByteSize:
		return "B"
	case instructions.WordSize:
		return "W"
	case instructions.LongSize:
		return "L"
	default:
		return "?"
	}
}

func formatEA(e instructions.EAExpr) string {
	switch e.Kind {
	case instructions.EAkImm:
		return "#" + formatSignedHex(e.Imm)
	case instructions.EAkDn:
		return formatDataRegister(e.Reg)
	case instructions.EAkAn:
		return formatAddrRegister(e.Reg)
	case instructions.EAkAddrPredec:
		return "-(" + formatAddrRegister(e.Reg) + ")"
	case instructions.EAkAddrPostinc:
		return "(" + formatAddrRegister(e.Reg) + ")+"
	case instructions.EAkAddrInd:
		return "(" + formatAddrRegister(e.Reg) + ")"
	case instructions.EAkAddrDisp16:
		return formatSignedHex(int64(e.Disp16)) + "(" + formatAddrRegister(e.Reg) + ")"
	case instructions.EAkPCDisp16:
		return formatSignedHex(int64(e.Disp16)) + "(PC)"
	case instructions.EAkIdxAnBrief:
		return formatIndexAddress(formatAddrRegister(e.Reg), e.Index)
	case instructions.EAkIdxPCBrief:
		return formatIndexAddress("PC", e.Index)
	case instructions.EAkAbsW:
		return "(" + formatUint32Hex(uint32(e.Abs16), 4) + ").W"
	case instructions.EAkAbsL:
		return "(" + formatUint32Hex(e.Abs32, 8) + ").L"
	case instructions.EAkSR:
		return "SR"
	case instructions.EAkCCR:
		return "CCR"
	case instructions.EAkUSP:
		return "USP"
	default:
		return ""
	}
}

func formatIndexAddress(base string, ix instructions.EAIndex) string {
	return formatSignedHex(int64(ix.Disp8)) + "(" + base + "," + formatIndexRegister(ix) + ")"
}

func formatIndexRegister(ix instructions.EAIndex) string {
	reg := formatDataRegister(ix.Reg)
	if ix.IsA {
		reg = formatAddrRegister(ix.Reg)
	}
	out := reg + "." + map[bool]string{false: "W", true: "L"}[ix.Long]
	if ix.Scale > 1 {
		out += fmt.Sprintf("*%d", ix.Scale)
	}
	return out
}

func formatRegList(mask uint16) string {
	parts := make([]string, 0, 4)
	parts = appendRegisterRuns(parts, mask&0x00FF, "D")
	parts = appendRegisterRuns(parts, (mask>>8)&0x00FF, "A")
	return strings.Join(parts, "/")
}

func appendRegisterRuns(parts []string, mask uint16, prefix string) []string {
	for reg := 0; reg < 8; reg++ {
		if mask&(1<<reg) == 0 {
			continue
		}
		start := reg
		for reg+1 < 8 && mask&(1<<(reg+1)) != 0 {
			reg++
		}
		end := reg
		if start == end {
			parts = append(parts, fmt.Sprintf("%s%d", prefix, start))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s%d-%s%d", prefix, start, prefix, end))
	}
	return parts
}

func formatDataRegister(reg int) string {
	return fmt.Sprintf("D%d", reg)
}

func formatAddrRegister(reg int) string {
	return fmt.Sprintf("A%d", reg)
}

func formatSignedHex(v int64) string {
	if v < 0 {
		return "-" + formatUint64Hex(uint64(-v), 0)
	}
	return formatUint64Hex(uint64(v), 0)
}

func formatUint64Hex(v uint64, width int) string {
	if width > 0 {
		return fmt.Sprintf("$%0*X", width, v)
	}
	return fmt.Sprintf("$%X", v)
}

func formatUint32Hex(v uint32, width int) string {
	return formatUint64Hex(uint64(v), width)
}
