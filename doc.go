// Package goznp provides a Go library for controlling Zigbee networks via
// Texas Instruments Z-Stack Network Processor (ZNP) adapters.
//
// The library is organized into layered packages:
//
//   - [github.com/marstid/goznp/pkg/adapter] — High-level API for device control,
//     network management, and diagnostics. This is the main entry point for most users.
//   - [github.com/marstid/goznp/pkg/znp] — Z-Stack Network Processor protocol layer
//     (ZDO, AF, SYS, UTIL commands).
//   - [github.com/marstid/goznp/pkg/zcl] — Zigbee Cluster Library types, frame
//     building/parsing, and cluster constants.
//   - [github.com/marstid/goznp/pkg/unpi] — Unified Network Processor Interface
//     for frame encoding and decoding.
//   - [github.com/marstid/goznp/pkg/ota] — OTA firmware update server and image parsing.
//   - [github.com/marstid/goznp/pkg/backup] — Coordinator NVRAM backup format.
//
// # Getting Started
//
// Most users should start with the adapter package:
//
//	a := adapter.New(adapter.WithSerialPath("/dev/ttyUSB0"))
//	if err := a.Open(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer a.Close()
//
// See the [examples] directory for runnable code samples covering device discovery,
// control, sensors, and advanced integrations.
//
// # Hardware
//
// This library supports Texas Instruments CC2652R/RB, CC2652P, and CC1352P
// coordinators running Koenkk Z-Stack 3.x.0 firmware.
//
// [examples]: https://github.com/marstid/goznp/tree/main/examples
package goznp
