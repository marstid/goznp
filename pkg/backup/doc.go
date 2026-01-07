// Package backup provides functionality for backing up and restoring Zigbee
// adapter configurations.
//
// The backup format captures all essential information needed to recreate a
// Zigbee network, including:
//
//   - Adapter firmware information (Z-Stack variant, SDK version)
//   - Coordinator address information
//   - Network configuration (PAN ID, channel, network key)
//   - Security settings (frame counters, trust center keys)
//   - Paired device list (IEEE addresses, network addresses, link keys)
//
// # Backup Format
//
// Backups are stored as JSON with the following structure:
//
//	{
//	  "version": 1,
//	  "created": "2025-01-01T12:00:00Z",
//	  "adapter": {
//	    "zstackVariant": 3,
//	    "sdkVersion": "5.30.00.67",
//	    "buildDate": 20230615
//	  },
//	  "coordinator": {
//	    "ieeeAddress": "00124b0024c8e8e8",
//	    "networkAddress": 0
//	  },
//	  "network": {
//	    "panId": 6738,
//	    "extendedPanId": "dddddddddddddddd",
//	    "channel": 15,
//	    "networkKey": "01030507090b0d0f00020406080a0c0d",
//	    "networkKeySequence": 0,
//	    "securityLevel": 5,
//	    "updateId": 0
//	  },
//	  "security": {
//	    "frameCounter": 12345
//	  },
//	  "devices": [
//	    {
//	      "ieeeAddress": "00158d00048e5f9a",
//	      "networkAddress": 4660,
//	      "type": "router",
//	      "linkKey": {
//	        "key": "5a6967426565416c6c69616e63653039",
//	        "txCounter": 100,
//	        "rxCounter": 200
//	      }
//	    }
//	  ]
//	}
//
// # Usage Example
//
//	// Create a new backup and populate it
//	backup := backup.New()
//	backup.Adapter.ZStackVariant = 3
//	backup.Adapter.SDKVersion = "5.30.00.67"
//	backup.Adapter.BuildDate = 20230615
//
//	// Set network configuration
//	backup.Network.PanID = 0x1a62
//	backup.Network.Channel = 15
//	backup.Network.NetworkKey = backup.EncodeHex(networkKeyBytes)
//
//	// Add device
//	backup.Devices = append(backup.Devices, backup.DeviceEntry{
//	    IEEEAddress:    backup.EncodeIEEEAddr(ieeeAddr),
//	    NetworkAddress: 0x1234,
//	    Type:           "router",
//	})
//
//	// Serialize to JSON
//	data, err := backup.ToJSON()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Save to file
//	err = os.WriteFile("zstack-backup.json", data, 0600)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Later, restore from backup
//	data, err := os.ReadFile("zstack-backup.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	restored, err := backup.FromJSON(data)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Validate before using
//	if err := restored.Validate(); err != nil {
//	    log.Fatal(err)
//	}
//
// # Security Considerations
//
// This package handles sensitive cryptographic material:
//
//   - NetworkKey - The Zigbee network encryption key (16 bytes)
//   - TrustCenterLinkKeySeed - Optional trust center key material
//   - Device LinkKeys - Per-device encryption keys
//
// Best practices for handling backup files:
//
//   - Never commit backup files to version control
//   - Store backup files with restricted file permissions (0600)
//   - Consider encrypting backup files at rest using tools like gpg
//   - Securely delete backup files when no longer needed (shred/secure-delete)
//   - Only restore backups to the same hardware/firmware combination
//
// The backup format stores all cryptographic keys in hexadecimal string format
// for JSON compatibility. These keys can be converted back to bytes using
// DecodeHex() and DecodeIEEEAddr().
//
// # Hex Encoding
//
// The package uses hex encoding for binary data in the JSON format:
//
//   - NetworkKey, ExtendedPanID - 16-byte keys as 32-character hex strings
//   - IEEEAddress - 8-byte addresses as 16-character hex strings
//   - All hex strings are lowercase with no separators
//
// Use EncodeHex() and DecodeHex() for general hex encoding, or EncodeIEEEAddr()
// and DecodeIEEEAddr() for Zigbee IEEE addresses specifically.
//
// # Versioning
//
// The backup format includes a version field to allow for schema evolution.
// Current FormatVersion is 1. When loading backups, the Validate() method
// checks that the version is supported. Future versions may add or modify
// fields while maintaining backward compatibility where possible.
package backup
