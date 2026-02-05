package state

import (
	"testing"
)

func TestSlugFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "Outdoor Plug",
			expected: "outdoor-plug",
		},
		{
			name:     "with hyphens",
			input:    "Garage-Outlet 2",
			expected: "garage-outlet-2",
		},
		{
			name:     "with emoji",
			input:    "Front Door 🚪",
			expected: "front-door",
		},
		{
			name:     "with underscores",
			input:    "Living_Room_Light",
			expected: "living-room-light",
		},
		{
			name:     "with dots",
			input:    "Bed.Room.Light",
			expected: "bed-room-light",
		},
		{
			name:     "with multiple spaces",
			input:    "  Spaces  Before/After  ",
			expected: "spaces-before-after",
		},
		{
			name:     "consecutive hyphens",
			input:    "Multiple---Hyphens",
			expected: "multiple-hyphens",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only special characters",
			input:    "!!!@@@###",
			expected: "",
		},
		{
			name:     "mixed case",
			input:    "MixedCaseString",
			expected: "mixedcasestring",
		},
		{
			name:     "with numbers",
			input:    "Living Room 2",
			expected: "living-room-2",
		},
		{
			name:     "leading/trailing hyphens",
			input:    "-Test-Data-",
			expected: "test-data",
		},
		{
			name:     "consecutive spaces and hyphens",
			input:    "Test  ---  Data",
			expected: "test-data",
		},
		{
			name:     "longer than 64 chars",
			input:    "This Is A Very Long Device Name That Should Be Truncated Because It Exceeds Sixty Four Characters",
			expected: "this-is-a-very-long-device-name-that-should-be-truncated-because",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SlugFormat(tt.input)
			if got != tt.expected {
				t.Errorf("SlugFormat(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsValidSlug(t *testing.T) {
	tests := []struct {
		name string
		slug string
		want bool
	}{
		{
			name: "valid simple slug",
			slug: "outdoor-plug",
			want: true,
		},
		{
			name: "valid with numbers",
			slug: "living-room-2",
			want: true,
		},
		{
			name: "valid with hyphens",
			slug: "test-data-here",
			want: true,
		},
		{
			name: "empty string",
			slug: "",
			want: false,
		},
		{
			name: "too long",
			slug: "this-is-a-very-slug-that-is-exactly-sixty-five-characters-long-a-b-c-d-e-f",
			want: false,
		},
		{
			name: "contains uppercase",
			slug: "Outdoor-Plug",
			want: false,
		},
		{
			name: "contains underscore",
			slug: "outdoor_plug",
			want: false,
		},
		{
			name: "contains space",
			slug: "outdoor plug",
			want: false,
		},
		{
			name: "leading hyphen",
			slug: "-outdoor-plug",
			want: false,
		},
		{
			name: "trailing hyphen",
			slug: "outdoor-plug-",
			want: false,
		},
		{
			name: "consecutive hyphens",
			slug: "outdoor--plug",
			want: false,
		},
		{
			name: "exactly 64 chars",
			slug: "this-is-a-slug-that-is-exactly-sixty-four-chars-long-abcdefghijk",
			want: true,
		},
		{
			name: "special characters",
			slug: "outdoor@plug",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidSlug(tt.slug)
			if got != tt.want {
				t.Errorf("IsValidSlug(%q) = %v, want %v", tt.slug, got, tt.want)
			}
		})
	}
}

func TestGetDefaultSlugForIEEE(t *testing.T) {
	tests := []struct {
		name     string
		ieeeAddr [8]byte
		expected string
	}{
		{
			name:     "all zeros",
			ieeeAddr: [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: "device-0000000000000000",
		},
		{
			name:     "non-zero address",
			ieeeAddr: [8]byte{0xa4, 0xc1, 0x38, 0x5d, 0x1a, 0x42, 0x7f, 0x16},
			expected: "device-a4c1385d1a427f16",
		},
		{
			name:     "another address",
			ieeeAddr: [8]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			expected: "device-ffffffffffffffff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDefaultSlugForIEEE(tt.ieeeAddr)
			if got != tt.expected {
				t.Errorf("GetDefaultSlugForIEEE(%x) = %q, want %q", tt.ieeeAddr, got, tt.expected)
			}
		})
	}
}

func TestSlugRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple", "Living Room Light"},
		{"with symbols", "Garage-Outlet #2"},
		{"with spaces", "  Front Door  "},
		{"mixed case", "Bedroom Ceiling Fan"},
		{"with numbers", "Kitchen Plug 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug := SlugFormat(tt.input)
			if !IsValidSlug(slug) {
				t.Errorf("SlugFormat(%q) produced invalid slug %q", tt.input, slug)
			}
		})
	}
}
