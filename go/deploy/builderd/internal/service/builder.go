package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	assetv1 "github.com/unkeyed/unkey/go/deploy/assetmanagerd/gen/asset/v1"
	builderv1 "github.com/unkeyed/unkey/go/deploy/builderd/gen/builder/v1"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/assetmanager"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/config"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/executor"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/observability"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BuilderService implements the BuilderService ConnectRPC service
type BuilderService struct {
	logger       *slog.Logger
	buildMetrics *observability.BuildMetrics
	config       *config.Config
	executors    *executor.Registry
	assetClient  *assetmanager.Client

	// TODO: Add these when implemented
	// db           *database.DB
	// storage      storage.Backend
	// docker       *docker.Client
	// tenantMgr    *tenant.Manager
}

// NewBuilderService creates a new BuilderService instance
func NewBuilderService(
	logger *slog.Logger,
	buildMetrics *observability.BuildMetrics,
	cfg *config.Config,
	assetClient *assetmanager.Client,
) *BuilderService {
	// Create executor registry
	executors := executor.NewRegistry(logger, cfg, buildMetrics)

	return &BuilderService{
		logger:       logger,
		buildMetrics: buildMetrics,
		config:       cfg,
		executors:    executors,
		assetClient:  assetClient,
	}
}

// CreateBuild creates a new build job
func (s *BuilderService) CreateBuild(
	ctx context.Context,
	req *connect.Request[builderv1.CreateBuildRequest],
) (*connect.Response[builderv1.CreateBuildResponse], error) {
	// Extract tenant info safely
	var tenantID, customerID string
	if req.Msg != nil && req.Msg.GetConfig() != nil && req.Msg.GetConfig().GetTenant() != nil {
		tenantID = req.Msg.GetConfig().GetTenant().GetTenantId()
		customerID = req.Msg.GetConfig().GetTenant().GetCustomerId()
	}

	s.logger.InfoContext(ctx, "create build request received",
		slog.String("tenant_id", tenantID),
		slog.String("customer_id", customerID),
	)

	// Validate build configuration first to prevent nil pointer dereference
	if err := s.validateBuildConfig(req.Msg.GetConfig()); err != nil {
		s.logger.WarnContext(ctx, "invalid build configuration",
			slog.String("error", err.Error()),
			slog.String("tenant_id", tenantID),
		)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// TODO: Check tenant quotas
	// TODO: Store build job in database

	// Execute the build immediately (for now)
	// In production, this would be queued and executed asynchronously
	buildResult, err := s.executors.Execute(ctx, req.Msg)
	if err != nil {
		s.logger.ErrorContext(ctx, "build execution failed",
			slog.String("error", err.Error()),
			slog.String("tenant_id", req.Msg.GetConfig().GetTenant().GetTenantId()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build execution failed: %w", err))
	}

	buildID := buildResult.BuildID

	s.logger.InfoContext(ctx, "build job completed successfully",
		slog.String("build_id", buildID),
		slog.String("tenant_id", req.Msg.GetConfig().GetTenant().GetTenantId()),
		slog.String("source_type", buildResult.SourceType),
		slog.String("rootfs_path", buildResult.RootfsPath),
		slog.Duration("duration", buildResult.EndTime.Sub(buildResult.StartTime)),
	)

	// Build state - since we executed immediately, it's either completed or failed
	buildState := builderv1.BuildState_BUILD_STATE_COMPLETED
	if buildResult.Status == "failed" {
		buildState = builderv1.BuildState_BUILD_STATE_FAILED
	}

	// Register the build artifact with assetmanagerd if build succeeded
	// AIDEV-NOTE: This enables the built rootfs to be used for VM creation
	var assetID string
	if buildState == builderv1.BuildState_BUILD_STATE_COMPLETED && s.assetClient.IsEnabled() {
		labels := map[string]string{
			"tenant_id":    tenantID,
			"customer_id":  customerID,
			"source_type":  buildResult.SourceType,
		}
		
		// Add source image if it's a Docker source
		if dockerSource := req.Msg.GetConfig().GetSource().GetDockerImage(); dockerSource != nil {
			labels["source_image"] = dockerSource.GetImageUri()
		}

		// Determine asset type based on target
		assetType := assetv1.AssetType_ASSET_TYPE_ROOTFS
		if req.Msg.GetConfig().GetTarget().GetMicrovmRootfs() != nil {
			assetType = assetv1.AssetType_ASSET_TYPE_ROOTFS
		}

		var err error
		assetID, err = s.assetClient.RegisterBuildArtifact(ctx, buildID, buildResult.RootfsPath, assetType, labels)
		if err != nil {
			// Log error but don't fail the build response
			s.logger.ErrorContext(ctx, "failed to register build artifact with assetmanagerd",
				slog.String("error", err.Error()),
				slog.String("build_id", buildID),
				slog.String("rootfs_path", buildResult.RootfsPath),
			)
		} else {
			s.logger.InfoContext(ctx, "registered build artifact with assetmanagerd",
				slog.String("asset_id", assetID),
				slog.String("build_id", buildID),
			)
		}
	}

	resp := &builderv1.CreateBuildResponse{
		BuildId:    buildID,
		State:      buildState,
		CreatedAt:  timestamppb.New(buildResult.StartTime),
		RootfsPath: buildResult.RootfsPath,
		// AIDEV-TODO: Add AssetId field to CreateBuildResponse proto to return registered asset ID
	}

	return connect.NewResponse(resp), nil
}

// GetBuild retrieves build status and information
func (s *BuilderService) GetBuild(
	ctx context.Context,
	req *connect.Request[builderv1.GetBuildRequest],
) (*connect.Response[builderv1.GetBuildResponse], error) {
	s.logger.InfoContext(ctx, "get build request received",
		slog.String("build_id", req.Msg.GetBuildId()),
		slog.String("tenant_id", req.Msg.GetTenantId()),
	)

	// TODO: Validate tenant has access to this build
	// TODO: Retrieve build from database

	// For now, return a placeholder response
	//nolint:exhaustruct // Placeholder response with only required fields
	build := &builderv1.BuildJob{
		BuildId: req.Msg.GetBuildId(),
		//nolint:exhaustruct // Config fields will be populated later
		Config: &builderv1.BuildConfig{
			//nolint:exhaustruct // Tenant context fields will be populated from actual data
			Tenant: &builderv1.TenantContext{
				TenantId:   req.Msg.GetTenantId(),
				CustomerId: "placeholder",
				Tier:       builderv1.TenantTier_TENANT_TIER_FREE,
			},
		},
		State:           builderv1.BuildState_BUILD_STATE_PENDING,
		CreatedAt:       timestamppb.Now(),
		StartedAt:       nil,
		CompletedAt:     nil,
		ProgressPercent: 0,
		CurrentStep:     "queued",
	}

	resp := &builderv1.GetBuildResponse{
		Build: build,
	}

	return connect.NewResponse(resp), nil
}

// ListBuilds lists builds for a tenant
func (s *BuilderService) ListBuilds(
	ctx context.Context,
	req *connect.Request[builderv1.ListBuildsRequest],
) (*connect.Response[builderv1.ListBuildsResponse], error) {
	s.logger.InfoContext(ctx, "list builds request received",
		slog.String("tenant_id", req.Msg.GetTenantId()),
		slog.Int("page_size", int(req.Msg.GetPageSize())),
	)

	// TODO: Retrieve builds from database with tenant filtering
	// TODO: Apply state filters
	// TODO: Implement pagination

	// For now, return empty list
	resp := &builderv1.ListBuildsResponse{
		Builds:        []*builderv1.BuildJob{},
		NextPageToken: "",
		TotalCount:    0,
	}

	return connect.NewResponse(resp), nil
}

// CancelBuild cancels a running build
func (s *BuilderService) CancelBuild(
	ctx context.Context,
	req *connect.Request[builderv1.CancelBuildRequest],
) (*connect.Response[builderv1.CancelBuildResponse], error) {
	s.logger.InfoContext(ctx, "cancel build request received",
		slog.String("build_id", req.Msg.GetBuildId()),
		slog.String("tenant_id", req.Msg.GetTenantId()),
	)

	// TODO: Validate tenant has access to this build
	// TODO: Cancel the running build process
	// TODO: Update build state in database

	// Record cancellation metrics
	if s.buildMetrics != nil {
		s.buildMetrics.RecordBuildCancellation(ctx, "unknown", "unknown", "unknown")
	}

	resp := &builderv1.CancelBuildResponse{
		Success: true,
		State:   builderv1.BuildState_BUILD_STATE_CANCELLED,
	}

	return connect.NewResponse(resp), nil
}

// DeleteBuild deletes a build and its artifacts
func (s *BuilderService) DeleteBuild(
	ctx context.Context,
	req *connect.Request[builderv1.DeleteBuildRequest],
) (*connect.Response[builderv1.DeleteBuildResponse], error) {
	s.logger.InfoContext(ctx, "delete build request received",
		slog.String("build_id", req.Msg.GetBuildId()),
		slog.String("tenant_id", req.Msg.GetTenantId()),
		slog.Bool("force", req.Msg.GetForce()),
	)

	// TODO: Validate tenant has access to this build
	// TODO: Check if build is running (and force flag)
	// TODO: Delete build from database
	// TODO: Delete build artifacts from storage

	resp := &builderv1.DeleteBuildResponse{
		Success: true,
	}

	return connect.NewResponse(resp), nil
}

// StreamBuildLogs streams build logs in real-time
func (s *BuilderService) StreamBuildLogs(
	ctx context.Context,
	req *connect.Request[builderv1.StreamBuildLogsRequest],
	stream *connect.ServerStream[builderv1.StreamBuildLogsResponse],
) error {
	s.logger.InfoContext(ctx, "stream build logs request received",
		slog.String("build_id", req.Msg.GetBuildId()),
		slog.String("tenant_id", req.Msg.GetTenantId()),
		slog.Bool("follow", req.Msg.GetFollow()),
	)

	// TODO: Validate tenant has access to this build
	// TODO: Stream existing logs
	// TODO: If follow=true, stream new logs as they arrive

	// For now, send a placeholder log entry
	logEntry := &builderv1.StreamBuildLogsResponse{
		Timestamp: timestamppb.New(time.Now()),
		Level:     "info",
		Message:   "Build logs streaming started",
		Component: "builder",
		Metadata:  make(map[string]string),
	}

	if err := stream.Send(logEntry); err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	return nil
}

// GetTenantQuotas retrieves tenant quota information
func (s *BuilderService) GetTenantQuotas(
	ctx context.Context,
	req *connect.Request[builderv1.GetTenantQuotasRequest],
) (*connect.Response[builderv1.GetTenantQuotasResponse], error) {
	s.logger.InfoContext(ctx, "get tenant quotas request received",
		slog.String("tenant_id", req.Msg.GetTenantId()),
	)

	// TODO: Retrieve tenant configuration
	// TODO: Calculate current usage
	// TODO: Check for quota violations

	// Return default quotas for now
	resp := &builderv1.GetTenantQuotasResponse{
		CurrentLimits: &builderv1.TenantResourceLimits{ //nolint:exhaustruct // AllowedRegistries, AllowedGitHosts, AllowPrivilegedBuilds, BlockedCommands, SandboxLevel are tenant-specific overrides not set in defaults
			MaxMemoryBytes:       s.config.Tenant.DefaultResourceLimits.MaxMemoryBytes,
			MaxCpuCores:          s.config.Tenant.DefaultResourceLimits.MaxCPUCores,
			MaxDiskBytes:         s.config.Tenant.DefaultResourceLimits.MaxDiskBytes,
			TimeoutSeconds:       s.config.Tenant.DefaultResourceLimits.TimeoutSeconds,
			MaxConcurrentBuilds:  s.config.Tenant.DefaultResourceLimits.MaxConcurrentBuilds,
			MaxDailyBuilds:       s.config.Tenant.DefaultResourceLimits.MaxDailyBuilds,
			MaxStorageBytes:      s.config.Tenant.DefaultResourceLimits.MaxStorageBytes,
			MaxBuildTimeMinutes:  s.config.Tenant.DefaultResourceLimits.MaxBuildTimeMinutes,
			AllowExternalNetwork: true,
		},
		CurrentUsage: &builderv1.TenantUsageStats{
			ActiveBuilds:         0,
			DailyBuildsUsed:      0,
			StorageBytesUsed:     0,
			ComputeMinutesUsed:   0,
			BuildsQueued:         0,
			BuildsCompletedToday: 0,
			BuildsFailedToday:    0,
		},
		Violations: []*builderv1.QuotaViolation{},
	}

	return connect.NewResponse(resp), nil
}

// GetBuildStats retrieves build statistics
func (s *BuilderService) GetBuildStats(
	ctx context.Context,
	req *connect.Request[builderv1.GetBuildStatsRequest],
) (*connect.Response[builderv1.GetBuildStatsResponse], error) {
	s.logger.InfoContext(ctx, "get build stats request received",
		slog.String("tenant_id", req.Msg.GetTenantId()),
	)

	// TODO: Calculate actual statistics from database

	resp := &builderv1.GetBuildStatsResponse{
		TotalBuilds:         0,
		SuccessfulBuilds:    0,
		FailedBuilds:        0,
		AvgBuildTimeMs:      0,
		TotalStorageBytes:   0,
		TotalComputeMinutes: 0,
		RecentBuilds:        []*builderv1.BuildJob{},
	}

	return connect.NewResponse(resp), nil
}

// validateBuildConfig validates the build configuration
func (s *BuilderService) validateBuildConfig(config *builderv1.BuildConfig) error {
	if config == nil {
		return fmt.Errorf("build config is required")
	}

	if config.GetTenant() == nil {
		return fmt.Errorf("tenant context is required")
	}

	if config.GetTenant().GetTenantId() == "" {
		return fmt.Errorf("tenant ID is required")
	}

	if config.GetSource() == nil {
		return fmt.Errorf("build source is required")
	}

	if config.GetTarget() == nil {
		return fmt.Errorf("build target is required")
	}

	if config.GetStrategy() == nil {
		return fmt.Errorf("build strategy is required")
	}

	// Validate source-specific requirements
	switch source := config.GetSource().GetSourceType().(type) {
	case *builderv1.BuildSource_DockerImage:
		if source.DockerImage.GetImageUri() == "" {
			return fmt.Errorf("docker image URI is required")
		}
	case *builderv1.BuildSource_GitRepository:
		if source.GitRepository.GetRepositoryUrl() == "" {
			return fmt.Errorf("git repository URL is required")
		}
	case *builderv1.BuildSource_Archive:
		if source.Archive.GetArchiveUrl() == "" {
			return fmt.Errorf("archive URL is required")
		}
	default:
		return fmt.Errorf("unsupported source type")
	}

	return nil
}
