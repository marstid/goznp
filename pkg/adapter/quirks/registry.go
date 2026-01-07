package quirks

import (
	"sync"

	"github.com/marstid/goznp/pkg/zcl"
)

// Registry manages device quirks.
type Registry struct {
	mu     sync.RWMutex
	quirks []*DeviceQuirk
}

// NewRegistry creates a new quirk registry.
func NewRegistry() *Registry {
	return &Registry{
		quirks: make([]*DeviceQuirk, 0),
	}
}

// Register adds a quirk to the registry.
func (r *Registry) Register(quirk *DeviceQuirk) error {
	// Compile matcher patterns
	if err := quirk.Matcher.Compile(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.quirks = append(r.quirks, quirk)
	return nil
}

// RegisterAll adds multiple quirks to the registry.
func (r *Registry) RegisterAll(quirks []*DeviceQuirk) error {
	for _, q := range quirks {
		if err := r.Register(q); err != nil {
			return err
		}
	}
	return nil
}

// FindQuirks returns all quirks that match the given device.
func (r *Registry) FindQuirks(info DeviceInfo) []*DeviceQuirk {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []*DeviceQuirk
	for _, q := range r.quirks {
		if q.Matcher.Matches(info.ManufacturerCode, info.Manufacturer, info.Model) {
			matches = append(matches, q)
		}
	}
	return matches
}

// GetAppliedQuirks returns a consolidated view of all quirks for a device.
func (r *Registry) GetAppliedQuirks(nwkAddr uint16, ieeeAddr [8]byte, info DeviceInfo) *AppliedQuirks {
	quirks := r.FindQuirks(info)

	applied := &AppliedQuirks{
		DeviceAddr:         nwkAddr,
		IEEEAddr:           ieeeAddr,
		Quirks:             quirks,
		AttributeOverrides: make(map[zcl.ClusterID]map[zcl.AttributeID]*AttributeOverride),
		ResponseOverrides:  make(map[zcl.ClusterID]map[uint8][]uint8),
		EnergyResetMethods: make([]EnergyResetMethod, 0),
	}

	// Consolidate all quirks
	for _, q := range quirks {
		// Attribute overrides
		for _, ao := range q.AttributeOverrides {
			if applied.AttributeOverrides[ao.ClusterID] == nil {
				applied.AttributeOverrides[ao.ClusterID] = make(map[zcl.AttributeID]*AttributeOverride)
			}
			aoCopy := ao // Create a copy to take address
			applied.AttributeOverrides[ao.ClusterID][ao.AttributeID] = &aoCopy
		}

		// Response overrides
		for _, ro := range q.ResponseOverrides {
			if applied.ResponseOverrides[ro.ClusterID] == nil {
				applied.ResponseOverrides[ro.ClusterID] = make(map[uint8][]uint8)
			}
			applied.ResponseOverrides[ro.ClusterID][ro.ExpectedCommand] = ro.AcceptCommands
		}

		// Energy reset methods
		applied.EnergyResetMethods = append(applied.EnergyResetMethods, q.EnergyResetMethods...)

		// Timing override (last one wins)
		if q.TimingOverride != nil {
			applied.TimingOverride = q.TimingOverride
		}
	}

	return applied
}

// All returns all registered quirks.
func (r *Registry) All() []*DeviceQuirk {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*DeviceQuirk, len(r.quirks))
	copy(result, r.quirks)
	return result
}

// Count returns the number of registered quirks.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.quirks)
}

// GetQuirkByID returns a quirk by its ID.
func (r *Registry) GetQuirkByID(id string) *DeviceQuirk {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, q := range r.quirks {
		if q.ID == id {
			return q
		}
	}
	return nil
}

// HasAttributeOverride checks if there's an override for the given attribute.
func (a *AppliedQuirks) HasAttributeOverride(clusterID zcl.ClusterID, attrID zcl.AttributeID) bool {
	if a == nil || a.AttributeOverrides == nil {
		return false
	}
	if cluster, ok := a.AttributeOverrides[clusterID]; ok {
		_, exists := cluster[attrID]
		return exists
	}
	return false
}

// GetAttributeOverride returns the override for the given attribute, or nil if none.
func (a *AppliedQuirks) GetAttributeOverride(clusterID zcl.ClusterID, attrID zcl.AttributeID) *AttributeOverride {
	if a == nil || a.AttributeOverrides == nil {
		return nil
	}
	if cluster, ok := a.AttributeOverrides[clusterID]; ok {
		return cluster[attrID]
	}
	return nil
}

// ApplyAttributeOverride applies any override to the given value.
func (a *AppliedQuirks) ApplyAttributeOverride(clusterID zcl.ClusterID, attrID zcl.AttributeID, value float64) float64 {
	override := a.GetAttributeOverride(clusterID, attrID)
	if override == nil {
		return value
	}

	result := value

	// Apply multiplier first
	if override.Multiplier != nil {
		result *= *override.Multiplier
	}

	// Then divisor
	if override.Divisor != nil && *override.Divisor != 0 {
		result /= *override.Divisor
	}

	// Then offset
	if override.Offset != nil {
		result += *override.Offset
	}

	return result
}

// IsAcceptableResponse checks if the given command is acceptable for the expected one.
func (a *AppliedQuirks) IsAcceptableResponse(clusterID zcl.ClusterID, expected, actual uint8) bool {
	// Always accept exact match
	if expected == actual {
		return true
	}

	if a == nil || a.ResponseOverrides == nil {
		return false
	}

	if cluster, ok := a.ResponseOverrides[clusterID]; ok {
		if acceptable, ok := cluster[expected]; ok {
			for _, cmd := range acceptable {
				if cmd == actual {
					return true
				}
			}
		}
	}
	return false
}

// GetEnergyResetMethods returns methods to try for energy reset.
func (a *AppliedQuirks) GetEnergyResetMethods() []EnergyResetMethod {
	if a == nil {
		return nil
	}
	return a.EnergyResetMethods
}

// GetTimingOverride returns timing overrides if any.
func (a *AppliedQuirks) GetTimingOverride() *TimingOverride {
	if a == nil {
		return nil
	}
	return a.TimingOverride
}

// QuirkNames returns the names of all applied quirks.
func (a *AppliedQuirks) QuirkNames() []string {
	if a == nil || len(a.Quirks) == 0 {
		return nil
	}
	names := make([]string, len(a.Quirks))
	for i, q := range a.Quirks {
		names[i] = q.ID
	}
	return names
}
