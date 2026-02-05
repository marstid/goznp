package adapter

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/marstid/goznp/pkg/znp"
)

// NetworkFormConfig contains configuration for forming a new network.
type NetworkFormConfig struct {
	// Channel is the Zigbee channel (11-26). If 0, uses a default channel.
	Channel uint8
	// PanID is the PAN ID. If 0, a random PAN ID is generated.
	PanID uint16
	// Profile is the Zigbee application profile. Default is "ha" (Home Automation).
	Profile string
}

// FormNetwork forms a new Zigbee network as coordinator.
func (a *Adapter) FormNetwork(ctx context.Context, config NetworkFormConfig) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	// Set default channel if not specified
	channel := config.Channel
	if channel == 0 {
		channel = 15 // Default to channel 15
	}

	// Validate channel range
	if channel < 11 || channel > 26 {
		return fmt.Errorf("invalid channel %d: must be 11-26", channel)
	}

	// Set PAN ID (random if not specified)
	panID := config.PanID
	if panID == 0 {
		var buf [2]byte
		if _, err := rand.Read(buf[:]); err != nil {
			return fmt.Errorf("failed to generate random PAN ID: %w", err)
		}
		panID = uint16(buf[0])<<8 | uint16(buf[1])
		// Avoid reserved PAN IDs
		if panID == 0xFFFF {
			panID = 0xFFFE
		}
	}

	a.options.Logger.Infof("Forming network on channel %d with PAN ID 0x%04X...", channel, panID)

	// Step 1: Write startup option to clear state and config
	// This is critical - without this, the device may retain previous network state
	a.options.Logger.Infof("Writing startup option to clear state...")
	if err := znpClient.NvWrite(ctx, znp.NvStartupOption, 0, []byte{StartupOptionClearState}); err != nil {
		return fmt.Errorf("failed to set startup option: %w", err)
	}

	// Step 2: Configure coordinator logical type
	a.options.Logger.Infof("Configuring as coordinator...")
	if err := znpClient.NvWrite(ctx, znp.NvLogicalType, 0, []byte{LogicalTypeCoordinator}); err != nil {
		return fmt.Errorf("failed to set logical type: %w", err)
	}

	// Configure profile-specific settings
	profile := config.Profile
	if profile == "" {
		profile = "ha" // Default to Home Automation
	}

	// Parse and validate profile
	appProfile, err := znp.ParseProfile(profile)
	if err != nil {
		return fmt.Errorf("unknown profile: %s (supported: ha, se, gp, zll)", profile)
	}
	a.options.Logger.Infof("Configuring for %s profile...", appProfile)

	// Initialize and store profile preference in NV for later retrieval
	profileData := make([]byte, 2)
	binary.LittleEndian.PutUint16(profileData, uint16(appProfile))
	if initErr := znpClient.NvItemInit(ctx, znp.NvZStackProfile, 2, profileData); initErr != nil {
		// Non-fatal, continue
		a.options.Logger.Warnf("could not initialize profile NV item: %v", initErr)
	}
	if writeErr := znpClient.NvWrite(ctx, znp.NvZStackProfile, 0, profileData); writeErr != nil {
		// Non-fatal, continue
		a.options.Logger.Warnf("could not store profile preference: %v", writeErr)
	}

	// Step 3: Enable ZDO callbacks to receive state change indications
	a.options.Logger.Infof("Enabling ZDO callbacks...")
	if err := znpClient.NvWrite(ctx, znp.NvZdoDirectCb, 0, []byte{ZdoDirectCbEnabled}); err != nil {
		return fmt.Errorf("failed to enable ZDO callbacks: %w", err)
	}

	// Step 4: Reset the device to apply NV changes
	// The device must be reset after writing NV items for them to take effect
	a.options.Logger.Infof("Resetting device to apply configuration...")
	_, resetErr := znpClient.Reset(ctx, znp.ResetTypeSoft)
	if resetErr != nil {
		return fmt.Errorf("failed to reset device: %w", resetErr)
	}
	a.options.Logger.Infof("Device reset complete")

	// Step 5: Set channel mask for network formation
	a.options.Logger.Infof("Setting channel mask for channel %d...", channel)
	channelMask := znp.ChannelToMask(channel)
	status, err := znpClient.BdbSetChannel(ctx, true, channelMask)
	if err != nil {
		return fmt.Errorf("failed to set primary channel: %w", err)
	}
	if status != 0 {
		return fmt.Errorf("BdbSetChannel failed with status %d", status)
	}

	// Clear secondary channel (single channel operation)
	_, err = znpClient.BdbSetChannel(ctx, false, 0)
	if err != nil {
		return fmt.Errorf("failed to clear secondary channel: %w", err)
	}

	// Step 6: Start network formation commissioning
	a.options.Logger.Infof("Starting network formation commissioning...")
	status, err = znpClient.BdbStartCommissioning(ctx, znp.BdbCommissioningModeNetworkFormation)
	if err != nil {
		return fmt.Errorf("failed to start commissioning: %w", err)
	}
	if status != 0 {
		return fmt.Errorf("BdbStartCommissioning failed with status %d", status)
	}

	// Step 7: Wait for coordinator state (40 second timeout)
	a.options.Logger.Infof("Waiting for network formation to complete...")
	formationCtx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	for {
		state, err := znpClient.WaitForStateChange(formationCtx, 5*time.Second)
		if err != nil {
			// Check if context canceled or timed out.
			if formationCtx.Err() != nil {
				return fmt.Errorf("network formation timed out: %w", formationCtx.Err())
			}
			// Individual wait timeout is ok, continue waiting
			continue
		}

		a.options.Logger.Infof("Device state changed to: %s", state)

		if state == znp.DevStateZbCoord {
			// Successfully formed network
			a.options.Logger.Infof("Network formation successful!")
			return nil
		}
		// Continue waiting for coordinator state
	}
}

// ChangeChannel changes the network channel.
// If seamless is true, broadcasts update to all devices (requires formed network).
// If seamless is false, forces local channel change (breaks existing network).
func (a *Adapter) ChangeChannel(ctx context.Context, channel uint8, seamless bool) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	// Validate channel range
	if channel < 11 || channel > 26 {
		return fmt.Errorf("invalid channel %d: must be 11-26", channel)
	}

	channelMask := znp.ChannelToMask(channel)

	if seamless {
		// Seamless channel change using MgmtNwkUpdateReq
		// Broadcast address mode 0x0F, scan duration 0xFE for channel change
		status, err := znpClient.MgmtNwkUpdateReq(ctx, BroadcastAddressNwk, 0x0F, channelMask, 0xFE, 0, 0x0000)
		if err != nil {
			return fmt.Errorf("channel change request failed: %w", err)
		}
		if status != 0 {
			return fmt.Errorf("MgmtNwkUpdateReq failed with status %d", status)
		}
	} else {
		// Forced channel change - update NV and reset
		// Read current NIB
		nibData, err := znpClient.NvReadAll(ctx, znp.NvNIB)
		if err != nil {
			return fmt.Errorf("failed to read NIB: %w", err)
		}

		// Update channel at offset 42 (logicalChannel field)
		if len(nibData) > 42 {
			nibData[42] = channel
			if err := znpClient.NvWriteAll(ctx, znp.NvNIB, nibData); err != nil {
				return fmt.Errorf("failed to write NIB: %w", err)
			}
		}

		// Reset to apply changes
		_, err = znpClient.Reset(ctx, znp.ResetTypeSoft)
		if err != nil {
			return fmt.Errorf("reset failed: %w", err)
		}
	}

	return nil
}

// SetTxPower sets the transmit power level.
// The value is stored in NV memory and persists across restarts.
func (a *Adapter) SetTxPower(ctx context.Context, power int8) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	_, err := znpClient.SetTxPower(ctx, power)
	if err != nil {
		return fmt.Errorf("failed to set TX power: %w", err)
	}

	// Also write to NV for persistence
	var powerUint uint8
	if power < 0 {
		powerUint = uint8(256 + int(power))
	} else {
		powerUint = uint8(power)
	}
	// Initialize NV item if it doesn't exist, then write
	//nolint:errcheck // Errors intentionally ignored
	_ = znpClient.NvItemInit(ctx, znp.NvTransmitPower, 1, []byte{powerUint})
	//nolint:errcheck // Errors intentionally ignored
	_ = znpClient.NvWrite(ctx, znp.NvTransmitPower, 0, []byte{powerUint})

	return nil
}

// GetTxPower reads the current TX power from NV memory.
// Returns the power in dBm, or an error if not available.
func (a *Adapter) GetTxPower(ctx context.Context) (int8, error) {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return 0, ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	data, err := znpClient.NvRead(ctx, znp.NvTransmitPower, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to read TX power from NV: %w", err)
	}

	if len(data) == 0 {
		return 0, fmt.Errorf("TX power NV item not found")
	}

	// Convert from unsigned to signed
	powerUint := data[0]
	var power int8
	if powerUint > 127 {
		power = int8(int(powerUint) - 256)
	} else {
		power = int8(powerUint)
	}

	return power, nil
}

// FactoryReset performs a factory reset, clearing all network configuration.
func (a *Adapter) FactoryReset(ctx context.Context) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	// Clear NIB (network information base)
	emptyNib := make([]byte, 116) // Standard NIB size
	//nolint:errcheck // Non-fatal error, continue on failure
	_ = znpClient.NvWriteAll(ctx, znp.NvNIB, emptyNib)

	// Clear active network key
	emptyKey := make([]byte, 17)
	//nolint:errcheck // Non-fatal error, continue on failure
	_ = znpClient.NvWrite(ctx, znp.NvNwkActiveKeyInfo, 0, emptyKey)

	// Clear alternate network key
	//nolint:errcheck // Non-fatal error, continue on failure
	_ = znpClient.NvWrite(ctx, znp.NvNwkAlternKeyInfo, 0, emptyKey)

	// Reset device to apply changes
	_, err := znpClient.Reset(ctx, znp.ResetTypeSoft)
	if err != nil {
		return fmt.Errorf("reset failed: %w", err)
	}

	return nil
}

// GetCurrentChannel returns the current network channel.
func (a *Adapter) GetCurrentChannel(ctx context.Context) (uint8, error) {
	networkInfo, err := a.GetNetworkInfo(ctx)
	if err != nil {
		return 0, err
	}
	return networkInfo.Channel, nil
}

// SupportedProfiles returns all Application Profiles that zigbee-herdsman supports.
// These are the profiles for which endpoints are registered on the coordinator.
func SupportedProfiles() []znp.ApplicationProfile {
	return znp.AvailableProfiles()
}
