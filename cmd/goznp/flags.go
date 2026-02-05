package main

import (
	"github.com/spf13/cobra"
)

// AddConnectionFlags adds standard connection flags (-p/--port, -b/--baud) to a command
func AddConnectionFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&portPath, "port", "p", "",
		"Serial port path (or set GOZNP_PORT env var)")
	cmd.Flags().IntVarP(&baudRate, "baud", "b", 115200,
		"Baud rate (default: 115200)")
}

// AddDeviceFlags adds device addressing flags (--addr, --ieee, --name, --endpoint) to a command
func AddDeviceFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&deviceAddr, "addr", "",
		"Device network address (hex, e.g., 0x1234)")
	cmd.Flags().StringVar(&deviceIEEE, "ieee", "",
		"Device IEEE address (e.g., 00:11:22:33:44:55:66:77)")
	cmd.Flags().StringVar(&deviceName, "name", "",
		"Device name (see 'goznp devices --name list' for available names)")
	cmd.Flags().Uint8Var(&deviceEndpoint, "endpoint", 1,
		"Device endpoint (default: 1)")
}

// AddGroupFlags adds group-related flags (--group, --group-name) to a command
func AddGroupFlags(cmd *cobra.Command) {
	cmd.Flags().Uint16Var(&groupID, "group", 0,
		"Group ID (0-65535)")
	cmd.Flags().StringVar(&groupName, "group-name", "",
		"Group name (user-friendly)")
}
