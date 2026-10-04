# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.6.1] - 2026-10-04

Found by round-tripping m68kdasm's output through the assembler.

### Added

- `MOVE CCR,<ea>` (68010+)
- `CMPI` with a PC-relative destination on CPU32/68020+
- `CAS2` takes data registers as well as address registers as pointers
- `FRESTORE`/`PRESTORE` with a PC-relative operand

### Fixed

- `DBcc` to the address right after its extension word encoded -2 instead
  of 0
- `CHK2`/`CMP2` emitted the operand's extension words before the register
  word; the register word directly follows the opcode
- `CAS2` left the D/A bit of its pointer registers clear, encoding
  `(A0):(A1)` as `(D0):(D1)`

## [1.6.0] - 2026-10-03

Checked against every opcode word of the 68000 (and, via m68kdasm, of the
68010-68060, CPU32, FPU and PMMU) by assembling each instruction's text and
comparing the result with the original encoding.

### Added

- `BCC`/`BCS`, `DBCC`/`DBCS`, `SCC`/`SCS` and `TRAPCC`/`TRAPCS`: the
  Motorola names for carry clear/set, next to the `HS`/`LO` aliases
- `MULU.L`/`MULS.L` (`<ea>,Dl` and `<ea>,Dh:Dl`), `DIVU.L`/`DIVS.L`
  (`<ea>,Dq` and `<ea>,Dr:Dq`), `CHK.L` and `LINK.L` for CPU32/68020+
- `MOVEC` to the control registers added after the 68010 — `CACR`, `CAAR`,
  `MSP`, `ISP`, `TC`, `ITT0`/`ITT1`, `DTT0`/`DTT1`, `MMUSR`, `URP`, `SRP`,
  `BUSCR`, `PCR` — each accepted only on the CPUs that have it
- `TST` with PC-relative, immediate and (word/long) address register
  operands on CPU32/68020+
- `BTST Dn,#<data>`, `CHK #<data>,Dn` and `NBCD` with every data alterable
  operand, as the 68000 allows

### Fixed

- `MOVEP` emitted the wrong opmode for all four forms (word and long,
  either direction were swapped)
- `MULU`/`MULS`/`DIVU`/`DIVS`/`DIVUL`/`DIVSL` with an immediate source
  dropped the immediate's extension word(s)
- FPU instructions accept a single-precision (`.s`) operand in a data
  register, like the integer formats
- Invalid 68000 forms are rejected: `BCHG`/`BCLR`/`BSET` with a PC-relative
  destination, `CMP.B An,Dn`, `CHK An,Dn`, `LEA` with a non-control
  source, `MOVEM` registers to `(An)+`, and `TST` with PC-relative operands

### Changed

- `DIVUL`/`DIVSL` follow the Motorola manual: `<ea>,Dr:Dq` divides a
  32-bit dividend and keeps the remainder. They previously used the GNU as
  meaning of `divul`/`divsl` and set the 64-bit dividend bit; that form is
  now `DIVU.L`/`DIVS.L <ea>,Dr:Dq`
- The 68060 emulation warning now names `DIVU.L`/`DIVS.L Dr:Dq` and
  `MULU.L`/`MULS.L Dh:Dl`

## [1.5.0] - 2026-09-14

### Added

- The full 68k family behind `--cpu`/`--fpu`/`--fpu-full`/`--mmu` (and the
  matching `Target` in the Go API): 68010/68012, CPU32, 68020, 68030, 68040
  and 68060, with 32-bit branches, memory-indirect addressing, scale
  factors, bit-field instructions, `CAS`/`CAS2`, `CHK2`/`CMP2`, `TRAPcc`,
  `PACK`/`UNPK`, `CALLM`/`RTM`, `MOVE16`, cache control and the CPU32
  table-lookup instructions
- The 68881/68882 FPU instruction set, including FPU conditionals,
  `FMOVEM`, floating-point and packed BCD immediates, and the
  transcendental and math-extension functions
- The 68851/68030 PMMU instruction set (`PMOVE` with every register,
  `PFLUSH`/`PLOAD`/`PTEST`, `PSAVE`/`PRESTORE`, `PMOVEFD`, the `P`
  conditionals) and the 68040's single-word PMMU forms
- Non-fatal warnings for forms the 68060 trap-emulates
- A user manual (`docs/user-manual.md`)

### Fixed

- `MOVEM` no longer accepts `-(An)` as a load source
- FPU coprocessor-ID and R/M-bit encodings, and `FSAVE`/`FRESTORE`'s
  coprocessor-ID bit

## [1.4.0] - 2026-09-03

### Changed

- Raised the minimum supported Go version to 1.27 and aligned CI with the new toolchain target
- Adopted Go 1.21+/1.27 standard-library idioms across the codebase (`slices`, `maps`, `min`, `strings.Cut`, `errors.AsType`, range-over-int)
- Split the monolithic parser into focused files (labels, statements, macros, operands, effective addresses) and de-duplicated the size-validation helper

## [1.3.2] - 2026-06-13

### Fixed

- Fixed PC-relative operands for bit operation encodings
- Corrected PC-relative operand handling to use the extension-word base consistently

## [1.3.1] - 2026-04-03

### Changed

- Raised the minimum supported Go version to 1.26 and aligned CI with the new toolchain target

### Documentation

- Normalized changelog formatting and updated README version references for the current release line

## [1.3.0] - 2026-03-28

### Fixed

- Corrected PC-relative displacement calculations for `d16(PC)` and `d8(PC,Xn)` addressing modes to use the extension word address as the base
- Fixed branch displacement calculations for word-sized branches (`BRA.W`, `BSR.W`, `DBcc`) to use the correct base PC
- Added support for `$` as current program counter in expressions
- Added support for `.w` and `.l` suffixes on labels in PC-relative expressions
- Updated test expectations to match correct displacement calculations

### Improved

- Improved error handling and diagnostics

## [2.0.0] - 2026-03-25

### Added

- ELF32 output format with standard sections and symbol tables
- Enhanced error diagnostics with source line context
- Support for `.text`, `.data`, `.bss` section directives
- Improved expression evaluation with better error messages

### Changed

- Major refactoring of parser and assembler internals
- Updated instruction encoding tables for better maintainability

## [1.2.1] - 2025-XX-XX

### Fixed

- Various bug fixes and improvements

### Added

- Additional instruction support
- Performance optimizations

## [1.2.0] - 2025-XX-XX

### Added

- S-record output format
- Source listing generation
- Macro support

## [1.1.5] - 2025-XX-XX

### Fixed

- Bug fixes

## [1.1.4] - 2025-XX-XX

### Added

- More instruction support

## [1.1.2.1] - 2025-XX-XX

### Fixed

- Patch release

## [1.1.2] - 2025-XX-XX

### Added

- Enhanced expression support

## [1.1.1] - 2025-XX-XX

### Fixed

- Bug fixes

## [1.1.0] - 2025-XX-XX

### Added

- Local numeric labels

## [1.0.1] - 2025-XX-XX

### Fixed

- Initial bug fixes

## [1.0.0] - 2025-XX-XX

### Added

- Initial release with basic 68000 instruction support
- Command-line interface
- Binary output format

## [0.4.0] - 2025-XX-XX

### Added

- More instructions

## [0.1.0] - 2025-XX-XX

### Added

- Initial prototype
