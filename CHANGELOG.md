# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.0] - 2026-03-28

### Fixed
- Corrected PC-relative displacement calculations for `d16(PC)` and `d8(PC,Xn)` addressing modes to use the extension word address as the base
- Fixed branch displacement calculations for word-sized branches (`BRA.W`, `BSR.W`, `DBcc`) to use the correct base PC
- Added support for `$` as current program counter in expressions
- Added support for `.w` and `.l` suffixes on labels in PC-relative expressions
- Updated test expectations to match correct displacement calculations

### Changed
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