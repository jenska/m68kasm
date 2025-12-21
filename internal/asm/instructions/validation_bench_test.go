package instructions

import "testing"

func BenchmarkValidateAddSubToDn(b *testing.B) {
	// Szenario: ADD.W (A0),D0
	// Dies nutzt validateAddSubToDn, wo wir redundante Checks entfernt haben.
	def := Instructions["ADD"]
	if def == nil {
		b.Fatal("ADD instruction not found")
	}
	form := &def.Forms[0] // Form 0: OpkEA, OpkDn
	args := &Args{
		Src:  EAExpr{Kind: EAkAddrInd, Reg: 0},
		Dst:  EAExpr{Kind: EAkDn, Reg: 0},
		Size: WordSize,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := form.Validate(args); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateAddSubDnToEA(b *testing.B) {
	// Szenario: ADD.W D0,(A0)
	// Dies nutzt validateAddSubDnToEA.
	def := Instructions["ADD"]
	if def == nil {
		b.Fatal("ADD instruction not found")
	}
	form := &def.Forms[1] // Form 1: OpkDn, OpkEA
	args := &Args{
		Src:  EAExpr{Kind: EAkDn, Reg: 0},
		Dst:  EAExpr{Kind: EAkAddrInd, Reg: 0},
		Size: WordSize,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := form.Validate(args); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateAddSubX(b *testing.B) {
	// Szenario: ADDX.W D0,D1
	// Dies nutzt validateAddSubX, welches nun fast leer sein sollte.
	def := Instructions["ADDX"]
	if def == nil {
		b.Fatal("ADDX instruction not found")
	}
	form := &def.Forms[0] // Form 0: OpkDn, OpkDn
	args := &Args{
		Src:  EAExpr{Kind: EAkDn, Reg: 0},
		Dst:  EAExpr{Kind: EAkDn, Reg: 1},
		Size: WordSize,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := form.Validate(args); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateAddSubQuick(b *testing.B) {
	// Szenario: ADDQ.W #1,D0
	def := Instructions["ADDQ"]
	if def == nil {
		b.Fatal("ADDQ instruction not found")
	}
	form := &def.Forms[0]
	args := &Args{
		Src:         EAExpr{Kind: EAkNone, Imm: 1},
		Dst:         EAExpr{Kind: EAkDn, Reg: 0},
		Size:        WordSize,
		HasImmQuick: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := form.Validate(args); err != nil {
			b.Fatal(err)
		}
	}
}
