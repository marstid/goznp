package event

import (
	"strings"
	"time"
)

// DeviceJoinedEvent is emitted when a device joins the network.
type DeviceJoinedEvent struct {
	IEEEAddr string    `json:"ieee_addr"`
	NwkAddr  string    `json:"nwk_addr"`
	Name     string    `json:"name,omitempty"`
	Model    string    `json:"model,omitempty"`
	JoinedAt time.Time `json:"joined_at"`
}

// DeviceLeftEvent is emitted when a device leaves the network.
type DeviceLeftEvent struct {
	IEEEAddr string    `json:"ieee_addr"`
	Name     string    `json:"name,omitempty"`
	LeftAt   time.Time `json:"left_at"`
}

// DeviceStateChangeEvent is emitted when a device's state changes.
type DeviceStateChangeEvent struct {
	IEEEAddr  string      `json:"ieee_addr"`
	Slug      string      `json:"slug,omitempty"`
	ClusterID uint16      `json:"cluster_id"`
	AttrID    uint16      `json:"attr_id"`
	OldValue  interface{} `json:"old_value,omitempty"`
	NewValue  interface{} `json:"new_value"`
	ChangedAt time.Time   `json:"changed_at"`
}

// SensorReportEvent is emitted when a device sends sensor data.
type SensorReportEvent struct {
	IEEEAddr   string      `json:"ieee_addr"`
	Slug       string      `json:"slug,omitempty"`
	ClusterID  uint16      `json:"cluster_id"`
	Attributes []Attribute `json:"attributes"`
	ReceivedAt time.Time   `json:"received_at"`
}

// Attribute represents a single attribute value.
type Attribute struct {
	ID    uint16      `json:"id"`
	Value interface{} `json:"value"`
}

// DeviceUpdatedEvent is emitted when a device's metadata is updated.
type DeviceUpdatedEvent struct {
	IEEEAddr   string    `json:"ieee_addr"`
	Slug       string    `json:"slug,omitempty"`
	UpdateType string    `json:"update_type"` // "name", "endpoints", "capabilities"
	OldValue   string    `json:"old_value,omitempty"`
	NewValue   string    `json:"new_value,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// AdapterOnlineEvent is emitted when the adapter comes online.
type AdapterOnlineEvent struct {
	OnlineAt time.Time `json:"online_at"`
}

// AdapterOfflineEvent is emitted when the adapter goes offline.
type AdapterOfflineEvent struct {
	Reason    string    `json:"reason,omitempty"`
	OfflineAt time.Time `json:"offline_at"`
}

// NetworkChangedEvent is emitted when network configuration changes.
type NetworkChangedEvent struct {
	ChangeType string                 `json:"change_type"` // "formed", "reformed", "channel", "pan_id"
	Details    map[string]interface{} `json:"details,omitempty"`
	ChangedAt  time.Time              `json:"changed_at"`
}

// Event represents a generic event.
type Event struct {
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// String returns a string representation of the event.
func (e *Event) String() string {
	var sb strings.Builder
	sb.WriteString(string(e.Type))
	if data, ok := e.Data.(stringer); ok {
		sb.WriteString(": ")
		sb.WriteString(data.String())
	}
	return sb.String()
}

type stringer interface {
	String() string
}
