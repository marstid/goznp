package adapter

// Adapter-level constants for magic numbers used throughout the code.

// Direction values for reporting configuration
const (
	// ReportingDirectionDeviceToCoordinator indicates the device reports to the coordinator.
	ReportingDirectionDeviceToCoordinator = 0x00
)

// Address modes for ZDO commands
const (
	// AddressModeGroup indicates the address is a group ID (16-bit).
	AddressModeGroup = 0x01
	// AddressModeIEEE indicates the address is an IEEE address (64-bit).
	AddressModeIEEE = 0x03
)

// Logical types for network devices
const (
	// LogicalTypeCoordinator represents a network coordinator.
	LogicalTypeCoordinator = 0x00
	// LogicalTypeRouter represents a router device.
	LogicalTypeRouter = 0x01
	// LogicalTypeEndDevice represents an end device.
	LogicalTypeEndDevice = 0x02
)

// Startup options for NV memory
const (
	// StartupOptionClearState clears the device state and configuration.
	StartupOptionClearState = 0x03
)

// ZDO callback configuration
const (
	// ZdoDirectCbEnabled enables ZDO direct callbacks.
	ZdoDirectCbEnabled = 0x01
)

// Special network addresses
const (
	// BroadcastAddressNwk is the network broadcast address.
	BroadcastAddressNwk = 0xFFFF
	// InvalidAddressNwk is the invalid network address marker.
	InvalidAddressNwk = 0xFFFE
	// CoordinatorAddressNwk is the network address of the coordinator.
	CoordinatorAddressNwk = 0x0000
)

// Magic numbers from Z-Stack NV items
const (
	// NvItemCoordinatorEP is the coordinator's primary endpoint NV item ID.
	NvItemCoordinatorEP = 0x0001
	// NvItemNIB is the Network Information Base NV item ID.
	NvItemNIB = 0x0021
	// NvItemNwkActiveKeyInfo is the active network key info NV item ID.
	NvItemNwkActiveKeyInfo = 0x003F
)
