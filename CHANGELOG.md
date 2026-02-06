# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.4.3] - 2026-02-06

### Fixed
- Increase timeouts in waiter tests for slow CI environments (ae59fbe)

## [v0.4.2] - 2026-02-06

### Fixed
- Increase registration wait time in TestWaiterMultiplePending and TestWaiterMultipleSameMatch (0a02495)

## [v0.4.1] - 2026-02-06

### Fixed
- Increase timeout tolerance in TestWaiterTimeout to reduce flakiness (9cce7f2)

## [v0.4.0] - 2026-02-06

### Added
- Auto-load .env file in Makefile for easier configuration (cec2914)

### Fixed
- Add NvLength to ZNPClient interface for integration tests (2dcc7c1)
- Resolve deadlock in TestBusClose (a78bb91)
- Resolve waiter delivery bug in concurrent scenarios (a4e3ce5)

### Changed
- Fix linting issues across codebase (89c3d67)

## [v0.3.0] - 2026-01-15

### Added
- Add REST API daemon for device management (5ca4e4e)
- Add device state management system (d4656ed)
- Add message parsing infrastructure (75204da)
- Add event bus system for device events (741a4e4)
- Add network tree command for topology visualization (2092070)
- Add adapter validation, constants, and logging (9e1f55f)
- Add NVRAM hardware integration tests (9d38ef1)
- Add integration test infrastructure (339d9b9)
- Add comprehensive documentation (ca71297)

### Changed
- Modernize CLI with improved UX and error messages (6872695)
- Enhance adapter with retry logic and improved error handling (47c8a6e)
- Extract shared flags and reorganize CLI commands (e5693e1)
- Update Go version to 1.25 (163004d)
- Update build configuration and documentation (32d61a2)

### Fixed
- Resolve NVRAM test timing issues (0e165da)
- Remove invalid errcheck config property (c3967c2)

### Removed
- Remove security workflow (fac1294)

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
