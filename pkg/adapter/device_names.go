package adapter

import (
	"context"
	"fmt"

	"github.com/marstid/goznp/pkg/znp"
)

// DeviceNameInfo contains the custom name and comment for a device.
type DeviceNameInfo struct {
	IEEEAddr [8]byte
	Name     string
	Comment  string
}

// IEEEAddrString formats the IEEE address as a hex string.
// IEEE addresses are stored in little-endian format (low byte first in memory),
// but displayed in big-endian format (high byte first) for human readability.
func (d *DeviceNameInfo) IEEEAddrString() string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x",
		d.IEEEAddr[7], d.IEEEAddr[6], d.IEEEAddr[5], d.IEEEAddr[4],
		d.IEEEAddr[3], d.IEEEAddr[2], d.IEEEAddr[1], d.IEEEAddr[0])
}

// SetDeviceName sets a custom name and comment for a device.
// Name is limited to 32 characters, comment to 64 characters.
// Names are stored in NVRAM and persist across restarts.
func (a *Adapter) SetDeviceName(ctx context.Context, ieeeAddr [8]byte, name, comment string) error {
	// Validate name and comment length
	if len(name) > znp.DeviceNameMaxName {
		return fmt.Errorf("name too long: %d bytes (max %d)", len(name), znp.DeviceNameMaxName)
	}
	if len(comment) > znp.DeviceNameMaxComment {
		return fmt.Errorf("comment too long: %d bytes (max %d)", len(comment), znp.DeviceNameMaxComment)
	}

	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	if err := znpClient.SetDeviceName(ctx, ieeeAddr, name, comment); err != nil {
		return fmt.Errorf("failed to set device name: %w", err)
	}

	return nil
}

// GetDeviceName retrieves the custom name for a device.
// Returns nil if no name is set for this device.
func (a *Adapter) GetDeviceName(ctx context.Context, ieeeAddr [8]byte) (*DeviceNameInfo, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	entry, err := znpClient.GetDeviceName(ctx, ieeeAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get device name: %w", err)
	}

	// Return nil if no name is set
	if entry == nil {
		return nil, nil
	}

	return &DeviceNameInfo{
		IEEEAddr: entry.IEEEAddr,
		Name:     entry.Name,
		Comment:  entry.Comment,
	}, nil
}

// DeleteDeviceName removes the custom name for a device.
func (a *Adapter) DeleteDeviceName(ctx context.Context, ieeeAddr [8]byte) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	if err := znpClient.DeleteDeviceName(ctx, ieeeAddr); err != nil {
		return fmt.Errorf("failed to delete device name: %w", err)
	}

	return nil
}

// ListDeviceNames returns all devices with custom names.
func (a *Adapter) ListDeviceNames(ctx context.Context) ([]DeviceNameInfo, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	table, err := znpClient.ReadDeviceNameTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read device name table: %w", err)
	}

	// Filter out empty entries and convert to DeviceNameInfo
	result := make([]DeviceNameInfo, 0, len(table.Entries))
	for i := range table.Entries {
		if !table.Entries[i].IsEmpty() {
			result = append(result, DeviceNameInfo{
				IEEEAddr: table.Entries[i].IEEEAddr,
				Name:     table.Entries[i].Name,
				Comment:  table.Entries[i].Comment,
			})
		}
	}

	return result, nil
}
