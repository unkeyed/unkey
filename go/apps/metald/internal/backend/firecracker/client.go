//go:build linux
// +build linux

package firecracker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/unkeyed/unkey/go/apps/metald/internal/assetmanager"
	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/types"
	"github.com/unkeyed/unkey/go/apps/metald/internal/config"
	"github.com/unkeyed/unkey/go/apps/metald/internal/jailer"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// NewClient creates a new SDK-based Firecracker backend client with integrated jailer
func NewClient(logger *slog.Logger,
	assetClient assetmanager.Client,
	jailerConfig *config.JailerConfig,
	baseDir string,
) (*Client, error) {
	tracer := otel.Tracer("metald.firecracker")
	meter := otel.Meter("metald.firecracker")

	vmCreateCounter, err := meter.Int64Counter("vm_create_total",
		metric.WithDescription("Total number of VM create operations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create vm_create counter: %w", err)
	}

	vmDeleteCounter, err := meter.Int64Counter("vm_delete_total",
		metric.WithDescription("Total number of VM delete operations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create vm_delete counter: %w", err)
	}

	vmBootCounter, err := meter.Int64Counter("vm_boot_total",
		metric.WithDescription("Total number of VM boot operations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create vm_boot counter: %w", err)
	}

	vmErrorCounter, err := meter.Int64Counter("vm_error_total",
		metric.WithDescription("Total number of VM operation errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create vm_error counter: %w", err)
	}

	// Create integrated jailer
	integratedJailer := jailer.NewJailer(logger, jailerConfig)

	return &Client{
		logger:          logger.With("backend", "firecracker"),
		assetClient:     assetClient,
		vmAssetLeases:   make(map[string][]string),
		jailer:          integratedJailer,
		jailerConfig:    jailerConfig,
		baseDir:         baseDir,
		tracer:          tracer,
		meter:           meter,
		vmCreateCounter: vmCreateCounter,
		vmDeleteCounter: vmDeleteCounter,
		vmBootCounter:   vmBootCounter,
		vmErrorCounter:  vmErrorCounter,
	}, nil
}

// Ping verifies the backend is operational
func (c *Client) Ping(ctx context.Context) error {
	c.logger.DebugContext(ctx, "pinging firecracker backend")
	return nil
}

// Shutdown gracefully shuts down the SDK client while preserving VMs
func (c *Client) Shutdown(ctx context.Context) error {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.shutdown")
	defer span.End()

	c.logger.InfoContext(ctx, "shutting down firecracker backend")

	return nil
}

// Type returns the backend type as a string for metrics
func (c *Client) Type() string {
	return string(types.BackendTypeFirecracker)
}

// Ensure Client implements Backend interface
var _ types.Backend = (*Client)(nil)
