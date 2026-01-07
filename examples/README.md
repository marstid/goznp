# goznp Examples

This directory contains example programs demonstrating how to use goznp to control Zigbee networks and devices.

## Running the Examples

All examples require a compatible Zigbee coordinator (e.g., CC2652R) connected via USB serial port.

### Set Serial Port

Before running any example, set the `GOZNP_PORT` environment variable to your serial port:

**Linux:**
```bash
export GOZNP_PORT=/dev/ttyUSB0
# or
export GOZNP_PORT=/dev/ttyACM0
```

**macOS:**
```bash
export GOZNP_PORT=/dev/tty.usbserial-110
```

**Windows:**
```powershell
$env:GOZNP_PORT="COM3"
```

### Running an Example

```bash
# Run any example directly
go run examples/basics/device_list.go

# Or build then run
go build -o myapp examples/basics/device_list.go
./myapp
```

### Using the existing goznp CLI

Many examples are also available as CLI commands:

```bash
# Build the CLI
make build

# List devices
./bin/goznp device list

# Control a device
./bin/goznp device on --addr 0x1234 --endpoint 1
./bin/goznp device off --addr 0x1234 --endpoint 1
```

## Example Categories

### Basics

Getting started examples for fundamental operations:

- `device_list.go` - List all paired devices on the network
- `network_info.go` - Get network status and coordinator information
- `device_interview.go` - Interview a device to discover its capabilities

### Control

Device control examples for common device types:

- `light_control.go` - Control smart bulbs (on/off, brightness, color)
- `sensor_read.go` - Read sensor data (temperature, humidity, occupancy)
- `switch_monitor.go` - Monitor and respond to switch events
- `group_control.go` - Control multiple devices with group messages

### Advanced

Advanced integration examples:

- `mqtt_bridge.go` - Bridge Zigbee events to MQTT (requires mqtt client library)
- `network_diagnostics.go` - Perform network health checks and diagnostics
- `ota_upgrade.go` - Perform OTA firmware updates on compatible devices
- `web_server.go` - Simple HTTP API for device control (requires web framework)

## Development Notes

### Using `// +build ignore`

Note that all example files use `// +build ignore` (or `//go:build ignore` in Go 1.18+) at the top, which prevents them from being included in package builds. This is intentional and allows you to run them directly as standalone programs.

### Error Handling

The examples demonstrate basic error handling patterns. Production code typically needs:

- Comprehensive error logging
- Retry logic for transient failures
- Graceful degradation when devices are unavailable
- State persistence across restarts

### Context Usage

All examples use `context.Context` for timeout control:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

Adjust timeouts based on your network conditions and device capabilities.

### Device Addresses

You'll need to know the network address (`nwkAddr`) and endpoint of your target device. Use `device_list.go` or the CLI to discover this:

```bash
./bin/goznp device list
```

Network addresses can change when devices rejoin, so in production code you should track devices by their IEEE address which is permanent.

## Troubleshooting

### "Device not found" errors

- Verify the device is paired: check with `./bin/goznp device list`
- The network address may have changed; refresh the device list
- The device may be asleep (battery-powered) - try waking it first

### "Timeout" errors

- Increase the context timeout
- Check the coordinator is powered on
- Verify network channel and PAN ID match expectations

### "Permission denied" on serial port

**Linux:**
```bash
sudo usermod -a -G dialout $USER
# Log out and back in for changes to take effect
```

**macOS:** No action needed typically.

**Windows:** Run as Administrator if needed.

### Garbled data on serial port

- Verify baud rate is 115200 (goznp default)
- Check no other process is using the serial port
- Try a different USB cable (some cables only carry power)

## Contributing Examples

We welcome contributions of new examples. Guidelines:

1. Keep examples simple and focused on a single concept
2. Use realistic error handling but don't over-engineer
3. Include comments explaining key decisions
4. Test on real hardware when possible
5. Update this README with new examples

## Related Resources

- [goznp Documentation](https://github.com/marstid/goznp)
- [DEVELOPING.md](../DEVELOPING.md) - Architecture and development guide
- [Zigbee Cluster Library Specification](https://zigbeealliance.org/zigbee-cluster-library-specification/)
- [TI Z-Stack Documentation](https://dev.ti.com/tirex/explore/file?alias=z-stack_home_1.02.001)
