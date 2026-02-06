package state

import (
	"fmt"
	"strings"
	"unicode"
)

// SlugFormat converts a device name to a URL-safe slug.
// Rules:
//   - Lowercase only
//   - Alphanumeric characters (a-z, 0-9) and hyphens only
//   - Replace spaces and other separators with single hyphen
//   - Remove any characters that aren't alphanumeric or hyphens
//   - Remove leading/trailing hyphens
//   - Remove consecutive hyphens
//   - Maximum 64 characters
//
// Examples:
//
//	"Outdoor Plug"                -> "outdoor-plug"
//	"Garage-Outlet 2"             -> "garage-outlet-2"
//	"Front Door 🚪"              -> "front-door"
//	"Living Room"                 -> "living-room"
//	"  Spaces  Before/After  "    -> "spaces-before-after"
//	"Multiple---Hyphens"          -> "multiple-hyphens"
//	""                            -> ""
func SlugFormat(name string) string {
	if name == "" {
		return ""
	}

	// Convert to lowercase
	result := strings.ToLower(name)

	// Replace non-alphanumeric characters (except hyphens) with spaces
	var builder strings.Builder
	for _, r := range result {
		switch {
		case r == '-':
			builder.WriteByte('-')
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			builder.WriteRune(r)
		case unicode.IsSpace(r) || r == '_' || r == '.' || r == '/':
			builder.WriteByte('-')
		default:
			// Other characters become hyphens (including emojis)
			builder.WriteByte('-')
		}
	}

	result = builder.String()

	// Replace multiple hyphens with single hyphen
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// Trim hyphens from start and end
	result = strings.Trim(result, "-")

	// Limit to 64 characters
	if len(result) > 64 {
		result = result[:64]
	}

	return result
}

// IsValidSlug checks if a slug is valid.
// Valid: lowercase alphanumeric + hyphens, not empty, max 64 chars
func IsValidSlug(slug string) bool {
	if slug == "" || len(slug) > 64 {
		return false
	}

	for _, r := range slug {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}

	// No leading or trailing hyphens
	if slug[0] == '-' || slug[len(slug)-1] == '-' {
		return false
	}

	// No consecutive hyphens
	if strings.Contains(slug, "--") {
		return false
	}

	return true
}

// GetDefaultSlugForIEEE generates a default slug from an IEEE address.
// Format: "device-xxxxxxxxxxxxxxxx" (lowercase hex)
func GetDefaultSlugForIEEE(ieeeAddr [8]byte) string {
	hexParts := make([]string, 8)
	for i, b := range ieeeAddr {
		hexParts[i] = fmt.Sprintf("%02x", b)
	}
	return "device-" + strings.Join(hexParts, "")
}
