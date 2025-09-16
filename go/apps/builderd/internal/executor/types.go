package executor

import (
	"context"
	"time"

	builderv1 "github.com/unkeyed/unkey/go/gen/proto/builderd/v1"
)

// Executor defines the interface for build executors
type Executor interface {
	// Execute processes a build request and returns the result
	Execute(ctx context.Context, request *builderv1.CreateBuildRequest) (*BuildResult, error)

	// ExecuteWithID processes a build request with a pre-assigned build ID
	ExecuteWithID(ctx context.Context, request *builderv1.CreateBuildRequest, buildID string) (*BuildResult, error)

	// GetSupportedSources returns the source types this executor supports
	GetSupportedSources() []string

	// Cleanup removes any temporary resources for the given build
	Cleanup(ctx context.Context, buildID string) error
}

// BuildResult represents the result of a build operation
type BuildResult struct {
	// BuildID is the unique identifier for this build
	BuildID string

	// SourceType indicates the type of source (docker, git, archive)
	SourceType string

	// SourceImage/URL is the original source reference
	SourceImage string

	// RootfsPath is the path to the extracted rootfs
	RootfsPath string

	// WorkspaceDir is the temporary workspace directory
	WorkspaceDir string

	// StartTime when the build began
	StartTime time.Time

	// EndTime when the build completed
	EndTime time.Time

	// Status of the build (completed, failed, in_progress)
	Status string

	// Error message if the build failed
	Error string

	// Metadata contains additional build information
	Metadata map[string]string

	// ImageMetadata contains container runtime configuration
	ImageMetadata *builderv1.ImageMetadata

	// Metrics contains build performance metrics
	Metrics BuildMetrics
}

// BuildMetrics contains performance and resource metrics for a build
type BuildMetrics struct {
	// DurationMs is the total build time in milliseconds
	DurationMs int64

	// RootfsSizeBytes is the final size of the rootfs in bytes
	RootfsSizeBytes int64

	// SourceSizeBytes is the original source size in bytes
	SourceSizeBytes int64

	// CompressionRatio is the compression ratio achieved (if applicable)
	CompressionRatio float64

	// FilesCount is the number of files in the rootfs
	FilesCount int64

	// CacheHit indicates if the build used cached results
	CacheHit bool
}

// BuildStatus represents the possible states of a build
type BuildStatus string

const (
	BuildStatusPending    BuildStatus = "pending"
	BuildStatusInProgress BuildStatus = "in_progress"
	BuildStatusCompleted  BuildStatus = "completed"
	BuildStatusFailed     BuildStatus = "failed"
	BuildStatusCancelled  BuildStatus = "cancelled"
)

// BuildError represents different types of build errors
type BuildError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Common build error types
const (
	ErrorTypeSourceNotFound   = "source_not_found"
	ErrorTypeSourceTooLarge   = "source_too_large"
	ErrorTypeExtractionFailed = "extraction_failed"
	ErrorTypePermissionDenied = "permission_denied"
	ErrorTypeQuotaExceeded    = "quota_exceeded"
	ErrorTypeTimeout          = "timeout"
	ErrorTypeInternalError    = "internal_error"
)

// NewBuildError creates a new build error
func NewBuildError(errorType, message string) *BuildError {
	return &BuildError{ //nolint:exhaustruct // Details field is optional and can be added via WithDetails() method
		Type:    errorType,
		Message: message,
	}
}

// Error implements the error interface
func (e *BuildError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}
