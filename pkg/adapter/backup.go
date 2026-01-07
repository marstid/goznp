package adapter

import (
	"context"
	"fmt"

	"github.com/marstid/goznp/pkg/backup"
	"github.com/marstid/goznp/pkg/znp"
)

// CreateBackup creates a full backup of the adapter configuration.
func (a *Adapter) CreateBackup(ctx context.Context) (*backup.Backup, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	a.mu.Unlock()

	b := backup.New()

	// 1. Get adapter version info.
	if a.version != nil {
		b.Adapter = backup.AdapterInfo{
			ZStackVariant: int(a.version.Product),
			SDKVersion:    fmt.Sprintf("%d.%d.%d", a.version.MajorRel, a.version.MinorRel, a.version.MaintRel),
			BuildDate:     a.version.Revision,
		}
	}

	// 2. Get device info (coordinator address).
	deviceInfo, err := a.znp.GetDeviceInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting device info: %w", err)
	}
	b.Coordinator = backup.CoordinatorInfo{
		IEEEAddress:    backup.EncodeIEEEAddr(deviceInfo.IEEEAddr),
		NetworkAddress: deviceInfo.ShortAddr,
	}

	// 3. Get network info from ZDO.
	extNwkInfo, err := a.znp.ExtNwkInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting network info: %w", err)
	}
	b.Network.PanID = extNwkInfo.PanID
	b.Network.ExtendedPanID = backup.EncodeIEEEAddr(extNwkInfo.ExtendedPanID)
	b.Network.Channel = extNwkInfo.Channel

	// 4. Read NIB for additional network config.
	nibData, err := a.znp.NvReadAll(ctx, znp.NvNIB)
	if err != nil {
		return nil, fmt.Errorf("reading NIB: %w", err)
	}
	if len(nibData) > 0 {
		nib, err := znp.ParseNIB(nibData)
		if err != nil {
			return nil, fmt.Errorf("parsing NIB: %w", err)
		}
		b.Network.SecurityLevel = nib.SecurityLevel
		b.Network.UpdateID = nib.UpdateID
	}

	// 5. Read network key.
	keyData, err := a.znp.NvReadAll(ctx, znp.NvNwkActiveKeyInfo)
	if err != nil {
		return nil, fmt.Errorf("reading network key: %w", err)
	}
	if len(keyData) >= 17 {
		keyDesc, err := znp.ParseNwkKeyDescriptor(keyData)
		if err != nil {
			return nil, fmt.Errorf("parsing network key: %w", err)
		}
		b.Network.NetworkKey = backup.EncodeHex(keyDesc.Key[:])
		b.Network.NetworkKeySequence = keyDesc.KeySeqNum
	}

	// 6. Read devices from NVRAM Address Manager table.
	addrMgrEntries, err := a.znp.ReadAddrMgrTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading address manager table: %w", err)
	}
	for _, dev := range addrMgrEntries {
		// Skip coordinator (address 0x0000).
		if dev.NwkAddr == 0x0000 {
			continue
		}

		entry := backup.DeviceEntry{
			IEEEAddress:    backup.EncodeIEEEAddr(dev.ExtAddr),
			NetworkAddress: dev.NwkAddr,
			Type:           "EndDevice", // Default type.
		}
		b.Devices = append(b.Devices, entry)
	}

	return b, nil
}

// RestoreBackup restores adapter configuration from a backup.
// WARNING: This will overwrite the current configuration!
func (a *Adapter) RestoreBackup(ctx context.Context, b *backup.Backup) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	a.mu.Unlock()

	// Validate backup.
	if err := b.Validate(); err != nil {
		return fmt.Errorf("invalid backup: %w", err)
	}

	// 1. Write coordinator IEEE address.
	coordAddr, err := backup.DecodeIEEEAddr(b.Coordinator.IEEEAddress)
	if err != nil {
		return fmt.Errorf("decoding coordinator address: %w", err)
	}
	if err := a.znp.NvWriteAll(ctx, znp.NvExtAddr, coordAddr[:]); err != nil {
		return fmt.Errorf("writing coordinator address: %w", err)
	}

	// 2. Write network key.
	networkKey, err := backup.DecodeHex(b.Network.NetworkKey)
	if err != nil {
		return fmt.Errorf("decoding network key: %w", err)
	}
	if len(networkKey) != 16 {
		return fmt.Errorf("invalid network key length: %d", len(networkKey))
	}

	keyData := make([]byte, 17)
	keyData[0] = b.Network.NetworkKeySequence
	copy(keyData[1:], networkKey)

	if err := a.znp.NvWriteAll(ctx, znp.NvNwkActiveKeyInfo, keyData); err != nil {
		return fmt.Errorf("writing network key: %w", err)
	}
	if err := a.znp.NvWriteAll(ctx, znp.NvNwkAlternKeyInfo, keyData); err != nil {
		return fmt.Errorf("writing alternate key: %w", err)
	}

	// 3. A full restore would require resetting and reconfiguring the NIB.
	// This is complex and version-dependent. For now, we restore the key data.
	// Full NIB restore would be implemented in a future version.

	return nil
}
