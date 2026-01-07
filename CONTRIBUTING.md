# Contributing to goznp

Thank you for your interest in contributing to goznp!

## Getting Started

### Prerequisites

- Go 1.21 or later
- A supported Zigbee coordinator (CC2652R/RB, CC2652P, CC1352P)
- Koenkk Z-Stack 3.x.0 firmware (recommended)

### Fork and Clone

```bash
# Fork the repository on GitHub
git clone https://github.com/your-username/goznp.git
cd goznp
git remote add upstream https://github.com/marstid/goznp.git
```

### Install Dependencies

```bash
go mod download
```

## Development Workflow

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out  # Open in browser

# Run tests with race detector
go test -race ./...

# Run tests in verbose mode
go test -v ./...
```

### Code Formatting

```bash
# Format code
go fmt ./...

# Or use goimports (included in golangci-lint)
goimports -w .
```

### Linting

```bash
# Run golangci-lint
golangci-lint run

# Run with all enabled linters
golangci-lint run --enable-all
```

### Building

```bash
# Build the CLI tool
go build -o bin/goznp ./cmd/goznp

# Build for all platforms using goreleaser
goreleaser build --snapshot --clean
```

## Coding Standards

### Naming Conventions

- Use **MixedCaps** for exported identifiers (no underscores or camelCase)
- Use **camelCase** for unexported variables
- Example: `NewAdapter`, `SendZCLFrame`, `deviceName`

### Error Handling

- Always wrap errors with context using `%w`:
  ```go
  return fmt.Errorf("failed to open port: %w", err)
  ```

- Check errors immediately, don't defer error checks
- Use sentinel errors for expected error cases (see `pkg/adapter/errors.go`)

### Context

- Accept `context.Context` as the first parameter for I/O operations
- Pass context down to all called functions
- Respect context cancellation in long-running operations

```go
func (a *Adapter) GetDevices(ctx context.Context) ([]*Device, error) {
    // Check context before starting work
    if err := ctx.Err(); err != nil {
        return nil, err
    }
    // ... implementation
}
```

### Documentation

- Exported functions must have godoc comments
- Comments should start with the function name:
  ```go
  // Ping sends a ping request to the adapter and returns capabilities.
  func (a *Adapter) Ping(ctx context.Context) (*znp.PingCapabilities, error)
  ```

- Package-level documentation should be in `doc.go`

### Testing

- Aim for 80%+ test coverage for new code
- Use table-driven tests for multiple scenarios:
  ```go
  tests := []struct {
      name     string
      input    string
      expected string
  }{
      {"simple", "test", "TEST"},
      {"empty", "", ""},
  }

  for _, tt := range tests {
      t.Run(tt.name, func(t *testing.T) {
          result := process(tt.input)
          if result != tt.expected {
              t.Errorf("got %v, want %v", result, tt.expected)
          }
      })
  }
  ```

- Mock the ZNP layer for adapter tests (see `pkg/adapter/mock_test.go`)
- Test with real hardware for protocol-level changes

## Testing with Hardware

For changes affecting device communication, test with real Zigbee devices:

```bash
# Set port for your coordinator
export GOZNP_PORT=/dev/ttyUSB0

# Verify connectivity
./bin/goznp ping

# Test device operations
./bin/goznp device list
./bin/goznp device on --addr 0x1234
```

**Test devices:**
- Smart bulbs (On/Off, Level Control, Color Control)
- Temperature/humidity sensors
- Power meters
- Various manufacturers (Tuya, IKEA, Aqara, Philips Hue)

## Submitting Changes

### Branching

```bash
# Create a feature branch
git checkout -b feature/my-feature main

# Or a bugfix branch
git checkout -b fix/my-bugfix main
```

### Commit Messages

Use conventional commits format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

Examples:
- `feat(clusters): add ColorControl temperature attributes`
- `fix(adapter): correct transaction ID handling`
- `docs(readme): add MQTT integration example`

### Pre-commit Checklist

- [ ] Code is formatted (`go fmt ./...`)
- [ ] Tests pass (`go test ./...`)
- [ ] Linting passes (`golangci-lint run`)
- [ ] New code has tests with 80%+ coverage
- [ ] Tested with real hardware if applicable
- [ ] Documentation is updated

### Pull Request Process

1. Push your branch to your fork
2. Create a pull request on GitHub
3. Fill out the PR template
4. Address review feedback
5. Wait for approval and merge

## Project Structure

```
goznp/
├── cmd/goznp/           # CLI application
│   ├── main.go          # Entry point
│   ├── device.go        # Device commands
│   ├── network.go       # Network commands
│   ├── backup.go        # Backup/restore
│   └── quirks.go        # Quirks management
├── pkg/
│   ├── adapter/         # High-level adapter API
│   │   ├── adapter.go   # Main adapter
│   │   ├── device.go    # Device management
│   │   ├── messaging.go # ZCL message handling
│   │   ├── interview.go # Device interview
│   │   └── quirks/      # Device quirks
│   ├── znp/             # ZNP protocol
│   ├── zcl/             # Zigbee cluster library
│   ├── unpi/            # UNPI frame protocol
│   ├── serial/          # Serial port handling
│   ├── backup/          # Backup format
│   └── devices/         # Device database
└── .github/             # GitHub workflows and templates
```

## Adding Device Support

To add a new device to the database:

1. Pair the device and get its fingerprint:
   ```bash
   goznp device pair
   goznp device info --addr <addr> --verbose
   ```

2. Edit `pkg/devices/devices.go`
3. Add the device to the `deviceFingerprints` array
4. Add necessary quirks if the device doesn't follow spec
5. Test all supported operations

## Adding Quirks

For devices with non-standard behavior:

1. Create a quirk in `pkg/adapter/quirks/builtin.go`
2. Add it to `DefaultRegistry`
3. Test with the specific device

Quirk types:
- `QuirkAttributeOverride`: Modify attribute read/write
- `QuirkCommandOverride`: Use different command
- `QuirkResponseOverride`: Accept non-standard responses

## Release Process

Releases follow semantic versioning. They are created by maintainers:

1. Update CHANGELOG.md
2. Tag release: `git tag v0.X.Y`
3. Push tag: `git push origin v0.X.Y`
4. GitHub Actions auto-create release with binaries

## Questions?

- Check the [README](README.md) for usage documentation
- Open a [GitHub Discussion](https://github.com/marstid/goznp/discussions) for questions
- Review existing issues and PRs for patterns

## Getting Help

- GitHub Issues: https://github.com/marstid/goznp/issues
- Discussions: https://github.com/marstid/goznp/discussions

Thanks for contributing to goznp!
