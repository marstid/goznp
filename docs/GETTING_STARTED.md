# Getting Started with goznp

This guide walks you through setting up a Zigbee coordinator, installing the CLI, and controlling your first device.

## Prerequisites

- Go 1.23 or later
- A supported Zigbee coordinator (CC2652R, CC2652P, or CC1352P) with [Z-Stack 3.x.0 firmware](https://github.com/Koenkk/Z-Stack-firmware)
- A USB cable connecting the coordinator to your computer

## 1. Install the CLI

```bash
go install github.com/marstid/goznp/cmd/goznp@latest
```

Verify the installation:

```bash
goznp version
```

## 2. Find Your Adapter

Plug in your Zigbee coordinator and scan for it:

```bash
goznp scan
```

You should see output like:

```
Available serial ports:
  /dev/tty.usbserial-110 (USB: VID=10C4 PID=EA60 Product=SONOFF Zigbee) [CC2652 Adapter]
```

Set the port as an environment variable so you don't have to pass `--port` every time:

```bash
# Linux
export GOZNP_PORT=/dev/ttyUSB0

# macOS
export GOZNP_PORT=/dev/tty.usbserial-110

# Windows (PowerShell)
$env:GOZNP_PORT="COM3"
```

> **Tip:** Add this to your shell profile (`~/.bashrc`, `~/.zshrc`) so it persists.

## 3. Connect to the Adapter

Test the connection and view adapter information:

```bash
goznp connect
```

You should see firmware version, capabilities, and network status. If the network shows "Not formed", continue to step 4. If the network is already formed, skip to step 5.

## 4. Form a Network

Create a new Zigbee network:

```bash
goznp network form --channel 15
```

This creates a network on Zigbee channel 15. Channels 11, 15, 20, and 25 are recommended as they don't overlap with common Wi-Fi channels.

Verify the network was created:

```bash
goznp connect
```

You should now see a PAN ID, channel, and the coordinator's IEEE address.

## 5. Pair a Device

Open the network for joining:

```bash
goznp network permit-join --seconds 60
```

Now put your Zigbee device into pairing mode. This varies by device — consult your device's manual. Common methods:
- **Smart plugs**: Hold the button for 5+ seconds until the LED blinks
- **Sensors**: Press the reset button 3 times quickly
- **Bulbs**: Power cycle 5 times (on for 2s, off for 2s)

Wait a few seconds, then verify the device joined:

```bash
goznp device list
```

You should see your new device with its IEEE address, network address, and manufacturer/model info.

## 6. Interview the Device

After pairing, interview the device to discover its capabilities:

```bash
goznp device interview --addr 0x1234
```

Replace `0x1234` with the network address shown in `device list`. The interview queries the device for its endpoints, supported clusters, and manufacturer information.

## 7. Control the Device

Now you can control the device. For a smart plug or light:

```bash
# Turn on
goznp device on --addr 0x1234 --endpoint 1

# Turn off
goznp device off --addr 0x1234 --endpoint 1

# Toggle
goznp device toggle --addr 0x1234 --endpoint 1
```

For lights with brightness control:

```bash
# Set to 50% brightness (0-254 scale)
goznp device brightness --addr 0x1234 --endpoint 1 --level 127
```

## 8. Name Your Device

Give your device a friendly name:

```bash
goznp device name set --ieee a4c138add6292966 --name "Kitchen Plug"
```

Names are stored on the coordinator and persist across reboots. View all named devices:

```bash
goznp device name list
```

## 9. Read Sensor Data

For sensor devices (temperature, humidity, etc.):

```bash
goznp device sensor --addr 0x5678 --endpoint 1
```

## Using the Library

For programmatic control, use the adapter package directly:

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

    // List devices
    devices, err := a.GetDevices(ctx)
    if err != nil {
        log.Fatal(err)
    }

    for _, dev := range devices {
        fmt.Printf("%s %s at 0x%04X\n", dev.Manufacturer, dev.Model, dev.NwkAddr)
    }

    // Turn on a device
    if err := a.TurnOn(ctx, 0x1234, 1); err != nil {
        log.Fatal(err)
    }
}
```

See the [examples](../examples/) directory for more detailed code samples.

## Next Steps

- Browse [examples/](../examples/) for control, sensor reading, and advanced integrations
- Read the [CLI reference](CLI.md) for all available commands
- Set up the [REST daemon](DAEMON.md) for HTTP-based control
- Read the [API reference](https://pkg.go.dev/github.com/marstid/goznp/pkg/adapter) on pkg.go.dev

## Troubleshooting

### "Permission denied" on serial port (Linux)

```bash
sudo usermod -a -G dialout $USER
```

Log out and back in for the change to take effect.

### "Timeout" connecting to adapter

- Verify the adapter is plugged in: `goznp scan`
- Check no other program is using the serial port (e.g., zigbee2mqtt)
- Try a different USB cable — some cables only carry power

### Device won't pair

- Make sure `permit-join` is active: `goznp network permit-join --seconds 120`
- Factory reset the device (refer to its manual)
- Move the device closer to the coordinator during pairing
- Some devices require multiple pairing attempts

### Garbled output

- Verify baud rate is 115200 (the default)
- Check that the adapter has Z-Stack 3.x.0 firmware, not older versions
