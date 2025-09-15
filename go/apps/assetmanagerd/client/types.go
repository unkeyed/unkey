package client

import (
	assetv1 "github.com/unkeyed/unkey/go/gen/proto/assetmanagerd/v1"
)

// AIDEV-NOTE: Type definitions for assetmanagerd client requests and responses
// These provide a clean interface that wraps the protobuf types

// RegisterAssetRequest represents a request to register a new asset
type RegisterAssetRequest struct {
	Name      string
	Type      assetv1.AssetType
	Backend   assetv1.StorageBackend
	Location  string
	SizeBytes int64
	Checksum  string
	Labels    map[string]string
	CreatedBy string
}

// RegisterAssetResponse represents the response from registering an asset
type RegisterAssetResponse struct {
	Asset *assetv1.Asset
}

// GetAssetResponse represents the response from getting an asset
type GetAssetResponse struct {
	Asset *assetv1.Asset
}

// ListAssetsRequest represents a request to list assets
type ListAssetsRequest struct {
	Type      assetv1.AssetType
	Status    assetv1.AssetStatus
	Labels    map[string]string
	PageSize  int32
	PageToken string
}

// ListAssetsResponse represents the response from listing assets
type ListAssetsResponse struct {
	Assets        []*assetv1.Asset
	NextPageToken string
}

// QueryAssetsRequest represents a request to query assets with auto-build
type QueryAssetsRequest struct {
	Type   assetv1.AssetType
	Labels map[string]string
}

// QueryAssetsResponse represents the response from querying assets
type QueryAssetsResponse struct {
	Assets []*assetv1.Asset
}

// PrepareAssetsRequest represents a request to prepare assets for a host
type PrepareAssetsRequest struct {
	AssetIds []string
	HostId   string
	JailerId string
	CacheDir string
}

// PrepareAssetsResponse represents the response from preparing assets
type PrepareAssetsResponse struct {
	PreparedPaths []string
	Success       bool
}

// AcquireAssetResponse represents the response from acquiring an asset
type AcquireAssetResponse struct {
	Success        bool
	ReferenceCount int32
}

// ReleaseAssetResponse represents the response from releasing an asset
type ReleaseAssetResponse struct {
	Success        bool
	ReferenceCount int32
}

// DeleteAssetResponse represents the response from deleting an asset
type DeleteAssetResponse struct {
	Success bool
}

// GarbageCollectRequest represents a request to perform garbage collection
type GarbageCollectRequest struct {
	DryRun       bool
	MaxAgeHours  int32
	ForceCleanup bool
}

// GarbageCollectResponse represents the response from garbage collection
type GarbageCollectResponse struct {
	RemovedAssets []string
	FreedBytes    int64
	Success       bool
}
