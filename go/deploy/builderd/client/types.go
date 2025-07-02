package client

import (
	builderv1 "github.com/unkeyed/unkey/go/deploy/builderd/gen/builder/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AIDEV-NOTE: Clean Go types wrapping builderd protobuf interfaces
// These types provide a simplified interface while maintaining compatibility with the underlying protobuf structures

// CreateBuildRequest wraps builderv1.CreateBuildRequest
type CreateBuildRequest struct {
	Config *builderv1.BuildConfig
}

// CreateBuildResponse wraps builderv1.CreateBuildResponse
type CreateBuildResponse struct {
	BuildID    string
	State      builderv1.BuildState
	CreatedAt  *timestamppb.Timestamp
	RootfsPath string
}

// GetBuildRequest wraps builderv1.GetBuildRequest
type GetBuildRequest struct {
	BuildID  string
	TenantID string
}

// GetBuildResponse wraps builderv1.GetBuildResponse
type GetBuildResponse struct {
	Build *builderv1.BuildJob
}

// ListBuildsRequest wraps builderv1.ListBuildsRequest
type ListBuildsRequest struct {
	TenantID  string
	State     []builderv1.BuildState
	PageSize  int32
	PageToken string
}

// ListBuildsResponse wraps builderv1.ListBuildsResponse
type ListBuildsResponse struct {
	Builds        []*builderv1.BuildJob
	NextPageToken string
	TotalCount    int32
}

// CancelBuildRequest wraps builderv1.CancelBuildRequest
type CancelBuildRequest struct {
	BuildID  string
	TenantID string
}

// CancelBuildResponse wraps builderv1.CancelBuildResponse
type CancelBuildResponse struct {
	Success bool
	State   builderv1.BuildState
}

// DeleteBuildRequest wraps builderv1.DeleteBuildRequest
type DeleteBuildRequest struct {
	BuildID  string
	TenantID string
	Force    bool
}

// DeleteBuildResponse wraps builderv1.DeleteBuildResponse
type DeleteBuildResponse struct {
	Success bool
}

// StreamBuildLogsRequest wraps builderv1.StreamBuildLogsRequest
type StreamBuildLogsRequest struct {
	BuildID  string
	TenantID string
	Follow   bool
}

// GetTenantQuotasRequest wraps builderv1.GetTenantQuotasRequest
type GetTenantQuotasRequest struct {
	TenantID string
}

// GetTenantQuotasResponse wraps builderv1.GetTenantQuotasResponse
type GetTenantQuotasResponse struct {
	Quotas     *builderv1.TenantResourceLimits
	Usage      *builderv1.TenantUsageStats
	Violations []*builderv1.QuotaViolation
}

// GetBuildStatsRequest wraps builderv1.GetBuildStatsRequest
type GetBuildStatsRequest struct {
	TenantID  string
	StartTime *timestamppb.Timestamp
	EndTime   *timestamppb.Timestamp
}

// GetBuildStatsResponse wraps builderv1.GetBuildStatsResponse
type GetBuildStatsResponse struct {
	Stats *builderv1.GetBuildStatsResponse
}
