# Developing goznp

This guide provides deep-dive information for developers working on goznp internals.

## Architecture Overview

goznp is structured as a layered architecture, from high-level application API down to low-level protocol handling:

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI / Application                      │
│                      (cmd/goznp, external apps)                │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                       pkg/adapter/                           │
│   High-level Zigbee device management and coordination       │
│   • Device discovery and lifecycle management                │
│   • ZCL cluster abstraction                                │
│   • Automatic device interview                             │
│   • Quirks system for non-compliant devices                 │
└───────────────────────┬─────────────────────────────────────┘
                        │
          ┌─────────────┴─────────────┐
          │                           │
          ▼                           ▼
┌─────────────────────┐   ┌───────────────────────────────────┐
│     pkg/zcl/       │   │              pkg/znp/              │
│  Zigbee Cluster    │   │     ZNP Protocol Implementation     │
│  Library Layer     │   │   • SYS subsystem (system mgmt)      │
│  • Cluster defs    │   │   • ZDO subsystem (dev disco)        │
│  • Frame encode/   │   │   • AF subsystem (app framework)    │
│    decoding        │   │   • UTIL subsystem (utilities)       │
│  • Attribute types │   │   • Request/response matching       │
└─────────┬───────────┘   └───────────────────┬───────────────┘
          │                                   │
          │                                   │
          └───────────────┬───────────────────┘
                          │
                          ▼
                ┌─────────────────────┐
                │      pkg/unpi/       │
                │  UNPI Frame Protocol │
                │  • Frame parsing     │
                │  • Checksum val      │
                │  • Streaming parser  │
                └─────────┬───────────┘
                          │
                          ▼
                ┌─────────────────────┐
                │     pkg/serial/     │
                │   Serial Port I/O   │
                │   (go.bug.st/serial)│
                └─────────────────────┘
```

## Core Components

### Adapter (pkg/adapter/)

The Adapter is the primary API for applications. It provides a high-level abstraction over the low-level ZNP protocol.

**Key responsibilities:**
- Device lifecycle management (discovery, interview, cleanup)
- ZCL message routing and handling
- Transaction ID management
- Message deduplication (mesh networks retransmit)
- Automatic endpoint registration
- Quirks system integration

**Thread safety:** The Adapter is not thread-safe. Only one goroutine should use an Adapter instance at a time. If concurrent access is needed, synchronize externally.

**Context usage:** All Adapter methods accept `context.Context`. Operations can be cancelled mid-flight by cancelling the context.

### ZNP (pkg/znp/)

The ZNP package implements the Zigbee Network Processor protocol used by TI Z-Stack firmware.

**Subsystems:**
| Subsystem | Purpose | Commands |
|-----------|---------|----------|
| SYS | System management | Ping, Version, Reset, NV operations |
| ZDO | Zigbee Device Objects | Device discovery, NWK management |
| AF | Application Framework | Endpoint registration, data transfer |
| UTIL | Utilities | Device info, channel management, assoc mgmt |
| APPCNF | Application Configuration | BDB commissioning |
| NV | Non-Volatile storage | Configuration persistence |

**Request-Response Pattern:**

ZNP uses synchronous requests (SREQ) with synchronous responses (SRSP), plus asynchronous messages (AREQ):

```go
// Synchronous request-response
ctx := context.Background()
reqFrame := NewFrame(SREQ, SYS, CmdPing, nil)
respFrame, err := z.Request(ctx, reqFrame)

// Wait for async notification
matcher := FrameMatcher{
    Type:      AREQ,
    Subsystem: AF,
    CommandID: CmdAfIncomingMsg.ID,
}
msgFrame, err := z.waiter.WaitFor(ctx, matcher, 5*time.Second)
```

**The Waiter:**
The waiter matches incoming frames to pending requests. Each `Request()` call registers a matcher and waits for a matching frame (usually SRSP, sometimes AREQ).

### ZCL (pkg/zcl/)

The Zigbee Cluster Library defines standardized commands and attributes for device communication.

**Key concepts:**
- **Clusters:** Logical groupings of related functionality (e.g., On/Off, Color Control)
- **Commands:** Actions performed on clusters (e.g., "Turn on", "Set color")
- **Attributes:** Data points on devices (e.g., "Current level", "Color temperature")
- **Transaction Sequence Number (transSeqNum):** Identifies individual transactions

**Frame structure:**
```go
type Frame struct {
    FrameControl     uint8  // Frame type, direction, disable default response
    TransSeqNum      uint8  // Transaction ID (monotonically increasing)
    CommandID        uint8  // Cluster-specific command or global command
    Data             []byte // Command-specific payload
}
```

**Global commands** (work on all clusters):
- Read Attributes (0x00)
- Write Attributes (0x03)
- Configure Reporting (0x06)
- Discover Attributes (0x01)

### UNPI (pkg/unpi/)

Unified Network Processor Interface is the framing protocol that wraps ZNP commands for serial transmission.

**Frame structure:**
```
┌────┬──────────┬───────┬───────┬─────────────┬───────┐
│ SOF│   Len    │ Cmd0  │ Cmd1  │    Data     │  FCS  │
│0xFE │ uint16  │ uint8 │ uint8 │  0-250 B    │ uint8 │
└────┴──────────┴───────┴───────┴─────────────┴───────┘
```

- **SOF:** Start of Frame (always 0xFE)
- **Len:** Length of Data (big-endian)
- **Cmd0:** Frame type (4 bits) + Subsystem (4 bits)
  - Types: SREQ (0x00), SRSP (0x01), AREQ (0x02)
  - Subsystems: SYS (0x00), AF (0x01), ZDO (0x02), UTIL (0x04), NV (0x05)
- **Cmd1:** Specific command ID within subsystem
- **Data:** Command-specific payload
- **FCS:** Frame Check Sequence (simple checksum: 0xFF - sum(mod 256))

## Package Interactions

### Device Discovery Flow

When a device joins the network:

1. ZNP layer receives `TcDeviceInd` AREQ → `onDeviceJoin` callback
2. Adapter creates a new Device entry
3. Adapter starts device interview:
   - Read basic cluster info (model ID, manufacturer)
   - Look up device quirks by fingerprint
   - Read cluster endpoints
   - Discover clusters and attributes
4. Device moves to state `DeviceStateInterviewed`

### Sending a ZCL Command

```
Application
    │
    ▼
adapter.SendZCLFrame(dev, clusterID, cmd, data)
    │
    ▼
1. Allocate transSeqNum
2. Build ZCL frame
3. Call znp.AfDataRequest()
    │
    ▼
4. Build AF data request frame
5. Send via znp.Request()
    │
    ▼
6. Serialize to UNPI frame
7. Write to serial port
    │
    ▼
┌───────────────────────┐
│   Zigbee Coordinator  │
└───────────────────────┘
```

### Handling Incoming Messages

```
Serial bytes
    │
    ▼
unpi.Parser.Feed()
    │ (streaming parser)
    ▼
unpi.Frame (parsed)
    │
    ▼
znp.waiter.Resolve(frame) OR callbacks
    │
    ├─► If matches pending request → return to Request()
    └─► If async event → callback (onFrame, onDeviceJoin, etc.)
        │
        ▼
adapter.handleIncomingFrame()
    │
    ├─► Parse ZCL frame
    ├─► Route to device handler
    └─► Invoke quirks if applicable
        │
        ▼
Application callbacks (via options)
```

## Adding Device Types

When supporting a new device type:

1. **Pair the device** to get its fingerprint:
   ```bash
   goznp device pair
   goznp device info --addr 0x1234 --verbose
   ```

2. **Add to device database:**
   Edit `pkg/devices/devices.go`:
   ```go
   var deviceFingerprints = []deviceFingerprint{
       {
           Manufacturer: "MyBrand",
           Model:        "MyDevice123",
       },
       // ... existing devices
   }
   ```

3. **Set supported clusters** in the same file:
   ```go
   var deviceSupport = map[string]DeviceSupport{
       "MyBrand:MyDevice123": {
           Clusters: []zcl.Cluster{
               zcl.ClusterBasic,
               zcl.ClusterIdentify,
               zcl.ClusterOnOff,
               zcl.ClusterLevelControl,
           },
       },
   }
   ```

4. **Add quirks if needed** (see below)

## Adding ZCL Clusters

For new clusters not yet defined in `pkg/zcl/clusters.go`:

1. **Define cluster ID:**
   ```go
   const ClusterMyCustom zcl.Cluster = 0xFC00
   ```

2. **Define command IDs:**
   ```go
   const (
       CmdMyCustomAction uint8 = 0x00
       CmdMyCustomGet    uint8 = 0x01
   )
   ```

3. **Define attribute IDs:**
   ```go
   const (
       AttrMyCustomValue  uint16 = 0x0000
       AttrMyCustomConfig uint16 = 0x0001
   )
   ```

4. **Add frame building/conversion functions** in `pkg/zcl/frame.go` if needed

## Debugging

### Enable Debug Logging

Set environment variable for verbose output:
```bash
export GOZNP_DEBUG=1
goznp device list
```

### Serial Protocol Debugging

To see raw serial traffic:
```bash
# Using socat to tee the serial port
socat -v /dev/ttyUSB0,b115200,raw PTY,link=/tmp/ttyUSB0.pty,raw,echo=0
# Then connect to /tmp/ttyUSB0.pty
```

### Common Issues

**Transactions timing out:**
- Check transaction ID handling (should monotonically increase)
- Verify frame FCS checksums
- Ensure context isn't cancelled too early

**Device not found:**
- Verify device is paired (check `goznp device list`)
- Check coordinator is in Coordinator state (not Router/EndDevice)
- Verify network channel matches

**Garbage data in responses:**
- Check byte order (ZNP uses little-endian for multi-byte values)
- Ensure IEEE addresses are interpreted correctly
- Verify response frame lengths

## Concurrency Model

### Goroutines in Adapter

1. **Main application goroutine:** Calls Adapter methods
2. **ZNP readLoop:** Reads bytes from serial port, feeds to parser
3. **ZNP frameLoop:** Processes incoming frames, resolves waiters, fires callbacks
4. **Device interview goroutine:** Started for each device during discovery

### Synchronization

- **Adapter:** One goroutine at a time (external sync needed for concurrent use)
- **ZNP:** Thread-safe for method calls (waiter is safe for concurrent Wait calls)
- **Waiter:** Thread-safe (uses mutex and channels)
- **Dedupe cache:** Thread-safe (uses mutex)

### Memory Management

- Device objects are retained in Adapter's device map until removed
- Old device states are garbage collected after removal
- Message deduplication cache automatically purges old entries

## Performance Considerations

### Serial Baud Rate

The recommended baud rate for Z-Stack coordinators is 115,200.

### Throughput

- ZNP can handle ~200 commands/second on typical hardware
- Battery-powered devices may delay responses to conserve power
- Mesh routing can add latency for multi-hop communication

### Memory Usage

- Each device consumes ~1-2KB of memory (struct + state)
- Deduplication cache size: ~N messages × 24 bytes × window time
- Per-transaction buffers are ~256 bytes

## Testing Strategies

### Unit Tests (without hardware)

- Use mock ZNP implementations (see `pkg/znp/sys_test.go`)
- Test frame encoding/decoding without real serial
- Test transaction ID allocation
- Test deduplication logic

### Integration Tests (with hardware)

- Requires actual coordinator and test devices
- Test end-to-end command flows
- Verify timing and state transitions
- Test real-world device quirks

### Coverage Goals

- pkg/unpi: 90%+
- pkg/znp: 50%+
- pkg/zcl: 60%+
- pkg/adapter: 50%+

## Further Reading

- [Zigbee Specification](https://zigbeealliance.org/zigbee-specification/)
- [TI ZNP Interface Guide](https://dev.ti.com/tirex/explore/file?alias=z-stack_home_1.02.001)
- [Zigbee Cluster Library Specification](https://zigbeealliance.org/zigbee-cluster-library-specification/)
