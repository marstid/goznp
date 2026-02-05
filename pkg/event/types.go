package event

// EventType represents the type of event.
type EventType string

const (
	// DeviceLifecycle events
	EventDeviceJoined  EventType = "device.joined"
	EventDeviceLeft    EventType = "device.left"
	EventDeviceUpdated EventType = "device.updated"

	// State events
	EventDeviceStateChange EventType = "device.state_change"

	// Sensor events
	EventSensorReport EventType = "sensor.report"

	// Adapter events
	EventAdapterOnline  EventType = "adapter.online"
	EventAdapterOffline EventType = "adapter.offline"

	// Network events
	EventNetworkChanged EventType = "network.changed"
)
