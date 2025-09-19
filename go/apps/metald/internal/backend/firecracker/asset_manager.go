//go:build linux
// +build linux

package firecracker

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	assetv1 "github.com/unkeyed/unkey/go/gen/proto/assetmanagerd/v1"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// releaseAssetLeases releases all asset leases for a VM
func (c *Client) releaseAssetLeases(ctx context.Context, vmID string) {
	if leaseIDs, ok := c.vmAssetLeases[vmID]; ok {
		c.logger.LogAttrs(ctx, slog.LevelInfo, "releasing asset leases",
			slog.String("vm_id", vmID),
			slog.Int("lease_count", len(leaseIDs)),
		)

		for _, leaseID := range leaseIDs {
			releaseCtx, releaseSpan := c.tracer.Start(ctx, "metald.firecracker.release_asset",
				trace.WithAttributes(
					attribute.String("vm.id", vmID),
					attribute.String("lease.id", leaseID),
				),
			)
			err := c.assetClient.ReleaseAsset(releaseCtx, leaseID)
			if err != nil {
				releaseSpan.RecordError(err)
				releaseSpan.SetStatus(codes.Error, err.Error())
				c.logger.ErrorContext(ctx, "failed to release asset lease",
					"vm_id", vmID,
					"lease_id", leaseID,
					"error", err,
				)
				// Continue with other leases even if one fails
			}
			releaseSpan.End()
		}
		delete(c.vmAssetLeases, vmID)
	}
}

// acquireAssetLeases acquires leases for VM assets after successful boot
func (c *Client) acquireAssetLeases(ctx context.Context, vmID string, assetMapping *assetMapping) {
	if assetMapping == nil || len(assetMapping.AssetIDs()) == 0 {
		return
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "acquiring asset leases for VM",
		slog.String("vm_id", vmID),
		slog.Int("asset_count", len(assetMapping.AssetIDs())),
	)

	leaseIDs := []string{}
	for _, assetID := range assetMapping.AssetIDs() {
		acquireCtx, acquireSpan := c.tracer.Start(ctx, "metald.firecracker.acquire_asset",
			trace.WithAttributes(
				attribute.String("vm.id", vmID),
				attribute.String("asset.id", assetID),
			),
		)

		leaseID, err := c.assetClient.AcquireAsset(acquireCtx, assetID, vmID)
		if err != nil {
			acquireSpan.RecordError(err)
			acquireSpan.SetStatus(codes.Error, err.Error())
			c.logger.ErrorContext(ctx, "failed to acquire asset lease",
				"vm_id", vmID,
				"asset_id", assetID,
				"error", err,
			)
			// Continue trying to acquire other leases even if one fails
		} else {
			acquireSpan.SetAttributes(attribute.String("lease.id", leaseID))
			leaseIDs = append(leaseIDs, leaseID)
		}
		acquireSpan.End()
	}

	// Store lease IDs for cleanup during VM deletion
	if len(leaseIDs) > 0 {
		c.vmAssetLeases[vmID] = leaseIDs
		c.logger.LogAttrs(ctx, slog.LevelInfo, "acquired asset leases",
			slog.String("vm_id", vmID),
			slog.Int("lease_count", len(leaseIDs)),
		)
	}
}

// generateAssetID generates a deterministic asset ID based on type and labels
func (c *Client) generateAssetID(assetType assetv1.AssetType, labels map[string]string) string {
	// Create a deterministic string from sorted labels
	var parts []string
	parts = append(parts, fmt.Sprintf("type=%s", assetType.String()))

	// Sort label keys for deterministic ordering
	var keys []string
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Add sorted labels
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, labels[k]))
	}

	// Create a hash of the combined string
	combined := strings.Join(parts, ",")
	hash := sha256.Sum256([]byte(combined))

	// Return a readable asset ID
	return fmt.Sprintf("asset-%x", hash[:8])
}

// prepareVMAssets prepares kernel and rootfs assets for the VM in the jailer chroot
// Returns the asset mapping for lease acquisition after successful boot
func (c *Client) prepareVMAssets(ctx context.Context, vmID string, config *metaldv1.VmConfig) (*assetMapping, map[string]string, error) {
	// Calculate the jailer chroot path
	jailerRoot := filepath.Join(
		c.jailerConfig.ChrootBaseDir,
		"firecracker",
		vmID,
		"root",
	)

	c.logger.LogAttrs(ctx, slog.LevelInfo, "preparing VM assets using assetmanager",
		slog.String("vm_id", vmID),
		slog.String("target_path", jailerRoot),
	)

	// Ensure the jailer root directory exists
	if err := os.MkdirAll(jailerRoot, 0o755); err != nil {
		return nil, nil, fmt.Errorf("failed to create jailer root directory: %w", err)
	}

	// Check if assetmanager is available, fallback to static if not
	// TODO: implement a check with backoff/deadline

	// Build asset requirements from VM configuration
	requiredAssets := c.buildAssetRequirements(config)
	c.logger.LogAttrs(ctx, slog.LevelDebug, "determined asset requirements",
		slog.String("vm_id", vmID),
		slog.Int("required_count", len(requiredAssets)),
	)

	// Query and build assets as needed
	allAssets, err := c.queryAndBuildAssets(ctx, vmID, config, requiredAssets)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query/build assets: %w", err)
	}

	// Match required assets with available ones
	assetMapping, err := c.matchAssets(requiredAssets, allAssets)
	if err != nil {
		c.logger.LogAttrs(ctx, slog.LevelError, "failed to match assets",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return nil, nil, fmt.Errorf("asset matching failed: %w", err)
	}

	// Prepare assets in target location
	preparedPaths, err := c.prepareAssetsInLocation(ctx, vmID, assetMapping, jailerRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prepare assets: %w", err)
	}

	// Copy metadata files alongside rootfs assets if they exist
	if err := c.copyMetadataFilesForAssets(ctx, vmID, config, preparedPaths, jailerRoot); err != nil {
		c.logger.WarnContext(ctx, "failed to copy metadata files",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		// Don't fail asset preparation for metadata issues
	}

	return assetMapping, preparedPaths, nil
}

// isAssetManagerAvailable checks if the asset manager service is available
func (c *Client) isAssetManagerAvailable(ctx context.Context, vmID string) bool {
	ctx, checkSpan := c.tracer.Start(ctx, "metald.firecracker.check_assetmanager",
		trace.WithAttributes(
			attribute.String("vm.id", vmID),
			attribute.String("asset.type", "KERNEL"),
		),
	)
	_, err := c.assetClient.QueryAssets(ctx, assetv1.AssetType_ASSET_TYPE_KERNEL, nil, nil)
	checkSpan.End()
	return err == nil
}

// queryAndBuildAssets queries assetmanager for available assets with automatic build support
func (c *Client) queryAndBuildAssets(ctx context.Context, vmID string, config *metaldv1.VmConfig, requiredAssets []assetRequirement) ([]*assetv1.Asset, error) {
	allAssets := []*assetv1.Asset{}

	// Extract tenant_id from VM metadata if available
	tenantID := "cli-tenant" // Default tenant for CLI operations
	if tid, ok := config.GetMetadata()["tenant_id"]; ok {
		tenantID = tid
	}

	// Group requirements by type and labels for efficient querying
	queryGroups := c.groupAssetRequirements(requiredAssets)

	// Query each unique combination of type and labels
	for key, reqs := range queryGroups {
		assets, err := c.queryAssetGroup(ctx, vmID, config, key, reqs[0], tenantID)
		if err != nil {
			return nil, err
		}
		allAssets = append(allAssets, assets...)
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "retrieved available assets",
		slog.String("vm_id", vmID),
		slog.Int("available_count", len(allAssets)),
	)

	// Log asset details for debugging
	for _, asset := range allAssets {
		c.logger.LogAttrs(ctx, slog.LevelInfo, "available asset",
			slog.String("asset_id", asset.GetId()),
			slog.String("asset_type", asset.GetType().String()),
			slog.Any("labels", asset.GetLabels()),
		)
	}

	return allAssets, nil
}

// groupAssetRequirements groups requirements by type and labels for efficient querying
func (c *Client) groupAssetRequirements(requiredAssets []assetRequirement) map[queryKey][]assetRequirement {
	queryGroups := make(map[queryKey][]assetRequirement)
	for _, req := range requiredAssets {
		// Serialize labels for grouping
		labelStr := ""
		for k, v := range req.Labels {
			if labelStr != "" {
				labelStr += ","
			}
			labelStr += fmt.Sprintf("%s=%s", k, v)
		}
		key := queryKey{assetType: req.Type, labels: labelStr}
		queryGroups[key] = append(queryGroups[key], req)
	}
	return queryGroups
}

// queryAssetGroup queries a specific group of assets with the same type and labels
func (c *Client) queryAssetGroup(ctx context.Context, vmID string, config *metaldv1.VmConfig, key queryKey, req assetRequirement, tenantID string) ([]*assetv1.Asset, error) {
	labels := req.Labels

	// Generate a deterministic asset ID
	assetID := c.generateAssetID(key.assetType, labels)

	c.logger.LogAttrs(ctx, slog.LevelInfo, "generated asset ID for query",
		slog.String("asset_id", assetID),
		slog.String("asset_type", key.assetType.String()),
		slog.Any("labels", labels),
	)

	// Configure build options
	buildOptions := c.createBuildOptions(config, labels, tenantID, assetID)

	// Record query initiation
	_, initSpan := c.tracer.Start(ctx, "metald.firecracker.query_assets",
		trace.WithAttributes(
			attribute.String("vm.id", vmID),
			attribute.String("asset.type", key.assetType.String()),
			attribute.StringSlice("asset.labels", formatLabels(labels)),
			attribute.String("tenant.id", tenantID),
			attribute.Bool("auto_build.enabled", buildOptions.GetEnableAutoBuild()),
			attribute.Int("build.timeout_seconds", int(buildOptions.GetBuildTimeoutSeconds())),
		),
	)
	initSpan.End()

	// Query assets
	resp, queryErr := c.assetClient.QueryAssets(ctx, key.assetType, labels, buildOptions)
	if queryErr != nil {
		return nil, fmt.Errorf("failed to query assets of type %s with labels %v: %w",
			key.assetType.String(), labels, queryErr)
	}

	// Record results
	_, resultSpan := c.tracer.Start(ctx, "metald.firecracker.query_assets_complete",
		trace.WithAttributes(
			attribute.String("vm.id", vmID),
			attribute.String("asset.type", key.assetType.String()),
			attribute.Int("assets.found", len(resp.GetAssets())),
			attribute.Int("builds.triggered", len(resp.GetTriggeredBuilds())),
		),
	)
	resultSpan.End()

	// Log triggered builds
	c.logTriggeredBuilds(ctx, vmID, resp.GetTriggeredBuilds())

	return resp.GetAssets(), nil
}

// createBuildOptions creates build options for asset queries
func (c *Client) createBuildOptions(config *metaldv1.VmConfig, labels map[string]string, tenantID, assetID string) *assetv1.BuildOptions {
	// Create build labels (copy asset labels and add force_rebuild if needed)
	buildLabels := make(map[string]string)
	for k, v := range labels {
		buildLabels[k] = v
	}

	// Check for force_rebuild in VM config metadata
	if forceRebuild, ok := config.GetMetadata()["force_rebuild"]; ok && forceRebuild == "true" {
		buildLabels["force_rebuild"] = "true"
	}

	return &assetv1.BuildOptions{
		EnableAutoBuild:     true,
		WaitForCompletion:   true, // Block VM creation until build completes
		BuildTimeoutSeconds: 1800, // 30 minutes maximum wait time
		SuggestedAssetId:    assetID,
		BuildLabels:         buildLabels,
	}
}

// logTriggeredBuilds logs information about builds that were triggered
func (c *Client) logTriggeredBuilds(ctx context.Context, vmID string, builds []*assetv1.BuildInfo) {
	for _, build := range builds {
		c.logger.LogAttrs(ctx, slog.LevelInfo, "automatic build triggered for missing asset",
			slog.String("vm_id", vmID),
			slog.String("build_id", build.GetBuildId()),
			slog.String("docker_image", build.GetDockerImage()),
			slog.String("status", build.GetStatus()),
		)

		if build.GetStatus() == "failed" {
			c.logger.LogAttrs(ctx, slog.LevelError, "automatic build failed",
				slog.String("vm_id", vmID),
				slog.String("build_id", build.GetBuildId()),
				slog.String("error", build.GetErrorMessage()),
			)
		}
	}
}

// prepareAssetsInLocation prepares assets in the target location
func (c *Client) prepareAssetsInLocation(ctx context.Context, vmID string, assetMapping *assetMapping, jailerRoot string) (map[string]string, error) {
	ctx, prepareSpan := c.tracer.Start(ctx, "metald.firecracker.prepare_assets",
		trace.WithAttributes(
			attribute.String("vm.id", vmID),
			attribute.StringSlice("asset.ids", assetMapping.AssetIDs()),
			attribute.String("target.path", jailerRoot),
		),
	)

	preparedPaths, err := c.assetClient.PrepareAssets(
		ctx,
		assetMapping.AssetIDs(),
		jailerRoot,
		vmID,
	)

	if err != nil {
		prepareSpan.RecordError(err)
		prepareSpan.SetStatus(codes.Error, err.Error())
	} else {
		prepareSpan.SetAttributes(
			attribute.Int("assets.prepared", len(preparedPaths)),
		)
	}
	prepareSpan.End()

	if err == nil {
		c.logger.LogAttrs(ctx, slog.LevelInfo, "assets prepared successfully",
			slog.String("vm_id", vmID),
			slog.Int("asset_count", len(preparedPaths)),
		)
	}

	return preparedPaths, err
}

// formatLabels formats labels for tracing attributes
func formatLabels(labels map[string]string) []string {
	var labelPairs []string
	for k, v := range labels {
		labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, v))
	}
	return labelPairs
}

// buildAssetRequirements analyzes VM config to determine required assets
func (c *Client) buildAssetRequirements(config *metaldv1.VmConfig) []assetRequirement {
	var reqs []assetRequirement

	// // DEBUG: Log VM config for docker image troubleshooting
	// c.logger.Info("DEBUG: analyzing VM config for assets",
	// 	"storage_count", len(config.GetStorage()),
	// 	"metadata", config.GetMetadata(),
	// )
	// for i, disk := range config.GetStorage() {
	// 	c.logger.Info("DEBUG: storage device",
	// 		"index", i,
	// 		"id", disk.GetId(),
	// 		"path", disk.GetPath(),
	// 		"is_root", disk.GetIsRootDevice(),
	// 		"options", disk.GetOptions(),
	// 	)
	// }

	// // Kernel requirement
	// if config.GetBoot() != nil && config.GetBoot().GetKernelPath() != "" {
	// 	reqs = append(reqs, assetRequirement{
	// 		Type:     assetv1.AssetType_ASSET_TYPE_KERNEL,
	// 		Required: true,
	// 	})
	// }

	// // Rootfs requirements from storage devices
	// for _, disk := range config.GetStorage() {
	// 	if disk.GetIsRootDevice() {
	// 		labels := make(map[string]string)
	// 		// Check for docker image in disk options first, then config metadata
	// 		if dockerImage, ok := disk.GetOptions()["docker_image"]; ok {
	// 			labels["docker_image"] = dockerImage
	// 		} else if dockerImage, ok := config.GetMetadata()["docker_image"]; ok {
	// 			labels["docker_image"] = dockerImage
	// 		}

	// 		reqs = append(reqs, assetRequirement{
	// 			Type:     assetv1.AssetType_ASSET_TYPE_ROOTFS,
	// 			Labels:   labels,
	// 			Required: true,
	// 		})
	// 	}
	// }

	// // Initrd requirement (optional)
	// if config.GetBoot() != nil && config.GetBoot().GetInitrdPath() != "" {
	// 	reqs = append(reqs, assetRequirement{
	// 		Type:     assetv1.AssetType_ASSET_TYPE_INITRD,
	// 		Required: false,
	// 	})
	// }

	return reqs
}

// matchAssets matches available assets to requirements
func (c *Client) matchAssets(reqs []assetRequirement, availableAssets []*assetv1.Asset) (*assetMapping, error) {
	mapping := &assetMapping{
		requirements: reqs,
		assets:       make(map[string]*assetv1.Asset),
		assetIDs:     []string{},
	}

	for i, req := range reqs {
		var matched *assetv1.Asset

		// Find best matching asset
		for _, asset := range availableAssets {
			if asset.GetType() != req.Type {
				continue
			}

			// Check if all required labels match
			labelMatch := true
			for k, v := range req.Labels {
				if assetLabel, ok := asset.GetLabels()[k]; !ok || assetLabel != v {
					labelMatch = false
					break
				}
			}

			if labelMatch {
				matched = asset
				break
			}
		}

		if matched == nil && req.Required {
			// Build helpful error message
			labelStr := ""
			for k, v := range req.Labels {
				if labelStr != "" {
					labelStr += ", "
				}
				labelStr += fmt.Sprintf("%s=%s", k, v)
			}
			return nil, fmt.Errorf("no matching asset found for type %s with labels {%s}",
				req.Type.String(), labelStr)
		}

		if matched != nil {
			mapping.assets[fmt.Sprintf("%d", i)] = matched
			mapping.assetIDs = append(mapping.assetIDs, matched.GetId())
		}
	}

	return mapping, nil
}
