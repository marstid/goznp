//go:build ignore

// Example: Read sensor data from Zigbee sensors.
// This demonstrates reading attributes from common sensor types.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/zcl"
)

func main() {
	// Get serial port from environment
	port := os.Getenv("GOZNP_PORT")
	if port == "" {
		port = "/dev/ttyUSB0"
		fmt.Printf("GOZNP_PORT not set, using default: %s\n", port)
	}

	// Device to read from (network address and endpoint)
	nwkAddrArg := os.Getenv("DEVICE_ADDR")
	if nwkAddrArg == "" {
		fmt.Println("Usage: DEVICE_ADDR=0x1234 ENDPOINT=1 [SENSOR_TYPE] go run sensor_read.go")
		fmt.Println("Sensor types: temp, humidity, press, illuminance, occupancy, motion")
		fmt.Println("\nExamples:")
		fmt.Println("  DEVICE_ADDR=0x1234 ENDPOINT=1 go run sensor_read.go              # Read all sensors")
		fmt.Println("  DEVICE_ADDR=0x1234 ENDPOINT=1 SENSOR_TYPE=temp go run sensor_read.go")
		return
	}

	var nwkAddr uint16
	if _, err := fmt.Sscanf(nwkAddrArg, "0x%04X", &nwkAddr); err != nil {
		fmt.Sscanf(nwkAddrArg, "%d", &nwkAddr)
	}

	// Get endpoint (default to 1)
	endpointArg := os.Getenv("ENDPOINT")
	endpoint := uint8(1)
	if endpointArg != "" {
		fmt.Sscanf(endpointArg, "%d", &endpoint)
	}

	// Get sensor type (optional)
	sensorType := os.Getenv("SENSOR_TYPE")

	// Create and open adapter connection
	adptr := adapter.New(adapter.WithSerialPath(port))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := adptr.Open(ctx); err != nil {
		log.Fatalf("Failed to open adapter: %v", err)
	}
	defer adptr.Close()

	fmt.Printf("=== Reading sensor at 0x%04X:%d ===\n\n", nwkAddr, endpoint)

	// Request specific sensor or all
	switch sensorType {
	case "temp":
		readTemperature(ctx, adptr, nwkAddr, endpoint)
	case "humidity":
		readHumidity(ctx, adptr, nwkAddr, endpoint)
	case "press":
		readPressure(ctx, adptr, nwkAddr, endpoint)
	case "illuminance":
		readIlluminance(ctx, adptr, nwkAddr, endpoint)
	case "occupancy":
		readOccupancy(ctx, adptr, nwkAddr, endpoint)
	case "motion":
		readMotion(ctx, adptr, nwkAddr, endpoint)
	default:
		// Try to read all common sensors
		readTemperature(ctx, adptr, nwkAddr, endpoint)
		fmt.Println()
		readHumidity(ctx, adptr, nwkAddr, endpoint)
		fmt.Println()
		readPressure(ctx, adptr, nwkAddr, endpoint)
		fmt.Println()
		readIlluminance(ctx, adptr, nwkAddr, endpoint)
		fmt.Println()
		readOccupancy(ctx, adptr, nwkAddr, endpoint)
	}
}

func readTemperature(ctx context.Context, adptr *adapter.Adapter, nwkAddr uint16, endpoint uint8) {
	fmt.Println("Temperature Sensor:")

	attrs, err := adptr.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterTemperature,
		zcl.AttrMeasuredValue, zcl.AttrMinMeasuredValue, zcl.AttrMaxMeasuredValue)
	if err != nil {
		log.Printf("Error reading temperature: %v", err)
		return
	}

	for _, attr := range attrs {
		if attr.Status != zcl.StatusSuccess {
			continue
		}

		value, ok := attr.Value.(int16)
		if !ok {
			continue
		}

		celsius := float64(value) / 100.0
		fahrenheit := celsius*1.8 + 32

		switch attr.AttributeID {
		case zcl.AttrMeasuredValue:
			fmt.Printf("  Current: %.2f°C (%.2f°F)\n", celsius, fahrenheit)
		case zcl.AttrMinMeasuredValue:
			fmt.Printf("  Minimum: %.2f°C (%.2f°F)\n", celsius, fahrenheit)
		case zcl.AttrMaxMeasuredValue:
			fmt.Printf("  Maximum: %.2f°C (%.2f°F)\n", celsius, fahrenheit)
		}
	}
}

func readHumidity(ctx context.Context, adptr *adapter.Adapter, nwkAddr uint16, endpoint uint8) {
	fmt.Println("Humidity Sensor:")

	attrs, err := adptr.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterHumidity,
		zcl.AttrMeasuredValue, zcl.AttrMinMeasuredValue, zcl.AttrMaxMeasuredValue)
	if err != nil {
		log.Printf("Error reading humidity: %v", err)
		return
	}

	for _, attr := range attrs {
		if attr.Status != zcl.StatusSuccess {
			continue
		}

		value, ok := attr.Value.(uint16)
		if !ok {
			continue
		}

		relative := float64(value) / 100.0

		switch attr.AttributeID {
		case zcl.AttrMeasuredValue:
			fmt.Printf("  Relative Humidity: %.2f%%\n", relative)
		case zcl.AttrMinMeasuredValue:
			fmt.Printf("  Minimum: %.2f%%\n", relative)
		case zcl.AttrMaxMeasuredValue:
			fmt.Printf("  Maximum: %.2f%%\n", relative)
		}
	}
}

func readPressure(ctx context.Context, adptr *adapter.Adapter, nwkAddr uint16, endpoint uint8) {
	fmt.Println("Pressure Sensor:")

	attrs, err := adptr.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterPressure,
		zcl.AttrMeasuredValue, zcl.AttrMinMeasuredValue, zcl.AttrMaxMeasuredValue)
	if err != nil {
		log.Printf("Error reading pressure: %v", err)
		return
	}

	for _, attr := range attrs {
		if attr.Status != zcl.StatusSuccess {
			continue
		}

		value, ok := attr.Value.(int16)
		if !ok {
			continue
		}

		// Pressure is in units of 1/10 kPa
		kpa := float64(value) / 10.0
		mbar := kpa
		atm := kpa / 101.325
		mmHg := kpa * 7.50062

		switch attr.AttributeID {
		case zcl.AttrMeasuredValue:
			fmt.Printf("  Pressure: %.2f kPa (%.2f mbar, %.3f atm, %.2f mmHg)\n", kpa, mbar, atm, mmHg)
		case zcl.AttrMinMeasuredValue:
			fmt.Printf("  Minimum: %.2f kPa\n", kpa)
		case zcl.AttrMaxMeasuredValue:
			fmt.Printf("  Maximum: %.2f kPa\n", kpa)
		}
	}
}

func readIlluminance(ctx context.Context, adptr *adapter.Adapter, nwkAddr uint16, endpoint uint8) {
	fmt.Println("Illuminance Sensor:")

	attrs, err := adptr.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterIlluminance,
		zcl.AttrMeasuredValue, zcl.AttrMinMeasuredValue, zcl.AttrMaxMeasuredValue)
	if err != nil {
		log.Printf("Error reading illuminance: %v", err)
		return
	}

	for _, attr := range attrs {
		if attr.Status != zcl.StatusSuccess {
			continue
		}

		value, ok := attr.Value.(uint16)
		if !ok {
			continue
		}

		// Illuminance is in units of 1 lux
		lux := float64(value)

		switch attr.AttributeID {
		case zcl.AttrMeasuredValue:
			fmt.Printf("  Illuminance: %.2f lux (%s)\n", lux, illuminanceLevel(lux))
		case zcl.AttrMinMeasuredValue:
			fmt.Printf("  Minimum: %.2f lux\n", lux)
		case zcl.AttrMaxMeasuredValue:
			fmt.Printf("  Maximum: %.2f lux\n", lux)
		}
	}
}

func readOccupancy(ctx context.Context, adptr *adapter.Adapter, nwkAddr uint16, endpoint uint8) {
	fmt.Println("Occupancy Sensor:")

	// Try standard occupancy cluster
	attrs, err := adptr.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterOccupancy, zcl.AttrOccupancy)
	if err != nil {
		log.Printf("Error reading occupancy: %v", err)
		return
	}

	for _, attr := range attrs {
		if attr.Status != zcl.StatusSuccess {
			continue
		}

		value, ok := attr.Value.(uint8)
		if !ok {
			continue
		}

		fmt.Printf("  Occupancy: %s\n", booleanString(value))
	}

	// Also try IAS Zone for motion sensors
	readMotion(ctx, adptr, nwkAddr, endpoint)
}

func readMotion(ctx context.Context, adptr *adapter.Adapter, nwkAddr uint16, endpoint uint8) {
	// Try IAS Zone cluster (common for motion sensors)
	attrs, err := adptr.ReadAttributes(ctx, nwkAddr, endpoint, zcl.ClusterIASZone, zcl.AttrZoneStatus)
	if err != nil {
		// Not an IAS zone device, that's okay
		return
	}

	for _, attr := range attrs {
		if attr.Status != zcl.StatusSuccess {
			continue
		}

		status, ok := attr.Value.(uint16)
		if !ok {
			continue
		}

		// Check for zone status bits
		alarmActive := (status & 0x0001) != 0
		batLow := (status & 0x0002) != 0
		supervision := (status & 0x0004) != 0
		testMode := (status & 0x0008) != 0
		ac := (status & 0x0010) != 0
		trouble := (status & 0x0020) != 0
		tamper := (status & 0x0080) != 0

		fmt.Printf("  Status: 0x%04X\n", status)
		fmt.Printf("    Alarm Active: %v\n", alarmActive)
		fmt.Printf("    Battery Low: %v\n", batLow)
		fmt.Printf("    Supervision Reports: %v\n", supervision)
		fmt.Printf("    Test Mode: %v\n", testMode)
		fmt.Printf("    AC Mains: %v\n", ac)
		fmt.Printf("    Trouble: %v\n", trouble)
		fmt.Printf("    Tamper: %v\n", tamper)
	}
}

func booleanString(value uint8) string {
	switch value {
	case 0:
		return "False / Unoccupied / Closed"
	case 1:
		return "True / Occupied / Open"
	default:
		return fmt.Sprintf("Unknown (%d)", value)
	}
}

func illuminanceLevel(lux float64) string {
	switch {
	case lux < 10:
		return "Pitch black"
	case lux < 50:
		return "Dim twilight"
	case lux < 100:
		return "Very dark"
	case lux < 200:
		return "Dark room"
	case lux < 400:
		return "Dimly lit"
	case lux < 800:
		return "Normal indoor lighting"
	case lux < 1500:
		return "Bright indoor"
	case lux < 5000:
		return "Overcast outdoor"
	case lux < 10000:
		return "Bright outdoor"
	case lux < 25000:
		return "Full daylight"
	case lux < 100000:
		return "Direct sunlight"
	default:
		return "Extremely bright"
	}
}
