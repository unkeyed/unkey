package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/builderd"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/config"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/registry"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/storage"
	assetv1 "github.com/unkeyed/unkey/go/gen/proto/deploy/assetmanagerd/v1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Service implements the AssetManagerService
type Service struct {
	cfg            *config.Config
	logger         *slog.Logger
	registry       *registry.Registry
	storage        storage.Backend
	builderdClient *builderd.Client
}

// New creates a new asset service
func New(cfg *config.Config, logger *slog.Logger, registry *registry.Registry, storage storage.Backend, builderdClient *builderd.Client) *Service {
	return &Service{
		cfg:            cfg,
		logger:         logger.With("component", "service"),
		registry:       registry,
		storage:        storage,
		builderdClient: builderdClient,
	}
}

// RegisterAsset registers a new asset
func (s *Service) RegisterAsset(
	ctx context.Context,
	req *connect.Request[assetv1.RegisterAssetRequest],
) (*connect.Response[assetv1.RegisterAssetResponse], error) {
	// AIDEV-NOTE: Assets are pre-stored before registration, this just adds metadata
	// This allows builderd to upload directly to storage then register

	// Validate request
	if req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}

	if req.Msg.GetLocation() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("location is required"))
	}

	// Verify asset exists in storage
	exists, err := s.storage.Exists(ctx, req.Msg.GetLocation())
	if err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to check asset existence",
			slog.String("location", req.Msg.GetLocation()),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to verify asset"))
	}

	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("asset not found at location: %s", req.Msg.GetLocation()))
	}

	// Get actual size and checksum from storage
	size := req.Msg.GetSizeBytes()
	if size == 0 {
		size, err = s.storage.GetSize(ctx, req.Msg.GetLocation())
		if err != nil {
			s.logger.LogAttrs(ctx, slog.LevelError, "failed to get asset size",
				slog.String("location", req.Msg.GetLocation()),
				slog.String("error", err.Error()),
			)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get asset size"))
		}
	}

	checksum := req.Msg.GetChecksum()
	if checksum == "" {
		checksum, err = s.storage.GetChecksum(ctx, req.Msg.GetLocation())
		if err != nil {
			s.logger.LogAttrs(ctx, slog.LevelError, "failed to get asset checksum",
				slog.String("location", req.Msg.GetLocation()),
				slog.String("error", err.Error()),
			)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get asset checksum"))
		}
	}

	// Create asset record
	//nolint:exhaustruct // Some fields may be auto-generated
	asset := &assetv1.Asset{
		Id:             req.Msg.GetId(), // Use provided ID if available
		Name:           req.Msg.GetName(),
		Type:           req.Msg.GetType(),
		Status:         assetv1.AssetStatus_ASSET_STATUS_AVAILABLE,
		Backend:        req.Msg.GetBackend(),
		Location:       req.Msg.GetLocation(),
		SizeBytes:      size,
		Checksum:       checksum,
		Labels:         req.Msg.GetLabels(),
		CreatedBy:      req.Msg.GetCreatedBy(),
		CreatedAt:      time.Now().Unix(),
		LastAccessedAt: time.Now().Unix(),
		ReferenceCount: 0,
		BuildId:        req.Msg.GetBuildId(),
		SourceImage:    req.Msg.GetSourceImage(),
	}

	// Save to registry
	if err := s.registry.CreateAsset(asset); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to create asset record",
			slog.String("name", req.Msg.GetName()),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to register asset"))
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "registered asset",
		slog.String("id", asset.GetId()),
		slog.String("name", asset.GetName()),
		slog.String("type", asset.GetType().String()),
		slog.Int64("size", asset.GetSizeBytes()),
	)

	return connect.NewResponse(&assetv1.RegisterAssetResponse{
		Asset: asset,
	}), nil
}

// GetAsset retrieves asset information
func (s *Service) GetAsset(
	ctx context.Context,
	req *connect.Request[assetv1.GetAssetRequest],
) (*connect.Response[assetv1.GetAssetResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}

	// Get asset from registry
	asset, err := s.registry.GetAsset(req.Msg.GetId())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to get asset",
			slog.String("id", req.Msg.GetId()),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get asset"))
	}

	//nolint:exhaustruct // LocalPath field is optional and set below if needed
	resp := &assetv1.GetAssetResponse{
		Asset: asset,
	}

	// Ensure local if requested
	if req.Msg.GetEnsureLocal() {
		localPath, err := s.storage.EnsureLocal(ctx, asset.GetLocation(), s.cfg.CacheDir)
		if err != nil {
			s.logger.LogAttrs(ctx, slog.LevelError, "failed to ensure asset is local",
				slog.String("id", req.Msg.GetId()),
				slog.String("location", asset.GetLocation()),
				slog.String("error", err.Error()),
			)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to ensure asset is local"))
		}
		resp.LocalPath = localPath
	}

	return connect.NewResponse(resp), nil
}

// ListAssets lists available assets
func (s *Service) ListAssets(
	ctx context.Context,
	req *connect.Request[assetv1.ListAssetsRequest],
) (*connect.Response[assetv1.ListAssetsResponse], error) {
	// Convert request to registry filters
	//nolint:exhaustruct // Limit and Offset are set below
	filters := registry.ListFilters{
		Type:   req.Msg.GetType(),
		Status: req.Msg.GetStatus(),
		Labels: req.Msg.GetLabelSelector(),
	}

	// Handle pagination
	pageSize := int(req.Msg.GetPageSize())
	if pageSize == 0 {
		pageSize = 100
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	filters.Limit = pageSize

	// Parse page token (simple offset-based pagination)
	if req.Msg.GetPageToken() != "" {
		var offset int
		if _, err := fmt.Sscanf(req.Msg.GetPageToken(), "offset:%d", &offset); err == nil {
			filters.Offset = offset
		}
	}

	// Get assets
	assets, err := s.registry.ListAssets(filters)
	if err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to list assets",
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list assets"))
	}

	// AIDEV-NOTE: Automatic asset building - if no rootfs found with docker_image label, trigger builderd
	if len(assets) == 0 && s.cfg.BuilderdEnabled && s.builderdClient != nil {
		// Check if this is a request for rootfs with docker_image label
		if req.Msg.GetType() == assetv1.AssetType_ASSET_TYPE_ROOTFS || req.Msg.GetType() == assetv1.AssetType_ASSET_TYPE_UNSPECIFIED {
			if dockerImage, ok := req.Msg.GetLabelSelector()["docker_image"]; ok && dockerImage != "" {
				s.logger.InfoContext(ctx, "no rootfs found, triggering automatic build",
					"docker_image", dockerImage,
				)

				// Trigger build and wait for completion
				if err := s.triggerAndWaitForBuild(ctx, dockerImage, req.Msg.GetLabelSelector()); err != nil {
					s.logger.ErrorContext(ctx, "failed to build rootfs automatically",
						"docker_image", dockerImage,
						"error", err,
					)
					// Return empty results but log the build failure
					// This allows the caller to handle the missing asset gracefully
				} else {
					// Re-query for assets after successful build
					assets, err = s.registry.ListAssets(filters)
					if err != nil {
						s.logger.LogAttrs(ctx, slog.LevelError, "failed to list assets after build",
							slog.String("error", err.Error()),
						)
					}
				}
			}
		}
	}

	//nolint:exhaustruct // NextPageToken is optional and set below if needed
	resp := &assetv1.ListAssetsResponse{
		Assets: assets,
	}

	// Set next page token if we hit the limit
	if len(assets) == pageSize {
		resp.NextPageToken = fmt.Sprintf("offset:%d", filters.Offset+pageSize)
	}

	return connect.NewResponse(resp), nil
}

// AcquireAsset acquires a reference to an asset
func (s *Service) AcquireAsset(
	ctx context.Context,
	req *connect.Request[assetv1.AcquireAssetRequest],
) (*connect.Response[assetv1.AcquireAssetResponse], error) {
	if req.Msg.GetAssetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("asset_id is required"))
	}

	if req.Msg.GetAcquiredBy() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("acquired_by is required"))
	}

	// Verify asset exists
	_, err := s.registry.GetAsset(req.Msg.GetAssetId())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get asset"))
	}

	// Create lease
	ttl := time.Duration(req.Msg.GetTtlSeconds()) * time.Second
	leaseID, err := s.registry.CreateLease(req.Msg.GetAssetId(), req.Msg.GetAcquiredBy(), ttl)
	if err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to create lease",
			slog.String("asset_id", req.Msg.GetAssetId()),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to acquire asset"))
	}

	// Get updated asset with incremented ref count
	asset, _ := s.registry.GetAsset(req.Msg.GetAssetId())

	s.logger.LogAttrs(ctx, slog.LevelInfo, "acquired asset",
		slog.String("asset_id", req.Msg.GetAssetId()),
		slog.String("lease_id", leaseID),
		slog.String("acquired_by", req.Msg.GetAcquiredBy()),
		slog.Int("ref_count", int(asset.GetReferenceCount())),
	)

	return connect.NewResponse(&assetv1.AcquireAssetResponse{
		Asset:   asset,
		LeaseId: leaseID,
	}), nil
}

// ReleaseAsset releases an asset reference
func (s *Service) ReleaseAsset(
	ctx context.Context,
	req *connect.Request[assetv1.ReleaseAssetRequest],
) (*connect.Response[assetv1.ReleaseAssetResponse], error) {
	if req.Msg.GetLeaseId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("lease_id is required"))
	}

	// Release lease
	if err := s.registry.ReleaseLease(req.Msg.GetLeaseId()); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to release lease",
			slog.String("lease_id", req.Msg.GetLeaseId()),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to release asset"))
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "released asset",
		slog.String("lease_id", req.Msg.GetLeaseId()),
	)

	// Return empty asset for now (could fetch if needed)
	return connect.NewResponse(&assetv1.ReleaseAssetResponse{
		//nolint:exhaustruct // Empty asset is intentional - could fetch if needed in future
		Asset: &assetv1.Asset{},
	}), nil
}

// DeleteAsset deletes an asset
func (s *Service) DeleteAsset(
	ctx context.Context,
	req *connect.Request[assetv1.DeleteAssetRequest],
) (*connect.Response[assetv1.DeleteAssetResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}

	// Get asset
	asset, err := s.registry.GetAsset(req.Msg.GetId())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get asset"))
	}

	// Check reference count
	if asset.GetReferenceCount() > 0 && !req.Msg.GetForce() {
		return connect.NewResponse(&assetv1.DeleteAssetResponse{
			Deleted: false,
			Message: fmt.Sprintf("asset has %d active references", asset.GetReferenceCount()),
		}), nil
	}

	// Delete from storage
	if err := s.storage.Delete(ctx, asset.GetLocation()); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to delete from storage",
			slog.String("id", req.Msg.GetId()),
			slog.String("location", asset.GetLocation()),
			slog.String("error", err.Error()),
		)
		// Continue with registry deletion even if storage deletion fails
	}

	// Delete from registry
	if err := s.registry.DeleteAsset(req.Msg.GetId()); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to delete from registry",
			slog.String("id", req.Msg.GetId()),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete asset"))
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "deleted asset",
		slog.String("id", req.Msg.GetId()),
		slog.String("name", asset.GetName()),
	)

	return connect.NewResponse(&assetv1.DeleteAssetResponse{
		Deleted: true,
		Message: "asset deleted successfully",
	}), nil
}

// GarbageCollect performs garbage collection
func (s *Service) GarbageCollect(
	ctx context.Context,
	req *connect.Request[assetv1.GarbageCollectRequest],
) (*connect.Response[assetv1.GarbageCollectResponse], error) {
	// AIDEV-NOTE: GC is critical for managing storage costs and disk space
	// This method handles both expired leases and unreferenced assets

	var deletedAssets []*assetv1.Asset
	var bytesFreed int64

	// Clean up expired leases first
	expiredLeases, err := s.registry.GetExpiredLeases()
	if err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to get expired leases",
			slog.String("error", err.Error()),
		)
	} else {
		for _, leaseID := range expiredLeases {
			if err := s.registry.ReleaseLease(leaseID); err != nil {
				s.logger.LogAttrs(ctx, slog.LevelWarn, "failed to release expired lease",
					slog.String("lease_id", leaseID),
					slog.String("error", err.Error()),
				)
			}
		}
		s.logger.LogAttrs(ctx, slog.LevelInfo, "cleaned up expired leases",
			slog.Int("count", len(expiredLeases)),
		)
	}

	// Get unreferenced assets
	//nolint:nestif // Nested conditions are clear and logical for GC operation
	if req.Msg.GetDeleteUnreferenced() {
		maxAge := time.Duration(req.Msg.GetMaxAgeSeconds()) * time.Second
		if maxAge == 0 {
			maxAge = s.cfg.GCMaxAge
		}

		unreferencedAssets, err := s.registry.GetUnreferencedAssets(maxAge)
		if err != nil {
			s.logger.LogAttrs(ctx, slog.LevelError, "failed to get unreferenced assets",
				slog.String("error", err.Error()),
			)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get unreferenced assets"))
		}

		for _, asset := range unreferencedAssets {
			if req.Msg.GetDryRun() {
				deletedAssets = append(deletedAssets, asset)
				bytesFreed += asset.GetSizeBytes()
				continue
			}

			// Delete from storage
			if err := s.storage.Delete(ctx, asset.GetLocation()); err != nil {
				s.logger.LogAttrs(ctx, slog.LevelWarn, "failed to delete asset from storage",
					slog.String("id", asset.GetId()),
					slog.String("location", asset.GetLocation()),
					slog.String("error", err.Error()),
				)
				continue
			}

			// Delete from registry
			if err := s.registry.DeleteAsset(asset.GetId()); err != nil {
				s.logger.LogAttrs(ctx, slog.LevelWarn, "failed to delete asset from registry",
					slog.String("id", asset.GetId()),
					slog.String("error", err.Error()),
				)
				continue
			}

			deletedAssets = append(deletedAssets, asset)
			bytesFreed += asset.GetSizeBytes()
		}
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "garbage collection completed",
		slog.Bool("dry_run", req.Msg.GetDryRun()),
		slog.Int("deleted_count", len(deletedAssets)),
		slog.Int64("bytes_freed", bytesFreed),
	)

	return connect.NewResponse(&assetv1.GarbageCollectResponse{
		DeletedAssets: deletedAssets,
		BytesFreed:    bytesFreed,
	}), nil
}

// PrepareAssets prepares assets for use (e.g., in jailer chroot)
func (s *Service) PrepareAssets(
	ctx context.Context,
	req *connect.Request[assetv1.PrepareAssetsRequest],
) (*connect.Response[assetv1.PrepareAssetsResponse], error) {
	if len(req.Msg.GetAssetIds()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("asset_ids is required"))
	}

	if req.Msg.GetTargetPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("target_path is required"))
	}

	assetPaths := make(map[string]string)

	for _, assetID := range req.Msg.GetAssetIds() {
		// Get asset
		asset, err := s.registry.GetAsset(assetID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("asset %s not found", assetID))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get asset %s", assetID))
		}

		// Ensure asset is available locally
		localPath, err := s.storage.EnsureLocal(ctx, asset.GetLocation(), s.cfg.CacheDir)
		if err != nil {
			s.logger.LogAttrs(ctx, slog.LevelError, "failed to ensure asset is local",
				slog.String("id", assetID),
				slog.String("location", asset.GetLocation()),
				slog.String("error", err.Error()),
			)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to prepare asset %s", assetID))
		}

		// Prepare the target file path with standardized names
		// AIDEV-NOTE: Use standardized names that Firecracker expects
		var filename string
		switch asset.GetType() {
		case assetv1.AssetType_ASSET_TYPE_KERNEL:
			filename = "vmlinux"
		case assetv1.AssetType_ASSET_TYPE_ROOTFS:
			filename = "rootfs.ext4"
		default:
			filename = filepath.Base(localPath)
		}
		targetFile := filepath.Join(req.Msg.GetTargetPath(), filename)

		// Create the target directory if it doesn't exist
		if err := os.MkdirAll(req.Msg.GetTargetPath(), 0755); err != nil {
			s.logger.LogAttrs(ctx, slog.LevelError, "failed to create target directory",
				slog.String("path", req.Msg.GetTargetPath()),
				slog.String("error", err.Error()),
			)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create target directory: %w", err))
		}

		// Try to create a hard link first (most efficient)
		if err := os.Link(localPath, targetFile); err != nil {
			// If hard link fails (e.g., different filesystems), copy the file
			s.logger.LogAttrs(ctx, slog.LevelDebug, "hard link failed, copying file",
				slog.String("source", localPath),
				slog.String("target", targetFile),
				slog.String("error", err.Error()),
			)

			// Copy the file
			if err := copyFile(localPath, targetFile); err != nil {
				s.logger.LogAttrs(ctx, slog.LevelError, "failed to copy asset to target",
					slog.String("source", localPath),
					slog.String("target", targetFile),
					slog.String("error", err.Error()),
				)
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to prepare asset %s: %w", assetID, err))
			}
		}

		// AIDEV-NOTE: For rootfs assets, also copy associated metadata file if it exists
		// This is needed for container initialization in microVMs
		if asset.GetType() == assetv1.AssetType_ASSET_TYPE_ROOTFS {
			// Look for metadata file alongside the rootfs asset
			metadataFileName := strings.TrimSuffix(filepath.Base(localPath), filepath.Ext(localPath)) + ".metadata.json"
			metadataSourcePath := filepath.Join(filepath.Dir(localPath), metadataFileName)

			if _, err := os.Stat(metadataSourcePath); err == nil {
				// Metadata file exists, copy it
				metadataTargetPath := filepath.Join(req.Msg.GetTargetPath(), "metadata.json")

				if err := os.Link(metadataSourcePath, metadataTargetPath); err != nil {
					// If hard link fails, copy the file
					if err := copyFile(metadataSourcePath, metadataTargetPath); err != nil {
						s.logger.LogAttrs(ctx, slog.LevelWarn, "failed to copy metadata file",
							slog.String("source", metadataSourcePath),
							slog.String("target", metadataTargetPath),
							slog.String("error", err.Error()),
						)
					} else {
						s.logger.LogAttrs(ctx, slog.LevelDebug, "copied metadata file for rootfs asset",
							slog.String("metadata_file", metadataTargetPath),
							slog.String("asset_id", assetID),
						)
					}
				} else {
					s.logger.LogAttrs(ctx, slog.LevelDebug, "linked metadata file for rootfs asset",
						slog.String("metadata_file", metadataTargetPath),
						slog.String("asset_id", assetID),
					)
				}
			} else {
				s.logger.LogAttrs(ctx, slog.LevelDebug, "no metadata file found for rootfs asset",
					slog.String("expected_path", metadataSourcePath),
					slog.String("asset_id", assetID),
				)
			}
		}

		assetPaths[assetID] = targetFile
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "prepared assets",
		slog.Int("count", len(assetPaths)),
		slog.String("target_path", req.Msg.GetTargetPath()),
		slog.String("prepared_for", req.Msg.GetPreparedFor()),
	)

	return connect.NewResponse(&assetv1.PrepareAssetsResponse{
		AssetPaths: assetPaths,
	}), nil
}

// StartGarbageCollector starts the background garbage collector
func (s *Service) StartGarbageCollector(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.GCInterval)
	defer ticker.Stop()

	s.logger.InfoContext(ctx, "started garbage collector",
		slog.Duration("interval", s.cfg.GCInterval),
		slog.Duration("max_age", s.cfg.GCMaxAge),
	)

	for {
		select {
		case <-ctx.Done():
			s.logger.InfoContext(ctx, "stopping garbage collector")
			return
		case <-ticker.C:
			// Run GC
			req := &assetv1.GarbageCollectRequest{
				MaxAgeSeconds:      int64(s.cfg.GCMaxAge.Seconds()),
				DeleteUnreferenced: true,
				DryRun:             false,
			}

			resp, err := s.GarbageCollect(ctx, connect.NewRequest(req))
			if err != nil {
				s.logger.ErrorContext(ctx, "garbage collection failed",
					slog.String("error", err.Error()),
				)
			} else {
				if len(resp.Msg.GetDeletedAssets()) > 0 {
					s.logger.InfoContext(ctx, "garbage collection completed",
						slog.Int("deleted_count", len(resp.Msg.GetDeletedAssets())),
						slog.Int64("bytes_freed", resp.Msg.GetBytesFreed()),
					)
				}
			}
		}
	}
}

// uploadAssetHelper handles direct asset uploads (helper method)
func (s *Service) uploadAssetHelper(ctx context.Context, name string, assetType assetv1.AssetType, reader io.Reader, size int64) (*assetv1.Asset, error) {
	// AIDEV-NOTE: This is a helper method for direct uploads
	// Currently, builderd uploads to storage directly then calls RegisterAsset
	// This method would be used for manual uploads or future integrations

	// Store asset
	id := fmt.Sprintf("%s-%d", name, time.Now().UnixNano())
	location, err := s.storage.Store(ctx, id, reader, size)
	if err != nil {
		return nil, fmt.Errorf("failed to store asset: %w", err)
	}

	// Get checksum
	checksum, err := s.storage.GetChecksum(ctx, location)
	if err != nil {
		// Clean up
		_ = s.storage.Delete(ctx, location)
		return nil, fmt.Errorf("failed to get checksum: %w", err)
	}

	// Register asset
	//nolint:exhaustruct // Optional fields not needed for manual upload
	req := &assetv1.RegisterAssetRequest{
		Name:      name,
		Type:      assetType,
		Backend:   assetv1.StorageBackend_STORAGE_BACKEND_LOCAL,
		Location:  location,
		SizeBytes: size,
		Checksum:  checksum,
		CreatedBy: "manual",
	}

	resp, err := s.RegisterAsset(ctx, connect.NewRequest(req))
	if err != nil {
		// Clean up
		_ = s.storage.Delete(ctx, location)
		return nil, err
	}

	return resp.Msg.GetAsset(), nil
}

// UploadAsset handles streaming asset uploads via gRPC
func (s *Service) UploadAsset(
	ctx context.Context,
	stream *connect.ClientStream[assetv1.UploadAssetRequest],
) (*connect.Response[assetv1.UploadAssetResponse], error) {
	// AIDEV-NOTE: Streaming upload RPC for builderd to upload assets before registering
	// First message should contain metadata, subsequent messages contain chunks

	// Read first message (metadata)
	if !stream.Receive() {
		if stream.Err() != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to receive metadata: %w", stream.Err()))
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("no metadata received"))
	}

	firstMsg := stream.Msg()
	metadata := firstMsg.GetMetadata()
	if metadata == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("first message must contain metadata"))
	}

	// Validate metadata
	if metadata.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}

	// Generate asset ID if not provided
	assetID := metadata.GetId()
	if assetID == "" {
		assetID = fmt.Sprintf("%s-%d", metadata.GetName(), time.Now().UnixNano())
	}

	// Create a pipe for streaming data to storage
	pipeReader, pipeWriter := io.Pipe()
	defer pipeReader.Close()

	// Start storing in background
	storeCh := make(chan struct {
		location string
		err      error
	}, 1)

	go func() {
		defer pipeWriter.Close()
		location, err := s.storage.Store(ctx, assetID, pipeReader, metadata.GetSizeBytes())
		storeCh <- struct {
			location string
			err      error
		}{location, err}
	}()

	// Stream data chunks
	var totalBytes int64
	for stream.Receive() {
		chunk := stream.Msg().GetChunk()
		if chunk == nil {
			continue
		}

		if _, err := pipeWriter.Write(chunk); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to write chunk: %w", err))
		}
		totalBytes += int64(len(chunk))
	}

	if stream.Err() != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("stream error: %w", stream.Err()))
	}

	// Close writer to signal end of data
	pipeWriter.Close()

	// Wait for storage to complete
	storeResult := <-storeCh
	if storeResult.err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to store asset: %w", storeResult.err))
	}

	// Get checksum
	checksum, err := s.storage.GetChecksum(ctx, storeResult.location)
	if err != nil {
		_ = s.storage.Delete(ctx, storeResult.location)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get checksum: %w", err))
	}

	// Register asset
	req := &assetv1.RegisterAssetRequest{
		Name:        metadata.GetName(),
		Type:        metadata.GetType(),
		Backend:     assetv1.StorageBackend_STORAGE_BACKEND_LOCAL,
		Location:    storeResult.location,
		SizeBytes:   totalBytes,
		Checksum:    checksum,
		Labels:      metadata.GetLabels(),
		CreatedBy:   metadata.GetCreatedBy(),
		BuildId:     metadata.GetBuildId(),
		SourceImage: metadata.GetSourceImage(),
		Id:          assetID,
	}

	resp, err := s.RegisterAsset(ctx, connect.NewRequest(req))
	if err != nil {
		_ = s.storage.Delete(ctx, storeResult.location)
		return nil, err
	}

	// Return response
	return connect.NewResponse(&assetv1.UploadAssetResponse{
		Asset: resp.Msg.GetAsset(),
	}), nil
}

// copyFile copies a file from source to destination
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy the file contents
	if _, copyErr := io.Copy(destFile, sourceFile); copyErr != nil {
		return fmt.Errorf("failed to copy file contents: %w", copyErr)
	}

	// Sync to ensure all data is written to disk
	if syncErr := destFile.Sync(); syncErr != nil {
		return fmt.Errorf("failed to sync destination file: %w", syncErr)
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	if chmodErr := os.Chmod(dst, sourceInfo.Mode()); chmodErr != nil {
		return fmt.Errorf("failed to set destination file permissions: %w", chmodErr)
	}

	return nil
}

// triggerAndWaitForBuild triggers builderd to create a rootfs and waits for completion
// AIDEV-NOTE: This implements the automatic asset creation workflow
func (s *Service) triggerAndWaitForBuild(ctx context.Context, dockerImage string, labels map[string]string) error {
	tracer := otel.Tracer("assetmanagerd")

	// Create build request
	ctx, buildSpan := tracer.Start(ctx, "assetmanagerd.service.trigger_build",
		trace.WithAttributes(
			attribute.String("docker.image", dockerImage),
			attribute.StringSlice("build.labels", func() []string {
				var labelPairs []string
				for k, v := range labels {
					labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, v))
				}
				return labelPairs
			}()),
		),
	)
	buildID, err := s.builderdClient.BuildDockerRootfs(ctx, dockerImage, labels)
	if err != nil {
		buildSpan.RecordError(err)
		buildSpan.SetStatus(codes.Error, err.Error())
		buildSpan.End()
		return fmt.Errorf("failed to trigger build: %w", err)
	}
	buildSpan.SetAttributes(attribute.String("build.id", buildID))
	buildSpan.End()

	s.logger.InfoContext(ctx, "build triggered",
		"build_id", buildID,
		"docker_image", dockerImage,
	)

	// Wait for build completion with polling
	pollInterval := 5 * time.Second
	ctx, waitSpan := tracer.Start(ctx, "assetmanagerd.service.wait_for_build",
		trace.WithAttributes(
			attribute.String("build.id", buildID),
			attribute.String("docker.image", dockerImage),
			attribute.String("poll.interval", pollInterval.String()),
		),
	)
	completedBuild, err := s.builderdClient.WaitForBuild(ctx, buildID, pollInterval)
	if err != nil {
		waitSpan.RecordError(err)
		waitSpan.SetStatus(codes.Error, err.Error())
	} else {
		waitSpan.SetAttributes(
			attribute.String("build.rootfs_path", completedBuild.Build.RootfsPath),
			attribute.String("build.status", completedBuild.Build.State.String()),
		)
	}
	waitSpan.End()
	if err != nil {
		return fmt.Errorf("build failed or timed out: %w", err)
	}

	s.logger.InfoContext(ctx, "build completed successfully",
		"build_id", completedBuild.Build.BuildId,
		"docker_image", dockerImage,
		"rootfs_path", completedBuild.Build.RootfsPath,
	)

	// If auto-register is enabled, the build should have been registered automatically
	// by builderd's post-build hook. If not, we'd need to register it here.
	if !s.cfg.BuilderdAutoRegister {
		// Manual registration would go here if needed
		// For now, we assume builderd handles registration
		s.logger.WarnContext(ctx, "auto-registration disabled, asset may need manual registration",
			"build_id", completedBuild.Build.BuildId,
		)
	}

	return nil
}

// QueryAssets queries assets with automatic build triggering if not found
// AIDEV-NOTE: This is the enhanced version of ListAssets that implements the complete
// asset query + automatic build workflow for metald
func (s *Service) QueryAssets(
	ctx context.Context,
	req *connect.Request[assetv1.QueryAssetsRequest],
) (*connect.Response[assetv1.QueryAssetsResponse], error) {
	// Convert request to registry filters
	//nolint:exhaustruct // Limit and Offset are set below
	filters := registry.ListFilters{
		Type:   req.Msg.GetType(),
		Status: req.Msg.GetStatus(),
		Labels: req.Msg.GetLabelSelector(),
	}

	// Handle pagination
	pageSize := int(req.Msg.GetPageSize())
	if pageSize == 0 {
		pageSize = 100
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	filters.Limit = pageSize

	// Parse page token (simple offset-based pagination)
	if req.Msg.GetPageToken() != "" {
		var offset int
		if _, err := fmt.Sscanf(req.Msg.GetPageToken(), "offset:%d", &offset); err == nil {
			filters.Offset = offset
		}
	}

	// Get assets
	assets, err := s.registry.ListAssets(filters)
	if err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to list assets",
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list assets"))
	}

	var triggeredBuilds []*assetv1.BuildInfo

	// Check if we should trigger automatic builds
	buildOpts := req.Msg.GetBuildOptions()
	if len(assets) == 0 && buildOpts != nil && buildOpts.GetEnableAutoBuild() && s.cfg.BuilderdEnabled && s.builderdClient != nil {
		// Check if this is a request for rootfs with docker_image label
		if req.Msg.GetType() == assetv1.AssetType_ASSET_TYPE_ROOTFS || req.Msg.GetType() == assetv1.AssetType_ASSET_TYPE_UNSPECIFIED {
			if dockerImage, ok := req.Msg.GetLabelSelector()["docker_image"]; ok && dockerImage != "" {
				s.logger.InfoContext(ctx, "no rootfs found, triggering automatic build",
					"docker_image", dockerImage,
					"tenant_id", buildOpts.GetTenantId(),
				)

				// Merge labels for the build
				buildLabels := make(map[string]string)
				for k, v := range req.Msg.GetLabelSelector() {
					buildLabels[k] = v
				}
				for k, v := range buildOpts.GetBuildLabels() {
					buildLabels[k] = v
				}

				s.logger.InfoContext(ctx, "triggering build with labels and asset ID",
					"build_labels", buildLabels,
					"suggested_asset_id", buildOpts.GetSuggestedAssetId(),
				)

				// Create build info
				buildInfo := &assetv1.BuildInfo{
					DockerImage: dockerImage,
					Status:      "pending",
				}

				// Set timeout
				timeout := time.Duration(buildOpts.GetBuildTimeoutSeconds()) * time.Second
				if timeout == 0 {
					timeout = 30 * time.Minute // Default timeout
				}

				// Create context with timeout
				buildCtx, cancel := context.WithTimeout(ctx, timeout)
				defer cancel()

				// Trigger build
				tracer := otel.Tracer("assetmanagerd")
				buildCtx, buildSpan := tracer.Start(buildCtx, "assetmanagerd.service.trigger_build_with_tenant",
					trace.WithAttributes(
						attribute.String("docker.image", dockerImage),
						attribute.String("tenant.id", buildOpts.GetTenantId()),
						attribute.StringSlice("build.labels", func() []string {
							var labelPairs []string
							for k, v := range buildLabels {
								labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, v))
							}
							return labelPairs
						}()),
					),
				)
				// AIDEV-NOTE: Extract proper customer ID from tenant context instead of using asset ID
				tenantID := buildOpts.GetTenantId()
				customerID := "cli-user" // Default fallback

				buildID, err := s.builderdClient.BuildDockerRootfsWithOptions(buildCtx, dockerImage, buildLabels, tenantID, customerID)
				if err != nil {
					buildSpan.RecordError(err)
					buildSpan.SetStatus(codes.Error, err.Error())
				} else {
					buildSpan.SetAttributes(attribute.String("build.id", buildID))
				}
				buildSpan.End()
				if err != nil {
					s.logger.ErrorContext(ctx, "failed to trigger build",
						"docker_image", dockerImage,
						"error", err,
					)
					buildInfo.Status = "failed"
					buildInfo.ErrorMessage = fmt.Sprintf("failed to trigger build: %v", err)
					triggeredBuilds = append(triggeredBuilds, buildInfo)
				} else {
					buildInfo.BuildId = buildID
					buildInfo.Status = "building"

					// Wait for completion if requested
					if buildOpts.GetWaitForCompletion() {
						// AIDEV-NOTE: Use proper build timeout instead of poll interval
						buildTimeout := time.Duration(buildOpts.GetBuildTimeoutSeconds()) * time.Second
						if buildTimeout == 0 {
							buildTimeout = 30 * time.Minute // Default timeout
						}

						buildCtx, waitSpan := tracer.Start(buildCtx, "assetmanagerd.service.wait_for_build_with_tenant",
							trace.WithAttributes(
								attribute.String("build.id", buildID),
								attribute.String("docker.image", dockerImage),
								attribute.String("tenant.id", buildOpts.GetTenantId()),
								attribute.String("build.timeout", buildTimeout.String()),
							),
						)
						completedBuild, err := s.builderdClient.WaitForBuildWithTenant(buildCtx, buildID, buildTimeout, buildOpts.GetTenantId())
						if err != nil {
							waitSpan.RecordError(err)
							waitSpan.SetStatus(codes.Error, err.Error())
						} else {
							waitSpan.SetAttributes(
								attribute.String("build.rootfs_path", completedBuild.Build.RootfsPath),
								attribute.String("build.status", completedBuild.Build.State.String()),
							)
						}
						waitSpan.End()
						if err != nil {
							s.logger.ErrorContext(ctx, "build failed or timed out",
								"build_id", buildID,
								"docker_image", dockerImage,
								"error", err,
							)
							buildInfo.Status = "failed"
							buildInfo.ErrorMessage = fmt.Sprintf("build failed: %v", err)
						} else {
							s.logger.InfoContext(ctx, "build completed successfully",
								"build_id", buildID,
								"docker_image", dockerImage,
								"rootfs_path", completedBuild.Build.RootfsPath,
							)
							buildInfo.Status = "completed"

							// Re-query for assets after successful build
							assets, err = s.registry.ListAssets(filters)
							if err != nil {
								s.logger.LogAttrs(ctx, slog.LevelError, "failed to list assets after build",
									slog.String("error", err.Error()),
								)
							} else if len(assets) > 0 {
								// Find the newly created asset
								buildInfo.AssetId = assets[0].GetId()
							}
						}
					}

					triggeredBuilds = append(triggeredBuilds, buildInfo)
				}
			}
		}
	}

	//nolint:exhaustruct // NextPageToken is optional and set below if needed
	resp := &assetv1.QueryAssetsResponse{
		Assets:          assets,
		TriggeredBuilds: triggeredBuilds,
	}

	// Set next page token if we hit the limit
	if len(assets) == pageSize {
		resp.NextPageToken = fmt.Sprintf("offset:%d", filters.Offset+pageSize)
	}

	return connect.NewResponse(resp), nil
}
