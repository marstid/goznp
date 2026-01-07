package adapter

import (
	"context"
	"fmt"

	"github.com/marstid/goznp/pkg/znp"
)

// NetworkInfo contains comprehensive network information.
type NetworkInfo struct {
	// Coordinator info
	IEEEAddr  [8]byte
	ShortAddr uint16

	// Network config
	PanID         uint16
	ExtendedPanID [8]byte
	Channel       uint8

	// Registered endpoint profiles
	Profiles []znp.ApplicationProfile

	// Device info
	DeviceType      uint8
	DeviceState     uint8
	NumAssocDevices uint8
}

// GetNetworkInfo retrieves comprehensive network information.
func (a *Adapter) GetNetworkInfo(ctx context.Context) (*NetworkInfo, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	a.mu.Unlock()

	// Get device info from UTIL
	deviceInfo, err := a.znp.GetDeviceInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting device info: %w", err)
	}

	// Get extended network info from ZDO
	extNwkInfo, err := a.znp.ExtNwkInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting extended network info: %w", err)
	}

	// Get registered profiles
	profiles := a.RegisteredProfiles()

	return &NetworkInfo{
		IEEEAddr:        deviceInfo.IEEEAddr,
		ShortAddr:       deviceInfo.ShortAddr,
		PanID:           extNwkInfo.PanID,
		ExtendedPanID:   extNwkInfo.ExtendedPanID,
		Channel:         extNwkInfo.Channel,
		Profiles:        profiles,
		DeviceType:      deviceInfo.DeviceType,
		DeviceState:     deviceInfo.DeviceState,
		NumAssocDevices: deviceInfo.NumAssocDevices,
	}, nil
}

// GetCoordinatorIEEE returns the coordinator's IEEE address.
func (a *Adapter) GetCoordinatorIEEE(ctx context.Context) ([8]byte, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return [8]byte{}, ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	deviceInfo, err := znpClient.GetDeviceInfo(ctx)
	if err != nil {
		return [8]byte{}, fmt.Errorf("getting device info: %w", err)
	}

	return deviceInfo.IEEEAddr, nil
}
