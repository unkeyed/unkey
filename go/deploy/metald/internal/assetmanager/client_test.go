package assetmanager

import (
	"context"
	"log/slog"
	"testing"

	"github.com/unkeyed/unkey/go/deploy/metald/internal/config"
	assetv1 "github.com/unkeyed/unkey/go/gen/proto/deploy/assetmanagerd/v1"
)

func TestNewClient(t *testing.T) {
	logger := slog.Default()

	tests := []struct {
		name     string
		config   *config.AssetManagerConfig
		wantErr  bool
		wantNoop bool
	}{
		{
			name: "enabled client",
			config: &config.AssetManagerConfig{
				Enabled:  true,
				Endpoint: "http://localhost:8082",
				CacheDir: "/tmp/assets",
			},
			wantErr:  false,
			wantNoop: false,
		},
		{
			name: "disabled client returns noop",
			config: &config.AssetManagerConfig{
				Enabled:  false,
				Endpoint: "http://localhost:8082",
				CacheDir: "/tmp/assets",
			},
			wantErr:  false,
			wantNoop: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check if we got a noop client
			_, isNoop := client.(*noopClient)
			if isNoop != tt.wantNoop {
				t.Errorf("NewClient() returned noop = %v, want %v", isNoop, tt.wantNoop)
			}
		})
	}
}

func TestNoopClient(t *testing.T) {
	ctx := context.Background()
	client := &noopClient{}

	// Test ListAssets returns empty list
	assets, err := client.ListAssets(ctx, assetv1.AssetType_ASSET_TYPE_KERNEL, nil)
	if err != nil {
		t.Errorf("ListAssets() unexpected error: %v", err)
	}
	if len(assets) != 0 {
		t.Errorf("ListAssets() expected empty list, got %d assets", len(assets))
	}

	// Test PrepareAssets returns empty map
	paths, err := client.PrepareAssets(ctx, []string{"asset1", "asset2"}, "/tmp", "vm-123")
	if err != nil {
		t.Errorf("PrepareAssets() unexpected error: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("PrepareAssets() expected empty map, got %d paths", len(paths))
	}

	// Test AcquireAsset returns empty lease
	lease, err := client.AcquireAsset(ctx, "asset1", "vm-123")
	if err != nil {
		t.Errorf("AcquireAsset() unexpected error: %v", err)
	}
	if lease != "" {
		t.Errorf("AcquireAsset() expected empty lease, got %s", lease)
	}

	// Test ReleaseAsset succeeds
	err = client.ReleaseAsset(ctx, "lease-123")
	if err != nil {
		t.Errorf("ReleaseAsset() unexpected error: %v", err)
	}
}
