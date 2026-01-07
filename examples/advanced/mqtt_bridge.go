//go:build ignore

// Example: Bridge Zigbee device events to MQTT.
// This demonstrates how to integrate goznp with MQTT for IoT automation.
// Note: You'll need MQTT broker (e.g., Mosquitto) running for this example.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/zcl"
)

// MQTT interface stub - replace with actual MQTT client library like
// https://github.com/eclipse/paho.mqtt.golang
//
// This example demonstrates the pattern. To run:
// 1. Install: go get github.com/eclipse/paho.mqtt.golang
// 2. Un-comment the MQTT code sections
// 3. Configure MQTT broker settings

type EventMessage struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Device    device    `json:"device"`
	Data      any       `json:"data,omitempty"`
}

type device struct {
	IEEEAddr string `json:"ieee_addr"`
	NwkAddr  string `json:"nwk_addr"`
}

var (
	// MQTT client - set up based on your MQTT library
	// var mqttClient mqtt.Client

	topics = struct {
		DeviceState    string
		DeviceAction   string
		Network        string
		GroupAction    string
		ZHAReceive     string
		Availability   string
		DeviceJoined   string
		DeviceLeft     string
	}{
		DeviceState:  "zigbee2mqtt/+/availability",
		DeviceAction: "zigbee2mqtt/+/set",
		Network:      "zigbee2mqtt/bridge/network",
		GroupAction:  "zigbee2mqtt/bridge/group/+/set",
		ZHAReceive:   "zigbee2mqtt/+/receive",
		Availability: "zigbee2mqtt/bridge/availability",
		DeviceJoined: "zigbee2mqtt/bridge/event/device_joined",
		DeviceLeft:   "zigbee2mqtt/bridge/event/device_left",
	}
)

func main() {
	// Get configuration from environment
	port := os.Getenv("GOZNP_PORT")
	if port == "" {
		port = "/dev/ttyUSB0"
	}
	mqttBroker := os.Getenv("MQTT_BROKER")
	if mqttBroker == "" {
		mqttBroker = "tcp://localhost:1883"
	}
	baseTopic := os.Getenv("MQTT_TOPIC")
	if baseTopic == "" {
		baseTopic = "goznp"
	}

	// Create adapter
	adptr := adapter.New(adapter.WithSerialPath(port))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := adptr.Open(ctx); err != nil {
		log.Fatalf("Failed to open adapter: %v", err)
	}
	defer adptr.Close()

	fmt.Println("=== goznp MQTT Bridge ===")
	fmt.Printf("Serial: %s\n", port)
	fmt.Printf("MQTT: %s\n", mqttBroker)
	fmt.Printf("Base Topic: %s\n\n", baseTopic)

	// Set up MQTT client (pseudo-code - uncomment with actual library)
	// opts := mqtt.NewClientOptions().AddBroker(mqttBroker)
	// opts.SetClientID("goznp-bridge")
	// mqttClient = mqtt.NewClient(opts)
	// if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
	//     log.Fatalf("Failed to connect to MQTT: %v", token.Error())
	// }
	// defer mqttClient.Disconnect(250)

	// Set up device event callbacks
	setupEventCallbacks(adptr, baseTopic)

	// Signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Announce bridge availability
	publishAvailability(baseTopic, "online")

	fmt.Println("Bridge running. Press Ctrl+C to stop...")

	// Wait for shutdown signal
	<-sigChan

	// Announce bridge going offline
	publishAvailability(baseTopic, "offline")

	fmt.Println("\nBridge stopped")
}

func setupEventCallbacks(adptr *adapter.Adapter, baseTopic string) {
	// Device event handler
	adptr.OnDeviceEvent(func(event adapter.DeviceEvent) {
		switch event.Type {
		case adapter.DeviceEventJoined:
			handleDeviceJoin(adptr, baseTopic, event)

		case adapter.DeviceEventLeft:
			handleDeviceLeave(baseTopic, event)

		case adapter.DeviceEventAnnounce:
			handleDeviceAnnounce(baseTopic, event)
		}
	})
}

func handleDeviceJoin(adptr *adapter.Adapter, baseTopic string, event adapter.DeviceEvent) {
	if event.Device == nil {
		return
	}

	dev := event.Device
	ieeeAddr := formatIEEEAddr(dev.IEEEAddr)

	// Publish device joined event
	msg := EventMessage{
		Type:      "device_joined",
		Timestamp: time.Now(),
		Device: device{
			IEEEAddr: ieeeAddr,
			NwkAddr:  fmt.Sprintf("0x%04X", dev.NwkAddr),
		},
	}

	// Interview the device to get full info
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	adptr.InterviewDevice(ctx, dev)

	// Refresh to get interview results
	devices, _ := adptr.GetDevices(ctx)
	for _, d := range devices {
		if d.IEEEAddr == dev.IEEEAddr {
			msg.Data = map[string]any{
				"manufacturer": d.Manufacturer,
				"model":        d.Model,
				"power_source": powerSourceString(d.Capabilities),
			}
			break
		}
	}

	publishJSON(fmt.Sprintf("%s/event/joined", baseTopic), msg)
	fmt.Printf("Device joined: %s (%s %s)\n", ieeeAddr, dev.Manufacturer, dev.Model)
}

func handleDeviceLeave(baseTopic string, event adapter.DeviceEvent) {
	msg := EventMessage{
		Type:      "device_left",
		Timestamp: time.Now(),
		Device: device{
			IEEEAddr: formatIEEEAddr(event.IEEEAddr),
		},
	}

	publishJSON(fmt.Sprintf("%s/event/left", baseTopic), msg)
	fmt.Printf("Device left: %s\n", formatIEEEAddr(event.IEEEAddr))
}

func handleDeviceAnnounce(baseTopic string, event adapter.DeviceEvent) {
	if event.Device == nil {
		return
	}

	dev := event.Device
	msg := EventMessage{
		Type:      "device_announce",
		Timestamp: time.Now(),
		Device: device{
			IEEEAddr: formatIEEEAddr(dev.IEEEAddr),
			NwkAddr:  fmt.Sprintf("0x%04X", dev.NwkAddr),
		},
	}

	publishJSON(fmt.Sprintf("%s/event/announce", baseTopic), msg)
	fmt.Printf("Device announced: %s at 0x%04X\n",
		formatIEEEAddr(dev.IEEEAddr), dev.NwkAddr)
}

// publishJSON marshals and publishes an event message
// Replace this with actual MQTT publish
func publishJSON(topic string, msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		return
	}

	// Actual MQTT publish:
	// token := mqttClient.Publish(topic, 0, false, string(data))
	// token.Wait()

	// For demonstration:
	log.Printf("[MQTT] %s: %s\n", topic, string(data))
}

func publishAvailability(baseTopic, status string) {
	msg := map[string]string{
		"status": status,
		"time":   time.Now().Format(time.RFC3339),
	}
	publishJSON(fmt.Sprintf("%s/availability", baseTopic), msg)
}

// publishDeviceState publishes a device's current state to MQTT
func publishDeviceState(baseTopic string, dev *adapter.Device, state map[string]any) {
	ieeeAddr := formatIEEEAddr(dev.IEEEAddr)
	topic := fmt.Sprintf("%s/%s/state", baseTopic, ieeeAddr)

	publishJSON(topic, state)
}

// Example subscription handler for incoming MQTT commands
// This would be used to control devices via MQTT
func startMQTTCommandHandler(adptr *adapter.Adapter, baseTopic string, wg *sync.WaitGroup) {
	defer wg.Done()

	// Subscribe to device command topics
	commandTopic := fmt.Sprintf("%s/+/command", baseTopic)

	// Pseudo-code for MQTT subscription:
	/*
		mqttClient.Subscribe(commandTopic, 0, func(client mqtt.Client, msg mqtt.Message) {
			// Parse topic to get device IEEE address
			topicParts := strings.Split(msg.Topic(), "/")
			if len(topicParts) < 2 {
				return
			}
			ieeeAddr := parseIEEEAddr(topicParts[len(topicParts)-2])

			// Parse command
			var cmd map[string]any
			if err := json.Unmarshal(msg.Payload(), &cmd); err != nil {
				log.Printf("Error parsing command: %v", err)
				return
			}

			// Execute command
			handleDeviceCommand(adptr, ieeeAddr, cmd)
		})
	*/
}

func handleDeviceCommand(adptr *adapter.Adapter, ieeeAddr [8]byte, cmd map[string]any) {
	// Find device
	devices, _ := adptr.GetDevices(context.Background())
	var target *adapter.Device
	for _, d := range devices {
		if d.IEEEAddr == ieeeAddr {
			target = d
			break
		}
	}
	if target == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Command: {"state": "on"} or {"state": "off"}
	if state, ok := cmd["state"].(string); ok {
		if state == "on" {
			adptr.TurnOn(ctx, target.NwkAddr, 1)
		} else if state == "off" {
			adptr.TurnOff(ctx, target.NwkAddr, 1)
		}
	}

	// Command: {"brightness": 128}
	if brightness, ok := cmd["brightness"].(float64); ok {
		adptr.SetBrightness(ctx, target.NwkAddr, 1, uint8(brightness))
	}

	// Command: {"color": {"r": 255, "g": 0, "b": 0}}
	if color, ok := cmd["color"].(map[string]any); ok {
		r := uint8(0)
		g := uint8(0)
		b := uint8(0)
		if rv, ok := color["r"].(float64); ok {
			r = uint8(rv)
		}
		if gv, ok := color["g"].(float64); ok {
			g = uint8(gv)
		}
		if bv, ok := color["b"].(float64); ok {
			b = uint8(bv)
		}
		adptr.SetColor(ctx, target.NwkAddr, 1, r, g, b)
	}
}

func formatIEEEAddr(addr [8]byte) string {
	return fmt.Sprintf("%02X%02X%02X%02X%02X%02X%02X%02X",
		addr[7], addr[6], addr[5], addr[4], addr[3], addr[2], addr[1], addr[0])
}

func powerSourceString(caps uint8) string {
	if caps&0x02 != 0 {
		return "Mains (single)"
	}
	if caps&0x04 != 0 {
		return "Battery"
	}
	return "Unknown"
}

// ZCL attribute change handler - could be integrated to report state changes
func onZCLAttributeChange(baseTopic string, nwkAddr uint16, endpoint uint8, clusterID uint16, attrID uint16, value any) {
	// Construct state change message
	change := map[string]any{
		"timestamp": time.Now().Unix(),
		"nwk_addr":  fmt.Sprintf("0x%04X", nwkAddr),
		"endpoint":  endpoint,
		"cluster":   clusterID,
		"attribute": attrID,
		"value":     value,
	}

	// Map to human-readable cluster/attribute names
	if clusterID == zcl.ClusterOnOff && attrID == zcl.AttrOnOff {
		change["cluster_name"] = "OnOff"
		change["attribute_name"] = "OnOff"
		if v, ok := value.(uint8); ok {
			change["value"] = map[string]any{
				"raw": v,
				"state": func() string {
					if v == 0 {
						return "OFF"
					}
					return "ON"
				}(),
			}
		}
	}

	publishJSON(fmt.Sprintf("%s/attribute_change", baseTopic), change)
}
