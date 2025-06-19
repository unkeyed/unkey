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
	assetv1 "github.com/unkeyed/unkey/go/deploy/assetmanagerd/gen/asset/v1"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/config"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/registry"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/storage"
)

// Service implements the AssetManagerService
type Service struct {
	cfg      *config.Config
	logger   *slog.Logger
	registry *registry.Registry
	storage  storage.Backend
}

// New creates a new asset service
func New(cfg *config.Config, logger *slog.Logger, registry *registry.Registry, storage storage.Backend) *Service {
	return &Service{
		cfg:      cfg,
		logger:   logger.With("component", "service"),
		registry: registry,
		storage:  storage,
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
	//nolint:exhaustruct // Id field will be auto-generated
	asset := &assetv1.Asset{
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

		// Prepare the target file path
		targetFile := filepath.Join(req.Msg.GetTargetPath(), filepath.Base(localPath))

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

// UploadAsset handles direct asset uploads (for future use)
func (s *Service) UploadAsset(ctx context.Context, name string, assetType assetv1.AssetType, reader io.Reader, size int64) (*assetv1.Asset, error) {
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
