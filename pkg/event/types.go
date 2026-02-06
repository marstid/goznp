package event

// Type represents the type of event.
type Type string

const (
	// DeviceLifecycle events
	EventDeviceJoined  Type = "device.joined"
	EventDeviceLeft    Type = "device.left"
	EventDeviceUpdated Type = "device.updated"

	// State events
	EventDeviceStateChange Type = "device.state_change"

	// Sensor events
	EventSensorReport Type = "sensor.report"

	// Adapter events
	EventAdapterOnline  Type = "adapter.online"
	EventAdapterOffline Type = "adapter.offline"

	// Network events
	EventNetworkChanged Type = "network.changed"
)
