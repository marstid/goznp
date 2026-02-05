package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/zcl"
	"github.com/marstid/goznp/pkg/znp"
)

var (
	bindCluster   string
	bindUnbind    bool
	reportCluster string
	reportAttr    string
	reportMin     uint16
	reportMax     uint16
)

// goznp device bind -p <port> --addr <nwk-addr> --cluster <cluster-id> [--unbind]
var deviceBindCmd = &cobra.Command{
	Use:   "bind",
	Short: "Bind device cluster to coordinator",
	Long: `Create or remove a binding from a device cluster to the coordinator.

Binding tells a device where to send attribute reports. Without binding,
devices don't know where to send automatic reports even if reporting is configured.

Common clusters to bind:
  0x0006 - OnOff (switch state)
  0x0008 - LevelControl (brightness)
  0x0300 - ColorControl (color)
  0x0402 - TemperatureMeasurement (temperature)
  0x0405 - RelativeHumidity (humidity)
  0x0001 - PowerConfiguration (battery)
  0x0702 - SimpleMetering (energy/consumption)
  0x0B04 - ElectricalMeasurement (power/voltage/current)

Use --unbind to remove an existing binding.

Examples:
  # Bind OnOff cluster by name
  goznp device bind --name "Living Room" --cluster 0x0006

  # bind PowerConfiguration for battery reporting
  goznp device bind --name "Motion Sensor" --cluster 0x0001

  # Bind temperature sensor
  goznp device bind --name "Temp Sensor" --cluster 0x0402

  # Unbind a cluster
  goznp device bind --name "Living Room" --cluster 0x0006 --unbind`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceBind(ctx)
	},
}

func runDeviceBind(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	clusterID, err := parseClusterID(bindCluster)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to connect to adapter: %w", err)
	}
	defer a.Close()

	nwkAddr, err := resolveDeviceAddr(ctx, a, deviceName, deviceIEEE, deviceAddr)
	if err != nil {
		return err
	}

	// Get device list to find IEEE address
	devices, err := a.GetDevices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get device list: %w", err)
	}

	var deviceIEEEAddr [8]byte
	found := false
	for _, dev := range devices {
		if dev.NwkAddr == nwkAddr {
			deviceIEEEAddr = dev.IEEEAddr
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("device 0x%04X not found in paired devices", nwkAddr)
	}

	// Get coordinator IEEE address
	info, err := a.GetInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get adapter info: %w", err)
	}

	// Get coordinator IEEE from network info
	coordIEEE, err := a.GetCoordinatorIEEE(ctx)
	if err != nil {
		return fmt.Errorf("failed to get coordinator IEEE: %w", err)
	}

	action := "Binding"
	if bindUnbind {
		action = "Unbinding"
	}

	fmt.Printf("%s device 0x%04X (%s) cluster 0x%04X to coordinator (%s)...\n",
		action,
		nwkAddr,
		znp.FormatIEEEAddr(deviceIEEEAddr),
		clusterID,
		znp.FormatIEEEAddr(coordIEEE),
	)
	_ = info // Used for debug if needed

	if bindUnbind {
		err = a.Unbind(ctx, deviceIEEEAddr, nwkAddr, deviceEndpoint, clusterID, coordIEEE, 1)
	} else {
		err = a.Bind(ctx, deviceIEEEAddr, nwkAddr, deviceEndpoint, clusterID, coordIEEE, 1)
	}

	if err != nil {
		return fmt.Errorf("%s failed: %w", strings.ToLower(action), err)
	}

	fmt.Printf("%s successful!\n", action)
	return nil
}

// goznp device configure-reporting -p <port> --addr <nwk-addr> --cluster <cluster-id> --attr <attr-id> --min <seconds> --max <seconds>
var deviceConfigureReportingCmd = &cobra.Command{
	Use:   "configure-reporting",
	Short: "Configure attribute reporting",
	Long: `Configure a device to automatically report attribute changes.

This tells the device when to send attribute reports:
  --min: Minimum interval between reports (even if value doesn't change)
  --max: Maximum interval between reports (even if value changes frequently)

Note: You must also bind the cluster to the coordinator for reports to be received.
Run 'goznp device bind --name "<device>" --cluster <id>' first.

Common cluster/attribute combinations:
  OnOff (0x0006):
    0x0000 - OnOff state (report 1s minimum, 300s max for switches)

  Power/Battery (0x0001):
    0x0021 - Battery voltage/percentage (report 3600s minimum)

  ElectricalMeasurement (0x0B04):
    0x0505 - RMS Voltage
    0x0508 - RMS Current
    0x050B - Active Power (report 5s minimum, 60s max for plugs)

  SimpleMetering (0x0702):
    0x0000 - Current Summation (energy, report 60s minimum, 300s max)

  Temperature (0x0402):
    0x0000 - Measured value (report 60s minimum, 300s max for sensors)

Examples:
  # Configure OnOff state reporting
  goznp device configure-reporting --name "Living Room" --cluster 0x0006 --attr 0x0000 --min 1 --max 300

  # Configure power reporting for smart plug
  goznp device configure-reporting --name "Smart Plug" --cluster 0x0B04 --attr 0x050B --min 5 --max 60

  # Configure temperature sensor reporting
  goznp device configure-reporting --name "Temp Sensor" --cluster 0x0402 --attr 0x0000 --min 60 --max 300`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceConfigureReporting(ctx)
	},
}

func runDeviceConfigureReporting(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	clusterID, err := parseClusterID(reportCluster)
	if err != nil {
		return err
	}

	attrID, err := parseClusterID(reportAttr)
	if err != nil {
		return fmt.Errorf("invalid attribute ID: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to connect to adapter: %w", err)
	}
	defer a.Close()

	nwkAddr, err := resolveDeviceAddr(ctx, a, deviceName, deviceIEEE, deviceAddr)
	if err != nil {
		return err
	}

	// Determine data type based on known attributes
	dataType := getDataTypeForAttribute(zcl.ClusterID(clusterID), zcl.AttributeID(attrID))

	fmt.Printf("Configuring reporting on device 0x%04X endpoint %d:\n", nwkAddr, deviceEndpoint)
	fmt.Printf("  Cluster:   0x%04X (%s)\n", clusterID, zcl.ClusterID(clusterID))
	fmt.Printf("  Attribute: 0x%04X\n", attrID)
	fmt.Printf("  Data Type: 0x%02X (%s)\n", uint8(dataType), dataType)
	fmt.Printf("  Min:       %d seconds\n", reportMin)
	fmt.Printf("  Max:       %d seconds\n", reportMax)

	err = a.ConfigureReporting(ctx, nwkAddr, deviceEndpoint,
		zcl.ClusterID(clusterID),
		zcl.AttributeID(attrID),
		dataType,
		reportMin, reportMax,
		nil, // No reportable change threshold for now
	)

	if err != nil {
		return fmt.Errorf("configure reporting failed: %w", err)
	}

	fmt.Println("Configure reporting successful!")
	fmt.Println("\nNote: Make sure the cluster is bound to the coordinator:")
	fmt.Printf("  goznp device bind -p %s --addr 0x%04X --cluster 0x%04X\n", portPath, nwkAddr, clusterID)
	return nil
}

// goznp device reporting -p <port> (--addr <nwk-addr> | --ieee <ieee-addr>) --cluster <cluster-id> --attr <attr-id>
var deviceReportingCmd = &cobra.Command{
	Use:   "reporting",
	Short: "Read reporting configuration",
	Long: `Query a device's current reporting configuration for an attribute.

This shows how the device is configured to send attribute reports:
  - Min Interval: Minimum seconds between reports (even if value changes)
  - Max Interval: Maximum seconds between reports (even if value doesn't change)
  - Reportable Change: How much the value must change to trigger a report

Common cluster/attribute combinations:
  OnOff (0x0006):
    0x0000 - OnOff state
  ElectricalMeasurement (0x0B04):
    0x0505 - RMS Voltage
    0x0508 - RMS Current
    0x050B - Active Power
  SimpleMetering (0x0702):
    0x0000 - Current Summation (energy)

Examples:
  goznp device reporting --addr 0xE2CA --cluster 0x0702 --attr 0x0000
  goznp device reporting --ieee 00:11:22:33:44:55:66:77 --cluster 0x0006 --attr 0x0000`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceReporting(ctx)
	},
}

func runDeviceReporting(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	// Validate that at least one address type is provided
	if deviceAddr == "" && deviceIEEE == "" {
		return fmt.Errorf("either --addr or --ieee must be specified")
	}

	clusterID, err := parseClusterID(reportCluster)
	if err != nil {
		return err
	}

	attrID, err := parseClusterID(reportAttr) // Same format as cluster
	if err != nil {
		return fmt.Errorf("invalid attribute ID: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	// Resolve network address
	var nwkAddr uint16
	if deviceAddr != "" {
		nwkAddr, err = parseNetworkAddr(deviceAddr)
		if err != nil {
			return err
		}
	} else {
		// Look up by IEEE address from paired devices
		ieeeAddr, err := znp.ParseIEEEAddr(deviceIEEE)
		if err != nil {
			return fmt.Errorf("invalid IEEE address %q: %w", deviceIEEE, err)
		}

		devices, err := a.GetDevices(ctx)
		if err != nil {
			return fmt.Errorf("failed to get paired devices: %w", err)
		}

		var found bool
		for _, dev := range devices {
			if dev.IEEEAddr == ieeeAddr {
				nwkAddr = dev.NwkAddr
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("device %s not found in paired devices", znp.FormatIEEEAddr(ieeeAddr))
		}
		fmt.Printf("Resolved IEEE %s to network address 0x%04X\n", deviceIEEE, nwkAddr)
	}

	fmt.Printf("Reading reporting config from device 0x%04X endpoint %d:\n", nwkAddr, deviceEndpoint)
	fmt.Printf("  Cluster:   0x%04X (%s)\n", clusterID, zcl.ClusterID(clusterID))
	fmt.Printf("  Attribute: 0x%04X\n", attrID)
	fmt.Println()

	results, err := a.ReadReportingConfig(ctx, nwkAddr, deviceEndpoint, zcl.ClusterID(clusterID), zcl.AttributeID(attrID))
	if err != nil {
		return fmt.Errorf("read reporting config failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No reporting configuration returned")
		return nil
	}

	for _, r := range results {
		fmt.Printf("Attribute 0x%04X:\n", r.AttributeID)
		if r.Status != 0x00 {
			fmt.Printf("  Status: 0x%02X (error)\n", r.Status)
			continue
		}

		if r.Direction == 0x00 {
			// Outgoing reports (device -> coordinator)
			fmt.Printf("  Direction:    Reported (device sends reports)\n")
			fmt.Printf("  Data Type:    0x%02X (%s)\n", uint8(r.DataType), r.DataType)
			fmt.Printf("  Min Interval: %d seconds\n", r.MinInterval)
			if r.MaxInterval == 0xFFFF {
				fmt.Printf("  Max Interval: disabled (no periodic reports)\n")
			} else {
				fmt.Printf("  Max Interval: %d seconds\n", r.MaxInterval)
			}
			if r.ReportableChange != nil {
				fmt.Printf("  Reportable Change: %v\n", r.ReportableChange)
			}
		} else {
			// Incoming reports (coordinator -> device)
			fmt.Printf("  Direction:      Received (device receives reports)\n")
			fmt.Printf("  Timeout Period: %d seconds\n", r.TimeoutPeriod)
		}
	}

	return nil
}

// goznp device bindings -p <port> --addr <nwk-addr>
var deviceBindingsCmd = &cobra.Command{
	Use:   "bindings",
	Short: "List device bindings",
	Long: `Display the binding table from a device.

The binding table shows where the device will send reports for each cluster.
Use this to verify that clusters are properly bound to the coordinator.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := setupSignalHandler()
		return runDeviceBindings(ctx)
	},
}

func runDeviceBindings(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	nwkAddr, err := parseNetworkAddr(deviceAddr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Printf("Querying binding table from device 0x%04X...\n\n", nwkAddr)

	bindings, err := a.GetBindingTable(ctx, nwkAddr)
	if err != nil {
		return fmt.Errorf("failed to get binding table: %w", err)
	}

	if len(bindings) == 0 {
		fmt.Println("No bindings found on device")
		fmt.Println("\nTip: Use 'goznp device bind' to create bindings")
		return nil
	}

	fmt.Printf("Found %d binding(s):\n\n", len(bindings))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Endpoint\tCluster\tDestination\tDst Endpoint")
	fmt.Fprintln(w, "--------\t-------\t-----------\t------------")

	for _, b := range bindings {
		dstStr := znp.FormatIEEEAddr(b.DstAddr)
		if b.DstAddrMode == 0x01 {
			// Group address - first 2 bytes
			groupID := uint16(b.DstAddr[0]) | uint16(b.DstAddr[1])<<8
			dstStr = fmt.Sprintf("Group 0x%04X", groupID)
		}

		fmt.Fprintf(w, "%d\t%s (0x%04X)\t%s\t%d\n",
			b.SrcEndpoint,
			zcl.ClusterID(b.ClusterID),
			b.ClusterID,
			dstStr,
			b.DstEndpoint,
		)
	}
	w.Flush()

	return nil
}

// getDataTypeForAttribute returns the ZCL data type for known cluster/attribute combinations.
// This is needed when configuring reporting to tell the device what type of data to expect.
func getDataTypeForAttribute(cluster zcl.ClusterID, attr zcl.AttributeID) zcl.DataType {
	switch cluster {
	case zcl.ClusterOnOff:
		if attr == zcl.AttrOnOff {
			return zcl.TypeBoolean
		}
	case zcl.ClusterElectricalMeas:
		switch attr {
		case zcl.AttrElecRMSVoltage, zcl.AttrElecRMSCurrent:
			return zcl.TypeUint16
		case zcl.AttrElecActivePower:
			return zcl.TypeInt16
		}
	case zcl.ClusterMeteringSimple:
		if attr == zcl.AttrMeterCurrentSummation {
			return zcl.TypeUint48
		}
	case zcl.ClusterTempMeasurement:
		if attr == zcl.AttrTempMeasuredValue {
			return zcl.TypeInt16
		}
	case zcl.ClusterHumidityMeas:
		if attr == zcl.AttrHumidityMeasuredValue {
			return zcl.TypeUint16
		}
	}
	// Default to boolean for unknown
	return zcl.TypeBoolean
}

func init() {
	// Use centralized flag helpers
	AddConnectionFlags(deviceBindCmd)
	AddConnectionFlags(deviceConfigureReportingCmd)
	AddConnectionFlags(deviceReportingCmd)
	AddConnectionFlags(deviceBindingsCmd)

	AddDeviceFlags(deviceBindCmd)
	AddDeviceFlags(deviceConfigureReportingCmd)
	AddDeviceFlags(deviceReportingCmd)
	AddDeviceFlags(deviceBindingsCmd)

	// Device bind specific flags
	deviceBindCmd.Flags().StringVar(&bindCluster, "cluster", "", "Cluster ID (hex, e.g., 0x0006)")
	deviceBindCmd.MarkFlagRequired("cluster")
	deviceBindCmd.Flags().BoolVar(&bindUnbind, "unbind", false, "Remove binding instead of creating")

	// Device configure-reporting specific flags
	deviceConfigureReportingCmd.Flags().StringVar(&reportCluster, "cluster", "", "Cluster ID (hex, e.g., 0x0006)")
	deviceConfigureReportingCmd.MarkFlagRequired("cluster")
	deviceConfigureReportingCmd.Flags().StringVar(&reportAttr, "attr", "", "Attribute ID (hex, e.g., 0x0000)")
	deviceConfigureReportingCmd.MarkFlagRequired("attr")
	deviceConfigureReportingCmd.Flags().Uint16Var(&reportMin, "min", 1, "Minimum reporting interval (seconds)")
	deviceConfigureReportingCmd.Flags().Uint16Var(&reportMax, "max", 300, "Maximum reporting interval (seconds)")

	// Device reporting specific flags
	deviceReportingCmd.Flags().StringVar(&reportCluster, "cluster", "", "Cluster ID (hex, e.g., 0x0006)")
	deviceReportingCmd.MarkFlagRequired("cluster")
	deviceReportingCmd.Flags().StringVar(&reportAttr, "attr", "", "Attribute ID (hex, e.g., 0x0000)")
	deviceReportingCmd.MarkFlagRequired("attr")

	// Register commands with deviceCmd
	deviceCmd.AddCommand(deviceBindCmd)
	deviceCmd.AddCommand(deviceConfigureReportingCmd)
	deviceCmd.AddCommand(deviceReportingCmd)
	deviceCmd.AddCommand(deviceBindingsCmd)
}
