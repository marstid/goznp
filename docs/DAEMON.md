# goznpd - Zigbee Daemon

A daemon that provides a REST API for managing Zigbee devices via a Z-Stack network coordinator. It maintains in-memory state for devices and their sensor measurements, and exposes a simple HTTP interface for querying and controlling devices.

## Features

- **Device Discovery**: Automatic device detection from coordinator
- **Device State Management**: Cached device state (on/off, brightness, etc.)
- **Sensor Data Caching**: Latest readings from sensors (temperature, humidity, power, battery, etc.)
- **ZCL Message Listener**: Real-time processing of incoming Zigbee messages and attribute reports
- **HTTP REST API**: Simple REST interface for device control and monitoring
- **Slug-Based Addressing**: Access devices by human-readable slugs or IEEE addresses
- **Event System**: Pub/sub event bus for device join/leave/state changes
- **Graceful Shutdown**: Proper signal handling (SIGTERM, SIGINT)
- **Logging**: Structured JSON logging with configurable levels

## Configuration

Configuration is done via environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GOZNP_PORT` | Yes | - | Serial port path (e.g., `/dev/tty.usbserial-110`) |
| `GOZNP_BAUD_RATE` | No | 115200 | Serial baud rate |
| `GOZNP_HTTP_ADDR` | No | `:8080` | HTTP bind address |
| `GOZNP_LOG_LEVEL` | No | `info` | Log level (debug, info, warn, error) |
| `GOZNP_POLL_INTERVAL` | No | `5m` | Interval for polling sleepy devices |
| `GOZNP_LISTEN_TIMEOUT` | No | `100ms` | Message listener timeout |
| `GOZNP_PERMIT_JOIN_DURATION` | No | `255` | Join window duration (255 = always open) |

## Building

```bash
make build-daemon
```

Or using go directly:

```bash
go build -o bin/goznpd ./cmd/goznpd
```

## Running

Start the daemon:

```bash
GOZNP_PORT=/dev/tty.usbserial-110 ./bin/goznpd
```

The daemon will:
1. Open the Zigbee adapter
2. Discover existing devices
3. Start the HTTP server
4. Listen for incoming device messages
5. Wait for shutdown signals

## API Endpoints

### Health Check

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "ok",
  "timestamp": "2026-01-09T17:30:15Z"
}
```

### List All Devices

```bash
curl http://localhost:8080/api/v1/devices
```

Response:
```json
{
  "devices": [
    {
      "ieee_addr": "a4:c1:38:ad:d6:29:29:66",
      "nwk_addr": "0xE2CA",
      "slug": "outdoor-plug",
      "name": "Outdoor Plug",
      "model": "TS011F",
      "manufacturer": "Tuya",
      "last_seen": "2026-01-09T17:30:10Z",
      "joined_at": "2026-01-08T10:23:15Z",
      "is_router": false,
      "is_battery_powered": false
    }
  ],
  "count": 6
}
```

### Get Device Details

By slug (human-readable):
```bash
curl http://localhost:8080/api/v1/devices/outdoor-plug
```

By IEEE address:
```bash
curl http://localhost:8080/api/v1/devices/a4c138add6292966
```

Response:
```json
{
  "ieee_addr": "a4:c1:38:ad:d6:29:29:66",
  "nwk_addr": "0xE2CA",
  "slug": "outdoor-plug",
  "name": "Optional friendly name",
  "model": "TS011F",
  "manufacturer": "Tuya",
  "capabilities": "0x00",
  "endpoints": [1, 242],
  "last_seen": "2026-01-09T17:30:10Z",
  "joined_at": "2026-01-08T10:23:15Z",
  "state": {
    "on_off": true,
    "brightness": null,
    "color_temp": null,
    "last_updated": "2026-01-09T17:30:05Z"
  }
}
```

### Get Device State

```bash
curl http://localhost:8080/api/v1/devices/outdoor-plug/state
```

Response:
```json
{
  "on_off": true,
  "brightness": 128,
  "color_temp": 4500,
  "color_hue": 120,
  "color_saturation": 200,
  "temperature": null,
  "humidity": null,
  "last_updated": "2026-01-09T17:30:15Z"
}
```

### Get Sensor Data

```bash
curl http://localhost:8080/api/v1/devices/outdoor-plug/sensor
```

Response:
```json
{
  "ieee_addr": "a4:c1:38:ad:d6:29:29:66",
  "slug": "outdoor-plug",
  "last_updated": "2026-01-09T17:30:15Z",
  "reading": {
    "temperature": {
      "value": 20.5,
      "timestamp": "2026-01-09T17:30:12Z",
      "received_at": "2026-01-09T17:30:12Z"
    },
    "humidity": {
      "value": 55.2,
      "timestamp": "2026-01-09T17:30:10Z",
      "received_at": "2026-01-09T17:30:10Z"
    },
    "voltage": {
      "value": 220.0,
      "timestamp": "2026-01-09T17:30:15Z",
      "received_at": "2026-01-09T17:30:15Z"
    },
    "current": {
      "value": 5.5,
      "timestamp": "2026-01-09T17:30:15Z",
      "received_at": "2026-01-09T17:30:15Z"
    },
    "power": {
      "value": 1210.0,
      "timestamp": "2026-01-09T17:30:15Z",
      "received_at": "202-01-09T17:30:15Z"
    }
  }
}
```

### Set Device Name

```bash
curl -X POST http://localhost:8080/api/v1/devices/outdoor-plug/name \
  -H "Content-Type: application/json" \
  -d '{"name": "Garage Outlet"}'
```

Response:
```json
{
  "success": true,
  "slug": "garage-outlet"
}
```

### Device Control

**Turn on:**
```bash
curl -X POST http://localhost:8080/api/v1/devices/outdoor-plug/on
```

**Turn off:**
```bash
curl -X POST http://localhost:8080/api/v1/devices/outdoor-plug/off
```

**Toggle:**
```bash
curl -X POST http://localhost:8080/api/v1/devices/outdoor-plug/toggle
```

**Set brightness:**
```bash
curl -X POST http://localhost:8080/api/v1/devices/outdoor-plug/brightness \
  -H "Content-Type: application/json" \
  -d '{"level": 128}'
```

**Set color temperature:**
```bash
curl -X POST http://localhost:8080/api/v1/devices/outdoor-plug/color-temp \
  -H "Content-Type: application/json" \
  -d '{"kelvin": 3000}'
```

**Identify device:**
```bash
curl -X POST http://localhost:8080/api/v1/devices/outdoor-plug/identify \
  -H "Content-Type: application/json" \
  -d '{"time": 10}'
```

The device will flash or blink for the specified duration (seconds).

## Device Addressing

Devices can be addressed in two ways:

1. **By slug** (human-readable name-based URL):
   ```json
   /api/v1/devices/outdoor-plug
   /api/v1/devices/garage-outlet
   ```

2. **By IEEE address** (hex string with or without colons):
   ```json
   /api/v1/devices/a4c138add6292966
   /api/v1/devices/a4:c1:38:ad:d6:29:29:66
   ```

**Slug Format Rules:**
- Lowercase only
- Alphanumeric characters + hyphens
- Spaces and other characters become hyphens
- No consecutive hyphens
- No leading/trailing hyphens
- Max 64 characters

**Examples:**
- `"Outdoor Plug"` → `"outdoor-plug"`
- `"Garage Outlet 2"` → `"garage-outlet-2"`
- `"Front Door 🚪"` → `"front-door"`
- `"Living_Room_Light"` → `"living-room-light"`

If no name is set, defaults to: `"device-<ieee-addr>"` (e.g., `"device-a4c138add6292966"`)

## Error Responses

All errors return appropriate HTTP status codes:

| Status Code | Description |
|-----------|-------------|
| 404 | Device not found |
| 409 | Name in use (slug conflict) |
| 400 | Bad request (invalid parameter) |
| 500 | Internal server error |
| 503 | Service unavailable (adapter offline) |

Example error:
```json
{
  "error": "Not Found",
  "message": "device not found"
}
```

## Device State Persistence

**MVP (Current Version):**
- Device state is kept in memory only
- State resets on daemon restart
- Device names are not persisted

**Future Enhancements:**
- Persistent storage (SQLite)
- Automatic state restoration on startup
- Configuration file support
- History/audit logs

## Signal Handling

The daemon handles shutdown gracefully:

- **SIGINT** (Ctrl+C): Cleanup and shutdown
- **SIGTERM**: Cleanup and shutdown

On shutdown, the daemon:
1. Stops message listener
2. Shuts down HTTP server (with 10s timeout)
3. Closes event bus
4. Closes state manager
5. Closes adapter connection

## Logging

Logs are written to stdout in JSON format with configurable levels:

```bash
# Debug level (verbose)
GOZNP_LOG_LEVEL=debug ./bin/goznpd

# Default log level
GOZNET_LOG_LEVEL=info ./bin/goznpd

# Errors only
GOZNP_LOG_LEVEL=error ./bin/goznpd
```

Example log output:
```json
{"time":"2026-01-09T17:30:00Z","level":"INFO","msg":"Starting goznpd","version":"dev","buildTime":"unknown"}
{"time":"2026-01-09T17:30:01Z","level":"INFO","msg":"Adapter opened","port":"/dev/tty.usbserial-110"}
{"time":"2026-01-09T17:30:02Z","level":"INFO","msg":"Adapter ping successful"}
{"time":"2026-01-09T17:30:02Z","level":"INFO","msg":"HTTP server starting","addr":":":8080"}
{"time":"2026-01-09T17:30:02Z","level":"INFO","msg":"Found devices","count":6"}
```

## Running with systemd

Create `/etc/systemd/system/goznpd.service`:

```ini
[Unit]
Description=GoZNP Zigbee Daemon
After=network.target

[Service]
Type=simple
User=zigbee
Group=zigbee
Environment="GOZNP_PORT=/dev/tty.usbserial-110"
Environment="GOZNP_HTTP_ADDR=:8080"
Environment="GOZNP_LOG_LEVEL=info"
Environment="GOZNP_POLL_INTERVAL=5m"
Restart=always
RestartSec=10
ExecStart=/usr/local/bin/goznpd

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable goznpd
sudo systemctl start goznpd
```

View logs:
```bash
sudo journalctl -u goznpd -f
```

## Running with Docker

```dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o goznpd ./cmd/goznpd

FROM alpine:latest
RUN apk add --no-cache udev
COPY --from=builder /app/goznpd /usr/local/bin/"

ENV GOZNP_HTTP_ADDR=:8080

CMD ["/usr/local/bin/goznpd"]
```

Build and run:
```bash
docker build -t goznpd .
docker run --device=/dev/tty.usbserial-110 goznpd
```

## Testing

Run unit tests:
```bash
go test ./pkg/state/... ./pkg/event/... ./pkg/message/... ./internal/server/... -v
```

Integration tests require actual hardware:
```bash
GOZNP_PORT=/dev/tty.usbserial-120 make test-integration
```

## Development

### Project Structure

```
cmd/goznpd/          # Daemon entrypoint
└── main.go

pkg/                   # Reusable libraries
├── state/           # Device state management
│   ├── slug.go
│   ├── identifiers.go
│   ├── device.go
│   ├── sensor.go
│   └── manager.go
├── event/           # Event pub/sub system
│   ├── types.go
│   ├── events.go
│   └── bus.go
└── message/         # Message handling
    ├── parser.go
    └── listener.go

internal/daemon/       # Daemon-specific
├── config.go
├── server.go
└── main.go

internal/server/       # HTTP API layer
├── handlers.go
├── middleware.go
├── routes.go
└── responses.go
```

### Available Packages

| Package | Purpose | Reusable |
|---------|---------|---------|
| `pkg/state` | Device state & sensor cache | ✅ |
| `pkg/event` | Event pub/sub system | ✅ |
| `pkg/message` | Message parsing & listening | ✅ |
| `pkg/adapter` | Zigbee adapter | ✅ (from CLI) |
| `internal/daemon` | Daemon configuration | ❌ |
| `internal/server` | HTTP API handling | ❌ |

### Adding New Features

1. **New Sensor Clusters**: Add to `pkg/state/sensor.go`
2. **New Device Commands**: Add to `internal/server/handlers.go`
3. **Persistence**: Add storage layer in `pkg/state/`
4. **Real-time Updates**: Add WebSocket support in `internal/server`
5. **Authentication**: Add middleware in `internal/server/middleware.go`

## Known Limitations

- **In-Memory Only**: State/reset on restart
- **No Device Interviews**: Device metadata not automatically populated
- **Message Listener**: Currently disabled (requires ZNP access from adapter.Adapter)
- **No Groups/Scenes**: Not implemented in MVP
- **No OTA Updates**: Not implemented in MVP

## License

Same as goznp project.

## Contributing

See CONTRIBUTING.md in the main goznp repository.
