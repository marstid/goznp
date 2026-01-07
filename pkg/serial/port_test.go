package serial

import (
	"runtime"
	"testing"
	"time"
)

func TestValidatePortPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		goos    string // empty means run on all platforms.
	}{
		// Linux paths.
		{
			name:    "valid Linux USB port",
			path:    "/dev/ttyUSB0",
			wantErr: runtime.GOOS != "linux",
			goos:    "linux",
		},
		{
			name:    "valid Linux ACM port",
			path:    "/dev/ttyACM0",
			wantErr: runtime.GOOS != "linux",
			goos:    "linux",
		},
		{
			name:    "valid Linux serial port",
			path:    "/dev/ttyS0",
			wantErr: runtime.GOOS != "linux",
			goos:    "linux",
		},
		{
			name:    "valid Linux AMA port",
			path:    "/dev/ttyAMA0",
			wantErr: runtime.GOOS != "linux",
			goos:    "linux",
		},
		{
			name:    "valid Linux serial by-id",
			path:    "/dev/serial/by-id/usb-device",
			wantErr: runtime.GOOS != "linux",
			goos:    "linux",
		},
		{
			name:    "invalid Linux path",
			path:    "/dev/sda1",
			wantErr: true,
			goos:    "linux",
		},

		// macOS paths.
		{
			name:    "valid macOS tty.usbserial",
			path:    "/dev/tty.usbserial",
			wantErr: runtime.GOOS != "darwin",
			goos:    "darwin",
		},
		{
			name:    "valid macOS tty.usbserial-1234",
			path:    "/dev/tty.usbserial-1234",
			wantErr: runtime.GOOS != "darwin",
			goos:    "darwin",
		},
		{
			name:    "valid macOS cu.usbmodem",
			path:    "/dev/cu.usbmodem",
			wantErr: runtime.GOOS != "darwin",
			goos:    "darwin",
		},
		{
			name:    "valid macOS cu.usbserial-ABCD",
			path:    "/dev/cu.usbserial-ABCD",
			wantErr: runtime.GOOS != "darwin",
			goos:    "darwin",
		},
		{
			name:    "invalid macOS ttyUSB",
			path:    "/dev/ttyUSB0",
			wantErr: true,
			goos:    "darwin",
		},

		// Windows paths.
		{
			name:    "valid Windows COM1",
			path:    "COM1",
			wantErr: runtime.GOOS != "windows",
			goos:    "windows",
		},
		{
			name:    "valid Windows COM10",
			path:    "COM10",
			wantErr: runtime.GOOS != "windows",
			goos:    "windows",
		},
		{
			name:    "valid Windows lowercase com1",
			path:    "com1",
			wantErr: runtime.GOOS != "windows",
			goos:    "windows",
		},
		{
			name:    "valid Windows \\\\.\\COM1",
			path:    "\\\\.\\COM1",
			wantErr: runtime.GOOS != "windows",
			goos:    "windows",
		},
		{
			name:    "valid Windows \\\\.\\COM10",
			path:    "\\\\.\\COM10",
			wantErr: runtime.GOOS != "windows",
			goos:    "windows",
		},
		{
			name:    "invalid Windows LPT1",
			path:    "LPT1",
			wantErr: true,
			goos:    "windows",
		},

		// Path traversal attacks (should fail on all platforms).
		{
			name:    "path traversal with ..",
			path:    "../etc/passwd",
			wantErr: true,
		},
		{
			name:    "path traversal in middle",
			path:    "/dev/../etc/passwd",
			wantErr: true,
		},
		{
			name:    "path traversal complex",
			path:    "/dev/ttyUSB0/../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "absolute path traversal",
			path:    "/../etc/passwd",
			wantErr: true,
		},

		// Invalid paths (should fail on all platforms).
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "random file path",
			path:    "/tmp/not-a-port",
			wantErr: true,
		},
		{
			name:    "home directory",
			path:    "/home/user/file",
			wantErr: true,
		},
		{
			name:    "root path",
			path:    "/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests for other platforms if goos is specified.
			if tt.goos != "" && tt.goos != runtime.GOOS {
				t.Skipf("skipping %s test on %s", tt.goos, runtime.GOOS)
			}

			err := validatePortPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePortPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BaudRate != 115200 {
		t.Errorf("DefaultConfig().BaudRate = %d, want 115200", cfg.BaudRate)
	}

	if cfg.ReadTimeout != 100*time.Millisecond {
		t.Errorf("DefaultConfig().ReadTimeout = %v, want 100ms", cfg.ReadTimeout)
	}

	if cfg.RTSCTSFlow != false {
		t.Errorf("DefaultConfig().RTSCTSFlow = %v, want false", cfg.RTSCTSFlow)
	}

	if cfg.Path != "" {
		t.Errorf("DefaultConfig().Path = %q, want empty string", cfg.Path)
	}
}

func TestIsCC2652Adapter(t *testing.T) {
	tests := []struct {
		name string
		info USBPortInfo
		want bool
	}{
		{
			name: "CH340 adapter (uppercase)",
			info: USBPortInfo{
				Name:  "/dev/ttyUSB0",
				IsUSB: true,
				VID:   "1A86",
				PID:   "55D4",
			},
			want: true,
		},
		{
			name: "CH340 adapter (lowercase)",
			info: USBPortInfo{
				Name:  "/dev/ttyUSB0",
				IsUSB: true,
				VID:   "1a86",
				PID:   "55d4",
			},
			want: true,
		},
		{
			name: "CH340 adapter (mixed case)",
			info: USBPortInfo{
				Name:  "/dev/ttyUSB0",
				IsUSB: true,
				VID:   "1a86",
				PID:   "55D4",
			},
			want: true,
		},
		{
			name: "CP2102 adapter (uppercase)",
			info: USBPortInfo{
				Name:  "/dev/ttyUSB1",
				IsUSB: true,
				VID:   "10C4",
				PID:   "EA60",
			},
			want: true,
		},
		{
			name: "CP2102 adapter (lowercase)",
			info: USBPortInfo{
				Name:  "/dev/ttyUSB1",
				IsUSB: true,
				VID:   "10c4",
				PID:   "ea60",
			},
			want: true,
		},
		{
			name: "CP2102 adapter (mixed case)",
			info: USBPortInfo{
				Name:  "/dev/ttyUSB1",
				IsUSB: true,
				VID:   "10C4",
				PID:   "ea60",
			},
			want: true,
		},
		{
			name: "non-USB port",
			info: USBPortInfo{
				Name:  "/dev/ttyS0",
				IsUSB: false,
				VID:   "",
				PID:   "",
			},
			want: false,
		},
		{
			name: "unknown USB device",
			info: USBPortInfo{
				Name:  "/dev/ttyUSB2",
				IsUSB: true,
				VID:   "DEAD",
				PID:   "BEEF",
			},
			want: false,
		},
		{
			name: "CH340 with wrong PID",
			info: USBPortInfo{
				Name:  "/dev/ttyUSB3",
				IsUSB: true,
				VID:   "1A86",
				PID:   "1234",
			},
			want: false,
		},
		{
			name: "CP2102 with wrong VID",
			info: USBPortInfo{
				Name:  "/dev/ttyUSB4",
				IsUSB: true,
				VID:   "1234",
				PID:   "EA60",
			},
			want: false,
		},
		{
			name: "empty VID/PID",
			info: USBPortInfo{
				Name:  "/dev/ttyUSB5",
				IsUSB: true,
				VID:   "",
				PID:   "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCC2652Adapter(tt.info)
			if got != tt.want {
				t.Errorf("IsCC2652Adapter(%+v) = %v, want %v", tt.info, got, tt.want)
			}
		})
	}
}

func TestToUpperHex(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercase hex",
			input: "1a86",
			want:  "1A86",
		},
		{
			name:  "uppercase hex",
			input: "1A86",
			want:  "1A86",
		},
		{
			name:  "mixed case hex",
			input: "1a8A",
			want:  "1A8A",
		},
		{
			name:  "all lowercase letters",
			input: "abcdef",
			want:  "ABCDEF",
		},
		{
			name:  "all uppercase letters",
			input: "ABCDEF",
			want:  "ABCDEF",
		},
		{
			name:  "numbers only",
			input: "123456",
			want:  "123456",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "single lowercase letter",
			input: "a",
			want:  "A",
		},
		{
			name:  "single uppercase letter",
			input: "F",
			want:  "F",
		},
		{
			name:  "CP2102 VID lowercase",
			input: "10c4",
			want:  "10C4",
		},
		{
			name:  "CP2102 PID lowercase",
			input: "ea60",
			want:  "EA60",
		},
		{
			name:  "CH340 VID mixed",
			input: "1a86",
			want:  "1A86",
		},
		{
			name:  "CH340 PID mixed",
			input: "55d4",
			want:  "55D4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toUpperHex(tt.input)
			if got != tt.want {
				t.Errorf("toUpperHex(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
