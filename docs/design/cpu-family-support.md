# Design Proposal: Multi-CPU Support for m68kasm

Status: **Draft / RFC** — no code changes implied by this document.
Author: Claude Sonnet 5, for discussion with @jenska.

## 1. Problem statement

m68kasm currently targets exactly one ISA level: the base MC68000. This is
stated explicitly as a known limitation in the README, and it is true at
every layer of the pipeline:

- The lexer has no notion of floating-point literals (`NUMBER` tokens are
  always `int64`), which blocks FPU immediates.
- `internal/asm/instructions` has a single global `Instructions` map keyed
  only by mnemonic (`types.go:104`), with no per-form or per-instruction
  concept of "which CPU(s) support this."
- `internal/asm/instructions/ea.go` encodes exactly the seven classic 68000
  addressing modes. Interestingly, `parser_ea.go` already *parses* `*1/*2/
  *4/*8` scale factors on indexed addressing (`parser_ea.go:343-356`) and
  `encodeBriefIndex` in `ea.go:75-93` already *encodes* scale bits — but
  scale ≠ 1 is a 68020+-only feature. Today it is silently accepted and
  encoded for a "68000-only" assembler, which is a real correctness gap:
  the assembler will happily emit an opcode a real 68000 cannot execute.
- `internal/asm/encode.go`'s branch handling only produces byte/word
  displacements (`prepared.BrDisp8/BrDisp16`); 68020 added 32-bit branch
  displacements (`Bcc.L`, `BSR.L`, etc.).
- There is no CPU selection surface anywhere: not in `ParseOptions`
  (`parser.go:73`), not in the CLI (`cmd/m68kasm/main.go`), not as a
  pseudo-op (`pseudos.go`).

The good news: the project already has two extension points that a
multi-CPU design can reuse almost as-is:

1. **`instructions.Table`** (`table.go`) is already a cloneable,
   swappable instruction registry, and `ParseOptions.InstrTable` already
   lets a caller hand the parser an alternate table. This is 90% of the
   plumbing needed for "parse this file against a different instruction
   set."
2. **Declarative `FormDef`/`EmitStep` tables** (`types.go`, `encode.go`)
   mean new opcodes are additive data, not new control flow. Adding a
   68020 instruction is, mechanically, the same kind of change as adding
   `CMPM` was.

The gap is a **CPU/feature axis that both the parser and the encoder can
see**, plus the actual 68010–68060/CPU32/FPU instruction and
addressing-mode data.

## 2. Scope

Target ISA levels, based on Motorola's actual family tree:

| Tier | Devices | Notes |
|---|---|---|
| `M68000` | 68000, 68008 | 68008 is 68000 with an 8-bit bus — no ISA difference, so it's `M68000` plus a "no `MOVE16`-style alignment assumptions" note, not a separate ISA tier |
| `M68010` | 68010, 68012 | Adds `MOVEC`, `MOVES`, `RTD`, `BKPT`; loop-mode `DBcc` |
| `CPU32` | 68330/68331/68332/683xx | 68010 superset plus most 68020 addressing modes and `TBLS`/`TBLU`/`TBLSN`/`TBLUN`; explicitly **omits** `BFxxx` bitfield ops, `CAS`/`CAS2`, `CALLM`/`RTM`, and full 68020 privileged instruction set. **Needs verification against the CPU32RM before the feature table is finalized** — flagged throughout this doc as the one tier with a materially different subset shape rather than a strict superset. |
| `M68020` | 68020, 68EC020 | Full extension-word addressing (memory indirect, base/outer displacement, scale 2/4/8), `BFxxx`, `CAS`/`CAS2`, `CHK2`/`CMP2`, `DIVSL`/`DIVUL`, `EXTB`, `PACK`/`UNPK`, `CALLM`/`RTM`, `TRAPcc`, 32-bit branch displacements, coprocessor interface (`cpGEN`/`cpBcc`/`cpDBcc`/`cpScc`/`cpTRAPcc`/`cpSAVE`/`cpRESTORE`) used by 68881/68882 |
| `M68030` | 68030, 68EC030 | 68020 superset plus on-chip MMU (`PMOVE`/`PFLUSH`/`PTEST`/`PLOAD`/`PVALID` and friends) |
| `M68040` | 68040, 68EC040, 68LC040 | 68030 superset; integrates FPU (subset — no transcendentals) and MMU on-chip; adds `MOVE16`, `CINV`/`CPUSH`; **drops** the general coprocessor interface (`cpGEN` etc.) and `CALLM`/`RTM` |
| `M68060` | 68060, 68LC060 | 68040 superset in the assembled-mnemonic sense, but several instructions (`CAS2`, `CHK2`/`CMP2`, `MOVEP`, dynamic `BFxxx` forms, 64-bit `DIVS`/`DIVU.L`) are trap-and-emulate rather than native. Assembler-relevant distinction: still *assembles*, just worth a severity tier (see §5.4). |
| FPU | 68881, 68882 (discrete); 68040/68060 (integrated subset) | Orthogonal capability, not a CPU tier — see §6 |
| MMU | 68851 (discrete, used with 68020); on-chip in 68030/68040/68060 | Orthogonal capability — mnemonics already covered by the M68030/040/060 tiers above; a discrete 68851 on a bare 68020 system is the one case where MMU support is needed *without* implying M68030+, so it's modeled as a capability, not folded into the CPU tier |

Explicitly **out of scope** for this proposal:
- Cycle-accurate timing/scheduling — this is an assembler, not a simulator.
- Relaxation/optimization across CPU tiers (already deferred in the
  README's "Next up" section).
- Disassembly.

## 3. Core model: CPU tiers + orthogonal feature flags

Two axes, not one enum, because FPU/MMU presence is independent of the
base integer ISA tier (a bare 68020 + external 68881 is common; a 68040
has FPU built in; a 68EC040 has no FPU at all).

```go
// internal/asm/instructions/cpu.go (new)

// CPUKind is the base integer-unit ISA level.
type CPUKind uint8

const (
    CPU68000 CPUKind = iota // also covers 68008
    CPU68010                // also covers 68012
    CPU32                   // 68330-family
    CPU68020
    CPU68030
    CPU68040
    CPU68060
)

// Feature is an orthogonal capability bit, independent of CPUKind.
type Feature uint32

const (
    FeatFPU        Feature = 1 << iota // 68881/68882 or integrated FPU
    FeatFPUFull                        // full transcendental set (discrete 68881/882), vs. 68040/060's reduced set
    FeatPMMU                           // 68851 or on-chip PMMU
    FeatEmulated                       // marks forms that assemble but are trap-emulated on this tier (68060)
)

// Target bundles the two axes plus the derived "which addressing modes
// and instruction forms are legal" query surface.
type Target struct {
    CPU      CPUKind
    Features Feature
}

var Target68000 = Target{CPU: CPU68000}

// Supports reports whether a form/instruction tagged with `req` is legal
// for this target. A CPUKind requirement is satisfied by >= req.CPU;
// a Feature requirement is satisfied by req.Features being a subset of
// t.Features.
func (t Target) Supports(req Target) bool {
    return t.CPU >= req.CPU && (req.Features&^t.Features) == 0
}
```

`CPUKind` is ordered so `t.CPU >= req.CPU` works for the strict-superset
tiers (68000 → 68010 → 68020 → 68030 → 68040 → 68060). **`CPU32` is
deliberately not on that line** — it sits beside 68010, not between it and
68020, because it is not a strict superset of 68010 in the same way the
68020+ chain is a strict superset of 68010. Concretely: `Supports` treats
`CPU32` as satisfying only `CPU68000`/`CPU68010`-tier requirements, plus a
dedicated allowlist of the CPU32-only forms (`TBLS`/`TBLU`/loop-mode
`DBcc`), never the `CPU68020`+ line. This keeps the "is it a superset"
question answerable by a single integer comparison for every tier except
one clearly-documented exception, rather than forcing a general subset
lattice for a family tree that is a line with one branch.

## 4. Threading `Target` through the pipeline

This is the part that isn't just "add more table rows" — CPU selection
changes *parsing*, not only encoding, because scale factors and full
extension-word syntax are decided while parsing operands, before any
`InstrDef` lookup happens.

```
CLI flag / .cpu directive / Go API option
              │
              ▼
        instructions.Target  ──────────┐
              │                        │
              ▼                        ▼
        Parser (parser_ea.go)    instructions.Table
   (gates scale>1, full ext      (gates which InstrDef/
    words, new EA syntax          FormDef are visible/
    at parse time)                 selectable)
              │                        │
              ▼                        ▼
           Program{ Target }  ──▶  assemble.go / encode.go
                                  (selectForm + EncodeEA respect
                                   Target; branch encode gains
                                   32-bit displacement path)
```

Concretely:

- **`ParseOptions`** (`parser.go:73`) gains a `Target instructions.Target`
  field, defaulting to `Target68000` when zero-valued — this preserves
  every existing caller's behavior with no source change, which matters
  because `ParseOptions` is a public, embedded type
  (`assemble_api.go:22`).
- **`Parser`** carries the active `Target` and consults it in
  `parser_ea.go`: the scale-factor branch (`parser_ea.go:343-356`) becomes
  a hard parse error ("scale factors require 68020 or later") when the
  target doesn't support it, instead of silently accepting syntax the
  target can't run. Same treatment for full extension-word syntax (base
  displacement, memory indirect, suppressed index/base) once that syntax
  is added at all — see §5.2.
- **`Program`** (`assemble.go:11`) gains a `Target` field, set once by the
  parser, so `internal/asm/assemble.go` and `encode.go` don't need it
  threaded through every function signature — they read it off the
  program they're already given.
- **`instructions.Table`** gains a `ForTarget(t Target) *Table` method
  that filters `Instructions` down to forms/defs whose `Requires` field
  `t.Supports(...)`. `DefaultTable()` stays what it is today (unfiltered,
  68000-only content is unaffected since existing defs simply carry a
  zero-value `Requires` = "68000 baseline"); new 68010+ content is added
  to the *same* global map with non-zero `Requires`, and
  `ForTarget(Target68000)` naturally excludes it. This means **no
  existing behavior changes for callers who never set `Target`.**
- **`selectForm`** (`assemble.go:155`) already loops `def.Forms` looking
  for a match; it gains one more filter condition alongside the existing
  size/operand-kind checks: skip forms whose `Requires` the active
  `Target` doesn't satisfy. This is what makes e.g. a 68020-only
  memory-indirect form invisible to `ADDA` on a 68000 target even though
  both forms live under the same `InstrDef`.
- **CLI** (`cmd/m68kasm/main.go`): new `--cpu` flag (`68000` default,
  `68008`, `68010`, `68012`, `cpu32`, `68020`, `68030`, `68040`, `68060`)
  and `--fpu`/`--mmu` boolean or `--fpu=68881` flags, parsed into a
  `instructions.Target` and passed via `ParseOptions`.
- **Pseudo-op** `.cpu <name>` (e.g. `.cpu 68020`, `.cpu 68040 fpu`) in
  `pseudos.go`, for source files that want to be self-describing rather
  than depend on build-time flags — mirrors how vasm/other multi-target
  assemblers do it. Design choice to settle with the maintainer: **one
  `Target` per assembly unit** (directive must appear before the first
  instruction, and the parser's first pass already runs top-to-bottom, so
  this is cheap to enforce) rather than allowing mid-file CPU switching.
  Real 68k systems don't hot-swap CPUs mid-program, and per-instruction
  target switching would explode the test matrix for a feature no known
  use case needs — flag this as a place to say no if asked to generalize
  further.

## 5. Instruction-set additions by tier

All additions follow the existing declarative pattern (`FormDef`/
`EmitStep`/`FieldRef`/`TrailerItem`) — the mechanism doesn't change, only
the data and the new `Requires` tag on each `FormDef`.

### 5.1 M68010 / M68012

Smallest tier, good first milestone:
- `MOVEC` (control register ↔ Dn/An: `SFC`, `DFC`, `USP`, `VBR`, and on
  68010 that's it — 68020+ adds `CACR`, `CAAR`, etc., so `MOVEC`'s valid
  register list is itself tier-gated, not just the mnemonic)
- `MOVES` (move with address space override)
- `RTD` (return and deallocate — like `RTS` but with a 16-bit stack
  adjustment operand)
- `BKPT` (breakpoint trap, `#0`-`#7` operand)
- Loop-mode `DBcc` is not a new mnemonic (already have `DBF`/`DBcc`
  encoding), just a documentation note that the CPU32/68010+ "loop mode"
  optimization is a hardware fast-path over the existing opcode, not
  something the assembler encodes differently.

### 5.2 M68020 (the big one)

Two independent additions, both large:

**a) Full extension-word addressing.** `EAExpr` needs new fields for
base displacement (0/16/32-bit), outer displacement (none/16/32-bit),
base/index suppress bits, and memory indirect (pre-indexed vs.
post-indexed). `eaTable`/`EncodeEA` (`ea.go`) need a new encoding path —
the brief extension word format (`encodeBriefIndex`) stays for
backward-compatible 68000 code, and a sibling `encodeFullExt` handles the
new cases. This is the single largest chunk of new logic in the whole
proposal; budget a dedicated milestone for it and get the addressing-mode
matrix reviewed against the 68020 User's Manual table before writing
`EmitStep`s, since off-by-one bit placement here is the classic 68020
assembler bug.

**b) New mnemonics:** `BFTST`/`BFCHG`/`BFCLR`/`BFSET`/`BFEXTU`/`BFEXTS`/
`BFFFO`/`BFINS` (bit fields — introduce `OpkBitField` operand kind for
the `{offset:width}` syntax, itself needing new lexer/parser support),
`CAS`/`CAS2`, `CHK2`/`CMP2`, `DIVSL`/`DIVUL` (64-bit divide forms),
`EXTB`, `PACK`/`UNPK`, `CALLM`/`RTM`, `TRAPcc` (with/without operand,
three encodings), and the coprocessor interface family (`cpGEN`, `cpBcc`,
`cpDBcc`, `cpScc`, `cpTRAPcc`, `cpSAVE`, `cpRESTORE`) that the FPU tier
(§6) rides on top of.

**c) 32-bit branch displacement.** `Bcc.L`/`BSR.L`: `prepared` in
`encode.go` needs a `BrDisp32 int32` alongside the existing 8/16-bit
fields, and `Encode`'s displacement-range switch (`encode.go:213-234`)
gains a `LongSize` case. Low-risk, mechanical, worth doing as its own PR
before the bitfield/full-EA work since it exercises the `Requires` gating
end-to-end on something simple.

### 5.3 M68030

Additive only: on-chip PMMU instructions (`PMOVE`, `PFLUSH`, `PTEST`,
`PLOAD`, `PVALID`, `PSAVE`/`PRESTORE` variants for the PMMU rather than
FPU). Gated behind `FeatPMMU`, not a new `CPUKind`, so the same mnemonics
also become available for a `{CPU68020, FeatPMMU}` target modeling a
68020 + discrete 68851.

### 5.4 M68040 / M68060

Adds `MOVE16` (16-byte aligned block move — new EA restriction: only
`(An)` and `(An)+`), `CINV`/`CPUSH` (cache control). Removes visibility of
`cpGEN`/`cpBcc`/etc. and `CALLM`/`RTM` at these tiers (`Requires` becomes
a *range*, e.g. "68020 through 68030 only," not just a floor — `Target
.Supports` needs a ceiling check for this handful of forms, or,
simpler, model removal as its own `Feature` bit
(`FeatCoprocessorInterface`) present on 68020/68030 targets and absent on
68040/68060, keeping `Supports` a pure floor+subset check everywhere
else).

For 68060's trap-emulated subset (`CAS2`, `CHK2`/`CMP2`, `MOVEP`, dynamic
`BFxxx`, 64-bit `DIVS`/`DIVU.L`): tag those forms `FeatEmulated` and have
`selectForm` still accept them but have the CLI/API surface a **warning**
(not an error) — "assembles, but traps and is software-emulated on
68060; expect a large performance cliff." This is a judgment call to
confirm with the maintainer: strict correctness says these are legal
68060 programs, but a silent performance cliff is exactly the kind of
thing this assembler's stated philosophy (`README.md`'s "binary
precision," "full control") suggests should be surfaced, not hidden.

## 6. FPU support (68881/68882/integrated) — separate milestone

This is comparable in size to the entire existing integer ISA and should
be scoped as its own phase after the integer-tier work lands, not bundled
into "68020 support." Requirements:

- **Lexer**: floating-point literal syntax (`#1.5e10`, `#0.1`) — today
  `NUMBER` tokens are always parsed as `int64` (`lexer.go:315`); this
  needs a new token kind or a `Value any` (int64 | float64) on the
  existing token, plus IEEE-754 extended (80-bit) conversion for the
  `TrailerItem` that emits it.
- **New operand/register kinds**: `FP0`–`FP7`, `FPCR`/`FPSR`/`FPIAR`,
  new `EAExprKind` values and `OperandKind` values (`OpkFPn`, `OpkFPCR`,
  etc.) parallel to the existing `Dn`/`An`/`SR`/`CCR` handling in
  `types.go` and `parser_operand.go`.
- **Mnemonics**: `FMOVE`, `FMOVEM`, `FADD`/`FSUB`/`FMUL`/`FDIV`/
  `FSGLDIV`/`FSGLMUL`, `FCMP`/`FTST`, `FABS`/`FNEG`/`FSQRT`, `FGETEXP`/
  `FGETMAN`/`FSCALE`/`FMOD`/`FREM`, the transcendental set (`FSIN`,
  `FCOS`, `FSINCOS`, `FTAN`, `FATAN`, `FASIN`, `FACOS`, `FATANH`, `FETOX`,
  `FETOXM1`, `FLOGN`, `FLOGNP1`, `FLOG10`, `FLOG2`, `FTWOTOX`, `FTENTOX`),
  `FMOVECR` (load constant ROM), `FBcc`/`FDBcc`/`FScc`/`FTRAPcc`,
  `FSAVE`/`FRESTORE`, `FNOP`. Gate the transcendental subset behind
  `FeatFPUFull` per §3, since 68040/68060's integrated FPU traps those to
  software.
- These all encode through the `cpGEN`-style coprocessor ID field
  introduced in §5.2b, so this phase has a hard dependency on that
  landing first.

MMU (68851/on-chip) mnemonics are already covered by §5.3/§5.4 and don't
need a separate phase.

## 7. Backward compatibility

- Every public API function in `assemble_api.go` keeps its current
  signature; `ParseOptions.Target` is an additive zero-valued field
  (Go's zero value for the new `CPUKind`/`Feature` types is exactly
  "68000, no coprocessors"), so existing callers — including every
  existing test — see no behavior change.
- `docs/syntax.md` and `docs/grammar.ebnf` get an addendum, not a rewrite,
  since 68000 syntax is a strict subset of what's being added.
- The scale-factor-on-68000 gap described in §1 becomes a **breaking bug
  fix** the moment `Target` gating lands: code that today assembles
  `(0,A0,D0.W*4)` on the implicit 68000 target will start failing with
  "scale factors require 68020+." This is correct behavior but is worth
  flagging explicitly in the changelog as a fix, since strictly speaking
  it rejects input the current version accepts.

## 8. Suggested milestone order

1. ✅ **Done.** `Target`/`CPUKind`/`Feature` plumbing with **zero new
   instructions** — proves the threading in §4 end-to-end against
   something trivial (gated the pre-existing scale-factor bug behind the
   new `Target`, fixing §1's latent issue as the first patch).
2. ✅ **Done.** M68010/68012 (§5.1): `MOVEC`, `MOVES`, `RTD`, `BKPT`,
   gated behind `Requires: Target{CPU: CPU68010}` on whole `InstrDef`s.
   Encodings were cross-checked against GNU binutils' GAS m68k backend
   (`opcodes/m68k-opc.c`, `gas/config/tc-m68k.c`) rather than taken from
   memory alone, given how easy a wrong bit placement is to miss by eye.
   This also introduced the general "extension-word register field"
   pattern (`FExtRegSrc`/`FExtRegDst`, bit 15 = A/D, bits 14-12 = regnum)
   that 68020's `CAS`/`CHK2`/`CMP2` in milestone 5 will reuse.
3. ✅ **Done.** 32-bit branch displacement (§5.2c): `BRA.L`/`BSR.L`/
   `Bcc.L`, gated behind `Requires: Target{CPU: CPU68020}` on a second
   `FormDef` added to each existing branch `InstrDef` — the byte/word
   form and the new long form share one literal `Steps` value, since
   `FBranchLow8`/`TBranchWordIfNeeded`/`TBranchLongIfNeeded` all key off
   which of `BrUseWord`/`BrUseLong` `Encode` resolved from `Args.Size`,
   not from which form matched.
4. ✅ **Done, with narrower scope than originally proposed.** 68020
   memory-indirect addressing (§5.2a): `([bd,An],Xn,od)`/`([bd,An,Xn],od)`
   and their PC-relative equivalents, cross-checked bit-for-bit against
   GNU binutils' `gas/config/tc-m68k.c` (the `POST`/`PRE`/`BASE` case).
   Three things from the original §5.2a description were deliberately
   **not** built, and are worth a second look before anyone assumes they
   exist:
   - **Base-register suppression** (an EA with no `An`/`PC` at all) isn't
     supported; every memory-indirect EA here has a real base register.
   - **`bd` must always be written explicitly** inside the brackets
     (`([0,An],...)`, not a `([An],...)` shorthand for zero) — this
     sidesteps a real parsing ambiguity (is the first bracketed token a
     displacement or the base register?) that the existing brief-EA
     parser resolves with a two-token lookahead heuristic; extending that
     heuristic into the bracket grammar was cut for time, not because
     it's hard in principle.
   - **The classic brief form `(d,An,Xn)`/`(d,PC,Xn)` was deliberately
     *not* extended** to auto-upgrade to full-format when `d` overflows a
     signed byte. Doing that would make the EA's *encoded length* depend
     on whether a value fits — safe for a compile-time constant, but a
     real two-pass hazard for a forward-referenced label (pass 1 could
     guess "brief," mis-sizing every address after it, before pass 2
     discovers the label's real value needs "full"). This is also why
     `bd`/`od` in the new bracket syntax require an explicit `.w`/`.l`
     suffix whenever they reference a symbol — the one existing
     mechanism for this (a label's own `.w`/`.l` suffix, e.g. `label.l`)
     turned out to already be silently broken for compound expressions,
     since the lexer folds a trailing dot into the identifier token
     itself (the same reason `stripRegSizeSuffix` exists for registers)
     rather than emitting a separate token `parseExprInfoUntil`'s
     existing dot-handling could see — `parseSizedDisp` (parser_ea.go)
     recovers it directly from the token text instead of trying to fix
     that shared code path. Net effect: a bare `label`/`label.w`/`label.l`
     works as `bd`/`od`, but `label+4` does not — only a full future pass
     at the expression evaluator itself would generalize that safely.
   - One incidental but real fix landed alongside this: the classic brief
     indexed form's base displacement was **silently truncated** to a
     signed byte with no range check at all (`ix.Disp8 = int8(disp)`,
     unconditionally) prior to this milestone — a latent correctness bug
     on every CPU tier, not just pre-68020 ones. It's now a clear parse
     error instead.
5. ✅ **Partly done.** §5.2b's mnemonics turned out to split cleanly into
   a "plain integer ISA" half and a "needs new operand-kind/lexer
   machinery" half; only the first is done:
   - **Done:** `EXTB.L`, `CHK2`/`CMP2` (all three sizes each), and
     `TRAPcc` (all 16 conditions × bare/`.w`/`.l` forms — 48 form
     combinations from one small per-condition loop, the same shape as
     `xcc.go`'s existing `Scc`/`DBcc` loops). `CHK2`/`CMP2` reuse the
     `FExtRegSrc`/`FExtRegDst` register-specifier field milestone 2
     introduced for `MOVEC`/`MOVES`, confirming that field is the
     general-purpose "register nibble in an extension word" primitive
     the original design note predicted it would become.
   - **Still open:** bit-field ops (`BFTST`/…/`BFINS` — need a new
     operand kind plus lexer support for `{offset:width}` syntax),
     `CAS`/`CAS2`, `PACK`/`UNPK`, `DIVSL`/`DIVUL` (its 64-bit-dividend
     `Dr:Dq` register-pair syntax also needs a new operand kind), and
     `CALLM`/`RTM` (low priority — obscure even in real 68020-era code).
   - **Bug found and fixed along the way:** `selectForm` (assemble.go)
     skipped its operand-kind check entirely whenever a form declared
     zero `OperKinds`, on the assumption that only truly-operand-less
     instructions (`NOP`, `RTS`, …) ever do that. `TRAPcc` was the first
     case where a zero-operand form and a same-size operand-bearing form
     share one `InstrDef` — `TRAPT` and `TRAPT.W #imm` are both
     word-sized — so the bare form matched first and silently swallowed
     the immediate. Fixed by always calling `operKindsMatch` (which
     already handles the empty case correctly on its own); every
     existing test still passes since a truly zero-operand `InstrDef`'s
     single form still matches an empty actual-kinds list either way.
   Then 68030 (§5.3), then 68040/68060 (§5.4).
6. ✅ **Partly done.** §6's FPU milestone, scoped down sharply to avoid
   its two hardest sub-problems for now:
   - **Done:** `FMOVE`, `FADD`, `FSUB`, `FMUL`, `FDIV`, `FCMP`, `FABS`,
     `FNEG`, `FSQRT`, `FTST`, `FNOP`, all with `.b`/`.w`/`.l`/`.s`/`.d`/`.x`
     sizes, register and EA (including integer immediate) operands, and
     the single-operand shorthand for the monadic ops (`FABS FPn`). Every
     bit position was decoded from GAS's `opcodes/m68k-opc.c` — the
     format-code field (bits 12-10) was reverse-engineered by comparing
     the literal word2 values across `faddb/w/l/s/d/x/p`, which was the
     only way to pin it down since the mask alone doesn't say what a
     variable field *means*.
   - **Not done, deliberately:** floating-point immediate literals
     (`#1.5`) — this needed the lexer changes §6 originally called for,
     and was cut rather than rushed; only integer immediates work today.
     Also still open: `FMOVEM`, `FMOVECR`, `FBcc`/`FDBcc`/`FScc`/
     `FTRAPcc` (their own 32-entry condition space), the transcendental
     functions, `FSAVE`/`FRESTORE`, packed BCD, and `FPCR`/`FPSR`/`FPIAR`.
   - **Gating:** `Requires: Target{Features: FeatFPU}`, not a CPU floor —
     the first real use of the `Feature` axis from milestone 1's design,
     confirming the CPU-tier/coprocessor split was worth keeping separate.
   - **Two more bugs found and fixed** in the course of getting FNOP and
     immediate/displacement operands byte-exact:
     1. `haveWord` (encode.go *and* parser_stmt.go — they must agree)
        treated a step with `WordBits==0` and no `Fields` as "no word
        here," which is right for a step that exists only to carry a
        `Trailer` but wrong for FNOP's second word, a genuinely fixed
        `0x0000`. It silently dropped that word entirely until the
        heuristic learned to tell "trailer-only" apart from "fixed zero."
     2. The first draft of every FPU form's `Steps` put the `Trailer`
        (the EA's own extension words — a displacement or an immediate
        value) *before* the opmode/format/register word, instead of
        after. `FADD.L #100,FP0` encoded as `F03C 0000 0064 0022` —
        opcode, then the immediate, then the opmode word — instead of the
        correct `F03C 0022 0000 0064`. Caught by hand-checking a case
        with a real displacement, not just the all-zero cases that had
        already passed.
7. ✅ **Done, modulo one deferred instruction family.** CPU32 (§2's
   flagged tier and §3's core-model prediction), verified against GNU
   binutils' `gas/config/tc-m68k.c` and `opcodes/m68k-opc.c` rather than
   the CPU32RM (not on hand), by grepping every mnemonic and
   architecture check GAS tags `cpu32`:
   - **`Target.Supports` gained the mechanism §3 predicted it would
     need**, but simpler than expected: most of "CPU32 shares 68020's
     integer additions" needs *no* special-casing at all — retagging
     `BRA.L`/`BSR.L`, `CHK2`/`CMP2`, `EXTB.L`, `TRAPcc`, and scale-factor
     addressing from `Target{CPU: CPU68020}` to `Target{CPU: CPU32}`
     is enough, since `Supports`'s plain `t.CPU >= req.CPU` comparison
     then grants them to CPU32 *and* every 68020+ tier from that one
     tag (CPU32's enum value sits below CPU68020). The one relationship
     plain ordering genuinely cannot express is a form CPU32 has that
     *no* higher tier does (68020's enum value is numerically greater,
     so a floor check alone would wrongly grant it there too) — that
     needed the one real addition: an unexported `featCPU32Only`
     `Feature` bit that `Supports` grants implicitly to any `CPU32`
     target (never something a caller sets themselves), paired with an
     exported `requireCPU32Only` `Target` value for forms to use it.
   - **Confirmed CPU32 does *not* get**, and left correctly gated on
     `Target{CPU: CPU68020}` exactly: 68020 memory-indirect addressing
     (`gas/config/tc-m68k.c` explicitly rejects it on cpu32 — "needs
     68020 or higher" — even though CPU32 has plain scale-factor
     addressing), and, unchanged from before, bit-field ops, `CAS`/
     `CAS2`, `PACK`/`UNPK`, `DIVSL`/`DIVUL`, `CALLM`/`RTM`, and the FPU/
     coprocessor interface.
   - **Added `BGND`** (enter background debug mode) as the proof case
     for `requireCPU32Only`: a real, simple, single-word CPU32-exclusive
     instruction, confirmed to assemble on `cpu32` and be rejected on
     every other tier including 68020+.
   - **Deliberately not implemented:** `TBLS`/`TBLU`/`TBLSN`/`TBLUN`
     (CPU32's table-lookup-and-interpolate instructions). Their GAS
     table entries use a macro-generated encoding with an unclear
     third register operand (`"DsD3D1"`) whose real-world syntax needs
     more research than could be responsibly rushed alongside the rest
     of this milestone — a good target for a small follow-up.
8. ✅ **Done, modulo `CAS2`.** Milestone 8 (chosen by the maintainer from
   the open items list after milestone 7): `PACK`, `UNPK`, `CAS`,
   `CALLM`, and `RTM` — the remaining §5.2b instructions milestone 5 had
   scoped out.
   - **The real work here was architectural, not encoding lookup:**
     `Args` had only ever needed a Src/Dst pair, but `CAS Dc,Du,<ea>` and
     `PACK Ds,Dd,#adjustment` both have three operands. Rather than
     generalize `tryParseForm`'s loop to arbitrary N (unbounded scope for
     a change touching every instruction in the codebase), `Args` gained
     one more field, `Aux EAExpr`, and the loop's `if i == 1` comma/
     assignment logic became `if i > 0` / a three-way `switch i` — the
     minimum extension that unblocks exactly the third-operand case that
     exists, not a general N-ary redesign. `operandKinds` (assemble.go),
     `Encode`/`applyField`/`emitTrailer` (encode.go), and
     `instructionWords` (parser_stmt.go) all needed the matching new
     `FAuxEA`/`FDstReg6`/`TAuxEAExt`/`TAuxImmWord` cases — the same
     "encode.go and parser_stmt.go must agree" discipline milestone 6's
     `haveWord` bug established.
   - **A second, smaller gating wrinkle, found by actually checking
     rather than assuming:** `CALLM`/`RTM` looked at first glance like
     ordinary 68020 additions (`Target{CPU: CPU68020}`, kept on every
     later tier, same as everything else in milestone 5). Checking GNU
     binutils' `include/opcode/m68k.h` instead of assuming showed they
     are tagged the single `m68020` bit (`0x004`), not the `m68020up`
     union (`m68020|m68030up|...`) every other 68020 addition uses in
     `opcodes/m68k-opc.c` — i.e. real 68030+ silicon dropped them. This
     is the exact same shape of problem milestone 7's `requireCPU32Only`
     solved (a tier-exclusive form a plain CPU-floor comparison can't
     express, since the excluded tiers have numerically *higher* CPU
     values), so `Supports` gained the general form of that mechanism:
     a `case CPU68020:` branch alongside the existing CPU32 one,
     granting an implicit `featM68020Only` bit, paired with an exported
     `requireM68020Only`. `PACK`/`UNPK`/`CAS` are tagged the `m68020up`
     union in GAS, so they correctly use the ordinary `Target{CPU:
     CPU68020}` and are kept through 68030+, unlike `CALLM`/`RTM`.
   - **`CALLM`'s word order was cross-checked, not guessed:** its
     `#vector` immediate and its `<ea>`'s own extension words are two
     independent trailers with no `two()` template to pin their
     relative order, so GAS's `insop()` (gas/config/tc-m68k.c) was
     checked directly — it explicitly inserts "before general operands,"
     confirming `[opcode][immediate][EA extension]` regardless of the
     mnemonic's own `"#b!s"` argument-string order.
   - **`CAS2` stays deliberately unimplemented**: six operands (two
     compare registers, two update registers, two memory pointers) is
     beyond what even the new `Aux` mechanism reaches, and generalizing
     further for one rare instruction isn't justified by this milestone
     alone.
9. ✅ **Done.** `BFxxx` bit-field ops (chosen by the maintainer from the
   open items list after milestone 8): `BFTST`, `BFCHG`, `BFCLR`,
   `BFSET`, `BFEXTU`, `BFEXTS`, `BFFFO`, `BFINS`.
   - **The `{offset:width}` syntax needed real new lexer/parser
     surface**, not just new instructions: `{`/`}` tokens (new, added
     alongside milestone 4's `[`/`]`), and a bit-field suffix parsed once
     in `parseEA` (renamed to wrap a new `parseEABase`) rather than
     duplicated across every one of `parseEABase`'s sub-paths — the
     suffix is legal on *any* EA (`D0{0:8}`, `(A0){0:8}`, `(100,A0){0:8}`
     all needed to work), so checking for a trailing `{` once after the
     base EA parses, regardless of which sub-path produced it, was the
     natural fit. Offset/width syntax has no `#` prefix, unlike every
     other immediate in this codebase — confirmed against GAS's own `'O'`
     argument case in `gas/config/tc-m68k.c`, which parses a bare
     expression or a bare `Dn`, matching real 68020 assembler convention
     for this one construct.
   - **The bit-field spec is a suffix on the EA, not a new operand or
     `EAExprKind`**: `EAExpr` gained `HasBitField`/`BFOffsetIsReg`/
     `BFOffsetVal`/`BFWidthIsReg`/`BFWidthVal` fields describing the
     field attached to whatever base addressing mode (`Kind`/`Reg`/etc.)
     was already there — reusing the base's existing classification
     (`EAkDn`, `memoryAlterableEA`, …) for validation rather than
     inventing a parallel one.
   - **The offset/width encoding was decoded from GAS's `'O'` case, not
     reconstructed from a half-remembered bit diagram**: both fields
     share one combined 6-bit format (bit 5 = "is a register," bits 4-0
     = the register number or the literal value) — `bitFieldSpecCode`
     (encode.go) — which is *simpler* than the "separate flag bit plus
     separate value field" layout a plain reading of the Motorola manual
     diagram suggests, and would have been easy to get subtly wrong by
     reconstructing it from memory instead of reading GAS's actual
     arithmetic. `BFEXTU`/`BFEXTS`/`BFFFO`'s result register and
     `BFINS`'s source register reuse milestone 2's `FExtRegSrc`/
     `FExtRegDst` nibble fields unchanged — a Dn-only register in that
     4-bit slot always has its A/D bit clear, so nothing new was needed
     there either.
   - **GAS masks a width of 32 down to 0 and an out-of-range offset
     silently (`value & 0x1F`), rather than rejecting either** — this
     codebase instead validates offset (0-31) and width (1-32) explicitly
     at parse time and errors on anything outside that range, consistent
     with every other place this project has chosen to reject silent
     truncation over matching GAS's permissiveness (milestone 4's brief-
     index-displacement fix being the first).
10. ✅ **Done.** `DIVSL`/`DIVUL` (chosen by the maintainer from the open
    items list after milestone 9): the 68020 64-bit-dividend divide
    forms, `<ea>,Dr:Dq` (or the `<ea>,Dq` shorthand).
    - **The hardest part was figuring out which physical bit position —
      14-12 or 2-0 — is the quotient register (Dq) versus the remainder
      register (Dr), which the opcode table's mask alone can't say** (it
      only says "two 3-bit fields live here," not which is which). GAS's
      disassembler was checked directly (`opcodes/m68k-dis.c`'s `case
      '1'`/`case '3'` in `print_insn_arg`, alongside `install_operand`'s
      matching cases) to confirm bits 14-12 = Dq and bits 2-0 = Dr — the
      kind of detail a plain reading of the manual's bit diagram would
      be easy to get backwards on without cross-checking against working
      code.
    - **The GAS source didn't settle the *source's* colon-vs-comma
      question, though**: its disassembler's generic operand-printing
      loop inserts a plain comma between every argument with no special
      case for adjacent register args, suggesting GAS itself may accept
      `<ea>,Dr,Dq` (comma-separated) rather than the `Dr:Dq` colon
      notation Motorola's manual documents. Rather than guess which one
      real-world source expects, this codebase commits to the
      manual's colon notation as its own explicit, documented syntax
      choice — consistent with milestone 9's bit-field `{offset:width}`
      also not chasing exact GAS syntax compatibility for the same
      reason (a rare, secondary construct where "make a documented
      choice" beats "guess at compatibility").
    - **New parsing surface, kept small**: one `OpkRegPair`/`EAkRegPair`
      pair and a dedicated `parseRegPair` (parser_ea.go) — parsing "a
      bare `Dn`, optionally followed by `:` and another `Dn`" — rather
      than reusing or extending milestone 8's `Aux` three-operand
      mechanism, since `Dr:Dq` is semantically *one* operand (colon-
      joined) from the parser's perspective, not a third comma-separated
      one.
    - **Gated on `Target{CPU: CPU32}`, not `Target{CPU: CPU68020}`**:
      confirmed via the same `m68020up|cpu32` tag milestone 7 first
      found on `Bcc.L`/`CHK2`/`EXTB`/`TRAPcc`, reusing that established
      pattern rather than re-deriving it.
11. ✅ **Done.** `CAS2` (chosen by the maintainer from the open items
    list after milestone 10) — the dual-location compare-and-swap
    milestone 8 deferred for needing more than the two-operand Src/Dst
    shape.
    - **It turned out not to need a bigger operand system after all.**
      CAS2's six registers (two compare, two update, two memory
      pointers) are written as exactly three colon-joined pairs —
      `Dc1:Dc2,Du1:Du2,(Rn1):(Rn2)` — which is three operands from the
      parser's point of view, fitting the *existing* Src/Dst/Aux slots
      milestone 8 introduced without any further generalization.
      Milestone 8's own note calling `CAS2` "beyond what even the new
      `Aux` mechanism reaches" undersold it slightly: the mechanism
      reaches fine once the six registers are recognized as three pairs
      rather than six independent operands.
    - **A comment in GAS's own source, not just its bit tables, settled
      the word count**: `install_operand`'s `case '6'` carries the
      comment "DANGER! This is a hack to force cas2l and cas2w cmds to
      be three words long" — direct confirmation, not inference, that
      `CAS2` has no classic `<ea>` at all (its first word's mask is
      `0xFFFF`, meaning literally no bits vary) and instead encodes both
      memory locations as bare register pointers across two more words.
    - **Reused `OpkRegPair` for the compare/update pairs rather than
      inventing a "mandatory-colon" sibling kind**: `CAS2`'s `Validate`
      just checks `RegPairWide` itself and errors clearly if a pair's
      colon was omitted, instead of adding a second `OperandKind` that
      would have needed its own entry in `operandKindByEA` and risked
      a form-matching mismatch against the existing `OpkRegPair`
      mapping. Only the pointer pair — a shape (`(An):(An)`) nothing
      else in this codebase parses — got a genuinely new
      `OpkAnIndPair`.
    - **Confirmed CAS2 is not on CPU32**, unlike milestone 7's `Bcc.L`/
      `CHK2`/`EXTB`/`TRAPcc` and milestone 10's `DIVSL`/`DIVUL`: GAS
      tags it plain `m68020up`, without `cpu32`, even though its
      single-location sibling `CAS` (milestone 8) shares that same
      plain tag too — so this one was never a special case, just a
      regression-test worth locking in given how often the
      cpu32-sharing question *has* gone the other way in this project.
12. ✅ **Done, with narrower scope than originally proposed (twice).**
    68030/68851 PMMU support (§5.3), chosen by the maintainer from the
    open items list after milestone 11. Landed as `PMOVE.L <ea>,TC` /
    `PMOVE.L TC,<ea>` and `PFLUSHA` only.
    - **The full PMMU family turned out to be by far the largest scope
      estimate of any milestone in this project**, closer in size to
      the entire 68020 memory-indirect-addressing milestone (milestone
      4) than to a typical single-instruction milestone. GAS's opcode
      table showed: a full `Pcc` family (`PBcc`/`PDBcc`/`PScc`/`PTRAPcc`)
      mirroring the integer `Bcc`/`DBcc`/`Scc`/`TRAPcc` families but for
      MMU status-register condition codes; `PFLUSH` alone with six
      different operand shapes (flush by EA, by FC, by FC+mask, all-but-
      current, …); and `PMOVE` covering half a dozen distinct MMU
      registers (`TC`, `CRP`, `SRP`, `TT0`, `TT1`, `MMUSR`, …), each
      with its *own* `install_operand` case letter and register-selector
      numbering scheme in `gas/config/tc-m68k.c` rather than one shared
      encoding. This was flagged to the maintainer before writing any
      code, rather than either silently ballooning the milestone or
      silently shipping something narrower than "68030 PMMU support"
      implies.
    - **First scope cut, made with the maintainer:** build only `PMOVE`
      and `PFLUSHA`, deferring every `Pcc` variant and every non-`PMOVE`
      `PFLUSH` shape entirely. Reasoning: `PMOVE` (loading/storing MMU
      control registers) and `PFLUSHA` (flush the entire ATC) are the
      two operations any basic MMU-setup code sequence actually needs;
      the `Pcc` conditionals and `PFLUSH`'s finer-grained variants are
      real but distinctly secondary, used by more sophisticated
      demand-paging logic this assembler has no other support for yet
      anyway (no OS/kernel-level examples or tests exist in this repo).
    - **Second scope cut, made unilaterally while implementing (not
      re-asking, since it's strictly inside the already-agreed
      boundary):** of `PMOVE`'s half-dozen register forms, implement
      only `TC` (Translation Control). Reasoning: `TC` is the single
      register that actually enables/configures address translation —
      the "turn the MMU on" register — while `CRP`/`SRP` (root pointers)
      and `TT0`/`TT1` (transparent translation) are refinements on top
      of a working `TC` setup, and `MMUSR` is a status/read-only
      register more naturally paired with the still-deferred `Pcc`
      conditionals. Confirmed via `tc-m68k.c`: TC's `install_operand`
      case (`'0'`) is the *only* one of the family gated with an
      explicit `if (opP->reg != TC) losing++;` validation, i.e. GAS
      itself treats TC as a case worth guarding on its own rather than
      folding into the shared multi-register cases (`'1'`/`'2'` for
      `CAL`/`VAL`/`SCC`/`AC`, `'W'` for `CRP`/`SRP`/`DRP`) — a small
      but real signal that TC is the more foundational, standalone one.
    - **Both `PMOVE` directions and `PFLUSHA` needed no new `FieldRef`
      or `TrailerItem`**: every one of these instructions' second words
      is a fully-fixed literal (`0x4000` for load, `0x4200` for store,
      `0x2400` for `PFLUSHA`) once TC's own register-selector value —
      confirmed as `0` by the same `tc-m68k.c` `install_operand` code
      that validates it — is folded in, contributing no variable bits.
      The `<ea>` word reuses the existing `FSrcEA`/`FDstEA`/`TSrcEAExt`/
      `TDstEAExt` machinery unchanged, same as every other single-`<ea>`
      instruction in this codebase.
    - **New `OpkTC`/`EAkTC` operand kind, following the established
      SR/CCR/USP pattern exactly**: `TC` is a fixed, unnumbered special
      register (not one of several numbered instances, unlike `Dn`/
      `An`/`FPn`), so it reuses the existing generic
      `parseExpectedSpecialRegister` helper rather than needing new
      parsing logic, plus the usual `eaTable` dummy entry (`Encode`
      unconditionally calls `EncodeEA` for any non-`EAkNone` operand)
      and `operandKindByEA` mapping.
    - **Gated purely on `Target{Features: FeatPMMU}`** (the new `--mmu`
      CLI flag), independent of `CPUKind` — matching milestone 6's FPU
      gating pattern rather than a CPU-tier `Requires`, since a bare
      68020 with an external 68851 and a 68030's on-chip PMMU both just
      need the feature bit set, not a specific integer-unit tier.
    - **Every other PMMU form** (`Pcc`, `PFLUSH`'s non-`PMOVE`-adjacent
      variants, and `PMOVE`'s `CRP`/`SRP`/`TT0`/`TT1`/`MMUSR` forms)
      remains explicitly open for a future, separately-scoped milestone.
13. ✅ **Done, plus an unplanned but necessary detour: a real encoding
    bug found and fixed in already-shipped milestone-6 code.** More FPU
    coverage (§6), chosen by the maintainer from the open items list
    after milestone 12: `FBcc`/`FDBcc`/`FScc`/`FTRAPcc` (the FPU's own
    32-condition branch/set/trap family), `FMOVECR` (ROM constant load),
    and `FSAVE`/`FRESTORE` (state-frame save/restore).
    - **The detour, found first, before any new code was written:**
      researching `FBcc`'s exact word1 layout required reading
      `tc-m68k.c`'s handling of the `'I'` argument-type letter, which
      every FPU "general instruction" mnemonic's args string starts
      with. That code (`m68k_ip`'s "fake a first entry of type COP#1"
      comment) showed GAS always synthesizes an *implicit* coprocessor-
      ID operand — value 1 (`COP1`) — for every plain float mnemonic,
      installed via `install_operand`'s case `'i'`/`'d'` as `opcode[0]
      |= val << 9`, i.e. bits 11-9 of word1 must be `001` (`0x0200`).
      Milestone 6's shipped code used a bare `0xF000` base for every
      EA-carrying FPU word1 (`FMOVE`/`FADD`/`FSUB`/`FMUL`/`FDIV`/`FCMP`/
      `FABS`/`FNEG`/`FSQRT`/`FTST`) — cpid 0, not 1. Two independent
      pieces of evidence confirmed this was wrong, not just a stylistic
      difference: GAS's disassembler (`m68k-dis.c`) only prints
      `"(cpid=N)"` when N isn't 1, treating 1 as the silent expected
      default; and `FNOP`'s own opcode table entry (`0xF280`, a fully
      fixed literal already copied verbatim into this codebase, so
      untouched by the bug) already has that same `0x0200` bit set,
      which is what first exposed the discrepancy against the EA-based
      forms' `0xF000`. Fixed by introducing a named `fpuWord1Base =
      0xF200` constant (`cpu020_fpu.go`) and re-deriving every affected
      test's expected bytes (`cpu020_fpu_test.go`) — GAS-standard tools
      reading output from before this fix would flag it as using a
      non-default, unusual coprocessor ID.
    - **`FBcc`'s two forms (word/long displacement) needed no shared-
      Steps trick** the way integer `Bcc.L`/`BSR.L` (milestone 3) did:
      FPU branches never inline an 8-bit displacement in the opcode
      (the condition already occupies those bits), so word1 itself
      differs per size (`0xF280|cc` vs `0xF2C0|cc`) and each form just
      carries its own single, size-matched trailer
      (`TBranchWordIfNeeded`/`TBranchLongIfNeeded`) — simpler than the
      integer case, not harder. GAS spells the long form as a
      completely separate mnemonic (`fbeql`, not `fbeq.l`); this
      codebase again commits to its own `.L`-suffix convention instead
      (as milestone 3 already did for `BRA.L`/`BSR.L`), consistent
      rather than chasing GAS's exact spelling.
    - **One condition-suffix table serves all four families.** Unlike
      the integer ISA's `branchConditions`/`dbConditions`/
      `sccConditions` (`xcc.go`), which duplicate 16 near-identical
      names three times because each integer family happens to use
      slightly different mnemonic conventions, the FPU's 32 condition
      names are spelled identically across `FBcc`/`FDBcc`/`FScc`/
      `FTRAPcc` — only the family prefix (`FB`/`FDB`/`FS`/`FTRAP`)
      changes. One `fpConditions` table plus prefix concatenation in
      the registration loop avoids retyping 32 names four times without
      losing the "index == condition code" property the integer tables
      rely on.
    - **`FSAVE`/`FRESTORE` don't need two `Form`s per mnemonic, even
      though GAS's own opcode table lists two rows each** (a general
      memory form and one restricted to the specific addressing mode
      real hardware needs — `-(An)` for `FSAVE`, `(An)+` for
      `FRESTORE`). Working out the two literals by hand (`0xF100`
      general / `0xF120` predecrement for `FSAVE`) showed the
      "dedicated" literal is exactly the general literal with that
      EA's own mode bits already OR'd in — precisely what `FSrcEA`
      computes at encode time for *any* EA, predecrement/postincrement
      included. So a single `Form` with the general literal, plus a
      `Validate` accepting the union of both GAS rows' restrictions,
      reproduces GAS's output exactly (confirmed: assembling `FSAVE
      -(A0)` through this one `Form` produces `0xF120`, GAS's own
      "dedicated" literal, byte-for-byte). Two separate `Form`s would
      have been actively wrong here, not just redundant: `selectForm`
      matches on `OperKinds`/`Sizes` alone before `Validate` ever runs,
      and both forms would share identical `OperKinds` (`[OpkEA]`) and
      `Sizes` — so whichever form came first would silently swallow
      every operand, valid or not, exactly the milestone-5 hazard
      `selectForm`'s own comment already warns about.
    - **Deliberately deferred again: `FMOVEM`.** Unlike everything else
      in this milestone, `FMOVEM` needs its own register-list
      infrastructure comparable in size to a new milestone on its own —
      a list syntax for `FPn`/`FPCR`/`FPSR`/`FPIAR` distinct from
      integer `MOVEM`'s `Dn`/`An` mask, a genuinely *dynamic* list form
      (the register set named by a `Dn` at runtime, not fixed at
      assembly time), and three separate EA-class/bit-pattern splits
      (control-alterable, predecrement, postincrement) each with their
      own static-vs-dynamic-list bit and register-order rules. Also
      still open: the transcendental function set, packed BCD (`.p`),
      and `FPCR`/`FPSR`/`FPIAR` as general operands.
14. ✅ **Done, plus a second unrelated pre-existing bug found and
    fixed.** `FMOVEM` (§6), chosen by the maintainer from the open items
    list after milestone 13 — the FPn (FP0-FP7) register-list save/
    restore deferred twice already. Scoped to the FP0-FP7 data-register
    list only, both static and dynamic, both directions; the separate
    `FPCR`/`FPSR`/`FPIAR` control-register list form remains deferred
    (see below).
    - **The unrelated bug, found first, before any new code was
      written:** working out FMOVEM's load/store EA restrictions meant
      re-reading integer `MOVEM`'s own `movemLoadEA` map
      (`validate.go`) for comparison, which turned up a real,
      pre-existing (predates every milestone in this document —
      confirmed via `git show` against the commit before milestone 1)
      bug: the map's own doc comment says predecrement is not a valid
      MOVEM load source, but `EAkAddrPredec: true` was in the map
      anyway, silently letting `MOVEM -(A0),D0-D7` assemble — a bit
      pattern real 68k hardware doesn't support for the load direction
      (predecrement is store-only, postincrement load-only). No test
      covered this case. Fixed in its own commit (one map entry plus a
      regression test) before starting `FMOVEM` itself, since `FMOVEM`
      needed the *correct* version of this exact restriction anyway.
    - **The central design problem, solved by extending a pattern
      milestone 13 had already established for FSAVE/FRESTORE:**
      GAS's opcode table gives FMOVEM's *load* direction a general-
      memory row and a postincrement row whose word2 values are
      identical (`0xD000` general, `0xD000` postincrement) — the same
      EA-artifact equivalence FSAVE/FRESTORE already rely on, so one
      `Form` covers both. But the *store* direction's general
      (`0xF000`) and predecrement (`0xE000`) rows are NOT equivalent —
      word2 differs by a genuine "direction/mode" bit, not something
      `FSrcEA`/`FDstEA` computes from the EA alone. Two `Form`s sharing
      identical `OperKinds` (`[OpkFPRegList, OpkEA]`) would have hit
      the exact `selectForm` hazard documented for FSAVE/FRESTORE: the
      first `Form` always wins before `Validate` runs, so the second
      would be unreachable — and, worse, a naive single-`Form`-with-
      one-literal design (which is what this file's first draft
      shipped, before being caught in review before any commit) would
      have *silently mis-encoded* predecrement stores with the general
      form's word2, rather than merely rejecting them. The fix: one
      `Form`, whose word2 is computed entirely from the Dst EA's
      runtime mode inside a single `FieldRef`
      (`FFPMovemStoreWord2`/`FFPMovemDynStoreWord2`, `encode.go`) —
      which turns out to be exactly what integer `MOVEM`'s own
      `TSrcRegMask` trailer already does for this identical
      general-vs-predecrement split (`if p.DstEA.Mode == 4 { reverse
      the mask }`), just applied to a whole word2 value instead of a
      trailing mask word. Confirmed byte-for-byte against the real CLI:
      `FMOVEM.X FP0-FP3,-(A7)` and `FMOVEM.X FP0-FP3,(A0)` produce
      `0xE0F0` and `0xF00F` respectively, both matching hand-derived
      GAS output.
    - **List bit order for predecrement matches integer `MOVEM`
      exactly**: the register list is bit-reversed only for the
      predecrement store direction (`reverse16(mask&0xFF)>>8`, reusing
      `encode.go`'s existing 16-bit `reverse16` rather than writing a
      dedicated 8-bit reversal), for the same real-hardware reason
      integer `MOVEM` already reverses its own list there.
    - **A new, separate `FPRegMaskSrc`/`FPRegMaskDst` pair on `Args`**,
      not a reuse of `RegMaskSrc`/`RegMaskDst`: both are plain `uint16`
      bitmasks, but integer `MOVEM`'s `Dn`/`An` list and `FMOVEM`'s FPn
      list are never used on the same instruction, and sharing one
      field would make `operandKinds` (`assemble.go`) unable to tell
      which kind of list it was looking at when choosing an
      `OperandKind`. A new `OpkFPRegList` plus a small `parseFPRegList`
      (mirroring `parseRegList`'s range/slash/comma syntax over the
      single FP0-FP7 namespace) was cheaper than generalizing the
      existing machinery to carry a register-class tag.
    - **Still deferred: the `FPCR`/`FPSR`/`FPIAR` control-register list
      form.** GAS encodes it with a real bit-width/position difference
      (a 3-bit selector at bits 12-10 of word2, not the 8-bit FPn mask
      at bits 7-0) and its own type-check rules (it allows a bare `Dn`/
      `An` destination, which the FPn form's EA restriction correctly
      rejects) — a second, smaller register-list subsystem in its own
      right, not a natural extension of what's built here. Given how
      rarely real code saves/restores the FPU's control registers
      compared to its data registers, this is left for a dedicated
      follow-up rather than folding it in here.
15. ✅ **Done.** 68040/68060 support (§5.4), chosen by the maintainer
    from the open items list after milestone 14: `MOVE16`, the
    cache-control instructions (`CINVA`/`CINVL`/`CINVP`, `CPUSHA`/
    `CPUSHL`/`CPUSHP`), and a non-fatal warning for 68060's
    trap-emulated subset.
    - **A critical `operandKindByEA` gap, caught before it shipped:**
      giving `EAkAddrPostinc` its own `OpkPostincAn` (needed so
      `MOVE16`'s register-to-register form, `"(An)+,(An)+"`, is
      distinguishable from its other forms at the `OperandKind` level —
      see below) meant postincrement operands stopped falling through
      to the generic `OpkEA` bucket. `operandKindCompatible`'s "a form
      expecting `OpkEA` also accepts `OpkPredecAn`" special case already
      existed for `-(An)`, but nothing analogous existed for `(An)+` —
      without adding `OpkPostincAn` to that same list, *every* existing
      instruction taking a generic `<ea>` operand would have silently
      stopped accepting postincrement addressing the moment the new
      mapping landed. Caught immediately (before running any tests) by
      re-reading `operandKindCompatible` right after adding the mapping,
      rather than after a regression surfaced; the full suite was run
      immediately afterward specifically because this class of change
      (touching the shared classification map every instruction's form
      matching goes through) is exactly the kind of edit where "it
      compiled" says nothing about correctness.
    - **`MOVE16`'s "`(An)`,absolute-long" pairing repeats the exact
      `selectForm` hazard `FFPMovemStoreWord2`/milestone 14 already
      named**, in a new shape: neither a bare `(An)` nor an absolute
      address has its own `OperandKind` (unlike `-(An)`/`(An)+`), so
      both directions — `"(An),$1000"` and `"$1000,(An)"` — classify as
      identical `[OpkEA, OpkEA]`. One `Form`
      (`FMove16AbsForm`/`types.go`) picks word1 (`0xF610` vs `0xF618`)
      from which operand is actually `(An)` at *encode* time, the same
      fix already used for `FMOVEM`'s store direction. It additionally
      includes *both* `TSrcEAExt` and `TDstEAExt` unconditionally rather
      than choosing one: `(An)` always contributes zero extension bytes
      and the absolute-long side always contributes its four, so both
      trailers firing is correct regardless of which side is which —
      no direction-conditional Step needed for that part, only for the
      word.
    - **The 68060 trap-emulated-subset warning is a per-`Args` runtime
      check (`checkEmulatedOn68060`, `internal/asm/emulated060.go`), not
      wired into `FeatEmulated`'s `Requires`/`Supports` gating at all.**
      `CAS2`/`CHK2`/`CMP2`/`MOVEP` are unconditionally emulated, but the
      68020 bit-field instructions and `DIVSL`/`DIVUL` are emulated only
      for specific *operand shapes* — a register-specified bit-field
      offset/width, or the wide `Dr:Dq` dividend form — that the *same*
      `Form` handles both natively and emulated for. A static per-`Form`
      flag genuinely cannot express that distinction; only inspecting
      the resolved `Args` after a successful `Encode` can. `Program`
      gained an additive `Warnings []string` field (populated by
      `assemble()`, printed by the CLI to stderr as `warning: ...`)
      rather than changing any existing function's return signature —
      assembly still succeeds and produces identical bytes; the warning
      is purely informational, matching §5.4's original framing of this
      as "assembles correctly but traps" rather than an error.
    - **`Target{CPU: CPU68040}` alone was enough to grant 68060 too**,
      via `Supports`' existing floor comparison (`CPU68060` sits above
      `CPU68040` in `CPUKind`'s enum order) — no new gating mechanism
      needed, confirmed against GNU binutils' own `m68040up` tag
      (`m68040 | m68060`, `include/opcode/m68k.h`).
16. ✅ **Done, with a verification caveat unlike any other milestone.**
    CPU32's `TBLS`/`TBLSN`/`TBLU`/`TBLUN` (table-lookup-and-interpolate),
    open since milestone 7, chosen by the maintainer from the open items
    list after milestone 15.
    - **GAS itself doesn't implement these — flagged to, and confirmed
      with, the maintainer before writing any code.** Every other
      instruction in this project was cross-checked against GNU
      binutils' actual, working parsing (`tc-m68k.c`) and/or
      disassembly (`m68k-dis.c`) logic. `opcodes/m68k-opc.c`'s `TBL(...)`
      macro produces four real opcode-table rows with plausible-looking
      argument-string codes (`` ` `` for the memory form's `<ea>`, plus
      the already-cross-checked-elsewhere `D`/`s`/`1`/`3` codes for the
      register form) — but no `` case '`': `` exists anywhere in
      `tc-m68k.c`'s parser, and `m68k-dis.c` has no TBL-printing logic
      either. These appear to be dormant table entries binutils never
      finished wiring into either direction of real tool support. The
      encoding here instead rests on the raw opcode/mask literals
      (real hardware bit positions, derived by hand from the `TBL1`
      macro) plus the generic install-code semantics already confirmed
      working for `D`/`s`/`1`/`3` elsewhere in this codebase — not on a
      working reference implementation. This is documented prominently
      in `cpu32_tbl.go`'s header comment and repeated in the test file
      and README, rather than presented with the same confidence as
      every other milestone's GAS-verified encoding.
    - **A real ambiguity, found and fixed before shipping (not by a
      user report):** an early draft's memory form accepted `Dn` in its
      `<ea>` (via `readableDataEA`), and — because `parseInstruction`
      accepts the first `Form` whose shape matches the token stream
      (`parser_stmt.go`) — a bare `"TBLS.B D0,D2"` (meant to exercise
      the register-pair form's mandatory-colon rejection) instead
      silently matched the *memory* form, parsing `D0` as if it were a
      memory address. Fixed two ways together: restricting the memory
      form's `<ea>` to `controlAlterableEA` (genuine memory reference
      only — also the more defensible semantic choice, matching
      `CHK2`/`CMP2`'s identical restriction, independent of the
      ambiguity), and listing the register-pair form *before* the
      memory form in `Forms` so that a bare two-`Dn` input reaches its
      specific, informative rejection rather than the memory form's
      generic one. `selectForm`'s own comment already warns about this
      class of hazard (two `Form`s that can both syntactically match
      the same input); this is the first time it showed up as an
      *ordering* problem between two genuinely different, non-identical
      `OperKinds` shapes rather than two literally-identical ones.
    - **This codebase's own `.B`/`.W`/`.L`-suffix convention** was used
      again (`TBLS`/`TBLSN`/`TBLU`/`TBLUN`, four mnemonics) rather than
      GAS's twelve separately-spelled `"tblsb"`/`"tblsw"`/`"tblsl"`/…
      names — the same choice already made for `BRA.L`/`BSR.L`
      (milestone 3), `FBcc.L` (milestone 13), and `DIVSL`/`DIVUL`'s
      `Dr:Dq` syntax (milestone 10).
    - **Gated `requireCPU32Only`, not `require68020orCPU32`**: GAS tags
      `TBL` plain `cpu32`, with no `m68020up` union, confirming it's not
      available on 68020+ at all — matching `BGND`
      (`cpu020_misc.go`), not `Bcc.L`/`CHK2`/`EXTB`/`TRAPcc`.
17. ✅ **Done.** `FMOVEM`'s `FPCR`/`FPSR`/`FPIAR` control-register list
    (§6), deferred at the end of milestone 14, chosen by the maintainer
    from the open items list after milestone 16. Unlike milestone 16,
    this one had a full working GAS reference to verify against.
    - **Two InstrDefs can't share one mnemonic — caught before it ever
      compiled wrong, by reading `registerInstrDef` first.** `FMOVEM`'s
      FPn-list forms (milestone 14) and this milestone's control-
      register-list forms are the same real mnemonic, and
      `registerInstrDef` panics on a duplicate name. Rather than a
      second `InstrDef`, `cpu020_fpu_movem2.go`'s `init` looks up the
      already-registered `"FMOVEM"` `InstrDef` from the package's
      `Instructions` map and appends its two new `Form`s onto it
      directly — relying on Go's (spec-recommended, and what `go
      build`/`go test` actually do) lexical file-name ordering of
      `init` functions across a package, with an explicit nil check
      turning a violated assumption into a clear panic rather than a
      silent nil dereference.
    - **GAS's own opcode table lists the store direction as two
      overlapping rows with a FIXME comment stating the intended rule
      directly**: a bare single register name may target `Dn`/`An` or
      memory, but real list syntax (two or three registers at once) may
      only target memory — "we should only permit %dn if the target is
      a single register." Rather than replicate two rows (which would
      revisit the exact `selectForm` hazard milestones 14/15/16 already
      named — both rows share `[OpkFPCtrlRegList, OpkEA]`), one
      `Validate` counts the parsed selector mask's set bits
      (`math/bits.OnesCount16`) and enforces the FIXME's rule directly.
      The load direction has no such split in GAS's table (its `<ea>`
      source accepts memory, `Dn`, `An`, or an immediate
      unconditionally), so it needed no equivalent check.
    - **A new `FPCtrlMaskSrc`/`FPCtrlMaskDst` pair on `Args`**, kept
      separate from `FPRegMaskSrc`/`Dst` (milestone 14) for the same
      reason that pair is kept separate from integer `MOVEM`'s
      `RegMaskSrc`/`Dst`: a different register namespace, encoded into
      a different bit width and position (a 3-bit selector at bits
      12-10, not the 8-bit FP0-FP7 mask at bits 7-0), needs its own
      `OperandKind` to stay distinguishable during form matching. A
      small `parseFPCtrlRegList` (bare name or slash-separated
      combination — no range syntax, since FPIAR/FPSR/FPCR have no
      numeric ordering to range over, unlike FP0-FP7 or D0-D7) mirrors
      `parseFPRegList`'s structure without its range-parsing code.
18. ✅ **Done, narrowed from a very large surface.** `PMOVE`'s remaining
    registers (§5.3), chosen by the maintainer after being shown the
    full scale of what's left: `PBcc`/`PDBcc`/`PScc`/`PTRAPcc` (112
    condition-family instructions), `PFLUSH`'s ~15 variants, `PLOAD`
    (6), `PMOVE`'s ~11 other registers, `PMOVEFD`, `PSAVE`, and
    `PTEST`'s ~14 variants — comparable in size to the entire FPU
    condition family plus `FMOVEM` combined. Scoped to `CRP`/`SRP`/
    `TT0`/`TT1`/`MMUSR`: the registers actually needed to configure
    address translation beyond `TC` (milestone 12), completing basic
    MMU table setup, while deferring the condition-branch family and
    the finer cache/TLB-management instructions.
    - **Every one of these five registers turned out to need no
      `FieldRef` at all — simpler than initially feared.** Each has a
      compile-time-known selector value (`gas/config/tc-m68k.c`'s
      `"case 'W':"` gives `DRP=1`/`SRP=2`/`CRP=3`; `"case '3':"` gives
      `TT0=2`/`TT1=3`; `MMUSR`'s row calls no `install_operand` at all,
      just asserting the register is exactly `PSR`/`MMUSR`), so every
      Form's word2 is a single fully-fixed literal computed once at
      write time — exactly like `TC`'s own already-shipped Forms, which
      already established this was possible rather than something new
      being discovered here.
    - **Appended onto the existing `defPMOVE` package-level `var`
      directly, not through the `Instructions` map** — safer than
      milestone 17's `FMOVEM` fix for the identical two-InstrDefs-one-
      mnemonic problem: since `defPMOVE.Forms` is a field on a named
      package-level variable both files can reference directly (not a
      lookup that only succeeds after the other file's `init` has run),
      the append is correct regardless of which file's `init` function
      executes first — no ordering dependency to document or guard.
    - **EA restrictions confirmed to genuinely differ by register, not
      guessed uniform**: `CRP`/`SRP` (GAS's `'|'`/`'~'` argument types)
      exclude `Dn`/`An`/immediate entirely — cross-checked against a
      GAS source comment noting these 64-bit "quad word" registers
      aren't even given immediate-operand support — while `TT0`/`TT1`/
      `MMUSR` (`'*'`/`'%'`) share `TC`'s own existing, broader
      `readableDataEA`/`dataAlterableEA` restriction. `MMUSR` is also
      the one register in this set sized `.W` (16 bits) rather than
      `.L`, per GAS's own size-hint letter on that row.
    - **A milestone-12 test's premise quietly went stale and needed
      updating, not just leaving green by accident**: `cpu030_pmmu_test.go`
      had `TestPmoveRejectsUnknownSecondRegister`, asserting `PMOVE
      (A0),CRP` was rejected — true when only `TC` existed, false the
      moment this milestone landed. Caught immediately by running the
      full suite (not a passive true-by-luck pass): retargeted to `DRP`,
      a register still genuinely unimplemented, preserving the test's
      actual intent (an unsupported `PMOVE` register must fail cleanly)
      rather than deleting it or leaving it silently checking the wrong
      thing.
    - **Still open**: `PBcc`/`PDBcc`/`PScc`/`PTRAPcc`, the rest of
      `PFLUSH`/`PLOAD`/`PTEST`, `PMOVEFD`, `PSAVE`, and `PMOVE`'s
      remaining registers (`DRP`, `CAL`, `VAL`, `SCC`, `AC`, `PSR`/
      `PCSR`, `BAD`/`BAC`) — deliberately deferred again, each its own
      future milestone rather than one more attempt to fit the whole
      PMMU surface into a single pass.
19. ✅ **Done.** The PMMU's own conditional branch/set/trap family —
    `PBcc`/`PDBcc`/`PScc`/`PTRAPcc`, 112 instructions across 16
    conditions — chosen by the maintainer from the open items list after
    milestone 18, and the largest remaining PMMU slice by instruction
    count.
    - **Structurally, nothing new: the third time this exact shape has
      been built.** The integer ISA's `Bcc`/`DBcc`/`Scc`/`TRAPcc`
      (`xcc.go`, `cpu020_misc.go`) and the FPU's `FBcc`/`FDBcc`/`FScc`/
      `FTRAPcc` (milestone 13, `cpu020_fpu2.go`) are the same family
      shape with different condition counts (16 vs 32) and base
      opcodes; the PMMU family reuses the FPU version's exact
      architecture (one condition-suffix table, `newPBccDef`/
      `newPDBccDef`/`newPSccDef`/`newPTrapccDef` builder functions) with
      the PMMU's own 16 conditions and bases substituted in — no new
      design decisions were needed, only new data. The one genuine
      difference: PMMU's word1 base is a bare `0xF0xx` (no coprocessor-
      ID bit to fold in), unlike the FPU family's `0xF2xx` — confirming,
      rather than contradicting, that the coprocessor-ID bit really is
      an FPU-specific requirement (`fpuWord1Base`'s own doc comment) and
      not something every `0xF`-line coprocessor instruction needs.
    - **Condition values cross-checked two independent ways before any
      code was written**: decoded once from `PBcc`'s own mnemonic
      literals (the low nibble of each opcode, e.g. `"pbac"` = `0xF087`
      → condition 7), then confirmed a second time against `Pscc`'s
      completely independent table — both produced the identical
      16-value assignment, the same cross-check discipline milestone 13
      used for the FPU's 32 conditions.
    - **`PBcc`'s bit-6 size flag reused `FBcc`'s own established
      pattern rather than replicating GAS's own mechanism**: GAS's bare
      `PBcc` mnemonic auto-upgrades from word to long displacement via a
      two-pass frag resolution (setting bit 6 only when the assembler
      later discovers the target doesn't fit in 16 bits), with a
      separately-spelled `-w`-suffixed mnemonic to force word size
      unconditionally. This codebase again commits to its own `.W`/`.L`
      suffix convention instead (two `Form`s, `Sizes: [WordSize]` and
      `Sizes: [LongSize]`, word1 literals `0xF080|cc` and `0xF0C0|cc`)
      — the same choice already made for `BRA.L`/`BSR.L` (milestone 3)
      and `FBcc.L` (milestone 13).
20. ✅ **Done.** `PMOVE`'s remaining registers (§5.3) — `DRP`, `CAL`,
    `VAL`, `SCC`, `AC`, `PSR`, `PCSR` — chosen by the maintainer from
    the open items list after milestone 19, completing `PMOVE` except
    for `BAD`/`BAC`.
    - **Sizes turned out to genuinely differ by register, not assumed
      uniform with `TC`'s own `.L`**: re-reading `gas/config/tc-m68k.c`'s
      type-check switch (not just its `install_operand` dispatch, which
      is shared and identical across `TC`/`AC`/`CAL`/`VAL`/`SCC`) showed
      three *separate* opcode-table rows restricting `TC` to `.L`, `AC`
      to `.W`, and `CAL`/`VAL`/`SCC` to `.B` — despite all five sharing
      the exact same selector-value computation. `newPmmuFixedReg`
      (milestone 18) gained an explicit `sz Size` parameter rather than
      inferring it, since inference (as milestone 18's own version did,
      hard-coding `MMUSR` as the one `.W` exception) would have
      silently gotten `AC`/`CAL`/`VAL`/`SCC` wrong.
    - **`CAL`/`VAL`/`SCC` needed no new mechanism at all, despite GAS
      giving them one shared opcode-table row with a runtime-computed
      selector**: since each of the three has its own `OperandKind` in
      this codebase (unlike GAS, which distinguishes them only via
      `opP->reg` at parse time within one shared row), each one's
      *individual* selector value (`4`/`5`/`6`) can just be folded into
      its own literal at write time — the same `newPmmuFixedReg` helper
      every other fixed-selector register already uses, no different
      from how `CRP`/`SRP`/`TT0`/`TT1` needed no shared dispatch despite
      also having one word2 base each.
    - **"PSR" added as a second accepted name for the already-shipped
      `MMUSR` encoding, not a new register.** The 68851 calls this
      register "PSR"; the 68030 calls the identical encoding "MMUSR" —
      real hardware, not two different things (confirmed via GAS's own
      comment on the register enum). Handled as a small addition to the
      existing `OpkMMUSR` parser case (accept either spelling, same
      `EAkMMUSR` result) rather than a new `EAExprKind`/`Form` pair.
    - **`PCSR` is store-only**, matching GAS's own table: there is no
      `"<ea>,PCSR"` row at all, only `"PCSR,<ea>"` — confirmed rather
      than assumed symmetric with every other `PMOVE` register.
    - **Deliberately still deferred: `BAD0`-`BAD7`/`BAC0`-`BAC7`**
      (breakpoint address/access registers, 8 numbered instances each).
      Their encoding turned out to need a register-*number* field (bits
      4-2) in addition to the usual selector, and — confirmed while
      researching this milestone, not assumed — an INVERTED load/store
      direction bit relative to every other `PMOVE` register (GAS's own
      table: load word2 `0x6200`, store `0x6000`, the reverse of the
      `0x...00`-load/`0x...200`-store convention every other register in
      this file follows). Different enough in shape, and obscure enough
      even among an already-rare instruction family, to warrant its own
      milestone rather than folding it in here.
    - **A second milestone-N test premise went stale, caught and
      updated the same way milestone 18's was**: `TestPmoveRejectsUnknownSecondRegister`
      (retargeted to `DRP` in milestone 18) needed retargeting again,
      this time to `BAD0` — the pattern of "this test's specific choice
      of still-unsupported register will keep going stale as PMOVE
      grows" is now expected and unsurprising, not a recurring bug.

Each milestone is independently shippable and testable against the real
opcode tables in `docs/M68kOpcodes.pdf`, and each one leaves
`go test ./...` green with zero changes required to any existing test
(modulo the intentional scale-factor fix in milestone 1).
