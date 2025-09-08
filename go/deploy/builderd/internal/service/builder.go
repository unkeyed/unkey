package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/assetmanager"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/config"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/executor"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/observability"
	assetv1 "github.com/unkeyed/unkey/go/gen/proto/deploy/assetmanagerd/v1"
	builderv1 "github.com/unkeyed/unkey/go/gen/proto/deploy/builderd/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BuilderService implements the BuilderService ConnectRPC service
type BuilderService struct {
	logger       *slog.Logger
	buildMetrics *observability.BuildMetrics
	config       *config.Config
	executors    *executor.Registry
	assetClient  *assetmanager.Client

	builds      map[string]*builderv1.BuildJob
	buildsMutex sync.RWMutex

	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
	buildWg        sync.WaitGroup
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

	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	return &BuilderService{
		logger:         logger,
		buildMetrics:   buildMetrics,
		config:         cfg,
		executors:      executors,
		assetClient:    assetClient,
		builds:         make(map[string]*builderv1.BuildJob),
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}
}

// generateBuildID generates a unique build ID
func generateBuildID() string {
	return fmt.Sprintf("build-%d", time.Now().UnixNano())
}

// selectKernelForImage determines which bundled kernel to use for a given Docker image
func (s *BuilderService) selectKernelForImage(imageName string) (kernelPath, kernelName string, err error) {
	// AIDEV-NOTE: Maps base images to appropriate bundled kernels
	// This ensures kernel/rootfs compatibility

	imageLower := strings.ToLower(imageName)
	kernelsDir := "/opt/builderd/kernels"

	// Determine kernel based on base image
	var kernelFile string
	var kernelDisplayName string

	switch {
	case strings.Contains(imageLower, "alpine"):
		kernelFile = "alpine-kernel"
		kernelDisplayName = "Alpine Linux Kernel"
	case strings.Contains(imageLower, "ubuntu"):
		kernelFile = "ubuntu-kernel"
		kernelDisplayName = "Ubuntu Linux Kernel"
	case strings.Contains(imageLower, "debian"):
		kernelFile = "ubuntu-kernel" // Use Ubuntu kernel for Debian (compatible)
		kernelDisplayName = "Ubuntu Linux Kernel (Debian compatible)"
	default:
		// Default to Alpine kernel for unknown images
		kernelFile = "alpine-kernel"
		kernelDisplayName = "Alpine Linux Kernel (default)"
	}

	kernelPath = filepath.Join(kernelsDir, kernelFile)

	// Verify kernel exists
	if _, err := os.Stat(kernelPath); err != nil {
		return "", "", fmt.Errorf("bundled kernel not found: %s", kernelPath)
	}

	return kernelPath, kernelDisplayName, nil
}

// extractOSFromImage extracts the OS type from a Docker image name
func extractOSFromImage(imageName string) string {
	imageLower := strings.ToLower(imageName)

	switch {
	case strings.Contains(imageLower, "alpine"):
		return "alpine"
	case strings.Contains(imageLower, "ubuntu"):
		return "ubuntu"
	case strings.Contains(imageLower, "debian"):
		return "debian"
	default:
		return "unknown"
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

	// Create build job record
	buildJob := &builderv1.BuildJob{
		BuildId:   generateBuildID(),
		Config:    req.Msg.GetConfig(),
		State:     builderv1.BuildState_BUILD_STATE_BUILDING,
		CreatedAt: timestamppb.Now(),
		StartedAt: timestamppb.Now(),
	}

	// Store build job in memory
	s.buildsMutex.Lock()
	s.builds[buildJob.BuildId] = buildJob
	s.buildsMutex.Unlock()

	// Execute the build asynchronously
	// AIDEV-NOTE: Launch build in a goroutine to avoid blocking the RPC call
	// AIDEV-BUSINESS_RULE: Use shutdown-aware context to prevent races during service shutdown
	s.buildWg.Add(1)
	go func() {
		defer s.buildWg.Done()

		// AIDEV-NOTE: Use shutdown context to coordinate with service lifecycle
		// This prevents builds from running indefinitely during shutdown
		buildCtx := s.shutdownCtx

		// AIDEV-NOTE: Preserve tenant context for asset registration
		tenantID := req.Msg.GetConfig().GetTenant().GetTenantId()
		customerID := req.Msg.GetConfig().GetTenant().GetCustomerId()

		s.logger.InfoContext(buildCtx, "starting async build execution",
			slog.String("build_id", buildJob.BuildId),
			slog.String("tenant_id", req.Msg.GetConfig().GetTenant().GetTenantId()),
		)

		// AIDEV-NOTE: Check for shutdown signal before starting expensive build operation
		select {
		case <-buildCtx.Done():
			s.logger.InfoContext(buildCtx, "build cancelled due to shutdown",
				slog.String("build_id", buildJob.BuildId),
			)
			// Update build job with cancelled state
			s.buildsMutex.Lock()
			buildJob.State = builderv1.BuildState_BUILD_STATE_CANCELLED
			buildJob.CompletedAt = timestamppb.Now()
			buildJob.ErrorMessage = "Build cancelled due to service shutdown"
			s.buildsMutex.Unlock()
			return
		default:
		}

		buildResult, err := s.executors.ExecuteWithID(buildCtx, req.Msg, buildJob.BuildId)
		if err != nil {
			// Update build job with error state
			s.buildsMutex.Lock()
			buildJob.State = builderv1.BuildState_BUILD_STATE_FAILED
			buildJob.CompletedAt = timestamppb.Now()
			buildJob.ErrorMessage = err.Error()
			s.buildsMutex.Unlock()

			s.logger.ErrorContext(buildCtx, "build execution failed",
				slog.String("error", err.Error()),
				slog.String("build_id", buildJob.BuildId),
				slog.String("tenant_id", req.Msg.GetConfig().GetTenant().GetTenantId()),
			)
			return
		}

		s.logger.InfoContext(buildCtx, "build job completed successfully",
			slog.String("build_id", buildJob.BuildId),
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

		// Update build job with completion info
		s.buildsMutex.Lock()
		buildJob.State = buildState
		buildJob.CompletedAt = timestamppb.Now()
		buildJob.RootfsPath = buildResult.RootfsPath
		buildJob.ImageMetadata = buildResult.ImageMetadata
		// TODO: Add checksum and size when available
		s.buildsMutex.Unlock()

		// Register the build artifact with assetmanagerd if build succeeded
		// AIDEV-NOTE: This enables the built rootfs to be used for VM creation
		if buildState == builderv1.BuildState_BUILD_STATE_COMPLETED && s.assetClient.IsEnabled() {
			labels := map[string]string{
				"source_type": buildResult.SourceType,
				"tenant_id":   tenantID,   // AIDEV-NOTE: Include tenant info for asset registration
				"customer_id": customerID, // AIDEV-NOTE: Include customer info for asset registration
			}

			// Add docker image label if it's a Docker source
			// AIDEV-NOTE: Must use "docker_image" label to match metald's query expectations
			if dockerSource := req.Msg.GetConfig().GetSource().GetDockerImage(); dockerSource != nil {
				labels["docker_image"] = dockerSource.GetImageUri()
			}

			// Determine asset type based on target
			assetType := assetv1.AssetType_ASSET_TYPE_ROOTFS
			if req.Msg.GetConfig().GetTarget().GetMicrovmRootfs() != nil {
				assetType = assetv1.AssetType_ASSET_TYPE_ROOTFS
			}

			// Use suggested asset ID if provided in the build config
			suggestedAssetID := req.Msg.GetConfig().GetSuggestedAssetId()

			s.logger.InfoContext(buildCtx, "registering build artifact with asset ID",
				slog.String("suggested_asset_id", suggestedAssetID),
				slog.String("build_id", buildJob.BuildId),
				slog.Any("labels", labels),
			)

			// First upload the rootfs
			assetID, err := s.assetClient.RegisterBuildArtifactWithID(buildCtx, buildJob.BuildId, buildResult.RootfsPath, assetType, labels, suggestedAssetID)
			if err != nil {
				// Log error but don't fail the build
				s.logger.ErrorContext(buildCtx, "failed to register rootfs with assetmanagerd",
					slog.String("error", err.Error()),
					slog.String("build_id", buildJob.BuildId),
					slog.String("rootfs_path", buildResult.RootfsPath),
				)
			} else {
				s.logger.InfoContext(buildCtx, "registered rootfs with assetmanagerd",
					slog.String("asset_id", assetID),
					slog.String("build_id", buildJob.BuildId),
				)
			}

			// Extract the source image from build config
			var sourceImage string
			if buildJob.Config != nil && buildJob.Config.Source != nil {
				if dockerSource := buildJob.Config.Source.GetDockerImage(); dockerSource != nil {
					sourceImage = dockerSource.ImageUri
				}
			}

			if sourceImage == "" {
				s.logger.WarnContext(buildCtx, "no Docker image source found, skipping kernel upload")
			} else {
				// Now upload the appropriate kernel
				kernelPath, kernelName, err := s.selectKernelForImage(sourceImage)
				if err != nil {
					s.logger.ErrorContext(buildCtx, "failed to select kernel for image",
						slog.String("error", err.Error()),
						slog.String("image", sourceImage),
					)
				} else {
					// Create kernel labels
					var tenantID, customerID string
					if buildJob.Config != nil && buildJob.Config.Tenant != nil {
						tenantID = buildJob.Config.Tenant.TenantId
						customerID = buildJob.Config.Tenant.CustomerId
					}

					kernelLabels := map[string]string{
						"kernel_type":   "bundled",
						"compatible_os": extractOSFromImage(sourceImage),
						"build_id":      buildJob.BuildId,
						"created_by":    "builderd",
						"tenant_id":     tenantID,
						"customer_id":   customerID,
					}

					kernelAssetID, err := s.assetClient.RegisterBuildArtifactWithID(
						buildCtx,
						buildJob.BuildId+"-kernel",
						kernelPath,
						assetv1.AssetType_ASSET_TYPE_KERNEL,
						kernelLabels,
						"", // Let assetmanagerd generate kernel asset ID
					)
					if err != nil {
						s.logger.ErrorContext(buildCtx, "failed to register kernel with assetmanagerd",
							slog.String("error", err.Error()),
							slog.String("kernel_path", kernelPath),
							slog.String("kernel_name", kernelName),
						)
					} else {
						s.logger.InfoContext(buildCtx, "registered kernel with assetmanagerd",
							slog.String("kernel_asset_id", kernelAssetID),
							slog.String("kernel_name", kernelName),
							slog.String("build_id", buildJob.BuildId),
						)
					}
				}
			}
		}
	}()

	// Return immediately with the build ID and "building" state
	resp := &builderv1.CreateBuildResponse{
		BuildId:    buildJob.BuildId,
		State:      builderv1.BuildState_BUILD_STATE_BUILDING,
		CreatedAt:  timestamppb.Now(),
		RootfsPath: "", // Not available yet
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

	// Retrieve build from memory storage
	s.buildsMutex.RLock()
	build, exists := s.builds[req.Msg.GetBuildId()]
	s.buildsMutex.RUnlock()

	if !exists {
		return nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("build not found: %s", req.Msg.GetBuildId()))
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
		s.buildMetrics.RecordBuildCancellation(ctx, "unknown", "unknown")
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

// Shutdown gracefully shuts down the BuilderService
// AIDEV-NOTE: This method coordinates shutdown of all running builds to prevent races
func (s *BuilderService) Shutdown(ctx context.Context) error {
	s.logger.InfoContext(ctx, "starting BuilderService shutdown")

	// Cancel all running builds
	s.shutdownCancel()

	// Wait for all builds to complete with timeout
	done := make(chan struct{})
	go func() {
		s.buildWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.InfoContext(ctx, "all builds completed during shutdown")
		return nil
	case <-ctx.Done():
		s.logger.WarnContext(ctx, "shutdown timeout reached, some builds may have been terminated")
		return ctx.Err()
	}
}
