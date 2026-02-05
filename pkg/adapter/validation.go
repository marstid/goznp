package adapter

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidIEEEAddress is returned when an IEEE address is invalid.
	ErrInvalidIEEEAddress = errors.New("invalid IEEE address")
	// ErrInvalidEndpoint is returned when an endpoint number is invalid.
	ErrInvalidEndpoint = errors.New("invalid endpoint number")
	// ErrInvalidNetworkAddress is returned when a network address is invalid.
	ErrInvalidNetworkAddress = errors.New("invalid network address")
)

// validateIEEEAddr validates that an IEEE address is not zero or broadcast.
func validateIEEEAddr(addr [8]byte) error {
	// Check for all zeros
	var zeroAddr [8]byte
	if addr == zeroAddr {
		return fmt.Errorf("%w: address cannot be all zeros", ErrInvalidIEEEAddress)
	}

	// Check for broadcast address (all 0xFF)
	var broadcastAddr [8]byte
	for i := range broadcastAddr {
		broadcastAddr[i] = 0xFF
	}
	if addr == broadcastAddr {
		return fmt.Errorf("%w: address cannot be broadcast (0x00FF...)...FF", ErrInvalidIEEEAddress)
	}

	return nil
}

// validateEndpoint validates that an endpoint number is within valid range (1-240).
func validateEndpoint(endpoint uint8) error {
	if endpoint < 1 || endpoint > 240 {
		return fmt.Errorf("%w: endpoint must be 1-240, got %d", ErrInvalidEndpoint, endpoint)
	}
	return nil
}

// validateNetworkAddress validates that a network address is not reserved.
func validateNetworkAddress(nwkAddr uint16) error {
	// 0xFFFF is the broadcast address
	if nwkAddr == BroadcastAddressNwk {
		return fmt.Errorf("%w: address cannot be broadcast (0x%04X)", ErrInvalidNetworkAddress, BroadcastAddressNwk)
	}
	// 0xFFFE is the invalid address
	if nwkAddr == InvalidAddressNwk {
		return fmt.Errorf("%w: address cannot be invalid (0x%04X)", ErrInvalidNetworkAddress, InvalidAddressNwk)
	}
	return nil
}
