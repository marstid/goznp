package state

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrDeviceNotFound indicates the requested device doesn't exist.
	ErrDeviceNotFound = errors.New("device not found")

	// ErrNameInUse indicates a device name conflicts with an existing device.
	ErrNameInUse = errors.New("name in use")

	// ErrInvalidIdentifier indicates the identifier is invalid.
	ErrInvalidIdentifier = errors.New("invalid identifier")
)

// IdentifierType represents the type of identifier (slug or IEEE address).
type IdentifierType string

const (
	IdentifierSlug IdentifierType = "slug"
	IdentifierIEEE IdentifierType = "ieee"
)

// Identifier represents either a slug or IEEE address.
type Identifier struct {
	Type  IdentifierType
	Value string
}

// ParseIEEEAddr parses an IEEE address string into an 8-byte array.
// Accepts formats: "aa:bb:cc:dd:ee:ff:00:11" or "aabbccddeeff0011"
func ParseIEEEAddr(s string) ([8]byte, error) {
	var ieeeAddr [8]byte

	// Remove colons if present
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ToLower(s)

	// Check length
	if len(s) != 16 {
		return ieeeAddr, fmt.Errorf("invalid IEEE address: must be 16 hex characters")
	}

	// Parse hex
	for i := 0; i < 16; i += 2 {
		var b byte
		_, err := fmt.Sscanf(s[i:i+2], "%02x", &b)
		if err != nil {
			return ieeeAddr, fmt.Errorf("invalid IEEE address at position %d: %w", i, err)
		}
		ieeeAddr[i/2] = b
	}

	return ieeeAddr, nil
}

// FormatIEEEAddr formats an 8-byte IEEE address as a colon-separated string.
// Format: "aa:bb:cc:dd:ee:ff:00:11"
func FormatIEEEAddr(ieeeAddr [8]byte) string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x",
		ieeeAddr[7], ieeeAddr[6], ieeeAddr[5], ieeeAddr[4],
		ieeeAddr[3], ieeeAddr[2], ieeeAddr[1], ieeeAddr[0])
}

// ParseIdentifier parses a path parameter into an Identifier.
// Priority: 1) Try as IEEE address (with or without colons), 2) Try as slug
func ParseIdentifier(param string) (*Identifier, error) {
	if param == "" {
		return nil, ErrInvalidIdentifier
	}

	// Try as IEEE address first
	if _, err := ParseIEEEAddr(param); err == nil {
		return &Identifier{
			Type:  IdentifierIEEE,
			Value: param,
		}, nil
	}

	// Must be a slug
	if !IsValidSlug(param) {
		return nil, ErrInvalidIdentifier
	}

	return &Identifier{
		Type:  IdentifierSlug,
		Value: param,
	}, nil
}

// ResolveIdentifier resolves an identifier to an IEEE address using a lookup function.
// Takes a slug-to-IEEE lookup function and returns the IEEE address.
func ResolveIdentifier(ident *Identifier, slugToIEEE func(string) ([8]byte, bool)) ([8]byte, error) {
	switch ident.Type {
	case IdentifierIEEE:
		return ParseIEEEAddr(ident.Value)
	case IdentifierSlug:
		if ieeeAddr, ok := slugToIEEE(ident.Value); ok {
			return ieeeAddr, nil
		}
		return [8]byte{}, ErrDeviceNotFound
	default:
		return [8]byte{}, ErrInvalidIdentifier
	}
}
