package backup

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// FormatVersion is the current backup format version.
const FormatVersion = 1

// Backup represents a complete adapter backup.
type Backup struct {
	Version     int             `json:"version"`
	Created     time.Time       `json:"created"`
	Adapter     AdapterInfo     `json:"adapter"`
	Coordinator CoordinatorInfo `json:"coordinator"`
	Network     NetworkConfig   `json:"network"`
	Security    SecurityInfo    `json:"security"`
	Devices     []DeviceEntry   `json:"devices"`
}

// AdapterInfo contains adapter firmware information.
type AdapterInfo struct {
	ZStackVariant int    `json:"zstackVariant"`
	SDKVersion    string `json:"sdkVersion"`
	BuildDate     uint32 `json:"buildDate"`
}

// CoordinatorInfo contains coordinator address information.
type CoordinatorInfo struct {
	IEEEAddress    string `json:"ieeeAddress"` // Hex string without colons
	NetworkAddress uint16 `json:"networkAddress"`
}

// NetworkConfig contains network configuration.
//
// SECURITY WARNING: This struct contains sensitive cryptographic material.
// The NetworkKey is the Zigbee network encryption key - if compromised,
// attackers can decrypt all network traffic and inject malicious commands.
//
// Best practices:
//   - Never commit backup files to version control
//   - Store backup files with restricted file permissions (0600)
//   - Consider encrypting backup files at rest
//   - Securely delete backup files when no longer needed
type NetworkConfig struct {
	PanID              uint16 `json:"panId"`
	ExtendedPanID      string `json:"extendedPanId"` // Hex string
	Channel            uint8  `json:"channel"`
	NetworkKey         string `json:"networkKey"` // Hex string (32 chars) - SENSITIVE
	NetworkKeySequence uint8  `json:"networkKeySequence"`
	SecurityLevel      uint8  `json:"securityLevel"`
	UpdateID           uint8  `json:"updateId"`
}

// SecurityInfo contains security-related information.
type SecurityInfo struct {
	FrameCounter           uint32 `json:"frameCounter"`
	TrustCenterLinkKeySeed string `json:"trustCenterLinkKeySeed,omitempty"` // Hex string
}

// DeviceEntry represents a paired device.
type DeviceEntry struct {
	IEEEAddress    string   `json:"ieeeAddress"` // Hex string
	NetworkAddress uint16   `json:"networkAddress"`
	Type           string   `json:"type"` // "router", "endDevice", "unknown"
	LinkKey        *LinkKey `json:"linkKey,omitempty"`
}

// LinkKey contains link key information for a device.
type LinkKey struct {
	Key       string `json:"key"` // Hex string (32 chars)
	TxCounter uint32 `json:"txCounter"`
	RxCounter uint32 `json:"rxCounter"`
}

// New creates a new backup with default values.
func New() *Backup {
	return &Backup{
		Version: FormatVersion,
		Created: time.Now().UTC(),
		Devices: make([]DeviceEntry, 0),
	}
}

// ToJSON serializes the backup to JSON.
func (b *Backup) ToJSON() ([]byte, error) {
	return json.MarshalIndent(b, "", "  ")
}

// FromJSON deserializes a backup from JSON.
func FromJSON(data []byte) (*Backup, error) {
	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, err
	}
	return &backup, nil
}

// Validate checks if the backup is valid.
func (b *Backup) Validate() error {
	if b.Version != FormatVersion {
		return fmt.Errorf("unsupported backup version: %d (expected %d)", b.Version, FormatVersion)
	}
	if b.Coordinator.IEEEAddress == "" {
		return fmt.Errorf("missing coordinator IEEE address")
	}
	if b.Network.NetworkKey == "" {
		return fmt.Errorf("missing network key")
	}
	return nil
}

// EncodeHex encodes bytes to hex string (no separators).
func EncodeHex(data []byte) string {
	return hex.EncodeToString(data)
}

// DecodeHex decodes a hex string to bytes.
func DecodeHex(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// EncodeIEEEAddr encodes an IEEE address to hex string.
func EncodeIEEEAddr(addr [8]byte) string {
	return hex.EncodeToString(addr[:])
}

// DecodeIEEEAddr decodes a hex string to IEEE address.
func DecodeIEEEAddr(s string) ([8]byte, error) {
	var addr [8]byte
	data, err := hex.DecodeString(s)
	if err != nil {
		return addr, err
	}
	if len(data) != 8 {
		return addr, fmt.Errorf("invalid IEEE address length: %d", len(data))
	}
	copy(addr[:], data)
	return addr, nil
}
