# CLI UX Improvement Implementation Plan

## Overview
Comprehensive refactoring of the `goznp` CLI tool to improve user experience, consistency, and usability.

**Status:** Ready for implementation  
**Breaking Changes:** Yes (approved)  
**Backward Compatibility:** Not required (approved)

---

## Priority 1: Command Renaming & Structure (Quick Wins)

### 1.1 Rename Top-Level Commands

**Changes:**
```
old: goznp info          →  new: goznp connect
old: goznp list          →  new: goznp scan
old: goznp reset         →  new: goznp soft-reset
old: goznp reset factory →  new: goznp factory-reset
```

**Files to modify:** `cmd/goznp/main.go`

**Implementation:**
1. Rename `var listCmd` → `var scanCmd`, update `Use` field to `"scan"`
2. Rename `var infoCmd` → `var connectCmd`, update `Use` field to `"connect"`
3. Rename `var resetCmd` → `var softResetCmd`
4. Keep `resetCmd` but make it a parent command with subcommands

**Code example:**
```go
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for available Zigbee adapters",
	Long:  "Shows USB ports with VID/PID info and highlights detected CC2652/SONOFF adapters",
	Run: func(_ *cobra.Command, _ []string) {
		if err := runScan(); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		},
}
```

---

## Priority 2: Move Groups and Colors to Top-Level

### 2.1 Create Top-Level Groups Commands

**New command structure:**
```
OLD: goznp device group add        NEW: goznp groups add
OLD: goznp device group list       NEW: goznp groups list
OLD: goznp device group on         NEW: goznp groups on
OLD: goznp device group off        NEW: goznp groups off
```

**New file:** `cmd/goznp/groups.go`

**Implementation:**
1. Create new file `cmd/goznp/groups.go`
2. Move logic from `cmd/goznp/device_groups.go`
3. Update all subcommands to use `groups` prefix
4. Remove from `deviceCmd` subcommands

### 2.2 Create Top-Level Colors Commands

**New command structure:**
```
OLD: goznp device color on        NEW: goznp color on
OLD: goznp device color kelvin     NEW: goznp color temperature
OLD: goznp device color hsl        NEW: goznp color hsl
OLD: goznp device color xy          NEW: goznp color xy
```

**New file:** `cmd/goznp/color.go`

**Implementation:**
1. Create new file `cmd/goznp/color.go` (or rename `device_color.go`)
2. Copy all color commands from `device_color.go`
3. Rename `kelvin` → `temperature` for clarity
4. Update help text with examples

**Code example:**
```go
var colorCmd = &cobra.Command{
	Use:   "color",
	Short: "Control device color and temperature",
	Long: "Commands for controlling RGB and CCT lights including on/off, " +
		"hue/saturation, XY coordinates, and color temperature",
}

var colorOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Turn color light on",
	Long:  "Send On command to a color-capable device",
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runColorControl(ctx, "on")
	},
}

var colorTemperatureCmd = &cobra.Command{
	Use:   "temperature [kelvin]",
	Short: "Set color temperature",
	Long: `Set the color temperature of a CCT (White Spectrum) light.
The kelvin parameter should be between 2700 (warm white) and 6500 (cool white).

Examples:
  goznp color temperature --name "Kitchen" 2700
  goznp color temperature --name " Living Room" 4000`,
	RunE: func(_ *cobra.Command, args []string) error {
		ctx := setupSignalHandler()
		return runColorTemperature(ctx, args)
	},
}
```

---

## Priority 3: Improved Error Messages

### 3.1 Update Error Messages Across All Commands

**Pattern for improved errors:**
- Always suggest what to do next
- Show examples when relevant
- Distinguish between user error vs system error

**Files to update:** All `cmd/goznp/*.go` files with CLI commands

**Example improvements:**

**Before:**
```go
if portPath == "" {
    return "", fmt.Errorf("serial port not specified")
}
```

**After:**
```go
if portPath == "" && os.Getenv("GOZNP_PORT") == "" {
    return "", fmt.Errorf("serial port not specified\n\n" +
        "Usage:\n" +
        "  goznp [command] --port <path>\n\n" +
        "Examples:\n" +
        "  goznp scan\n" +
        "  goznp connect --port /dev/tty.usbserial-110\n" +
        "  export GOZNP_PORT=/dev/tty.usbserial-110\n")
}
```

**Device not found error:**
```go
if !found {
    return 0, fmt.Errorf("device %q not found\n\n" +
        "Available devices:\n" +
        "  Use 'goznp devices list' to see all devices\n\n" +
        "Or specify by:\n" +
        "  --name \"Kitchen Light\"  (recommended)\n" +
        "  --ieee 00:11:22:33:44:55:66:77\n" +
        "  --addr 0x1234",
        deviceName)
}
```

**Commands to update:**
- All `runRun()` functions in all command files
- `resolveDeviceAddr()` if kept
- Connection errors in all commands

---

## Priority 4: Centralized Flag Management

### 4.1 Create `flags.go` Helper File

**New file:** `cmd/goznp/flags.go`

**Purpose:** Centralize common flag definitions and helper functions

**Implementation:**

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/znp"
)

// AddConnectionFlags adds -p and -b flags to a command
func AddConnectionFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&portPath, "port", "p", "",
		"Serial port path (or set GOZNP_PORT env var)")
	cmd.Flags().IntVarP(&baudRate, "baud", "b", 115200,
		"Baud rate (default: 115200)")
}

// AddDeviceFlags adds device addressing flags to a command
func AddDeviceFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&deviceAddr, "addr", "",
		"Device network address (hex, e.g., 0x1234)")
	cmd.Flags().StringVar(&deviceIEEE, "ieee", "",
		"Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
	cmd.Flags().StringVar(&deviceName, "name", "",
		"Device name (use --name list to see available names)")
	cmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1,
		"Device endpoint (default: 1)")
}

// AddGroupFlags adds group-related flags
func AddGroupFlags(cmd *cobra.Command) {
	cmd.Flags().Uint16Var(&groupID, "group", 0,
		"Group ID (0-65535)")
	cmd.Flags().StringVar(&groupName, "group-name", "",
		"Group name (user-friendly)")
}

// AddColorFlags adds color-related flags
func AddColorFlags(cmd *cobra.Command) {
	cmd.Flags().Uint8Var(&hue, "hue", 0,
		"Hue (0-254, color wheel position)")
	cmd.Flags().Uint8Var(&saturation, "saturation", 254,
		"Saturation (0-254, 0=white, 254=full color)")
	cmd.Flags().Uint16Var(&x, "x", 0,
		"X coordinate (0-65279)")
	cmd.Flags().Uint16Var(&y, "y", 0,
		"Y coordinate (0-65279)")
}
```

### 4.2 Update All Commands to Use Centralized Flags

**Pattern:**
Replace manual flag registration with centralized helpers.

**Example before:**
```go
deviceOnCmd.Flags().StringVarP(&portPath, "port", "p", "", "...")
deviceOnCmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "...")
deviceOnCmd.Flags().StringVar(&deviceAddr, "addr", "", "...")
deviceOnCmd.Flags().StringVar(&deviceIEEE, "ieee", "", "...")
deviceOnCmd.Flags().StringVar(&deviceName, "name", "", "...")
deviceOnCmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1, "...")
```

**Example after:**
```go
AddConnectionFlags(deviceOnCmd)
AddDeviceFlags(deviceOnCmd)
```

**Files to update:**
- `cmd/goznp/main.go` - connection-related commands
- `cmd/goznp/device_onoff.go`
- `cmd/goznp/device_color.go`
- `cmd/goznp/device_groups.go`
- `cmd/goznp/device_sensors.go`
- `cmd/goznp/device_power.go`
- `cmd/goznp/device_binding.go`
- `cmd/goznp/device_identify.go`
- `cmd/goznp/device_scenes.go`
- `cmd/goznp/backup.go`
- `cmd/goznp/network.go`
- `cmd/goznp/ota.go`
- `cmd/goznp/quirks.go`

---

## Priority 5: Update Help Text and Examples

### 5.1 Update Command Short and Long Descriptions

**Guidelines:**
- Use verbs for top-level commands (`scan`, `connect`, `devices`)
- Use clear, user-friendly language
- Include practical examples in `Long` description
- Prioritize `--name` flag in examples

**Example improvements:**

**Device on command:**
```go
var deviceOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Turn device on",
	Long: `Send On command to a Zigbee device.

The device can be specified by:
  - Custom name (recommended): --name "Kitchen Light"
  - IEEE address: --ieee 00:11:22:33:44:55:66:77
  - Network address: --addr 0x1234

Examples:
  # Turn on by name (recommended)
  goznp devices on --name "Kitchen Light"

  # Turn on by address
  goznp devices on --addr 0x1234

  # Turn on all devices in a group
  goznp groups on --group 1`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceControl(ctx, "on")
	},
}
```

### 5.2 Add Examples to All Commands

**Commands that need examples:**
- All device control commands (on, off, toggle, brightness, color)
- All group commands
- Pairing and device discovery
- Network commands (form, channel, power)

---

## Priority 6: Add Defaults to Common Commands

### 6.1 Add Default Values

**Pair command defaults:**
```go
var devicePairCmd = &cobra.Command{
	Use:   "pair",
	Short: "Pair a new device to the network",
	Long: `Open network for pairing, watch for device joins, interview them,
and bind their clusters to the coordinator.

This command:
1. Opens the network for 180 seconds (configurable with --timeout)
2. Watches for device join events
3. Interviews each new device (endpoints, clusters, attributes)
4. Bind clusters to coordinator for automatic reporting
5. Automatically closes the network when done

Default behavior:
- Timeout: 180 seconds
- Auto-interview: enabled
- Auto-bind: enabled
- Use --no-bind to skip binding (some devices don't support it)

Put your device in pairing mode (hold reset button for 5 seconds)
before running this command.

Examples:
  # Default pairing
  goznp devices pair

  # Pair faster (60 second timeout)
  goznp devices pair --timeout 60

  # Pair without auto-binding
  goznp devices pair --no-bind`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDevicePair(ctx)
	},
}
```

**Network form defaults:**
```go
var networkFormCmd = &cobra.Command{
	Use:   "form",
	Short: "Form a new Zigbee network",
	Long: `Form a new Zigbee network as coordinator.

Default configuration:
- Channel: 15 (Home Automation default)
- PAN ID: Random (auto-generated)
- Profile: HA (Home Automation)

Forming a network will:
1. Clear any existing network state
2. Configure device as coordinator
3. Generate network keys
4. Start the Zigbee stack

Examples:
  # Form network with defaults
  goznp network form

  # Form on specific channel
  goznp network form --channel 11

  # Form with custom PAN ID
  goznp network form --channel 15 --pan-id 0x1234`,
}
```

---

## Implementation Order

### Phase 1: Foundation (30-45 min)
1. ✅ Create `cmd/goznp/flags.go` with centralized helpers
2. ✅ Update error messages in all `Run()` functions
3. ✅ Add `GetPortPath()` with improved error text

### Phase 2: Structure (45-60 min)
1. ✅ Rename commands (`info` → `connect`, `list` → `scan`)
2. ✅ Create `cmd/goznp/groups.go` (move from device_groups.go)
3. ✅ Create `cmd/goznp/color.go` (rename from device_color.go)
4. ✅ Update `main.go` to register new commands
5. ✅ Remove group/color from deviceCmd subcommands

### Phase 3: Apply Centralized Flags (30-45 min)
1. ✅ Add `AddConnectionFlags()` calls to all commands
2. ✅ Add `AddDeviceFlags()` calls to device commands
3. ✅ Add `AddGroupFlags()` calls to group commands
4. ✅ Add `AddColorFlags()` calls to color commands
5. ✅ Remove manual flag registrations

### Phase 4: Update Help Text (45-60 min)
1. ✅ Update all `Short` and `Long` descriptions
2. ✅ Add examples to all commands
3. ✅ Prefer `--name` flag in all examples

### Phase 5: Add Defaults (15-30 min)
1. ✅ Add default values to `pair` command
2. ✅ Add default values to `network form` command

### Phase 6: Test & Verify (15-30 min)
1. ✅ Run `goznp --help` check output
2. ✅ Run `goznp scan` verify works
3. ✅ Run `goznp connect --port <path>` verify works
4. ✅ Test all command help outputs
5. ✅ Verify binary builds
6. ✅ Run all existing tests pass

---

## File Changes Summary

**New Files:**
- `cmd/goznp/flags.go` - Centralized flag helpers
- `cmd/goznp/groups.go` - Top-level group commands
- `cmd/goznp/color.go` - Top-level color commands

**Modified Files:**
- `cmd/goznp/main.go` - Command registration, renaming
- `cmd/goznp/device.go` - Device commands, improved errors
- `cmd/goznp/device_onoff.go` - Use centralized flags
- `cmd/goznp/device_color.go` - Use centralized flags, possibly move to color.go
- `cmd/goznp/device_groups.go` - Use centralized flags, move to groups.go
- `cmd/goznp/device_sensors.go` - Use centralized flags
- `cmd/goznp/device_power.go` - Use centralized flags
- `cmd/goznp/device_binding.go` - Use centralized flags
- `cmd/goznp/device_identify.go` - Use centralized flags
- `cmd/goznp/device_scenes.go` - Use centralized flags
- `cmd/goznp/backup.go` - Use centralized flags
- `cmd/goznp/network.go` - Use centralized flags
- `cmd/goznp/ota.go` - Use centralized flags
- `cmd/goznp/quirks.go` - Use centralized flags

**Estimated Total Time:** 3-4 hours

---

## Before Starting

1. **Backup current implementation**
   ```bash
   cd /Users/martinstidelius/code/goznp
   git add -A
   git commit -m "Backup before CLI refactor"
   ```

2. **Create feature branch**
   ```bash
   git checkout -b feature/cli-ux-improvements
   ```

---

## After Implementation

1. **Update README.md** with new command structure
2. **Update documentation** with examples using new commands
3. **Test thoroughly** with actual hardware

---

## Example CLI Usage After Refactoring

```bash
# Scan for adapters
goznp scan

# Connect and view info
goznp connect --port /dev/tty.usbserial-110

# Form network
goznp network form --channel 15

# Pair device
goznp devices pair

# List devices with names
goznp devices list

# Control by name (recommended)
goznp devices on --name "Kitchen Light"
goznp devices brightness --name "Kitchen Light" 180

# Group control
goznp groups add --name "Living Room" --group 1
goznp groups on --group 1

# Color control
goznp color on --name "Bedroom Bulb"
goznp color temperature --name "Bedroom Bulb" 2700
goznp color hsl --name "Bedroom Bulb" --hue 120 --saturation 200

# Sensor data
goznp devices sensor --name "Temp Sensor"

# Network info
goznp network topology
```
