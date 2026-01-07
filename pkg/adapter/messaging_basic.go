package adapter

import (
	"context"
	"fmt"

	"github.com/marstid/goznp/pkg/zcl"
)

// Basic Cluster (0x0000)

// ResetToFactoryDefaults sends the reset to factory defaults command to a device.
// This resets all writeable attributes in the Basic cluster to their factory default values.
// Note: The device behavior may vary - some devices may reset all settings, while others
// may only reset specific attributes. Refer to the device documentation for details.
func (a *Adapter) ResetToFactoryDefaults(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterBasic, zcl.CmdBasicResetToFactoryDefaults, nil)
}

// DeviceInfo contains all Basic cluster attributes for a device.
type DeviceInfo struct {
	// Core Device Information
	ZCLVersion       *uint8  // ZCL version supported
	AppVersion       *uint8  // Application version
	StackVersion     *uint8  // Stack version
	HWVersion        *uint8  // Hardware version
	ManufacturerName *string // Manufacturer name
	ModelIdentifier  *string // Model identifier
	DateCode         *string // Manufacturing date code (YYYYMMDD format)
	PowerSource      *zcl.PowerSource
	SWBuildID        *string // Software build ID

	// Extended Device Information
	ProductCode                *string // Product code (octet string)
	ProductURL                 *string // Product URL
	ManufacturerVersionDetails *string // Manufacturer version details
	SerialNumber               *string // Serial number
	ProductLabel               *string // Product label

	// Optional Device Information
	LocationDescription *string // Physical location description (max 16 chars)
	PhysicalEnvironment *uint8  // Physical environment type (enum8)
	DeviceEnabled       *bool   // Whether device is enabled
	AlarmMask           *uint8  // Alarm mask (bitmap8)
	DisableLocalConfig  *uint8  // Local config disable mask (bitmap8)

	// Generic Device Information (ZCL 7+)
	GenericDeviceClass *uint8 // Generic device class (enum8)
	GenericDeviceType  *uint8 // Generic device type (enum8)
}

// ReadDeviceInfo reads all Basic cluster attributes from a device.
// Returns a DeviceInfo struct with all available attributes. Attributes that
// are not supported or fail to read will be nil.
func (a *Adapter) ReadDeviceInfo(ctx context.Context, nwkAddr uint16, endpoint uint8) (*DeviceInfo, error) {
	// Define all Basic cluster attributes to read
	attrs := []zcl.AttributeID{
		zcl.AttrBasicZCLVersion,
		zcl.AttrBasicAppVersion,
		zcl.AttrBasicStackVersion,
		zcl.AttrBasicHWVersion,
		zcl.AttrBasicManufacturerName,
		zcl.AttrBasicModelIdentifier,
		zcl.AttrBasicDateCode,
		zcl.AttrBasicPowerSource,
		zcl.AttrBasicProductCode,
		zcl.AttrBasicProductURL,
		zcl.AttrBasicManufacturerVersionDetails,
		zcl.AttrBasicSerialNumber,
		zcl.AttrBasicProductLabel,
		zcl.AttrBasicLocationDescription,
		zcl.AttrBasicPhysicalEnvironment,
		zcl.AttrBasicDeviceEnabled,
		zcl.AttrBasicAlarmMask,
		zcl.AttrBasicDisableLocalConfig,
		zcl.AttrBasicSWBuildID,
		zcl.AttrBasicGenericDeviceClass,
		zcl.AttrBasicGenericDeviceType,
	}

	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterBasic, attrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to read basic cluster attributes: %w", err)
	}

	info := &DeviceInfo{}

	// Parse results and populate DeviceInfo struct
	for _, r := range results {
		if r.Status != zcl.StatusSuccess {
			continue // Skip attributes that failed to read
		}

		switch r.AttributeID {
		case zcl.AttrBasicZCLVersion:
			if v, ok := r.Value.(uint8); ok {
				info.ZCLVersion = &v
			}
		case zcl.AttrBasicAppVersion:
			if v, ok := r.Value.(uint8); ok {
				info.AppVersion = &v
			}
		case zcl.AttrBasicStackVersion:
			if v, ok := r.Value.(uint8); ok {
				info.StackVersion = &v
			}
		case zcl.AttrBasicHWVersion:
			if v, ok := r.Value.(uint8); ok {
				info.HWVersion = &v
			}
		case zcl.AttrBasicManufacturerName:
			if v, ok := r.Value.(string); ok {
				info.ManufacturerName = &v
			}
		case zcl.AttrBasicModelIdentifier:
			if v, ok := r.Value.(string); ok {
				info.ModelIdentifier = &v
			}
		case zcl.AttrBasicDateCode:
			if v, ok := r.Value.(string); ok {
				info.DateCode = &v
			}
		case zcl.AttrBasicPowerSource:
			if v, ok := r.Value.(uint8); ok {
				ps := zcl.PowerSource(v)
				info.PowerSource = &ps
			}
		case zcl.AttrBasicLocationDescription:
			if v, ok := r.Value.(string); ok {
				info.LocationDescription = &v
			}
		case zcl.AttrBasicPhysicalEnvironment:
			if v, ok := r.Value.(uint8); ok {
				info.PhysicalEnvironment = &v
			}
		case zcl.AttrBasicDeviceEnabled:
			if v, ok := r.Value.(bool); ok {
				info.DeviceEnabled = &v
			} else if v, ok := r.Value.(uint8); ok {
				enabled := v != 0
				info.DeviceEnabled = &enabled
			}
		case zcl.AttrBasicAlarmMask:
			if v, ok := r.Value.(uint8); ok {
				info.AlarmMask = &v
			}
		case zcl.AttrBasicDisableLocalConfig:
			if v, ok := r.Value.(uint8); ok {
				info.DisableLocalConfig = &v
			}
		case zcl.AttrBasicSWBuildID:
			if v, ok := r.Value.(string); ok {
				info.SWBuildID = &v
			}
		case zcl.AttrBasicProductCode:
			if v, ok := r.Value.(string); ok {
				info.ProductCode = &v
			}
		case zcl.AttrBasicProductURL:
			if v, ok := r.Value.(string); ok {
				info.ProductURL = &v
			}
		case zcl.AttrBasicManufacturerVersionDetails:
			if v, ok := r.Value.(string); ok {
				info.ManufacturerVersionDetails = &v
			}
		case zcl.AttrBasicSerialNumber:
			if v, ok := r.Value.(string); ok {
				info.SerialNumber = &v
			}
		case zcl.AttrBasicProductLabel:
			if v, ok := r.Value.(string); ok {
				info.ProductLabel = &v
			}
		case zcl.AttrBasicGenericDeviceClass:
			if v, ok := r.Value.(uint8); ok {
				info.GenericDeviceClass = &v
			}
		case zcl.AttrBasicGenericDeviceType:
			if v, ok := r.Value.(uint8); ok {
				info.GenericDeviceType = &v
			}
		}
	}

	return info, nil
}

// On/Off Cluster (0x0006)

// GetOnOffState reads the current on/off state.
func (a *Adapter) GetOnOffState(ctx context.Context, nwkAddr uint16, endpoint uint8) (bool, error) {
	results, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterOnOff, zcl.AttrOnOff)
	if err != nil {
		return false, fmt.Errorf("failed to read on/off state: %w", err)
	}

	if len(results) == 0 {
		return false, fmt.Errorf("no attributes returned")
	}

	result := results[0]
	if result.Status != zcl.StatusSuccess {
		return false, fmt.Errorf("read on/off state returned status 0x%02X", result.Status)
	}

	// Convert value to bool
	switch v := result.Value.(type) {
	case bool:
		return v, nil
	case uint8:
		return v != 0, nil
	default:
		return false, fmt.Errorf("unexpected value type: %T", result.Value)
	}
}

// DeviceStatus contains common device status information.
type DeviceStatus struct {
	NwkAddr        uint16
	Manufacturer   string
	Model          string
	PowerSource    zcl.PowerSource
	BatteryPercent *uint8 // nil if not available
	OnOff          *bool  // nil if not on/off device
}

// GetDeviceStatus queries common status attributes from a device.
func (a *Adapter) GetDeviceStatus(ctx context.Context, nwkAddr uint16, endpoint uint8) (*DeviceStatus, error) {
	status := &DeviceStatus{
		NwkAddr: nwkAddr,
	}

	// Read basic cluster attributes
	basicAttrs := []zcl.AttributeID{
		zcl.AttrBasicManufacturerName,
		zcl.AttrBasicModelIdentifier,
		zcl.AttrBasicPowerSource,
	}

	basicResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterBasic, basicAttrs...)
	if err == nil {
		for _, r := range basicResults {
			if r.Status != zcl.StatusSuccess {
				continue
			}

			switch r.AttributeID {
			case zcl.AttrBasicManufacturerName:
				if s, ok := r.Value.(string); ok {
					status.Manufacturer = s
				}
			case zcl.AttrBasicModelIdentifier:
				if s, ok := r.Value.(string); ok {
					status.Model = s
				}
			case zcl.AttrBasicPowerSource:
				if ps, ok := r.Value.(uint8); ok {
					status.PowerSource = zcl.PowerSource(ps)
				}
			}
		}
	}

	// Try to read battery percentage (may not be supported)
	batteryResults, err := a.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPowerConfig, zcl.AttrPowerBatteryPercentage)
	if err == nil && len(batteryResults) > 0 && batteryResults[0].Status == zcl.StatusSuccess {
		if bp, ok := batteryResults[0].Value.(uint8); ok {
			status.BatteryPercent = &bp
		}
	}

	// Try to read on/off state (may not be supported)
	onOffState, err := a.GetOnOffState(ctx, nwkAddr, endpoint)
	if err == nil {
		status.OnOff = &onOffState
	}

	return status, nil
}
