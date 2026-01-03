package adapter

import (
	"context"
	"fmt"

	"github.com/marstid/goznp/pkg/znp"
)

// DeviceNameInfo contains the custom name and description for a device.
type DeviceNameInfo struct {
	IEEEAddr    [8]byte
	Name        string
	Description string
}

// IEEEAddrString formats the IEEE address as a hex string.
// Returns format: "00:11:22:33:44:55:66:77" (reversed byte order, high byte first in display).
func (d *DeviceNameInfo) IEEEAddrString() string {
	// IEEE addresses are stored in little-endian format (low byte first)
	// Display in big-endian format (high byte first) for human readability
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X:%02X:%02X",
		d.IEEEAddr[7], d.IEEEAddr[6], d.IEEEAddr[5], d.IEEEAddr[4],
		d.IEEEAddr[3], d.IEEEAddr[2], d.IEEEAddr[1], d.IEEEAddr[0])
}

// SetDeviceName sets a custom name and description for a device.
// Name is limited to 32 characters, description to 64 characters.
// Names are stored in NVRAM and persist across restarts.
func (a *Adapter) SetDeviceName(ctx context.Context, ieeeAddr [8]byte, name, description string) error {
	// Validate name and description length
	if len(name) > znp.DeviceNameMaxName {
		return fmt.Errorf("name too long: %d bytes (max %d)", len(name), znp.DeviceNameMaxName)
	}
	if len(description) > znp.DeviceNameMaxDescription {
		return fmt.Errorf("description too long: %d bytes (max %d)", len(description), znp.DeviceNameMaxDescription)
	}

	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	if err := znpClient.SetDeviceName(ctx, ieeeAddr, name, description); err != nil {
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
		IEEEAddr:    entry.IEEEAddr,
		Name:        entry.Name,
		Description: entry.Description,
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
				IEEEAddr:    table.Entries[i].IEEEAddr,
				Name:        table.Entries[i].Name,
				Description: table.Entries[i].Description,
			})
		}
	}

	return result, nil
}
