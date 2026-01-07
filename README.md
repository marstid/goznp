# goznp

A Go library and CLI tool for managing Zigbee networks using Texas Instruments CC2652 coordinators with Z-Stack firmware.  
SONOFF ZBDongle-P has been used as the reference hardware.

[![Go Reference](https://pkg.go.dev/badge/github.com/marstid/goznp.svg)](https://pkg.go.dev/github.com/marstid/goznp)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

## Supported Hardware

| Coordinator | Status | Notes |
|-------------|--------|-------|
| SONOFF ZBDongle-P (CC2652P) | Tested | Primary development device |
| CC2652R/RB | Compatible | Lower TX power (max 5 dBm) |
| CC2652P/CC1352P | Compatible | Higher TX power (max 20 dBm) |

**Firmware**: [Koenkk Z-Stack 3.x.0](https://github.com/Koenkk/Z-Stack-firmware/tree/master/coordinator/Z-Stack_3.x.0)

---

# CLI Tool

The `goznp` command-line tool provides a complete interface for managing Zigbee networks.

## Installation

### Pre-built Binaries

Download pre-built binaries from [GitHub Releases](https://github.com/marstid/goznp/releases):

| Platform | Architecture | Download |
|----------|--------------|----------|
| Linux | amd64 | `goznp_*_linux_amd64.tar.gz` |
| Linux | arm64 | `goznp_*_linux_arm64.tar.gz` |
| macOS | Intel | `goznp_*_darwin_amd64.tar.gz` |
| macOS | Apple Silicon | `goznp_*_darwin_arm64.tar.gz` |
| Windows | amd64 | `goznp_*_windows_amd64.zip` |

### From Source

```bash
go install github.com/marstid/goznp/cmd/goznp@latest
```

## Quick Start

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

# List paired devices (shows custom names if set)
goznp device list
```

## Device Naming

Custom device names are stored on the coordinator and persist across restarts.

```bash
# Set a device name
goznp device name set --ieee 00:11:22:33:44:55:66:77 --name "Kitchen Light" --description "Main ceiling"

# List all named devices
goznp device name list

# Control devices by name, IEEE address, or network address
goznp device on --name "Kitchen Light"
goznp device on --ieee 00:11:22:33:44:55:66:77
goznp device on --addr 0xBE87
```

## Command Reference

### Network Management

| Command | Description |
|---------|-------------|
| `network form` | Create a new Zigbee network |
| `network channel` | View or change network channel |
| `network power` | View or set TX power level |
| `network topology` | Display mesh network topology |
| `reset` | Soft reset the adapter |
| `reset factory` | Factory reset (erase all configuration) |

### Device Management

| Command | Description |
|---------|-------------|
| `device pair` | Pair new devices (with auto-interview and binding) |
| `device permit` | Open/close network for pairing |
| `device list` | List paired devices (with custom names) |
| `device watch` | Monitor device join/leave events |
| `device remove` | Remove a device from the network |
| `device info` | Query device endpoints and clusters |
| `device status` | Read device status (manufacturer, model, battery) |
| `device name set/get/list/delete` | Manage custom device names |

### Device Control

| Command | Description |
|---------|-------------|
| `device on/off/toggle` | Control On/Off cluster |
| `device brightness` | Control Level cluster (dimmable lights) |
| `device color kelvin` | Set color temperature in Kelvin |
| `device color hue-sat` | Set color by hue/saturation |
| `device color xy` | Set color by CIE XY coordinates |
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

### Bindings and Reporting

| Command | Description |
|---------|-------------|
| `device bind` | Bind device cluster to coordinator |
| `device bindings` | List device binding table |
| `device configure-reporting` | Configure attribute reporting |

### Backup and Restore

| Command | Description |
|---------|-------------|
| `backup create` | Create network backup (JSON) |
| `backup restore` | Restore from backup |
| `backup show` | Display backup contents |

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

### Device Addressing

| Flag | Description | Example |
|------|-------------|---------|
| `--addr` | Network address (hex) | `--addr 0x1234` |
| `--ieee` | IEEE address | `--ieee 00:11:22:33:44:55:66:77` |
| `--name` | Custom device name | `--name "Kitchen Light"` |

## CLI Examples

### Control Smart Bulbs

```bash
# Name your devices for easy identification
goznp device name set --ieee 00:11:22:33:44:55:66:77 --name "Living Room"

# Control by name
goznp device on --name "Living Room"
goznp device brightness --name "Living Room" --level 200 --transition 500
goznp device color kelvin --name "Living Room" --kelvin 2700
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

# Restore (after replacing adapter)
goznp backup restore -i network-backup.json --force
```

---

# Go Library

Import goznp into your Go applications for custom Zigbee integrations.

## Installation

```bash
go get github.com/marstid/goznp
```

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

## Adapter API

### Network Operations

```go
// Get adapter and network info
info, _ := a.GetInfo(ctx)
network, _ := a.GetNetworkInfo(ctx)

// Form a new network
a.FormNetwork(ctx, channel, panID)

// Permit joining
a.PermitJoin(ctx, duration)
```

### Device Operations

```go
// List devices
devices, _ := a.GetDevices(ctx)

// Interview a device
result, _ := a.InterviewDevice(ctx, nwkAddr)

// Control devices
a.TurnOn(ctx, nwkAddr, endpoint)
a.TurnOff(ctx, nwkAddr, endpoint)
a.Toggle(ctx, nwkAddr, endpoint)
a.SetBrightness(ctx, nwkAddr, endpoint, level, transitionTime)
a.SetColorTemperature(ctx, nwkAddr, endpoint, mireds, transitionTime)
```

### Device Names

```go
// Set a custom name (stored on coordinator NVRAM)
a.SetDeviceName(ctx, ieeeAddr, "Kitchen Light", "Main ceiling light")

// Get a device name
name, _ := a.GetDeviceName(ctx, ieeeAddr)

// List all named devices
names, _ := a.ListDeviceNames(ctx)

// Delete a name
a.DeleteDeviceName(ctx, ieeeAddr)
```

### Event Handling

```go
// Register for device events
a.OnDeviceEvent(func(event adapter.DeviceEvent) {
    switch event.Type {
    case adapter.DeviceEventJoined:
        fmt.Printf("Device joined: %v\n", event.Device)
    case adapter.DeviceEventLeft:
        fmt.Printf("Device left: %v\n", event.IEEEAddr)
    }
})
```

## Package Structure

| Package | Description |
|---------|-------------|
| `pkg/adapter` | High-level adapter API for network and device management |
| `pkg/znp` | ZNP command interface (SYS, AF, ZDO, NV storage) |
| `pkg/zcl` | ZCL cluster definitions, frame encoding, data types |
| `pkg/unpi` | UNPI frame parsing and serialization |
| `pkg/serial` | Serial port enumeration and CC2652 detection |
| `pkg/backup` | Backup file format handling |

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

---

# Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      CLI (cmd/goznp)                       │
├─────────────────────────────────────────────────────────────┤
│                    Adapter (pkg/adapter)                    │
│  - Network management    - Device interview                 │
│  - Device control        - Device names (NVRAM)             │
│  - Backup/restore        - Event handling                   │
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

---

# Development

## Building

```bash
# Build CLI
go build -o bin/goznp ./cmd/goznp

# Run tests
go test ./...

# Format code
go fmt ./...
```

## Project Structure

```
goznp/
├── cmd/goznp/           # CLI application
├── pkg/
│   ├── adapter/         # High-level adapter API
│   ├── backup/          # Backup file format
│   ├── devices/         # Device database
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
