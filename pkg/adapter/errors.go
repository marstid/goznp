package adapter

import "errors"

// Sentinel errors for adapter operations.
// These errors can be checked with errors.Is() for specific error handling.
var (
	// ErrNotOpen indicates the adapter is not open.
	ErrNotOpen = errors.New("adapter: not open")

	// ErrDeviceNotFound indicates the requested device was not found.
	ErrDeviceNotFound = errors.New("adapter: device not found")

	// ErrClusterNotSupported indicates the device doesn't support the cluster.
	ErrClusterNotSupported = errors.New("adapter: cluster not supported by device")

	// ErrEndpointNotFound indicates the endpoint doesn't exist on the device.
	ErrEndpointNotFound = errors.New("adapter: endpoint not found")

	// ErrAttributeNotSupported indicates the attribute is not supported.
	ErrAttributeNotSupported = errors.New("adapter: attribute not supported")

	// ErrDeviceSleeping indicates the device is a sleepy end device and not responding.
	ErrDeviceSleeping = errors.New("adapter: device is sleeping")

	// ErrTransactionTimeout indicates a ZCL transaction timed out.
	ErrTransactionTimeout = errors.New("adapter: transaction timeout")

	// ErrInvalidResponse indicates an invalid or malformed response was received.
	ErrInvalidResponse = errors.New("adapter: invalid response")

	// ErrNetworkNotFormed indicates no network has been formed yet.
	ErrNetworkNotFormed = errors.New("adapter: network not formed")

	// ErrPermitJoinDisabled indicates permit join is not currently enabled.
	ErrPermitJoinDisabled = errors.New("adapter: permit join disabled")

	// ErrBindingFailed indicates a binding operation failed.
	ErrBindingFailed = errors.New("adapter: binding failed")

	// ErrReportingConfigFailed indicates reporting configuration failed.
	ErrReportingConfigFailed = errors.New("adapter: reporting configuration failed")
)
