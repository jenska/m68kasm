package instructions

import "fmt"

// swapSrcDstIfDstNone swaps Src and Dst if Dst is empty.
// Handles single-operand instructions parsed with operand in source position.
func swapSrcDstIfDstNone(a *Args) {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
		a.Src = EAExpr{}
	}
}

// Control addressable EA kinds: (An), d(An), d(An,Rx), xxx.W, xxx.L, PC-relative
var controlAlterableEA = map[EAExprKind]bool{
	EAkAddrInd:    true,
	EAkAddrDisp16: true,
	EAkIdxAnBrief: true,
	EAkAbsW:       true,
	EAkAbsL:       true,
	EAkPCDisp16:   true,
	EAkIdxPCBrief: true,
}

func validateControlEA(name string, a *Args) error {
	swapSrcDstIfDstNone(a)
	if a.Dst.Kind == EAkNone {
		return fmt.Errorf("%s requires destination", name)
	}
	if !controlAlterableEA[a.Dst.Kind] {
		return fmt.Errorf("%s requires control addressing mode", name)
	}
	return nil
}

func checkImmediateRange(v int64, sz Size) error {
	switch sz {
	case ByteSize:
		if v < -128 || v > 255 {
			return fmt.Errorf("immediate out of range for .b: %d", v)
		}
	case WordSize:
		if v < -32768 || v > 65535 {
			return fmt.Errorf("immediate out of range for .w: %d", v)
		}
	case LongSize:
		if v < -0x80000000 || v > 0xFFFFFFFF {
			return fmt.Errorf("immediate out of range for .l: %d", v)
		}
	}
	return nil
}

var pcRelativeEA = map[EAExprKind]bool{
	EAkPCDisp16:   true,
	EAkIdxPCBrief: true,
}

func isPCRelativeKind(k EAExprKind) bool {
	return pcRelativeEA[k]
}

// Memory alterable EA kinds: (An), (An)+, -(An), d(An), d(An,Rx), xxx.W, xxx.L
var memoryAlterableEA = map[EAExprKind]bool{
	EAkAddrInd:     true,
	EAkAddrPostinc: true,
	EAkAddrPredec:  true,
	EAkAddrDisp16:  true,
	EAkIdxAnBrief:  true,
	EAkAbsW:        true,
	EAkAbsL:        true,
}

// Data alterable EA kinds: Dn, plus memory alterable
var dataAlterableEA = map[EAExprKind]bool{
	EAkDn:          true,
	EAkAddrInd:     true,
	EAkAddrPostinc: true,
	EAkAddrPredec:  true,
	EAkAddrDisp16:  true,
	EAkIdxAnBrief:  true,
	EAkAbsW:        true,
	EAkAbsL:        true,
}

// isMemoryAlterable checks if an EA kind is memory-alterable (but not Dn/An).
func isMemoryAlterable(k EAExprKind) bool {
	return memoryAlterableEA[k]
}

// isDataAlterable checks if an EA kind is data-alterable (Dn or memory alterable).
func isDataAlterable(k EAExprKind) bool {
	return dataAlterableEA[k]
}

// readableDataEA kinds: data registers, memory addresses, PC-relative, and immediates
var readableDataEA = map[EAExprKind]bool{
	EAkDn:          true,
	EAkAddrInd:     true,
	EAkAddrPostinc: true,
	EAkAddrPredec:  true,
	EAkAddrDisp16:  true,
	EAkIdxAnBrief:  true,
	EAkAbsW:        true,
	EAkAbsL:        true,
	EAkPCDisp16:    true,
	EAkIdxPCBrief:  true,
	EAkImm:         true,
}

// isReadableDataEA checks if EA can be used as a source for data operations (like MOVE to SR/CCR).
func isReadableDataEA(k EAExprKind) bool {
	return readableDataEA[k]
}

// movemLoadEA kinds: memory addresses that can be sources for MOVEM (includes PC-relative and postincrement, but not predecrement)
var movemLoadEA = map[EAExprKind]bool{
	EAkPCDisp16:    true,
	EAkIdxPCBrief:  true,
	EAkAddrInd:     true,
	EAkAddrPostinc: true,
	EAkAddrDisp16:  true,
	EAkAddrPredec:  true,
	EAkIdxAnBrief:  true,
	EAkAbsW:        true,
	EAkAbsL:        true,
}

// isMovemLoadEA checks if EA can be used as a source for MOVEM.
func isMovemLoadEA(k EAExprKind) bool {
	return movemLoadEA[k]
}
