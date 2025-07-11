package firecracker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	assetv1 "github.com/unkeyed/unkey/go/deploy/assetmanagerd/gen/asset/v1"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1"
)

// MockAssetClient is a mock implementation of the assetmanager.Client interface
type MockAssetClient struct {
	mock.Mock
}

func (m *MockAssetClient) ListAssets(ctx context.Context, assetType assetv1.AssetType, labels map[string]string) ([]*assetv1.Asset, error) {
	args := m.Called(ctx, assetType, labels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*assetv1.Asset), args.Error(1)
}

func (m *MockAssetClient) PrepareAssets(ctx context.Context, assetIDs []string, targetPath string, vmID string) (map[string]string, error) {
	args := m.Called(ctx, assetIDs, targetPath, vmID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockAssetClient) AcquireAsset(ctx context.Context, assetID string, vmID string) (string, error) {
	args := m.Called(ctx, assetID, vmID)
	return args.String(0), args.Error(1)
}

func (m *MockAssetClient) ReleaseAsset(ctx context.Context, leaseID string) error {
	args := m.Called(ctx, leaseID)
	return args.Error(0)
}

func TestBuildAssetRequirements(t *testing.T) {
	client := &SDKClientV4{}

	tests := []struct {
		name     string
		config   *metaldv1.VmConfig
		expected int
	}{
		{
			name: "basic VM with kernel and rootfs",
			config: &metaldv1.VmConfig{
				Boot: &metaldv1.BootConfig{
					KernelPath: "/path/to/kernel",
				},
				Storage: []*metaldv1.StorageDevice{
					{
						IsRootDevice: true,
						Options: map[string]string{
							"docker_image": "ghcr.io/unkeyed/unkey:latest",
						},
					},
				},
			},
			expected: 2, // kernel + rootfs
		},
		{
			name: "VM with docker image in metadata",
			config: &metaldv1.VmConfig{
				Boot: &metaldv1.BootConfig{
					KernelPath: "/path/to/kernel",
				},
				Storage: []*metaldv1.StorageDevice{
					{
						IsRootDevice: true,
					},
				},
				Metadata: map[string]string{
					"docker_image": "nginx:alpine",
				},
			},
			expected: 2, // kernel + rootfs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqs := client.buildAssetRequirements(tt.config)
			assert.Equal(t, tt.expected, len(reqs))
		})
	}
}

func TestMatchAssets(t *testing.T) {
	client := &SDKClientV4{}

	// Test successful matching
	reqs := []assetRequirement{
		{
			Type:     assetv1.AssetType_ASSET_TYPE_KERNEL,
			Required: true,
		},
		{
			Type: assetv1.AssetType_ASSET_TYPE_ROOTFS,
			Labels: map[string]string{
				"docker_image": "ghcr.io/unkeyed/unkey:latest",
			},
			Required: true,
		},
	}

	availableAssets := []*assetv1.Asset{
		{
			Id:   "kernel-123",
			Type: assetv1.AssetType_ASSET_TYPE_KERNEL,
		},
		{
			Id:   "rootfs-456",
			Type: assetv1.AssetType_ASSET_TYPE_ROOTFS,
			Labels: map[string]string{
				"docker_image": "ghcr.io/unkeyed/unkey:latest",
			},
		},
	}

	mapping, err := client.matchAssets(reqs, availableAssets)
	assert.NoError(t, err)
	assert.NotNil(t, mapping)
	assert.Equal(t, 2, len(mapping.AssetIDs()))
	assert.Contains(t, mapping.AssetIDs(), "kernel-123")
	assert.Contains(t, mapping.AssetIDs(), "rootfs-456")

	// Test missing required asset
	reqsMissing := []assetRequirement{
		{
			Type: assetv1.AssetType_ASSET_TYPE_ROOTFS,
			Labels: map[string]string{
				"docker_image": "nonexistent:latest",
			},
			Required: true,
		},
	}

	_, err = client.matchAssets(reqsMissing, availableAssets)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no matching asset found")
}

// AIDEV-NOTE: These are basic unit tests for the asset integration.
// More comprehensive integration tests would require:
// 1. A running assetmanagerd instance or more sophisticated mocking
// 2. Tests for the full VM creation flow with asset preparation
// 3. Tests for lease acquisition and release
// 4. Tests for error handling and rollback scenarios
