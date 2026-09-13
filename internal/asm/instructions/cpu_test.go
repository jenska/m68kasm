package instructions

import "testing"

func TestTargetSupportsCPUOrdering(t *testing.T) {
	cases := []struct {
		target CPUKind
		req    CPUKind
		want   bool
	}{
		{CPU68000, CPU68000, true},
		{CPU68000, CPU68010, false},
		{CPU68010, CPU68000, true},
		{CPU68020, CPU68010, true},
		{CPU68060, CPU68000, true},
		{CPU68010, CPU68020, false},
	}
	for _, c := range cases {
		got := Target{CPU: c.target}.Supports(Target{CPU: c.req})
		if got != c.want {
			t.Errorf("Target{%s}.Supports(Target{%s}) = %v, want %v", c.target, c.req, got, c.want)
		}
	}
}

func TestTargetSupportsFeatureSubset(t *testing.T) {
	full := Target{CPU: CPU68040, Features: FeatFPU | FeatPMMU}

	if !full.Supports(Target{Features: FeatFPU}) {
		t.Error("target with FeatFPU|FeatPMMU should support a FeatFPU requirement")
	}
	if full.Supports(Target{Features: FeatFPUFull}) {
		t.Error("target without FeatFPUFull should not support a FeatFPUFull requirement")
	}
	if !Target68000.Supports(Target{}) {
		t.Error("every target should support a requirement with no CPU/feature floor")
	}
}

// TestTargetSupportsCPU32Lattice locks in milestone 7's core model:
// CPU32 sits beside the 68010->68020 chain rather than on it. A form
// tagged Target{CPU: CPU32} (the 68020-era additions CPU32 backported —
// Bcc.L, CHK2/CMP2, EXTB, TRAPcc, scale factors) is satisfied by CPU32
// *and* every 68020+ tier via the plain CPU-floor comparison; a form
// tagged requireCPU32Only (CPU32-exclusive: TBLS/TBLU, BGND) is
// satisfied by CPU32 alone, which the floor comparison could never
// express by itself since 68020's enum value is numerically higher.
func TestTargetSupportsCPU32Lattice(t *testing.T) {
	cpu32 := Target{CPU: CPU32}
	sharedReq := Target{CPU: CPU32} // e.g. Bcc.L, CHK2/CMP2, EXTB, TRAPcc

	for _, tier := range []CPUKind{CPU68020, CPU68030, CPU68040, CPU68060} {
		if !(Target{CPU: tier}).Supports(sharedReq) {
			t.Errorf("Target{%s} should support the shared CPU32/68020+ requirement", tier)
		}
	}
	if !cpu32.Supports(sharedReq) {
		t.Error("CPU32 should support its own shared-with-68020+ requirement")
	}
	for _, tier := range []CPUKind{CPU68000, CPU68010} {
		if (Target{CPU: tier}).Supports(sharedReq) {
			t.Errorf("Target{%s} should NOT support the CPU32/68020+ requirement", tier)
		}
	}

	// requireCPU32Only: CPU32 alone, never 68020+ even though its enum
	// value is higher (the memory-indirect-addressing asymmetry, and
	// BGND/TBLS's actual gate).
	if !cpu32.Supports(requireCPU32Only) {
		t.Error("CPU32 should support a CPU32-exclusive requirement")
	}
	for _, tier := range []CPUKind{CPU68000, CPU68010, CPU68020, CPU68030, CPU68040, CPU68060} {
		if (Target{CPU: tier}).Supports(requireCPU32Only) {
			t.Errorf("Target{%s} should NOT support a CPU32-exclusive requirement", tier)
		}
	}

	// Memory-indirect addressing is tagged Target{CPU: CPU68020} exactly
	// (not CPU32, and not requireCPU32Only) precisely because CPU32 lacks
	// it despite otherwise sharing most 68020 additions.
	memIndirectReq := Target{CPU: CPU68020}
	if cpu32.Supports(memIndirectReq) {
		t.Error("CPU32 should NOT support a plain CPU68020 requirement (memory-indirect addressing)")
	}
	if !(Target{CPU: CPU68020}).Supports(memIndirectReq) {
		t.Error("a 68020 target should support a plain CPU68020 requirement")
	}
}

func TestParseCPUKind(t *testing.T) {
	cases := map[string]CPUKind{
		"68000": CPU68000,
		"68008": CPU68000,
		"68010": CPU68010,
		"68012": CPU68010,
		"CPU32": CPU32,
		"68020": CPU68020,
		"68030": CPU68030,
		"68040": CPU68040,
		"68060": CPU68060,
	}
	for name, want := range cases {
		got, ok := ParseCPUKind(name)
		if !ok || got != want {
			t.Errorf("ParseCPUKind(%q) = (%v, %v), want (%v, true)", name, got, ok, want)
		}
	}
	if _, ok := ParseCPUKind("68050"); ok {
		t.Error("ParseCPUKind(\"68050\") should fail: no such CPU")
	}
}

// synthTable builds a standalone Table from the given defs, independent
// of the DefaultTable() singleton and the shared Instructions registry,
// so ForTarget can be tested in isolation.
func synthTable(defs ...*InstrDef) *Table {
	m := make(map[string]*InstrDef, len(defs))
	for _, d := range defs {
		m[d.Mnemonic] = d
	}
	return &Table{defs: m}
}

func TestTableForTargetFiltersGatedForms(t *testing.T) {
	base := synthTable(&InstrDef{
		Mnemonic: "MIXED",
		Forms: []FormDef{
			{OperKinds: []OperandKind{OpkNone}},
			{OperKinds: []OperandKind{OpkImm}, Requires: Target{CPU: CPU68020}},
		},
	})

	on68000 := base.ForTarget(Target68000)
	def := on68000.Lookup("MIXED")
	if def == nil {
		t.Fatal("MIXED should still be visible on a 68000 target (it has a baseline form)")
	}
	if len(def.Forms) != 1 {
		t.Errorf("MIXED on 68000 target: got %d forms, want 1 (the 68020-only form must be filtered out)", len(def.Forms))
	}

	on68020 := base.ForTarget(Target{CPU: CPU68020})
	def = on68020.Lookup("MIXED")
	if def == nil || len(def.Forms) != 2 {
		t.Errorf("MIXED on 68020 target: got %v, want both forms visible", def)
	}
}

func TestTableForTargetDropsMnemonicWithNoSupportedForms(t *testing.T) {
	base := synthTable(&InstrDef{
		Mnemonic: "ONLY68020",
		Forms: []FormDef{
			{OperKinds: []OperandKind{OpkNone}, Requires: Target{CPU: CPU68020}},
		},
	})

	on68000 := base.ForTarget(Target68000)
	if def := on68000.Lookup("ONLY68020"); def != nil {
		t.Errorf("ONLY68020 should be entirely absent on a 68000 target, got %v", def)
	}
	on68020 := base.ForTarget(Target{CPU: CPU68020})
	if def := on68020.Lookup("ONLY68020"); def == nil {
		t.Error("ONLY68020 should be visible on a 68020 target")
	}
}
