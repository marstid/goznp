# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.2.1] - 2026-01-07

### Added
- Display device names in list and health commands (defbcf3)
- Add hardware integration test framework (4327851)

### Fixed
- Return context error from device list function (be9fc6f)
- Resolve flaky waiter tests by increasing timing tolerance (545b30a)

### Changed
- Improve code comments and fix misspellings (bd77d7f)
- Relax linter rules for existing codebase (dd055c0)

## [v0.2.0] - 2026-01-07

### Added
- Add network diagnostics to adapter (69d7b85)
- Add attribute reading infrastructure to adapter (22f427d)
- Add OTA firmware update support (74e6596)
- Add quirks system for non-compliant devices (a7dc997)
- Add routing table queries and device announce handling to ZNP (d349adc)
- Expand ZCL support with new data types and frame builders (20affd6)

### Changed
- Split device commands into focused files (9077bc5)
- Split messaging into domain-specific files and add interview retries (f81aeb4)
- Add ZNP interface abstraction and improve device handling (f12b5e3)

### Fixed
- Improve buffer overflow recovery by searching for next frame (f3443c3)

### Documentation
- Add package-level documentation (49b2500)
- Add comprehensive usage examples (495fb17)
