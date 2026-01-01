package zcl

import (
	"math"
	"testing"
)

func TestRGBToXY(t *testing.T) {
	tests := []struct {
		name         string
		r, g, b      uint8
		wantX, wantY uint16
		tolerance    uint16
	}{
		{
			name: "pure red",
			r:    255, g: 0, b: 0,
			wantX: 41779, wantY: 21542, // Approximately (0.640, 0.330) - sRGB red primary
			tolerance: 500,
		},
		{
			name: "pure green",
			r:    0, g: 255, b: 0,
			wantX: 19584, wantY: 39167, // Approximately (0.300, 0.600) - sRGB green primary
			tolerance: 500,
		},
		{
			name: "pure blue",
			r:    0, g: 0, b: 255,
			wantX: 9792, wantY: 3917, // Approximately (0.150, 0.060) - sRGB blue primary
			tolerance: 500,
		},
		{
			name: "white (all max)",
			r:    255, g: 255, b: 255,
			wantX: 20414, wantY: 21478, // Approximately (0.3127, 0.3290) - D65 white point
			tolerance: 500,
		},
		{
			name: "black (all zero)",
			r:    0, g: 0, b: 0,
			wantX: 0, wantY: 0,
			tolerance: 0,
		},
		{
			name: "gray (half brightness)",
			r:    128, g: 128, b: 128,
			wantX: 20414, wantY: 21478, // Should have same chromaticity as white
			tolerance: 500,
		},
		{
			name: "yellow (red + green)",
			r:    255, g: 255, b: 0,
			wantX: 27373, wantY: 32982, // Approximately (0.419, 0.505)
			tolerance: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotX, gotY := RGBToXY(tt.r, tt.g, tt.b)

			diffX := absDiff(gotX, tt.wantX)
			diffY := absDiff(gotY, tt.wantY)

			if diffX > tt.tolerance || diffY > tt.tolerance {
				t.Errorf("RGBToXY(%d, %d, %d) = (%d, %d), want approximately (%d, %d) with tolerance %d",
					tt.r, tt.g, tt.b, gotX, gotY, tt.wantX, tt.wantY, tt.tolerance)
			}
		})
	}
}

func TestXYToRGB(t *testing.T) {
	tests := []struct {
		name                string
		x, y                uint16
		brightness          uint8
		wantR, wantG, wantB uint8
		tolerance           uint8
	}{
		{
			name: "red with full brightness",
			x:    41779, y: 21542, brightness: 255,
			wantR: 255, wantG: 0, wantB: 0,
			tolerance: 15,
		},
		{
			name: "green with full brightness",
			x:    19584, y: 39167, brightness: 255,
			wantR: 0, wantG: 255, wantB: 0,
			tolerance: 15,
		},
		{
			name: "blue with full brightness",
			x:    9792, y: 3917, brightness: 255,
			wantR: 0, wantG: 0, wantB: 255,
			tolerance: 15,
		},
		{
			name: "white with full brightness",
			x:    20414, y: 21478, brightness: 255,
			wantR: 255, wantG: 255, wantB: 255,
			tolerance: 15,
		},
		{
			name: "white with half brightness",
			x:    20414, y: 21478, brightness: 128,
			wantR: 128, wantG: 128, wantB: 128,
			tolerance: 15,
		},
		{
			name: "black (zero brightness)",
			x:    20414, y: 21478, brightness: 0,
			wantR: 0, wantG: 0, wantB: 0,
			tolerance: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotR, gotG, gotB := XYToRGB(tt.x, tt.y, tt.brightness)

			diffR := absDiffUint8(gotR, tt.wantR)
			diffG := absDiffUint8(gotG, tt.wantG)
			diffB := absDiffUint8(gotB, tt.wantB)

			if diffR > tt.tolerance || diffG > tt.tolerance || diffB > tt.tolerance {
				t.Errorf("XYToRGB(%d, %d, %d) = RGB(%d, %d, %d), want approximately RGB(%d, %d, %d) with tolerance %d",
					tt.x, tt.y, tt.brightness, gotR, gotG, gotB, tt.wantR, tt.wantG, tt.wantB, tt.tolerance)
			}
		})
	}
}

func TestRGBToXYToRGB_RoundTrip(t *testing.T) {
	// Test that converting RGB -> XY -> RGB gives us back similar values
	tests := []struct {
		name      string
		r, g, b   uint8
		tolerance uint8
	}{
		{name: "red", r: 255, g: 0, b: 0, tolerance: 15},
		{name: "green", r: 0, g: 255, b: 0, tolerance: 15},
		{name: "blue", r: 0, g: 0, b: 255, tolerance: 15},
		{name: "white", r: 255, g: 255, b: 255, tolerance: 15},
		{name: "cyan", r: 0, g: 255, b: 255, tolerance: 15},
		{name: "magenta", r: 255, g: 0, b: 255, tolerance: 15},
		{name: "yellow", r: 255, g: 255, b: 0, tolerance: 15},
		{name: "gray", r: 128, g: 128, b: 128, tolerance: 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert RGB -> XY
			x, y := RGBToXY(tt.r, tt.g, tt.b)

			// Calculate brightness from original RGB
			brightness := maxUint8(tt.r, tt.g, tt.b)

			// Convert XY -> RGB
			gotR, gotG, gotB := XYToRGB(x, y, brightness)

			diffR := absDiffUint8(gotR, tt.r)
			diffG := absDiffUint8(gotG, tt.g)
			diffB := absDiffUint8(gotB, tt.b)

			if diffR > tt.tolerance || diffG > tt.tolerance || diffB > tt.tolerance {
				t.Errorf("Round trip RGB(%d,%d,%d) -> XY(%d,%d) -> RGB(%d,%d,%d), diff too large",
					tt.r, tt.g, tt.b, x, y, gotR, gotG, gotB)
			}
		})
	}
}

func TestKelvinToMireds(t *testing.T) {
	tests := []struct {
		name   string
		kelvin uint16
		want   uint16
	}{
		{name: "warm white (2700K)", kelvin: 2700, want: 370},
		{name: "warm white (3000K)", kelvin: 3000, want: 333},
		{name: "neutral white (4000K)", kelvin: 4000, want: 250},
		{name: "cool white (5000K)", kelvin: 5000, want: 200},
		{name: "daylight (6500K)", kelvin: 6500, want: 153},
		{name: "zero kelvin", kelvin: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := KelvinToMireds(tt.kelvin)
			if got != tt.want {
				t.Errorf("KelvinToMireds(%d) = %d, want %d", tt.kelvin, got, tt.want)
			}
		})
	}
}

func TestMiredsToKelvin(t *testing.T) {
	tests := []struct {
		name   string
		mireds uint16
		want   uint16
	}{
		{name: "370 mireds", mireds: 370, want: 2702},
		{name: "333 mireds", mireds: 333, want: 3003},
		{name: "250 mireds", mireds: 250, want: 4000},
		{name: "200 mireds", mireds: 200, want: 5000},
		{name: "153 mireds", mireds: 153, want: 6535},
		{name: "zero mireds", mireds: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MiredsToKelvin(tt.mireds)
			if got != tt.want {
				t.Errorf("MiredsToKelvin(%d) = %d, want %d", tt.mireds, got, tt.want)
			}
		})
	}
}

func TestKelvinMiredsRoundTrip(t *testing.T) {
	// Test that conversions are reversible (within rounding errors)
	kelvins := []uint16{2000, 2700, 3000, 4000, 5000, 6500}

	for _, k := range kelvins {
		t.Run("", func(t *testing.T) {
			mireds := KelvinToMireds(k)
			kelvinBack := MiredsToKelvin(mireds)

			// Allow small rounding error (larger tolerance for higher Kelvin values)
			diff := absDiff(k, kelvinBack)
			if diff > 50 {
				t.Errorf("Round trip Kelvin(%d) -> Mireds(%d) -> Kelvin(%d), diff = %d",
					k, mireds, kelvinBack, diff)
			}
		})
	}
}

func TestClampToGamut(t *testing.T) {
	tests := []struct {
		name       string
		x, y       float64
		gamut      Gamut
		wantInside bool
	}{
		{
			name: "point inside GamutC",
			x:    0.4, y: 0.4,
			gamut:      GamutC,
			wantInside: true,
		},
		{
			name: "point outside GamutC (too red)",
			x:    0.8, y: 0.2,
			gamut:      GamutC,
			wantInside: false,
		},
		{
			name: "point outside GamutC (too green)",
			x:    0.1, y: 0.8,
			gamut:      GamutC,
			wantInside: false,
		},
		{
			name: "point at red vertex of GamutA",
			x:    0.704, y: 0.296,
			gamut:      GamutA,
			wantInside: true,
		},
		{
			name: "point at green vertex of GamutB",
			x:    0.409, y: 0.518,
			gamut:      GamutB,
			wantInside: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clampedX, clampedY := ClampToGamut(tt.x, tt.y, tt.gamut)

			// Check if the clamped point is within the gamut
			if !isPointInGamut(Point{X: clampedX, Y: clampedY}, tt.gamut) {
				t.Errorf("ClampToGamut(%f, %f) = (%f, %f) is not inside the gamut",
					tt.x, tt.y, clampedX, clampedY)
			}

			// If point was inside, it should remain unchanged
			if tt.wantInside {
				if math.Abs(clampedX-tt.x) > 0.001 || math.Abs(clampedY-tt.y) > 0.001 {
					t.Errorf("ClampToGamut(%f, %f) = (%f, %f), point was inside so should be unchanged",
						tt.x, tt.y, clampedX, clampedY)
				}
			}
		})
	}
}

func TestIsPointInGamut(t *testing.T) {
	tests := []struct {
		name  string
		point Point
		gamut Gamut
		want  bool
	}{
		{
			name:  "center of GamutC",
			point: Point{X: 0.4, Y: 0.4},
			gamut: GamutC,
			want:  true,
		},
		{
			name:  "red vertex of GamutC",
			point: Point{X: 0.6915, Y: 0.3083},
			gamut: GamutC,
			want:  true,
		},
		{
			name:  "green vertex of GamutC",
			point: Point{X: 0.17, Y: 0.7},
			gamut: GamutC,
			want:  true,
		},
		{
			name:  "blue vertex of GamutC",
			point: Point{X: 0.1532, Y: 0.0475},
			gamut: GamutC,
			want:  true,
		},
		{
			name:  "outside GamutC (far right)",
			point: Point{X: 0.9, Y: 0.3},
			gamut: GamutC,
			want:  false,
		},
		{
			name:  "outside GamutC (far up)",
			point: Point{X: 0.3, Y: 0.9},
			gamut: GamutC,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPointInGamut(tt.point, tt.gamut)
			if got != tt.want {
				t.Errorf("isPointInGamut(%+v, %+v) = %v, want %v",
					tt.point, tt.gamut, got, tt.want)
			}
		})
	}
}

func TestGamutConstants(t *testing.T) {
	// Verify the gamut constants are defined correctly
	gamuts := []struct {
		name  string
		gamut Gamut
	}{
		{"GamutA", GamutA},
		{"GamutB", GamutB},
		{"GamutC", GamutC},
	}

	for _, g := range gamuts {
		t.Run(g.name, func(t *testing.T) {
			// All points should have valid coordinates (0-1 range)
			checkPoint := func(name string, p Point) {
				if p.X < 0 || p.X > 1 || p.Y < 0 || p.Y > 1 {
					t.Errorf("%s.%s = (%f, %f), coordinates should be in range [0, 1]",
						g.name, name, p.X, p.Y)
				}
			}

			checkPoint("Red", g.gamut.Red)
			checkPoint("Green", g.gamut.Green)
			checkPoint("Blue", g.gamut.Blue)
		})
	}
}

// Helper functions

func absDiff(a, b uint16) uint16 {
	if a > b {
		return a - b
	}
	return b - a
}

func absDiffUint8(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

func maxUint8(values ...uint8) uint8 {
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}
