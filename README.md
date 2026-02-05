# goznp

Go library for controlling Zigbee networks via Texas Instruments Z-Stack (ZNP) adapters.

[![CI](https://github.com/marstid/goznp/actions/workflows/ci.yml/badge.svg)](https://github.com/marstid/goznp/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/marstid/goznp.svg)](https://pkg.go.dev/github.com/marstid/goznp)
[![Go Report Card](https://goreportcard.com/badge/github.com/marstid/goznp)](https://goreportcard.com/report/github.com/marstid/goznp)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- **Device control** — on/off, brightness, color temperature, hue/saturation
- **Device discovery** — automatic interview of endpoints, clusters, and attributes
- **Sensor reading** — temperature, humidity, pressure, power, battery, occupancy
- **Groups and scenes** — control multiple devices with a single command
- **Network management** — form/join networks, permit joining, topology mapping
- **Network diagnostics** — LQI/RSSI neighbor tables, route discovery
- **OTA firmware updates** — upgrade device firmware over the air
- **Quirks system** — automatic workarounds for non-compliant devices
- **Device naming** — persistent friendly names stored on the coordinator
- **Backup/restore** — full coordinator NVRAM backup and restore
- **REST API daemon** ([goznpd](docs/DAEMON.md)) — HTTP interface for device control
- **CLI tool** ([goznp](docs/CLI.md)) — command-line interface for all operations

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/marstid/goznp/pkg/adapter"
)

func main() {
    a := adapter.New(adapter.WithSerialPath("/dev/ttyUSB0"))

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := a.Open(ctx); err != nil {
        log.Fatal(err)
    }
    defer a.Close()

    devices, err := a.GetDevices(ctx)
    if err != nil {
        log.Fatal(err)
    }

    for _, dev := range devices {
        fmt.Printf("Device: %s %s (0x%04X)\n", dev.Manufacturer, dev.Model, dev.NwkAddr)
    }
}
```

## Installation

### Library

```bash
go get github.com/marstid/goznp
```

### CLI tool

```bash
go install github.com/marstid/goznp/cmd/goznp@latest
```

### Daemon

```bash
go install github.com/marstid/goznp/cmd/goznpd@latest
```

Or build from source:

```bash
make build       # CLI only
make build-all   # CLI + daemon
```

## Hardware

Supported coordinators running [Koenkk Z-Stack 3.x.0](https://github.com/Koenkk/Z-Stack-firmware) firmware:

| Chip | Boards |
|------|--------|
| CC2652R / CC2652RB | SONOFF ZBDongle-E, Electrolama zig-a-zig-ah! |
| CC2652P | SONOFF ZBDongle-P, Tube's CC2652P2 |
| CC1352P | LAUNCHXL-CC1352P |

Connect the coordinator via USB. The serial port will appear as:
- **Linux**: `/dev/ttyUSB0` or `/dev/ttyACM0`
- **macOS**: `/dev/tty.usbserial-110`
- **Windows**: `COM3`

Use `goznp scan` to auto-detect connected adapters.

## CLI

The `goznp` CLI provides commands for all adapter operations:

```bash
# Scan for adapters
goznp scan

# Connect and show adapter info
goznp connect --port /dev/ttyUSB0

# List paired devices
goznp device list --port /dev/ttyUSB0

# Control a device
goznp device on --addr 0x1234 --endpoint 1
goznp device off --addr 0x1234 --endpoint 1

# Set device brightness
goznp device brightness --addr 0x1234 --endpoint 1 --level 128

# Network management
goznp network info
goznp network form --channel 15
goznp network permit-join --seconds 60
goznp network tree
```

Set `GOZNP_PORT` to avoid passing `--port` every time:

```bash
export GOZNP_PORT=/dev/ttyUSB0
```

See [docs/CLI.md](docs/CLI.md) for the full command reference.

## Daemon

`goznpd` runs as a background service exposing a REST API:

```bash
GOZNP_PORT=/dev/ttyUSB0 goznpd
```

```bash
# List devices
curl http://localhost:8080/api/v1/devices

# Control by slug
curl -X POST http://localhost:8080/api/v1/devices/living-room-light/on

# Read sensor data
curl http://localhost:8080/api/v1/devices/outdoor-plug/sensor
```

See [docs/DAEMON.md](docs/DAEMON.md) for the full API reference, Docker setup, and systemd configuration.

## Documentation

| Resource | Description |
|----------|-------------|
| [Getting Started](docs/GETTING_STARTED.md) | Hardware setup, first pairing, first control |
| [Examples](examples/) | Runnable code samples (basics, control, advanced) |
| [API Reference](https://pkg.go.dev/github.com/marstid/goznp/pkg/adapter) | Go package documentation |
| [CLI Reference](docs/CLI.md) | Full CLI command reference |
| [Daemon Reference](docs/DAEMON.md) | REST API, Docker, systemd |
| [Architecture](DEVELOPING.md) | Internal design and development guide |
| [Contributing](CONTRIBUTING.md) | Coding standards and hardware testing |

## Architecture

goznp is a layered library with minimal dependencies (2 direct: `cobra`, `serial`):

```
Application
    |
    v
pkg/adapter    High-level API (device control, network management)
    |
    v
pkg/znp        Z-Stack Network Processor protocol (ZDO, AF, SYS, UTIL)
    |
    v
pkg/unpi       Unified Network Processor Interface (frame encoding/decoding)
    |
    v
pkg/serial     Serial port communication
```

Supporting packages:
- `pkg/zcl` — Zigbee Cluster Library (frame building, data types, cluster constants)
- `pkg/ota` — OTA firmware update server and image parsing
- `pkg/backup` — Coordinator NVRAM backup format

## License

[MIT](LICENSE)
