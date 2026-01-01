package zcl

import "math"

// Point represents a 2D coordinate in CIE color space.
type Point struct {
	X float64
	Y float64
}

// Gamut defines a color gamut triangle using three primary color points.
// Each gamut represents the range of colors a specific device can produce.
type Gamut struct {
	Red   Point
	Green Point
	Blue  Point
}

// Common Philips Hue color gamuts for different generations of bulbs.
var (
	// GamutA is used by older Hue bulbs (A19 1st gen, BR30, etc.)
	GamutA = Gamut{
		Red:   Point{X: 0.704, Y: 0.296},
		Green: Point{X: 0.2151, Y: 0.7106},
		Blue:  Point{X: 0.138, Y: 0.08},
	}

	// GamutB is used by some Hue bulbs (A19 2nd gen, BR30 2nd gen, etc.)
	GamutB = Gamut{
		Red:   Point{X: 0.675, Y: 0.322},
		Green: Point{X: 0.409, Y: 0.518},
		Blue:  Point{X: 0.167, Y: 0.04},
	}

	// GamutC is used by newer Hue bulbs (A19 3rd gen+, LightStrips Plus, etc.)
	GamutC = Gamut{
		Red:   Point{X: 0.6915, Y: 0.3083},
		Green: Point{X: 0.17, Y: 0.7},
		Blue:  Point{X: 0.1532, Y: 0.0475},
	}
)

// RGBToXY converts RGB color values to CIE 1931 XY color space coordinates.
// RGB values are in range 0-255, output XY values are in range 0-65279 (uint16).
// This function applies sRGB gamma correction and uses the sRGB to XYZ transformation.
func RGBToXY(r, g, b uint8) (x, y uint16) {
	// Normalize RGB values to 0.0-1.0
	red := float64(r) / 255.0
	green := float64(g) / 255.0
	blue := float64(b) / 255.0

	// Apply reverse gamma correction (sRGB to linear)
	red = gammaCorrect(red)
	green = gammaCorrect(green)
	blue = gammaCorrect(blue)

	// Convert to XYZ using sRGB to XYZ transformation matrix (D65 illuminant)
	X := red*0.4124564 + green*0.3575761 + blue*0.1804375
	Y := red*0.2126729 + green*0.7151522 + blue*0.0721750
	Z := red*0.0193339 + green*0.1191920 + blue*0.9503041

	// Avoid division by zero
	sum := X + Y + Z
	if sum < 0.00001 {
		// Return black (0, 0) for very dark colors
		return 0, 0
	}

	// Convert XYZ to xy chromaticity coordinates
	xFloat := X / sum
	yFloat := Y / sum

	// Convert to uint16 range (0-65279)
	x = uint16(math.Round(xFloat * 65279.0))
	y = uint16(math.Round(yFloat * 65279.0))

	return x, y
}

// gammaCorrect applies reverse sRGB gamma correction to convert from sRGB to linear RGB.
func gammaCorrect(value float64) float64 {
	if value > 0.04045 {
		return math.Pow((value+0.055)/1.055, 2.4)
	}
	return value / 12.92
}

// XYToRGB converts CIE 1931 XY color space coordinates back to RGB color values.
// XY values are in range 0-65279 (uint16), brightness is 0-255.
// Output RGB values are in range 0-255.
// The brightness parameter is required since XY coordinates don't encode luminance.
func XYToRGB(x, y uint16, brightness uint8) (r, g, b uint8) {
	// Normalize xy to 0.0-1.0
	xFloat := float64(x) / 65279.0
	yFloat := float64(y) / 65279.0

	// Avoid division by zero
	if yFloat < 0.00001 {
		return 0, 0, 0
	}

	// Calculate z
	z := 1.0 - xFloat - yFloat

	// Convert xy to XYZ (Y component comes from brightness)
	Y := float64(brightness) / 255.0
	X := (Y / yFloat) * xFloat
	Z := (Y / yFloat) * z

	// Convert XYZ to RGB using XYZ to sRGB transformation matrix (D65 illuminant)
	red := X*3.2404542 + Y*-1.5371385 + Z*-0.4985314
	green := X*-0.9692660 + Y*1.8760108 + Z*0.0415560
	blue := X*0.0556434 + Y*-0.2040259 + Z*1.0572252

	// Apply gamma correction (linear to sRGB)
	red = reverseGammaCorrect(red)
	green = reverseGammaCorrect(green)
	blue = reverseGammaCorrect(blue)

	// Scale to match the desired brightness
	// The brightness parameter should correspond to the maximum RGB component
	maxRGB := math.Max(red, math.Max(green, blue))
	if maxRGB > 0.001 {
		scale := (float64(brightness) / 255.0) / maxRGB
		red *= scale
		green *= scale
		blue *= scale
	}

	// Clamp to valid range and convert to uint8
	r = uint8(math.Max(0, math.Min(255, math.Round(red*255.0))))
	g = uint8(math.Max(0, math.Min(255, math.Round(green*255.0))))
	b = uint8(math.Max(0, math.Min(255, math.Round(blue*255.0))))

	return r, g, b
}

// reverseGammaCorrect applies sRGB gamma correction to convert from linear RGB to sRGB.
func reverseGammaCorrect(value float64) float64 {
	if value > 0.0031308 {
		return 1.055*math.Pow(value, 1.0/2.4) - 0.055
	}
	return value * 12.92
}

// KelvinToMireds converts color temperature from Kelvin to mireds.
// Mireds (micro reciprocal degrees) = 1,000,000 / Kelvin
func KelvinToMireds(kelvin uint16) uint16 {
	if kelvin == 0 {
		return 0
	}
	return uint16(1000000 / uint32(kelvin))
}

// MiredsToKelvin converts color temperature from mireds to Kelvin.
// Kelvin = 1,000,000 / mireds
func MiredsToKelvin(mireds uint16) uint16 {
	if mireds == 0 {
		return 0
	}
	return uint16(1000000 / uint32(mireds))
}

// ClampToGamut clamps an XY color point to be within the specified gamut triangle.
// If the point is already within the gamut, it returns the original point.
// If the point is outside, it returns the nearest point on the gamut triangle.
func ClampToGamut(x, y float64, gamut Gamut) (float64, float64) {
	point := Point{X: x, Y: y}

	// Check if point is inside the gamut triangle
	if isPointInGamut(point, gamut) {
		return x, y
	}

	// Find the closest point on the triangle edges
	closestPoint := closestPointOnTriangle(point, gamut)
	return closestPoint.X, closestPoint.Y
}

// isPointInGamut checks if a point is inside the gamut triangle using barycentric coordinates.
func isPointInGamut(p Point, gamut Gamut) bool {
	const epsilon = 0.0001 // Small tolerance for floating point comparison

	v0 := Point{
		X: gamut.Blue.X - gamut.Red.X,
		Y: gamut.Blue.Y - gamut.Red.Y,
	}
	v1 := Point{
		X: gamut.Green.X - gamut.Red.X,
		Y: gamut.Green.Y - gamut.Red.Y,
	}
	v2 := Point{
		X: p.X - gamut.Red.X,
		Y: p.Y - gamut.Red.Y,
	}

	dot00 := v0.X*v0.X + v0.Y*v0.Y
	dot01 := v0.X*v1.X + v0.Y*v1.Y
	dot02 := v0.X*v2.X + v0.Y*v2.Y
	dot11 := v1.X*v1.X + v1.Y*v1.Y
	dot12 := v1.X*v2.X + v1.Y*v2.Y

	denom := dot00*dot11 - dot01*dot01
	if math.Abs(denom) < epsilon {
		return false // Degenerate triangle
	}

	invDenom := 1.0 / denom
	u := (dot11*dot02 - dot01*dot12) * invDenom
	v := (dot00*dot12 - dot01*dot02) * invDenom

	return (u >= -epsilon) && (v >= -epsilon) && (u+v <= 1.0+epsilon)
}

// closestPointOnTriangle finds the closest point on the gamut triangle to the given point.
func closestPointOnTriangle(p Point, gamut Gamut) Point {
	// Check distance to each edge of the triangle
	edges := []struct {
		start Point
		end   Point
	}{
		{gamut.Red, gamut.Green},
		{gamut.Green, gamut.Blue},
		{gamut.Blue, gamut.Red},
	}

	var closestPoint Point
	minDistance := math.MaxFloat64

	for _, edge := range edges {
		point := closestPointOnSegment(p, edge.start, edge.end)
		distance := distanceSquared(p, point)
		if distance < minDistance {
			minDistance = distance
			closestPoint = point
		}
	}

	return closestPoint
}

// closestPointOnSegment finds the closest point on a line segment to a given point.
func closestPointOnSegment(p, start, end Point) Point {
	dx := end.X - start.X
	dy := end.Y - start.Y

	if dx == 0 && dy == 0 {
		return start
	}

	t := ((p.X-start.X)*dx + (p.Y-start.Y)*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))

	return Point{
		X: start.X + t*dx,
		Y: start.Y + t*dy,
	}
}

// distanceSquared calculates the squared distance between two points.
func distanceSquared(p1, p2 Point) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return dx*dx + dy*dy
}
