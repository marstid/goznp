// Package devices provides a lookup table for Zigbee device identification.
// It maps manufacturer names and model IDs to friendly vendor names and descriptions,
// using data sourced from the zigbee2mqtt community database.
package devices

// Info contains friendly device information looked up from the database.
type Info struct {
	Vendor      string // Friendly vendor name (e.g., "NOUS", "IKEA", "Philips").
	Model       string // Product model name (e.g., "A6Z", "TRADFRI bulb E27").
	Description string // Brief device description.
}

// Fingerprint uniquely identifies a device type by its Zigbee identifiers.
type Fingerprint struct {
	ManufacturerName string // Manufacturer name from Basic cluster (e.g., "_TZ3000_266azbg3").
	ModelID          string // Model identifier from Basic cluster (e.g., "TS011F").
}

// Lookup finds device info by manufacturer name and model ID.
// Returns nil if no matching device is found.
func Lookup(manufacturerName, modelID string) *Info {
	fp := Fingerprint{
		ManufacturerName: manufacturerName,
		ModelID:          modelID,
	}

	if info, ok := deviceDatabase[fp]; ok {
		return &info
	}

	return nil
}

// LookupByManufacturer finds device info by manufacturer name only.
// This is a fallback when the exact manufacturer+model combination isn't found.
// Returns nil if no matching device is found.
func LookupByManufacturer(manufacturerName string) *Info {
	// First try exact match with empty model.
	fp := Fingerprint{
		ManufacturerName: manufacturerName,
		ModelID:          "",
	}
	if info, ok := deviceDatabase[fp]; ok {
		return &info
	}

	// Search for any device with matching manufacturer.
	for k, v := range deviceDatabase {
		if k.ManufacturerName == manufacturerName {
			return &v
		}
	}

	return nil
}

// LookupWithFallback tries exact match first, then manufacturer-only fallback.
// Returns nil if no matching device is found.
func LookupWithFallback(manufacturerName, modelID string) *Info {
	// Try exact match first.
	if info := Lookup(manufacturerName, modelID); info != nil {
		return info
	}

	// Fallback to manufacturer-only lookup.
	return LookupByManufacturer(manufacturerName)
}

// Count returns the number of devices in the database.
func Count() int {
	return len(deviceDatabase)
}

// AllFingerprints returns all known device fingerprints.
// Useful for debugging or listing supported devices.
func AllFingerprints() []Fingerprint {
	fps := make([]Fingerprint, 0, len(deviceDatabase))
	for fp := range deviceDatabase {
		fps = append(fps, fp)
	}
	return fps
}
