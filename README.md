# goznp

A Go library and CLI for controlling Zigbee networks using Texas Instruments CC2652 coordinators with Z-Stack firmware.

[![Go Reference](https://pkg.go.dev/badge/github.com/marstid/goznp.svg)](https://pkg.go.dev/github.com/marstid/goznp)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

## Overview

goznp provides a complete solution for building Zigbee network applications in Go. It includes:

- **Low-level protocol support**: UNPI framing, ZNP commands, and ZCL cluster implementations
- **High-level adapter API**: Easy-to-use interface for network management and device control
- **Feature-rich CLI**: Ready-to-use command-line tool for network operations

## Supported Hardware

| Coordinator | Status | Notes |
|-------------|--------|-------|
| SONOFF ZBDongle-P (CC2652P) | Tested | Primary development device |
| CC2652R/RB | Compatible | Lower TX power (max 5 dBm) |
| CC2652P/CC1352P | Compatible | Higher TX power (max 20 dBm) |

**Firmware**: [Koenkk Z-Stack 3.x.0](https://github.com/Koenkk/Z-Stack-firmware/tree/master/coordinator/Z-Stack_3.x.0)

## Installation

### CLI Tool

```bash
go install github.com/marstid/goznp/cmd/goznp@latest
```

### Library

```bash
go get github.com/marstid/goznp
```

## Quick Start

### Using the CLI

```bash
# List available serial ports (detects CC2652 adapters)
goznp list

# Set port via environment variable
export GOZNP_PORT=/dev/ttyUSB0

# Check adapter connectivity
goznp ping

# View adapter and network information
goznp info

# Form a new network
goznp network form --channel 15

# Pair a device (opens network, waits for device, interviews, binds)
goznp device pair

# List paired devices
goznp device list --interview

# Control a smart plug
goznp device on --addr 0xBE87
goznp device off --addr 0xBE87
goznp device toggle --addr 0xBE87

# Control a dimmable light
goznp device brightness --addr 0xBE87 --level 128 --transition 1000

# Read sensor data
goznp device sensor --addr 0x1234

# Read power consumption from a smart plug
goznp device power --addr 0xBE87
```

### Using the Library

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
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Create and open adapter
    a := adapter.New(
        adapter.WithSerialPath("/dev/ttyUSB0"),
        adapter.WithBaudRate(115200),
    )

    if err := a.Open(ctx); err != nil {
        log.Fatal(err)
    }
    defer a.Close()

    // Get adapter info
    info, err := a.GetInfo(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Z-Stack: %s\n", info.Version.Variant())

    // Get network info
    network, err := a.GetNetworkInfo(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("PAN ID: 0x%04X, Channel: %d\n", network.PanID, network.Channel)

    // List devices
    devices, err := a.GetDevices(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Devices: %d\n", len(devices))

    // Control a device
    if len(devices) > 0 {
        if err := a.TurnOn(ctx, devices[0].NwkAddr, 1); err != nil {
            log.Printf("Turn on failed: %v", err)
        }
    }
}
```

## Features

### Network Management

| Command | Description |
|---------|-------------|
| `network form` | Create a new Zigbee network |
| `network channel` | View or change network channel |
| `network power` | View or set TX power level |
| `network profile` | View registered application profiles |
| `network topology` | Display mesh network topology |
| `reset` | Soft reset the adapter |
| `reset factory` | Factory reset (erase all configuration) |

### Device Management

| Command | Description |
|---------|-------------|
| `device pair` | Pair new devices (with auto-interview and binding) |
| `device permit` | Open/close network for pairing |
| `device list` | List paired devices |
| `device watch` | Monitor device join/leave events |
| `device remove` | Remove a device from the network |
| `device info` | Query device endpoints and clusters |
| `device status` | Read device status (manufacturer, model, battery) |

### Device Control

| Command | Description |
|---------|-------------|
| `device on/off/toggle` | Control On/Off cluster |
| `device brightness` | Control Level cluster (dimmable lights) |
| `device color-temp` | Control Color Temperature (tunable white) |
| `device color` | Control Color (hue/saturation, XY) |
| `device identify` | Trigger device identification (blinking) |

### Groups and Scenes

| Command | Description |
|---------|-------------|
| `device group add/remove/list` | Manage device group membership |
| `device group on/off/toggle` | Control all devices in a group |
| `device group brightness` | Set brightness for a group |
| `device scene store/recall/remove/list` | Manage device scenes |

### Sensors and Power Monitoring

| Command | Description |
|---------|-------------|
| `device sensor` | Read temperature, humidity, battery |
| `device power` | Read voltage, current, power, energy |
| `device listen` | Listen for incoming sensor reports |
| `device reset-energy` | Reset energy counter |

### Bindings and Reporting

| Command | Description |
|---------|-------------|
| `device bind` | Bind device cluster to coordinator |
| `device bindings` | List device binding table |
| `device configure-reporting` | Configure attribute reporting |
| `device reporting` | Read reporting configuration |

### Backup and Restore

| Command | Description |
|---------|-------------|
| `backup create` | Create network backup (JSON) |
| `backup restore` | Restore from backup |
| `backup show` | Display backup contents |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      CLI (cmd/goznp)                       │
├─────────────────────────────────────────────────────────────┤
│                    Adapter (pkg/adapter)                    │
│  - Network management    - Device interview                 │
│  - Device control        - Backup/restore                   │
├─────────────────────────────────────────────────────────────┤
│         ZCL (pkg/zcl)         │        ZNP (pkg/znp)        │
│  - Cluster definitions        │  - SYS commands             │
│  - Frame encoding             │  - AF commands              │
│  - Data types                 │  - ZDO commands             │
│  - Color control              │  - NV storage               │
├─────────────────────────────────────────────────────────────┤
│                     UNPI (pkg/unpi)                         │
│  - Frame parsing/serialization                              │
│  - Checksum validation                                      │
├─────────────────────────────────────────────────────────────┤
│                    Serial (pkg/serial)                      │
│  - Port enumeration                                         │
│  - CC2652 adapter detection                                 │
└─────────────────────────────────────────────────────────────┘
```

### Protocol Layers

- **UNPI** (Unified Network Processor Interface): TI's serial protocol for Z-Stack communication
- **ZNP** (Zigbee Network Processor): Command interface for the Z-Stack firmware
- **ZCL** (Zigbee Cluster Library): Standard device control clusters

## Supported ZCL Clusters

| Cluster | ID | Description |
|---------|-----|-------------|
| Basic | 0x0000 | Device information (manufacturer, model) |
| Power Configuration | 0x0001 | Battery level reporting |
| Identify | 0x0003 | Device identification |
| Groups | 0x0004 | Group membership |
| Scenes | 0x0005 | Scene storage |
| On/Off | 0x0006 | Binary switch control |
| Level Control | 0x0008 | Brightness/dimming |
| Color Control | 0x0300 | Color temperature, hue/saturation, XY |
| Temperature Measurement | 0x0402 | Temperature sensors |
| Humidity Measurement | 0x0405 | Humidity sensors |
| Occupancy Sensing | 0x0406 | Motion sensors |
| Electrical Measurement | 0x0B04 | Voltage, current, power |
| Simple Metering | 0x0702 | Energy consumption |

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GOZNP_PORT` | Serial port path | (none) |

### Common Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-p, --port` | Serial port path | `$GOZNP_PORT` |
| `-b, --baud` | Baud rate | 115200 |

## Examples

### Form Network and Pair Devices

```bash
# Form a network on channel 20
goznp network form --channel 20

# Pair devices (180 second window with auto-binding)
goznp device pair --timeout 180

# View paired devices with full details
goznp device list --interview
```

### Control Smart Bulbs

```bash
# Turn on and set brightness
goznp device on --addr 0x1234
goznp device brightness --addr 0x1234 --level 200 --transition 500

# Set warm white color temperature
goznp device color-temp --addr 0x1234 --kelvin 2700

# Set color by hue/saturation
goznp device color hue --addr 0x1234 --hue 120 --sat 100
```

### Group Control

```bash
# Add devices to a group
goznp device group add --addr 0x1234 --group 1 --name "Living Room"
goznp device group add --addr 0x5678 --group 1

# Control all devices in group
goznp device group on --group 1
goznp device group brightness --group 1 --level 128
```

### Monitor Sensors

```bash
# Read sensor data directly
goznp device sensor --addr 0xABCD

# Listen for automatic reports
goznp device listen
```

### Backup and Restore

```bash
# Create backup
goznp backup create -o network-backup.json

# View backup contents
goznp backup show -i network-backup.json

# Restore (after replacing adapter)
goznp backup restore -i network-backup.json --force
```

## Development

### Building

```bash
# Build CLI
go build -o bin/goznp ./cmd/goznp

# Run tests
go test ./...

# Format code
go fmt ./...
```

### Project Structure

```
goznp/
├── cmd/goznp/          # CLI application
├── pkg/
│   ├── adapter/         # High-level adapter API
│   ├── backup/          # Backup file format
│   ├── serial/          # Serial port handling
│   ├── unpi/            # UNPI protocol
│   ├── zcl/             # ZCL clusters and types
│   └── znp/             # ZNP commands
└── README.md
```

## Acknowledgments

This project was built with help from these excellent resources:

- [zigbee-herdsman](https://github.com/Koenkk/zigbee-herdsman) - TypeScript Zigbee library
- [zigpy-znp](https://github.com/zigpy/zigpy-znp) - Python ZNP implementation
- [Koenkk Z-Stack Firmware](https://github.com/Koenkk/Z-Stack-firmware) - Coordinator firmware

## License

MIT License - see [LICENSE](LICENSE) for details.
