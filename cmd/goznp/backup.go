package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/backup"
	"github.com/marstid/goznp/pkg/serial"
	"github.com/marstid/goznp/pkg/znp"
)

var (
	outputFile   string
	inputFile    string
	forceRestore bool
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup and restore adapter configuration",
	Long:  "Create backups of adapter configuration and restore from backup files",
}

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a backup of the adapter configuration",
	Long:  "Creates a JSON backup file containing network configuration and paired devices",
	Run: func(_ *cobra.Command, _ []string) {
		ctx := setupSignalHandler()
		if err := runBackupCreate(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore adapter configuration from a backup",
	Long:  "Restores network configuration from a JSON backup file",
	Run: func(_ *cobra.Command, _ []string) {
		ctx := setupSignalHandler()
		if err := runBackupRestore(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

var backupShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display contents of a backup file",
	Long:  "Shows the contents of a backup file without connecting to adapter",
	Run: func(_ *cobra.Command, _ []string) {
		if err := runBackupShow(); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

var backupDebugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug NVRAM content",
	Long:  "Dump raw NVRAM Address Manager table for debugging",
	Run: func(_ *cobra.Command, _ []string) {
		ctx := setupSignalHandler()
		if err := runBackupDebug(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Add flags to backup create.
	AddConnectionFlags(backupCreateCmd)
	backupCreateCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (required)")
	backupCreateCmd.MarkFlagRequired("output")

	// Add flags to backup restore.
	AddConnectionFlags(backupRestoreCmd)
	backupRestoreCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input file path (required)")
	backupRestoreCmd.Flags().BoolVarP(&forceRestore, "force", "f", false, "Skip confirmation prompt")
	backupRestoreCmd.MarkFlagRequired("input")

	// Add flags to backup show.
	backupShowCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input file path (required)")
	backupShowCmd.MarkFlagRequired("input")

	// Add flags to backup debug.
	backupDebugCmd.Flags().StringVarP(&portPath, "port", "p", "", "Serial port path (or set GOZNP_PORT)")
	backupDebugCmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")

	// Build command hierarchy.
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupShowCmd)
	backupCmd.AddCommand(backupDebugCmd)
}

func runBackupCreate(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Println("Creating backup...")

	b, err := a.CreateBackup(ctx)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	data, err := b.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize backup: %w", err)
	}

	if err := os.WriteFile(outputFile, data, 0o600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	fmt.Printf("Backup created successfully: %s\n", outputFile)
	fmt.Printf("  Coordinator: %s\n", b.Coordinator.IEEEAddress)
	fmt.Printf("  PAN ID:      0x%04X\n", b.Network.PanID)
	fmt.Printf("  Channel:     %d\n", b.Network.Channel)
	fmt.Printf("  Devices:     %d\n", len(b.Devices))

	return nil
}

func runBackupRestore(ctx context.Context) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	// Read backup file.
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	b, err := backup.FromJSON(data)
	if err != nil {
		return fmt.Errorf("failed to parse backup file: %w", err)
	}

	if err := b.Validate(); err != nil {
		return fmt.Errorf("invalid backup: %w", err)
	}

	// Show what will be restored.
	fmt.Println("Backup contents:")
	fmt.Printf("  Created:     %s\n", b.Created.Format(time.RFC3339))
	fmt.Printf("  Coordinator: %s\n", b.Coordinator.IEEEAddress)
	fmt.Printf("  PAN ID:      0x%04X\n", b.Network.PanID)
	fmt.Printf("  Channel:     %d\n", b.Network.Channel)
	fmt.Printf("  Devices:     %d\n", len(b.Devices))

	if !forceRestore {
		fmt.Print("\nWARNING: This will overwrite the current configuration. Continue? [y/N]: ")
		var response string
		//nolint:errcheck // User input, error intentionally ignored
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Restore canceled.")
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	fmt.Println("\nRestoring backup...")

	if err := a.RestoreBackup(ctx, b); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	fmt.Println("Backup restored successfully!")
	fmt.Println("Note: You may need to reset the adapter for changes to take effect.")

	return nil
}

func runBackupShow() error {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	b, err := backup.FromJSON(data)
	if err != nil {
		return fmt.Errorf("failed to parse backup file: %w", err)
	}

	fmt.Println("Backup File Contents")
	fmt.Println("====================")
	fmt.Printf("Version:       %d\n", b.Version)
	fmt.Printf("Created:       %s\n", b.Created.Format(time.RFC3339))

	fmt.Println("\nAdapter:")
	fmt.Printf("  Z-Stack Variant: %d\n", b.Adapter.ZStackVariant)
	fmt.Printf("  SDK Version:     %s\n", b.Adapter.SDKVersion)
	fmt.Printf("  Build Date:      %d\n", b.Adapter.BuildDate)

	fmt.Println("\nCoordinator:")
	fmt.Printf("  IEEE Address:    %s\n", b.Coordinator.IEEEAddress)
	fmt.Printf("  Network Address: 0x%04X\n", b.Coordinator.NetworkAddress)

	fmt.Println("\nNetwork:")
	fmt.Printf("  PAN ID:           0x%04X\n", b.Network.PanID)
	fmt.Printf("  Extended PAN ID:  %s\n", b.Network.ExtendedPanID)
	fmt.Printf("  Channel:          %d\n", b.Network.Channel)
	fmt.Printf("  Security Level:   %d\n", b.Network.SecurityLevel)
	fmt.Printf("  Network Key:      %s...\n", b.Network.NetworkKey[:8]) // Only show first 8 chars.

	if len(b.Devices) > 0 {
		fmt.Printf("\nDevices (%d):\n", len(b.Devices))
		for i, d := range b.Devices {
			fmt.Printf("  %d. %s (0x%04X) - %s\n", i+1, d.IEEEAddress, d.NetworkAddress, d.Type)
		}
	} else {
		fmt.Println("\nDevices: (none)")
	}

	return nil
}

func runBackupDebug(ctx context.Context) error {
	portPath, err := getPortPath()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cfg := serial.DefaultConfig()
	cfg.Path = portPath
	cfg.BaudRate = baudRate

	port, err := serial.Open(cfg)
	if err != nil {
		return fmt.Errorf("failed to open serial port: %w", err)
	}
	defer port.Close()

	z := znp.New(port)
	if err := z.Open(ctx); err != nil {
		return fmt.Errorf("failed to open ZNP: %w", err)
	}
	defer z.Close()

	// Ping.
	if _, err := z.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	fmt.Println("=== NVRAM Contents for Paired Devices ===")
	fmt.Println()

	// 1. Address Manager - IEEE/NWK address mappings.
	fmt.Println("1. ADDRESS MANAGER (device addresses)")
	addrEntries, err := z.ReadAddrMgrTable(ctx)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Found %d device(s)\n", len(addrEntries))
		for i, e := range addrEntries {
			typeStr := ""
			if e.Type&znp.AddrMgrAssoc != 0 {
				typeStr += "Assoc "
			}
			if e.Type&znp.AddrMgrSecurity != 0 {
				typeStr += "Security "
			}
			if e.Type&znp.AddrMgrBinding != 0 {
				typeStr += "Binding "
			}
			fmt.Printf("   %d: IEEE=%s NwkAddr=0x%04X Flags=[%s]\n",
				i, znp.FormatIEEEAddr(e.ExtAddr), e.NwkAddr, typeStr)
		}
	}

	// 2. TCLK Table - Trust Center Link Keys.
	fmt.Println("\n2. TCLK TABLE (Trust Center Link Keys)")
	tclkEntries, err := z.NvReadTable(ctx, znp.NvSysZStack, znp.NvExTclkTable)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		nonEmpty := 0
		for _, data := range tclkEntries {
			isEmpty := true
			for _, b := range data {
				if b != 0x00 && b != 0xFF {
					isEmpty = false
					break
				}
			}
			if !isEmpty {
				nonEmpty++
				// TCLK entry format: txFrmCntr(4) + rxFrmCntr(4) + extAddr(8) + keyAttr(1) + keyType(1) + seedShift(1) + padding(1) = 20 bytes.
				if len(data) >= 20 {
					var ieee [8]byte
					copy(ieee[:], data[8:16])
					txCtr := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
					rxCtr := uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24
					fmt.Printf("   Device: IEEE=%s TxFrameCtr=%d RxFrameCtr=%d\n",
						znp.FormatIEEEAddr(ieee), txCtr, rxCtr)
				}
			}
		}
		if nonEmpty == 0 {
			fmt.Println("   (no entries)")
		}
	}

	// 3. APS Key Data - Application layer keys.
	fmt.Println("\n3. APS KEY DATA TABLE (Application Keys)")
	apsEntries, err := z.NvReadTable(ctx, znp.NvSysZStack, znp.NvExApsKeyData)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		nonEmpty := 0
		for i, data := range apsEntries {
			isEmpty := true
			for _, b := range data {
				if b != 0x00 && b != 0xFF {
					isEmpty = false
					break
				}
			}
			if !isEmpty && i < 5 {
				nonEmpty++
				fmt.Printf("   Entry %d: len=%d data=%x...\n", i, len(data), data[:min(20, len(data))])
			}
		}
		if nonEmpty == 0 {
			fmt.Println("   (no entries)")
		}
	}

	// 4. Legacy OSAL items.
	fmt.Println("\n4. LEGACY OSAL NV ITEMS")

	osalItems := []struct {
		name string
		id   znp.NvItemID
	}{
		{"NIB (Network Info Base)", znp.NvNIB},
		{"TCLK Seed", znp.NvTclkSeed},
		{"Network Active Key", znp.NvNwkActiveKeyInfo},
		{"APS Link Key Table", znp.NvApsLinkKeyTable},
	}

	for _, item := range osalItems {
		length, err := z.NvLength(ctx, item.id)
		if err != nil {
			fmt.Printf("   %s: error\n", item.name)
			continue
		}
		if length == 0 {
			fmt.Printf("   %s: (empty)\n", item.name)
			continue
		}
		fmt.Printf("   %s: %d bytes\n", item.name, length)
	}

	return nil
}
