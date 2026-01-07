package main

import (
	"context"
	"fmt"
	"time"

	"github.com/marstid/goznp/pkg/adapter"
)

// WithAdapter executes a function with a ready-to-use adapter.
// It handles port resolution, adapter creation, opening, and cleanup.
// The provided context will have a timeout applied.
func WithAdapter(ctx context.Context, timeout time.Duration, fn func(ctx context.Context, a *adapter.Adapter) error) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	return fn(ctx, a)
}

// WithAdapterNoTimeout executes a function with a ready-to-use adapter without applying a timeout.
// It handles port resolution, adapter creation, opening, and cleanup.
// Use this for long-running operations like watch commands.
func WithAdapterNoTimeout(ctx context.Context, fn func(ctx context.Context, a *adapter.Adapter) error) error {
	port, err := getPortPath()
	if err != nil {
		return err
	}

	a := adapter.New(
		adapter.WithSerialPath(port),
		adapter.WithBaudRate(baudRate),
	)

	if err := a.Open(ctx); err != nil {
		return fmt.Errorf("failed to open adapter: %w", err)
	}
	defer a.Close()

	return fn(ctx, a)
}

// WithAdapterDevice executes a function with a ready-to-use adapter and resolved device address.
// It handles port resolution, adapter creation, opening, device resolution, and cleanup.
// The device is resolved using the global flags: deviceNameLookup, deviceIEEE, or deviceAddr.
// Priority: --name > --ieee > --addr
func WithAdapterDevice(ctx context.Context, timeout time.Duration, fn func(ctx context.Context, a *adapter.Adapter, nwkAddr uint16) error) error {
	return WithAdapter(ctx, timeout, func(ctx context.Context, a *adapter.Adapter) error {
		nwkAddr, err := resolveDeviceAddr(ctx, a, deviceNameLookup, deviceIEEE, deviceAddr)
		if err != nil {
			return err
		}
		return fn(ctx, a, nwkAddr)
	})
}
