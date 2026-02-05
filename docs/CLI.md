# goznp CLI Reference

Command-line tool for interacting with Zigbee adapters via the Z-Stack Network Processor (ZNP) protocol.

## Global Options

All commands that communicate with the adapter accept:

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `-p`, `--port` | `GOZNP_PORT` | - | Serial port path (required) |
| `-b`, `--baud` | - | 115200 | Baud rate |

Set `GOZNP_PORT` to avoid passing `--port` every time:

```bash
export GOZNP_PORT=/dev/ttyUSB0
```

## Device Addressing

Many commands accept a target device. Devices can be specified by:

| Flag | Format | Example |
|------|--------|---------|
| `--name` | Friendly name | `--name "Kitchen Light"` |
| `--ieee` | IEEE address | `--ieee 00:11:22:33:44:55:66:77` |
| `--addr` | Network address (hex) | `--addr 0x1234` |

Priority: `--name` > `--ieee` > `--addr`

## Commands

### Top-Level

| Command | Description |
|---------|-------------|
| `goznp scan` | Scan for available Zigbee adapters on USB |
| `goznp connect` | Connect to adapter and display firmware/network info |
| `goznp ping` | Ping the adapter and measure response time |
| `goznp reset` | Perform soft reset of the adapter |
| `goznp reset factory` | Factory reset (erases all configuration) |
| `goznp version` | Print version information |

### Device Management (`goznp device`)

| Command | Description |
|---------|-------------|
| `device list` | List paired devices |
| `device list --interview` | List with full device interview (slower) |
| `device permit` | Open network for device joining |
| `device pair` | Pair a device with auto-interview and binding |
| `device watch` | Watch for device join/leave events in real-time |
| `device status` | Query device status (manufacturer, on/off, battery) |
| `device info` | Query device endpoints and clusters |
| `device interview` | Interview device to discover capabilities |
| `device remove` | Remove a device from the network |

**Device Control:**

| Command | Description |
|---------|-------------|
| `device on` | Turn device on |
| `device off` | Turn device off |
| `device toggle` | Toggle device state |
| `device brightness` | Get or set brightness (0-254) |
| `device identify` | Make device flash/blink to locate it |
| `device sensor` | Read temperature, humidity, battery |

**Device Names:**

| Command | Description |
|---------|-------------|
| `device name set` | Set friendly name for a device |
| `device name get` | Get device name |
| `device name list` | List all named devices |
| `device name delete` | Delete a device name |

**Binding and Reporting:**

| Command | Description |
|---------|-------------|
| `device bind` | Bind device cluster to coordinator |
| `device bind --unbind` | Unbind a cluster |
| `device report` | Configure attribute reporting |
| `device bindings` | List active bindings |

**Scenes:**

| Command | Description |
|---------|-------------|
| `device scene store` | Store current state as a scene |
| `device scene recall` | Recall a stored scene |
| `device scene remove` | Remove a scene |
| `device scene list` | List scenes on a device |

### Color Control (`goznp color`)

| Command | Description |
|---------|-------------|
| `color hue` | Set hue and saturation |
| `color xy` | Set CIE x,y color coordinates |
| `color rgb` | Set color from RGB values |
| `color temp` | Set color temperature (Kelvin) |
| `color read` | Read current color state |

### Group Control (`goznp groups`)

| Command | Description |
|---------|-------------|
| `groups add` | Add a device to a group |
| `groups remove` | Remove a device from a group |
| `groups list` | List group memberships for a device |
| `groups on` | Turn on all devices in a group |
| `groups off` | Turn off all devices in a group |
| `groups toggle` | Toggle all devices in a group |
| `groups brightness` | Set brightness for a group |
| `groups recall` | Recall a scene on a group |

### Network Management (`goznp network`)

| Command | Description |
|---------|-------------|
| `network form` | Form a new Zigbee network |
| `network channel` | Show or change network channel |
| `network power` | Show or set TX power (dBm) |
| `network profile` | Show registered application profiles |
| `network topology` | Show mesh network topology |
| `network tree` | Display topology as ASCII tree |
| `network health` | Show comprehensive network health |
| `network routes` | Show routing table |

### Backup/Restore (`goznp backup`)

| Command | Description |
|---------|-------------|
| `backup create` | Create JSON backup of adapter configuration |
| `backup restore` | Restore from a backup file |
| `backup show` | Display backup file contents |
| `backup debug` | Dump raw NVRAM content |

### OTA Firmware Updates (`goznp ota`)

| Command | Description |
|---------|-------------|
| `ota list` | List available OTA firmware images |
| `ota info` | Display detailed image information |
| `ota check` | Check image compatibility with a device |
| `ota server` | Start OTA server for firmware updates |
| `ota image` | Get OTA firmware info from a device |

### Device Quirks (`goznp quirks`)

| Command | Description |
|---------|-------------|
| `quirks list` | List all registered quirks |
| `quirks device` | Show quirks for a specific device |
| `quirks match` | Test quirk matching without hardware |

## Examples

### First Setup

```bash
# Find the adapter
goznp scan

# Connect and verify
goznp connect --port /dev/ttyUSB0

# Form a network
goznp network form --channel 15

# Pair a device (auto-interview + auto-bind)
goznp device pair
```

### Daily Use

```bash
# Control by name
goznp device on --name "Kitchen Light"
goznp device off --name "Kitchen Light"
goznp device brightness --name "Kitchen Light" --level 128

# Control a group
goznp groups on --group 1
goznp groups brightness --group 1 --level 200

# Check sensors
goznp device sensor --name "Temperature Sensor"

# View network topology
goznp network tree
```

### Backup

```bash
# Create a backup
goznp backup create --output backup.json

# View backup contents
goznp backup show --input backup.json

# Restore (with confirmation)
goznp backup restore --input backup.json
```
