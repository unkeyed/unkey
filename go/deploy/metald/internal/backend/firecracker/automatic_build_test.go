package firecracker

import (
	"context"
	"os"
	"testing"
	"time"

	"log/slog"

	assetv1 "github.com/unkeyed/unkey/go/deploy/assetmanagerd/gen/asset/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/config"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1"
)

// mockAssetClient implements assetmanager.Client for testing automatic builds
type mockAssetClient struct {
	// Control behavior
	triggerBuild bool
	buildDelay   time.Duration
	buildError   error

	// Track calls
	queryCalls []queryCall
	lastQuery  *assetv1.QueryAssetsRequest
}

type queryCall struct {
	assetType assetv1.AssetType
	labels    map[string]string
	buildOpts *assetv1.BuildOptions
}

func (m *mockAssetClient) QueryAssets(ctx context.Context, assetType assetv1.AssetType, labels map[string]string, buildOptions *assetv1.BuildOptions) (*assetv1.QueryAssetsResponse, error) {
	m.queryCalls = append(m.queryCalls, queryCall{
		assetType: assetType,
		labels:    labels,
		buildOpts: buildOptions,
	})

	// For initial kernel check, return a kernel asset to indicate assetmanager is enabled
	if assetType == assetv1.AssetType_ASSET_TYPE_KERNEL && buildOptions == nil {
		return &assetv1.QueryAssetsResponse{
			Assets: []*assetv1.Asset{
				{
					Id:   "kernel-test",
					Type: assetv1.AssetType_ASSET_TYPE_KERNEL,
				},
			},
		}, nil
	}

	// For kernel queries with build options, return a kernel asset
	if assetType == assetv1.AssetType_ASSET_TYPE_KERNEL && buildOptions != nil {
		return &assetv1.QueryAssetsResponse{
			Assets: []*assetv1.Asset{
				{
					Id:     "kernel-123",
					Name:   "vmlinux",
					Type:   assetv1.AssetType_ASSET_TYPE_KERNEL,
					Status: assetv1.AssetStatus_ASSET_STATUS_AVAILABLE,
				},
			},
		}, nil
	}

	// Simulate no assets initially for rootfs queries
	resp := &assetv1.QueryAssetsResponse{
		Assets: []*assetv1.Asset{},
	}

	// If build is triggered and enabled for rootfs
	if m.triggerBuild && buildOptions != nil && buildOptions.EnableAutoBuild && assetType == assetv1.AssetType_ASSET_TYPE_ROOTFS {
		dockerImage := labels["docker_image"]

		// Create build info
		buildInfo := &assetv1.BuildInfo{
			BuildId:     "test-build-123",
			DockerImage: dockerImage,
			Status:      "building",
		}

		// Simulate build delay
		if m.buildDelay > 0 && buildOptions.WaitForCompletion {
			select {
			case <-time.After(m.buildDelay):
				// Build completed
				if m.buildError != nil {
					buildInfo.Status = "failed"
					buildInfo.ErrorMessage = m.buildError.Error()
				} else {
					buildInfo.Status = "completed"
					buildInfo.AssetId = "test-asset-456"

					// Add the built asset to response
					resp.Assets = append(resp.Assets, &assetv1.Asset{
						Id:     "test-asset-456",
						Name:   "rootfs-" + dockerImage,
						Type:   assetv1.AssetType_ASSET_TYPE_ROOTFS,
						Status: assetv1.AssetStatus_ASSET_STATUS_AVAILABLE,
						Labels: labels,
					})
				}
			case <-ctx.Done():
				buildInfo.Status = "failed"
				buildInfo.ErrorMessage = "context cancelled"
			}
		}

		resp.TriggeredBuilds = append(resp.TriggeredBuilds, buildInfo)
	}

	return resp, nil
}

func (m *mockAssetClient) ListAssets(ctx context.Context, assetType assetv1.AssetType, labels map[string]string) ([]*assetv1.Asset, error) {
	// Not used in this test
	return []*assetv1.Asset{}, nil
}

func (m *mockAssetClient) PrepareAssets(ctx context.Context, assetIDs []string, targetPath string, vmID string) (map[string]string, error) {
	// Return mock paths
	paths := make(map[string]string)
	for _, id := range assetIDs {
		paths[id] = targetPath + "/asset-" + id
	}
	return paths, nil
}

func (m *mockAssetClient) AcquireAsset(ctx context.Context, assetID string, vmID string) (string, error) {
	return "lease-" + assetID, nil
}

func (m *mockAssetClient) ReleaseAsset(ctx context.Context, leaseID string) error {
	return nil
}

// TestAutomaticAssetBuilding tests the automatic build flow
func TestAutomaticAssetBuilding(t *testing.T) {
	// AIDEV-NOTE: This test verifies the complete automatic build flow:
	// 1. VM requests rootfs with docker_image label
	// 2. Asset doesn't exist, so QueryAssets triggers a build
	// 3. Build completes and asset is registered
	// 4. VM uses the newly built asset

	tests := []struct {
		name         string
		dockerImage  string
		tenantID     string
		triggerBuild bool
		buildDelay   time.Duration
		buildError   error
		expectError  bool
		expectBuild  bool
	}{
		{
			name:         "successful automatic build",
			dockerImage:  "alpine:latest",
			tenantID:     "test-tenant",
			triggerBuild: true,
			buildDelay:   100 * time.Millisecond,
			expectBuild:  true,
		},
		{
			name:         "build failure",
			dockerImage:  "invalid:image",
			tenantID:     "test-tenant",
			triggerBuild: true,
			buildDelay:   100 * time.Millisecond,
			buildError:   context.DeadlineExceeded,
			expectError:  true,
			expectBuild:  true,
		},
		{
			name:         "no automatic build when disabled",
			dockerImage:  "alpine:latest",
			tenantID:     "test-tenant",
			triggerBuild: false,
			expectError:  true, // Should fail due to missing asset
			expectBuild:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock asset client
			mockClient := &mockAssetClient{
				triggerBuild: tt.triggerBuild,
				buildDelay:   tt.buildDelay,
				buildError:   tt.buildError,
			}

			// Create SDK client with mock
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
			client := &SDKClientV4{
				assetClient: mockClient,
				logger:      logger,
				jailerConfig: &config.JailerConfig{
					ChrootBaseDir: "/tmp/test-jailer",
					UID:           1000,
					GID:           1000,
				},
			}

			// Create VM config with docker_image metadata
			vmConfig := &metaldv1.VmConfig{
				Boot: &metaldv1.BootConfig{
					KernelPath: "/test/kernel",
				},
				Storage: []*metaldv1.StorageDevice{
					{
						Path:         "", // Should be populated by asset
						ReadOnly:     false,
						IsRootDevice: true,
					},
				},
				Metadata: map[string]string{
					"docker_image": tt.dockerImage,
					"tenant_id":    tt.tenantID,
				},
			}

			// Test prepareVMAssets which triggers the automatic build
			ctx := context.Background()
			assetMapping, paths, err := client.prepareVMAssets(ctx, "test-vm-123", vmConfig)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// Verify QueryAssets was called with correct parameters
			if len(mockClient.queryCalls) == 0 {
				t.Fatal("QueryAssets was not called")
			}

			lastCall := mockClient.queryCalls[len(mockClient.queryCalls)-1]

			// Check asset type
			if lastCall.assetType != assetv1.AssetType_ASSET_TYPE_ROOTFS {
				t.Errorf("expected ASSET_TYPE_ROOTFS, got %v", lastCall.assetType)
			}

			// Check docker_image label
			if lastCall.labels["docker_image"] != tt.dockerImage {
				t.Errorf("expected docker_image=%s, got %s", tt.dockerImage, lastCall.labels["docker_image"])
			}

			// Check build options
			if lastCall.buildOpts == nil {
				t.Fatal("build options were not provided")
			}

			if !lastCall.buildOpts.EnableAutoBuild {
				t.Error("expected EnableAutoBuild to be true")
			}

			if !lastCall.buildOpts.WaitForCompletion {
				t.Error("expected WaitForCompletion to be true")
			}

			if lastCall.buildOpts.TenantId != tt.tenantID {
				t.Errorf("expected tenant_id=%s, got %s", tt.tenantID, lastCall.buildOpts.TenantId)
			}

			// If successful, verify asset mapping
			if !tt.expectError && assetMapping != nil {
				if len(assetMapping.assets) == 0 {
					t.Error("expected assets in mapping but got none")
				}

				if len(paths) == 0 {
					t.Error("expected prepared paths but got none")
				}
			}
		})
	}
}

// TestAutomaticBuildTimeout tests build timeout handling
func TestAutomaticBuildTimeout(t *testing.T) {
	// Create mock that simulates a long build
	mockClient := &mockAssetClient{
		triggerBuild: true,
		buildDelay:   5 * time.Second, // Longer than our context timeout
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	client := &SDKClientV4{
		assetClient: mockClient,
		logger:      logger,
		jailerConfig: &config.JailerConfig{
			ChrootBaseDir: "/tmp/test-jailer",
			UID:           1000,
			GID:           1000,
		},
	}

	vmConfig := &metaldv1.VmConfig{
		Boot: &metaldv1.BootConfig{
			KernelPath: "/test/kernel",
		},
		Storage: []*metaldv1.StorageDevice{
			{IsRootDevice: true},
		},
		Metadata: map[string]string{
			"docker_image": "slow:build",
		},
	}

	// Use a short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// This should timeout
	_, _, err := client.prepareVMAssets(ctx, "test-vm-timeout", vmConfig)

	if err == nil {
		t.Error("expected timeout error but got none")
	}
}
