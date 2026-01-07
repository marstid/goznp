// Package adapter provides a high-level API for controlling Zigbee networks
// through Texas Instruments Z-Stack Network Processor (ZNP) adapters.
//
// # Basic Usage
//
// Create and open an adapter:
//
//	a := adapter.New(adapter.WithSerialPath("/dev/ttyUSB0"))
//	if err := a.Open(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer a.Close()
//
// # Network Operations
//
// Form a new network:
//
//	err := a.FormNetwork(ctx, adapter.NetworkFormConfig{Channel: 15})
//
// Permit device joining:
//
//	err := a.PermitJoin(ctx, 254) // 254 seconds
//
// # Device Operations
//
// List all paired devices:
//
//	devices, err := a.GetDevices(ctx)
//
// Control a device:
//
//	err := a.TurnOn(ctx, nwkAddr, endpoint)
//
// Read device status:
//
//	status, err := a.GetDeviceStatus(ctx, nwkAddr)
//
// # Device Naming
//
// Set a custom name stored on the coordinator:
//
//	a.SetDeviceName(ctx, ieeeAddr, "Living Room Light", "Main ceiling light")
//
// Get a device name:
//
//	name, _ := a.GetDeviceName(ctx, ieeeAddr)
//
// # Events
//
// Register for device join/leave events:
//
//	a.OnDeviceEvent(func(event adapter.DeviceEvent) {
//	    fmt.Printf("Device event: %v\n", event)
//	})
//
// # Messaging
//
// Read attributes:
//
//	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, clusterID, []zcl.AttributeID{
//	    zcl AttrOnOff,
//	})
//
// Write attributes:
//
//	err := a.WriteAttributes(ctx, nwkAddr, endpoint, clusterID, []zcl.AttributeRecord{
//	    {ID: zcl.AttrOnOff, DataType: zcl.TypeData8, Value: uint8(1)},
//	})
//
// # Thread Safety
//
// The Adapter is safe for concurrent use from multiple goroutines.
// All methods use internal mutex protection.
//
// # Context Support
//
// All methods accept a context.Context for cancellation and timeout control.
// Recommended timeout: 10-30 seconds for most operations.
//
// # Hardware Support
//
// This library supports Texas Instruments CC2652R/RB, CC2652P, and CC1352P
// coordinators running Koenkk Z-Stack 3.x.0 firmware.
//
// # Device Management
//
// Device information is maintained by the device manager, which tracks
// devices by their IEEE address and network address. When devices rejoin
// with a new network address, the device manager automatically updates.
//
// # Quirks
//
// Some devices don't follow the Zigbee specification exactly. The library
// includes a quirks system that can override default behavior for specific
// manufacturers or models. Quirks are automatically applied during device
// interview.
package adapter
