package executor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	builderv1 "github.com/unkeyed/unkey/go/deploy/builderd/gen/proto/builder/v1"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/config"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/observability"
)

// DockerExecutor handles Docker image extraction to rootfs
type DockerExecutor struct {
	logger       *slog.Logger
	config       *config.Config
	buildMetrics *observability.BuildMetrics
}

// Ensure DockerExecutor implements Executor interface
var _ Executor = (*DockerExecutor)(nil)

// NewDockerExecutor creates a new Docker executor
func NewDockerExecutor(logger *slog.Logger, cfg *config.Config, metrics *observability.BuildMetrics) *DockerExecutor {
	return &DockerExecutor{
		logger:       logger,
		config:       cfg,
		buildMetrics: metrics,
	}
}

// ExtractDockerImage pulls a Docker image and extracts it to a rootfs directory
func (d *DockerExecutor) ExtractDockerImage(ctx context.Context, request *builderv1.CreateBuildRequest) (*BuildResult, error) {
	start := time.Now()

	// Get tenant context for logging and metrics
	tenantID := "unknown"
	if auth, ok := observability.TenantFromContext(ctx); ok {
		tenantID = auth.TenantID
	}

	logger := d.logger.With(
		slog.String("tenant_id", tenantID),
		slog.String("image_uri", request.Config.Source.GetDockerImage().ImageUri),
	)

	logger.Info("starting Docker image extraction")

	// Record build start metrics
	if d.buildMetrics != nil {
		d.buildMetrics.RecordBuildStart(ctx, "docker", "docker", tenantID)
	}

	defer func() {
		duration := time.Since(start)
		logger.Info("Docker image extraction completed", slog.Duration("duration", duration))
	}()

	dockerSource := request.Config.Source.GetDockerImage()
	if dockerSource == nil {
		return nil, fmt.Errorf("docker image source is required")
	}

	// Create build workspace
	buildID := generateBuildID()
	workspaceDir := filepath.Join(d.config.Builder.WorkspaceDir, buildID)
	rootfsDir := filepath.Join(d.config.Builder.RootfsOutputDir, buildID)

	logger = logger.With(
		slog.String("build_id", buildID),
		slog.String("workspace_dir", workspaceDir),
		slog.String("rootfs_dir", rootfsDir),
	)

	// Create directories
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		logger.Error("failed to create workspace directory", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}

	if err := os.MkdirAll(rootfsDir, 0755); err != nil {
		logger.Error("failed to create rootfs directory", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to create rootfs directory: %w", err)
	}

	// Use the full image URI directly
	fullImageName := dockerSource.ImageUri
	if fullImageName == "" {
		return nil, fmt.Errorf("docker image URI is required")
	}

	logger = logger.With(slog.String("full_image_name", fullImageName))

	// Step 1: Pull the Docker image
	if err := d.pullDockerImage(ctx, logger, fullImageName); err != nil {
		logger.Error("failed to pull Docker image",
			slog.String("error", err.Error()),
			slog.String("image", fullImageName),
		)
		if d.buildMetrics != nil {
			d.buildMetrics.RecordBuildComplete(ctx, "docker", "docker", tenantID, time.Since(start), false)
		}
		return nil, fmt.Errorf("failed to pull Docker image: %w", err)
	}

	// Step 2: Create container from image (without running)
	containerID, err := d.createContainer(ctx, logger, fullImageName)
	if err != nil {
		logger.Error("failed to create container",
			slog.String("error", err.Error()),
			slog.String("image", fullImageName),
		)
		if d.buildMetrics != nil {
			d.buildMetrics.RecordBuildComplete(ctx, "docker", "docker", tenantID, time.Since(start), false)
		}
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Ensure cleanup of container
	defer func() {
		if cleanupErr := d.removeContainer(ctx, logger, containerID); cleanupErr != nil {
			logger.Warn("failed to cleanup container", slog.String("error", cleanupErr.Error()))
		}
	}()

	// Step 3: Extract filesystem from container
	if err := d.extractFilesystem(ctx, logger, containerID, rootfsDir); err != nil {
		logger.Error("failed to extract filesystem",
			slog.String("error", err.Error()),
			slog.String("container_id", containerID),
			slog.String("rootfs_dir", rootfsDir),
		)
		if d.buildMetrics != nil {
			d.buildMetrics.RecordBuildComplete(ctx, "docker", "docker", tenantID, time.Since(start), false)
		}
		return nil, fmt.Errorf("failed to extract filesystem: %w", err)
	}

	// Step 4: Optimize rootfs (remove unnecessary files, etc.)
	if err := d.optimizeRootfs(ctx, logger, rootfsDir); err != nil {
		logger.Warn("failed to optimize rootfs", slog.String("error", err.Error()))
		// Don't fail the build for optimization errors
	}

	// Create build result
	result := &BuildResult{
		BuildID:      buildID,
		SourceType:   "docker",
		SourceImage:  fullImageName,
		RootfsPath:   rootfsDir,
		WorkspaceDir: workspaceDir,
		TenantID:     tenantID,
		StartTime:    start,
		EndTime:      time.Now(),
		Status:       "completed",
	}

	// Record successful build
	if d.buildMetrics != nil {
		d.buildMetrics.RecordBuildComplete(ctx, "docker", "docker", tenantID, time.Since(start), true)
	}

	logger.Info("Docker image extraction successful",
		slog.String("rootfs_path", rootfsDir),
		slog.Duration("total_duration", time.Since(start)),
	)

	return result, nil
}

// pullDockerImage pulls the specified Docker image
func (d *DockerExecutor) pullDockerImage(ctx context.Context, logger *slog.Logger, imageName string) error {
	// AIDEV-NOTE: Comprehensive observability for Docker pull step
	tracer := otel.Tracer("builderd/docker")
	stepStart := time.Now()

	// Start OpenTelemetry span for this build step
	ctx, span := tracer.Start(ctx, "docker.pull",
		trace.WithAttributes(
			attribute.String("step", "pull"),
			attribute.String("image", imageName),
			attribute.String("source_type", "docker"),
		),
	)
	defer span.End()

	// Record step start metrics
	if d.buildMetrics != nil {
		d.buildMetrics.RecordBuildStepStart(ctx, "pull", "docker")
	}

	logger.Info("pulling Docker image", slog.String("image", imageName))

	// Create context with timeout for docker pull
	pullCtx, cancel := context.WithTimeout(ctx, d.config.Docker.PullTimeout)
	defer cancel()

	cmd := exec.CommandContext(pullCtx, "docker", "pull", imageName)

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()

	// Record step completion
	stepDuration := time.Since(stepStart)
	success := err == nil

	if d.buildMetrics != nil {
		d.buildMetrics.RecordBuildStepComplete(ctx, "pull", "docker", stepDuration, success)
		if success {
			d.buildMetrics.RecordPullDuration(ctx, "docker", stepDuration)
		}
	}

	if err != nil {
		span.SetAttributes(
			attribute.String("error", err.Error()),
			attribute.String("output", string(output)),
		)
		logger.Error("docker pull failed",
			slog.String("error", err.Error()),
			slog.String("output", string(output)),
			slog.Duration("duration", stepDuration),
		)
		return fmt.Errorf("docker pull failed: %w", err)
	}

	span.SetAttributes(attribute.String("status", "success"))
	logger.Info("docker pull completed",
		slog.String("image", imageName),
		slog.Duration("duration", stepDuration),
	)
	return nil
}

// createContainer creates a container from the image without running it
func (d *DockerExecutor) createContainer(ctx context.Context, logger *slog.Logger, imageName string) (string, error) {
	// AIDEV-NOTE: Comprehensive observability for Docker create step
	tracer := otel.Tracer("builderd/docker")
	stepStart := time.Now()

	// Start OpenTelemetry span for this build step
	ctx, span := tracer.Start(ctx, "docker.create",
		trace.WithAttributes(
			attribute.String("step", "create"),
			attribute.String("image", imageName),
			attribute.String("source_type", "docker"),
		),
	)
	defer span.End()

	// Record step start metrics
	if d.buildMetrics != nil {
		d.buildMetrics.RecordBuildStepStart(ctx, "create", "docker")
	}

	logger.Info("creating container from image", slog.String("image", imageName))

	cmd := exec.CommandContext(ctx, "docker", "create", imageName)
	output, err := cmd.Output()

	// Record step completion
	stepDuration := time.Since(stepStart)
	success := err == nil

	if d.buildMetrics != nil {
		d.buildMetrics.RecordBuildStepComplete(ctx, "create", "docker", stepDuration, success)
	}

	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		logger.Error("docker create failed",
			slog.String("error", err.Error()),
			slog.String("image", imageName),
			slog.Duration("duration", stepDuration),
		)
		return "", fmt.Errorf("docker create failed: %w", err)
	}

	containerID := strings.TrimSpace(string(output))
	span.SetAttributes(
		attribute.String("status", "success"),
		attribute.String("container_id", containerID),
	)
	logger.Info("container created",
		slog.String("container_id", containerID),
		slog.Duration("duration", stepDuration),
	)

	return containerID, nil
}

// extractFilesystem extracts the filesystem from the container to the rootfs directory
func (d *DockerExecutor) extractFilesystem(ctx context.Context, logger *slog.Logger, containerID, rootfsDir string) error {
	// AIDEV-NOTE: Comprehensive observability for filesystem extraction step
	tracer := otel.Tracer("builderd/docker")
	stepStart := time.Now()

	// Start OpenTelemetry span for this build step
	ctx, span := tracer.Start(ctx, "docker.extract",
		trace.WithAttributes(
			attribute.String("step", "extract"),
			attribute.String("container_id", containerID),
			attribute.String("rootfs_dir", rootfsDir),
			attribute.String("source_type", "docker"),
		),
	)
	defer span.End()

	// Record step start metrics
	if d.buildMetrics != nil {
		d.buildMetrics.RecordBuildStepStart(ctx, "extract", "docker")
	}

	logger.Info("extracting filesystem from container",
		slog.String("container_id", containerID),
		slog.String("rootfs_dir", rootfsDir),
	)

	// Use docker export to get the full filesystem as a tar stream
	cmd := exec.CommandContext(ctx, "docker", "export", containerID)

	// Create tar extraction command
	tarCmd := exec.CommandContext(ctx, "tar", "-xf", "-", "-C", rootfsDir)

	// Connect docker export output to tar input
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		stepDuration := time.Since(stepStart)
		if d.buildMetrics != nil {
			d.buildMetrics.RecordBuildStepComplete(ctx, "extract", "docker", stepDuration, false)
		}
		span.SetAttributes(attribute.String("error", err.Error()))
		logger.Error("failed to create pipe",
			slog.String("error", err.Error()),
			slog.Duration("duration", stepDuration),
		)
		return fmt.Errorf("failed to create pipe: %w", err)
	}

	tarCmd.Stdin = pipe

	// Start both commands
	if err := cmd.Start(); err != nil {
		stepDuration := time.Since(stepStart)
		if d.buildMetrics != nil {
			d.buildMetrics.RecordBuildStepComplete(ctx, "extract", "docker", stepDuration, false)
		}
		span.SetAttributes(attribute.String("error", err.Error()))
		logger.Error("failed to start docker export",
			slog.String("error", err.Error()),
			slog.Duration("duration", stepDuration),
		)
		return fmt.Errorf("failed to start docker export: %w", err)
	}

	if err := tarCmd.Start(); err != nil {
		cmd.Process.Kill()
		stepDuration := time.Since(stepStart)
		if d.buildMetrics != nil {
			d.buildMetrics.RecordBuildStepComplete(ctx, "extract", "docker", stepDuration, false)
		}
		span.SetAttributes(attribute.String("error", err.Error()))
		logger.Error("failed to start tar extraction",
			slog.String("error", err.Error()),
			slog.Duration("duration", stepDuration),
		)
		return fmt.Errorf("failed to start tar extraction: %w", err)
	}

	// Wait for docker export to complete
	if err := cmd.Wait(); err != nil {
		tarCmd.Process.Kill()
		stepDuration := time.Since(stepStart)
		if d.buildMetrics != nil {
			d.buildMetrics.RecordBuildStepComplete(ctx, "extract", "docker", stepDuration, false)
		}
		span.SetAttributes(attribute.String("error", err.Error()))
		logger.Error("docker export failed",
			slog.String("error", err.Error()),
			slog.Duration("duration", stepDuration),
		)
		return fmt.Errorf("docker export failed: %w", err)
	}

	// Close the pipe and wait for tar to complete
	pipe.Close()
	if err := tarCmd.Wait(); err != nil {
		stepDuration := time.Since(stepStart)
		if d.buildMetrics != nil {
			d.buildMetrics.RecordBuildStepComplete(ctx, "extract", "docker", stepDuration, false)
		}
		span.SetAttributes(attribute.String("error", err.Error()))
		logger.Error("tar extraction failed",
			slog.String("error", err.Error()),
			slog.Duration("duration", stepDuration),
		)
		return fmt.Errorf("tar extraction failed: %w", err)
	}

	// Record successful completion
	stepDuration := time.Since(stepStart)
	if d.buildMetrics != nil {
		d.buildMetrics.RecordBuildStepComplete(ctx, "extract", "docker", stepDuration, true)
		d.buildMetrics.RecordExtractDuration(ctx, "docker", stepDuration)
	}

	span.SetAttributes(attribute.String("status", "success"))
	logger.Info("filesystem extraction completed",
		slog.Duration("duration", stepDuration),
	)
	return nil
}

// removeContainer removes the temporary container
func (d *DockerExecutor) removeContainer(ctx context.Context, logger *slog.Logger, containerID string) error {
	logger.Debug("removing container", slog.String("container_id", containerID))

	cmd := exec.CommandContext(ctx, "docker", "rm", containerID)
	if err := cmd.Run(); err != nil {
		logger.Error("failed to remove container",
			slog.String("error", err.Error()),
			slog.String("container_id", containerID),
		)
		return fmt.Errorf("failed to remove container: %w", err)
	}

	logger.Debug("container removed", slog.String("container_id", containerID))
	return nil
}

// optimizeRootfs removes unnecessary files and optimizes the rootfs
func (d *DockerExecutor) optimizeRootfs(ctx context.Context, logger *slog.Logger, rootfsDir string) error {
	// AIDEV-NOTE: Comprehensive observability for rootfs optimization step
	tracer := otel.Tracer("builderd/docker")
	stepStart := time.Now()

	// Start OpenTelemetry span for this build step
	ctx, span := tracer.Start(ctx, "docker.optimize",
		trace.WithAttributes(
			attribute.String("step", "optimize"),
			attribute.String("rootfs_dir", rootfsDir),
			attribute.String("source_type", "docker"),
		),
	)
	defer span.End()

	// Record step start metrics
	if d.buildMetrics != nil {
		d.buildMetrics.RecordBuildStepStart(ctx, "optimize", "docker")
	}

	logger.Info("optimizing rootfs", slog.String("rootfs_dir", rootfsDir))

	// List of directories/files to remove for optimization
	removePatterns := []string{
		"var/cache/*",
		"var/lib/apt/lists/*",
		"tmp/*",
		"var/tmp/*",
		"usr/share/doc/*",
		"usr/share/man/*",
		"usr/share/info/*",
		"var/log/*",
	}

	var lastError error
	removedPatterns := 0

	for _, pattern := range removePatterns {
		fullPattern := filepath.Join(rootfsDir, pattern)

		// Use rm command to remove files matching pattern
		cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("rm -rf %s", fullPattern))
		if err := cmd.Run(); err != nil {
			logger.Debug("failed to remove pattern",
				slog.String("pattern", pattern),
				slog.String("error", err.Error()),
			)
			lastError = err
			// Continue with other patterns even if one fails
		} else {
			removedPatterns++
		}
	}

	// Get rootfs size after optimization
	var finalSize int64
	if size, err := d.getRootfsSize(rootfsDir); err == nil {
		finalSize = size
		if d.buildMetrics != nil {
			d.buildMetrics.RecordRootfsSize(ctx, size)
		}
	} else {
		logger.Warn("failed to calculate rootfs size", slog.String("error", err.Error()))
	}

	// Record step completion
	stepDuration := time.Since(stepStart)
	success := lastError == nil

	if d.buildMetrics != nil {
		d.buildMetrics.RecordBuildStepComplete(ctx, "optimize", "docker", stepDuration, success)
		d.buildMetrics.RecordOptimizeDuration(ctx, stepDuration)
	}

	if lastError != nil {
		span.SetAttributes(
			attribute.String("error", lastError.Error()),
			attribute.Int("patterns_removed", removedPatterns),
			attribute.Int("total_patterns", len(removePatterns)),
		)
		logger.Warn("rootfs optimization completed with errors",
			slog.String("error", lastError.Error()),
			slog.Int("patterns_removed", removedPatterns),
			slog.Int("total_patterns", len(removePatterns)),
			slog.Int64("size_bytes", finalSize),
			slog.Duration("duration", stepDuration),
		)
	} else {
		span.SetAttributes(
			attribute.String("status", "success"),
			attribute.Int("patterns_removed", removedPatterns),
		)
		logger.Info("rootfs optimization completed",
			slog.Int64("size_bytes", finalSize),
			slog.Int("patterns_removed", removedPatterns),
			slog.Duration("duration", stepDuration),
		)
	}

	return lastError
}

// getRootfsSize calculates the total size of the rootfs directory
func (d *DockerExecutor) getRootfsSize(rootfsDir string) (int64, error) {
	var totalSize int64

	err := filepath.Walk(rootfsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			d.logger.Debug("error walking rootfs path",
				slog.String("path", path),
				slog.String("error", err.Error()),
			)
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		d.logger.Error("failed to calculate rootfs size",
			slog.String("error", err.Error()),
			slog.String("rootfs_dir", rootfsDir),
		)
	}

	return totalSize, err
}

// Execute implements the Executor interface
func (d *DockerExecutor) Execute(ctx context.Context, request *builderv1.CreateBuildRequest) (*BuildResult, error) {
	return d.ExtractDockerImage(ctx, request)
}

// GetSupportedSources implements the Executor interface
func (d *DockerExecutor) GetSupportedSources() []string {
	return []string{"docker"}
}

// Cleanup implements the Executor interface
func (d *DockerExecutor) Cleanup(ctx context.Context, buildID string) error {
	logger := d.logger.With(slog.String("build_id", buildID))

	// Clean up workspace directory
	workspaceDir := filepath.Join(d.config.Builder.WorkspaceDir, buildID)
	if err := os.RemoveAll(workspaceDir); err != nil {
		logger.Error("failed to cleanup workspace directory",
			slog.String("workspace_dir", workspaceDir),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to cleanup workspace: %w", err)
	}

	logger.Info("build cleanup completed", slog.String("workspace_dir", workspaceDir))
	return nil
}

// generateBuildID generates a unique build ID
func generateBuildID() string {
	return fmt.Sprintf("build-%d", time.Now().UnixNano())
}
