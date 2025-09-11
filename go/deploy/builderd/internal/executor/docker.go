package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/unkeyed/unkey/go/deploy/builderd/internal/config"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/observability"
	builderv1 "github.com/unkeyed/unkey/go/gen/proto/deploy/builderd/v1"
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
	// Generate build ID for backward compatibility
	return d.ExtractDockerImageWithID(ctx, request, generateBuildID())
}

// ExtractDockerImageWithID pulls a Docker image and extracts it with a pre-assigned build ID
func (d *DockerExecutor) ExtractDockerImageWithID(ctx context.Context, request *builderv1.CreateBuildRequest, buildID string) (*BuildResult, error) {
	start := time.Now()

	logger := d.logger.With(
		slog.String("image_uri", request.GetConfig().GetSource().GetDockerImage().GetImageUri()),
	)

	logger.InfoContext(ctx, "starting Docker image extraction")

	// Record build start metrics
	if d.buildMetrics != nil {
		d.buildMetrics.RecordBuildStart(ctx, "docker", "docker")
	}

	defer func() {
		duration := time.Since(start)
		logger.InfoContext(ctx, "Docker image extraction completed", slog.Duration("duration", duration))
	}()

	dockerSource := request.GetConfig().GetSource().GetDockerImage()
	if dockerSource == nil {
		return nil, fmt.Errorf("docker image source is required")
	}

	// Use the provided build ID
	workspaceDir := filepath.Join(d.config.Builder.WorkspaceDir, buildID)
	rootfsDir := filepath.Join(d.config.Builder.RootfsOutputDir, buildID)

	logger = logger.With(
		slog.String("build_id", buildID),
		slog.String("workspace_dir", workspaceDir),
		slog.String("rootfs_dir", rootfsDir),
	)

	// Create directories
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		logger.ErrorContext(ctx, "failed to create workspace directory", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}

	if err := os.MkdirAll(rootfsDir, 0755); err != nil {
		logger.ErrorContext(ctx, "failed to create rootfs directory", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to create rootfs directory: %w", err)
	}

	// Use the full image URI directly
	fullImageName := dockerSource.GetImageUri()
	if fullImageName == "" {
		return nil, fmt.Errorf("docker image URI is required")
	}

	logger = logger.With(slog.String("full_image_name", fullImageName))

	// Step 1: Pull the Docker image
	if err := d.pullDockerImage(ctx, logger, fullImageName); err != nil {
		logger.ErrorContext(ctx, "failed to pull Docker image",
			slog.String("error", err.Error()),
			slog.String("image", fullImageName),
		)
		if d.buildMetrics != nil {
			d.buildMetrics.RecordBuildComplete(ctx, "docker", "docker", time.Since(start), false)
		}
		return nil, fmt.Errorf("failed to pull Docker image: %w", err)
	}

	// Step 2: Create container from image (without running)
	containerID, err := d.createContainer(ctx, logger, fullImageName)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create container",
			slog.String("error", err.Error()),
			slog.String("image", fullImageName),
		)
		if d.buildMetrics != nil {
			d.buildMetrics.RecordBuildComplete(ctx, "docker", "docker", time.Since(start), false)
		}
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Ensure cleanup of container
	defer func() {
		if cleanupErr := d.removeContainer(ctx, logger, containerID); cleanupErr != nil {
			logger.WarnContext(ctx, "failed to cleanup container", slog.String("error", cleanupErr.Error()))
		}
	}()

	// Step 3: Extract container metadata (entrypoint, cmd, env, etc.)
	metadata, err := d.extractContainerMetadata(ctx, logger, fullImageName)
	if err != nil {
		logger.ErrorContext(ctx, "failed to extract container metadata",
			slog.String("error", err.Error()),
			slog.String("image", fullImageName),
		)
		if d.buildMetrics != nil {
			d.buildMetrics.RecordBuildComplete(ctx, "docker", "docker", time.Since(start), false)
		}
		return nil, fmt.Errorf("failed to extract container metadata: %w", err)
	}

	// Step 4: Extract filesystem from container
	if err := d.extractFilesystem(ctx, logger, containerID, rootfsDir, metadata); err != nil {
		logger.ErrorContext(ctx, "failed to extract filesystem",
			slog.String("error", err.Error()),
			slog.String("container_id", containerID),
			slog.String("rootfs_dir", rootfsDir),
		)
		if d.buildMetrics != nil {
			d.buildMetrics.RecordBuildComplete(ctx, "docker", "docker", time.Since(start), false)
		}
		return nil, fmt.Errorf("failed to extract filesystem: %w", err)
	}

	// Step 5: Optimize rootfs (remove unnecessary files, etc.)
	if err := d.optimizeRootfs(ctx, logger, rootfsDir); err != nil {
		logger.WarnContext(ctx, "failed to optimize rootfs", slog.String("error", err.Error()))
		// Don't fail the build for optimization errors
	}

	// Step 6: Create ext4 filesystem image
	ext4Path := filepath.Join(d.config.Builder.RootfsOutputDir, buildID+".ext4")
	if err := d.createExt4Image(ctx, logger, rootfsDir, ext4Path); err != nil {
		logger.ErrorContext(ctx, "failed to create ext4 image",
			slog.String("error", err.Error()),
			slog.String("build_id", buildID),
		)
		return nil, fmt.Errorf("failed to create ext4 image: %w", err)
	}

	// Step 7: Save container metadata alongside the rootfs
	metadataPath := filepath.Join(d.config.Builder.RootfsOutputDir, buildID+".metadata.json")
	if err := d.saveContainerMetadata(ctx, logger, metadata, metadataPath); err != nil {
		logger.ErrorContext(ctx, "failed to save container metadata",
			slog.String("error", err.Error()),
			slog.String("metadata_path", metadataPath),
		)
		return nil, fmt.Errorf("failed to save container metadata: %w", err)
	}

	// Create build result
	result := &BuildResult{ //nolint:exhaustruct // Error, Metadata, and Metrics fields are set after successful build
		BuildID:       buildID,
		SourceType:    "docker",
		SourceImage:   fullImageName,
		RootfsPath:    ext4Path, // Use the ext4 image path instead of directory
		WorkspaceDir:  workspaceDir,
		StartTime:     start,
		EndTime:       time.Now(),
		Status:        "completed",
		ImageMetadata: metadata, // Include the extracted metadata
	}

	// Record successful build
	if d.buildMetrics != nil {
		d.buildMetrics.RecordBuildComplete(ctx, "docker", "docker", time.Since(start), true)
	}

	logger.InfoContext(ctx, "Docker image extraction successful",
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
	ctx, span := tracer.Start(ctx, "builderd.docker.pull_image",
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

	logger.InfoContext(ctx, "pulling Docker image", slog.String("image", imageName))

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
		logger.ErrorContext(ctx, "docker pull failed",
			slog.String("error", err.Error()),
			slog.String("output", string(output)),
			slog.Duration("duration", stepDuration),
		)
		return fmt.Errorf("docker pull failed: %w", err)
	}

	span.SetAttributes(attribute.String("status", "success"))
	logger.InfoContext(ctx, "docker pull completed",
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
	ctx, span := tracer.Start(ctx, "builderd.docker.create_container",
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

	logger.InfoContext(ctx, "creating container from image", slog.String("image", imageName))

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
		logger.ErrorContext(ctx, "docker create failed",
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
	logger.InfoContext(ctx, "container created",
		slog.String("container_id", containerID),
		slog.Duration("duration", stepDuration),
	)

	return containerID, nil
}

// extractFilesystem extracts the filesystem from the container to the rootfs directory
func (d *DockerExecutor) extractFilesystem(ctx context.Context, logger *slog.Logger, containerID, rootfsDir string, metadata *builderv1.ImageMetadata) error {
	// AIDEV-NOTE: Comprehensive observability for filesystem extraction step
	tracer := otel.Tracer("builderd/docker")
	stepStart := time.Now()

	// Start OpenTelemetry span for this build step
	ctx, span := tracer.Start(ctx, "builderd.docker.extract_filesystem",
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

	logger.InfoContext(ctx, "extracting filesystem from container",
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
		logger.ErrorContext(ctx, "failed to create pipe",
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
		logger.ErrorContext(ctx, "failed to start docker export",
			slog.String("error", err.Error()),
			slog.Duration("duration", stepDuration),
		)
		return fmt.Errorf("failed to start docker export: %w", err)
	}

	if err := tarCmd.Start(); err != nil {
		_ = cmd.Process.Kill()
		stepDuration := time.Since(stepStart)
		if d.buildMetrics != nil {
			d.buildMetrics.RecordBuildStepComplete(ctx, "extract", "docker", stepDuration, false)
		}
		span.SetAttributes(attribute.String("error", err.Error()))
		logger.ErrorContext(ctx, "failed to start tar extraction",
			slog.String("error", err.Error()),
			slog.Duration("duration", stepDuration),
		)
		return fmt.Errorf("failed to start tar extraction: %w", err)
	}

	// Wait for docker export to complete
	if err := cmd.Wait(); err != nil {
		_ = tarCmd.Process.Kill()
		stepDuration := time.Since(stepStart)
		if d.buildMetrics != nil {
			d.buildMetrics.RecordBuildStepComplete(ctx, "extract", "docker", stepDuration, false)
		}
		span.SetAttributes(attribute.String("error", err.Error()))
		logger.ErrorContext(ctx, "docker export failed",
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
		logger.ErrorContext(ctx, "tar extraction failed",
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
	logger.InfoContext(ctx, "filesystem extraction completed",
		slog.Duration("duration", stepDuration),
	)

	// AIDEV-NOTE: CRITICAL FIX - Inject metald-init into rootfs after extraction
	// This ensures every container has the required init process for VM execution
	if err := d.injectMetaldInit(ctx, logger, rootfsDir); err != nil {
		logger.WarnContext(ctx, "failed to inject metald-init (non-fatal)",
			slog.String("error", err.Error()),
			slog.String("rootfs_dir", rootfsDir),
		)
		// Continue anyway - this is not fatal, VM might still work with container's original init
	}

	// AIDEV-NOTE: Create container command file for metald-init
	// This tells metald-init what command to run when the microVM starts
	if err := d.createContainerCmd(ctx, logger, rootfsDir, metadata); err != nil {
		logger.WarnContext(ctx, "failed to create container.cmd (non-fatal)",
			slog.String("error", err.Error()),
			slog.String("rootfs_dir", rootfsDir),
		)
		// Continue anyway - this is not fatal if there's a fallback command
	}

	// AIDEV-NOTE: Create container environment file for metald-init
	// This provides complete container runtime environment replication
	if err := d.createContainerEnv(ctx, logger, rootfsDir, metadata); err != nil {
		logger.WarnContext(ctx, "failed to create container.env (non-fatal)",
			slog.String("error", err.Error()),
			slog.String("rootfs_dir", rootfsDir),
		)
		// Continue anyway - basic environment will still work
	}

	return nil
}

// removeContainer removes the temporary container
func (d *DockerExecutor) removeContainer(ctx context.Context, logger *slog.Logger, containerID string) error {
	logger.DebugContext(ctx, "removing container", slog.String("container_id", containerID))

	cmd := exec.CommandContext(ctx, "docker", "rm", containerID)
	if err := cmd.Run(); err != nil {
		logger.ErrorContext(ctx, "failed to remove container",
			slog.String("error", err.Error()),
			slog.String("container_id", containerID),
		)
		return fmt.Errorf("failed to remove container: %w", err)
	}

	logger.DebugContext(ctx, "container removed", slog.String("container_id", containerID))
	return nil
}

// optimizeRootfs removes unnecessary files and optimizes the rootfs
func (d *DockerExecutor) optimizeRootfs(ctx context.Context, logger *slog.Logger, rootfsDir string) error {
	// AIDEV-NOTE: Comprehensive observability for rootfs optimization step
	tracer := otel.Tracer("builderd/docker")
	stepStart := time.Now()

	// Start OpenTelemetry span for this build step
	ctx, span := tracer.Start(ctx, "builderd.docker.optimize_rootfs",
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

	logger.InfoContext(ctx, "optimizing rootfs", slog.String("rootfs_dir", rootfsDir))

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

		// Validate and sanitize the path before executing
		if err := validateAndSanitizePath(rootfsDir, fullPattern); err != nil {
			logger.WarnContext(ctx, "skipping unsafe pattern",
				slog.String("pattern", pattern),
				slog.String("error", err.Error()),
			)
			continue
		}

		// Use rm command to remove files matching pattern
		// Note: fullPattern is now validated and sanitized
		//nolint:gosec // G204: Path is validated and sanitized above to prevent injection
		cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("rm -rf %s", fullPattern))
		if err := cmd.Run(); err != nil {
			logger.DebugContext(ctx, "failed to remove pattern",
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
		logger.WarnContext(ctx, "failed to calculate rootfs size", slog.String("error", err.Error()))
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
		logger.WarnContext(ctx, "rootfs optimization completed with errors",
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
		logger.InfoContext(ctx, "rootfs optimization completed",
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

// createExt4Image creates an ext4 filesystem image from the rootfs directory
func (d *DockerExecutor) createExt4Image(ctx context.Context, logger *slog.Logger, rootfsDir, outputPath string) error {
	// AIDEV-NOTE: Create ext4 filesystem image for Firecracker VMs
	tracer := otel.Tracer("builderd/docker")
	stepStart := time.Now()

	// Start OpenTelemetry span for this build step
	ctx, span := tracer.Start(ctx, "builderd.docker.create_ext4_image",
		trace.WithAttributes(
			attribute.String("step", "create_ext4"),
			attribute.String("rootfs_dir", rootfsDir),
			attribute.String("output_path", outputPath),
		),
	)
	defer span.End()

	logger.InfoContext(ctx, "creating ext4 filesystem image",
		slog.String("rootfs_dir", rootfsDir),
		slog.String("output_path", outputPath),
	)

	// Calculate size needed (rootfs size + 20% overhead)
	rootfsSize, err := d.getRootfsSize(rootfsDir)
	if err != nil {
		return fmt.Errorf("failed to calculate rootfs size: %w", err)
	}

	// Add 20% overhead for filesystem metadata and future growth
	imageSize := int64(float64(rootfsSize) * 1.2)
	// Minimum 100MB, round up to nearest MB
	minSize := int64(100 * 1024 * 1024)
	if imageSize < minSize {
		imageSize = minSize
	}
	imageSize = (imageSize + 1024*1024 - 1) / (1024 * 1024) * (1024 * 1024) // Round up to MB

	logger.InfoContext(ctx, "calculated image size",
		slog.Int64("rootfs_bytes", rootfsSize),
		slog.Int64("image_bytes", imageSize),
	)

	// Step 1: Create sparse file
	createCmd := exec.CommandContext(ctx, "truncate", "-s", fmt.Sprintf("%d", imageSize), outputPath)
	if output, err := createCmd.CombinedOutput(); err != nil {
		logger.ErrorContext(ctx, "failed to create sparse file",
			slog.String("error", err.Error()),
			slog.String("output", string(output)),
		)
		return fmt.Errorf("failed to create sparse file: %w", err)
	}

	// Step 2: Create ext4 filesystem
	mkfsCmd := exec.CommandContext(ctx, "mkfs.ext4", "-F", "-d", rootfsDir, outputPath)
	if output, err := mkfsCmd.CombinedOutput(); err != nil {
		logger.ErrorContext(ctx, "failed to create ext4 filesystem",
			slog.String("error", err.Error()),
			slog.String("output", string(output)),
		)
		// Clean up the sparse file
		_ = os.Remove(outputPath)
		return fmt.Errorf("failed to create ext4 filesystem: %w", err)
	}

	// Step 3: Optimize the filesystem (optional)
	e2fsckCmd := exec.CommandContext(ctx, "e2fsck", "-f", "-y", outputPath)
	if output, err := e2fsckCmd.CombinedOutput(); err != nil {
		// Log but don't fail - e2fsck returns non-zero for fixes
		logger.WarnContext(ctx, "e2fsck completed with warnings",
			slog.String("output", string(output)),
		)
	}

	// Get final file size
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		logger.ErrorContext(ctx, "failed to stat ext4 image",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to stat ext4 image: %w", err)
	}

	// Set file permissions to be world-readable for other services
	// AIDEV-NOTE: Running as root, make files readable by other services
	if err := os.Chmod(outputPath, 0644); err != nil {
		logger.ErrorContext(ctx, "failed to set file permissions",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	stepDuration := time.Since(stepStart)
	span.SetAttributes(
		attribute.String("status", "success"),
		attribute.Int64("final_size", fileInfo.Size()),
	)

	logger.InfoContext(ctx, "ext4 filesystem image created successfully",
		slog.String("path", outputPath),
		slog.Int64("size_bytes", fileInfo.Size()),
		slog.Duration("duration", stepDuration),
	)

	return nil
}

// Execute implements the Executor interface
func (d *DockerExecutor) Execute(ctx context.Context, request *builderv1.CreateBuildRequest) (*BuildResult, error) {
	// Generate a new build ID for backward compatibility
	return d.ExecuteWithID(ctx, request, generateBuildID())
}

// ExecuteWithID implements the Executor interface for Docker builds with a pre-assigned ID
func (d *DockerExecutor) ExecuteWithID(ctx context.Context, request *builderv1.CreateBuildRequest, buildID string) (*BuildResult, error) {
	return d.ExtractDockerImageWithID(ctx, request, buildID)
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
		logger.ErrorContext(ctx, "failed to cleanup workspace directory",
			slog.String("workspace_dir", workspaceDir),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to cleanup workspace: %w", err)
	}

	logger.InfoContext(ctx, "build cleanup completed", slog.String("workspace_dir", workspaceDir))
	return nil
}

// generateBuildID generates a unique build ID
func generateBuildID() string {
	return fmt.Sprintf("build-%d", time.Now().UnixNano())
}

// validateAndSanitizePath validates that the target path is within the rootfs directory
// and doesn't contain dangerous characters or path traversal attempts
func validateAndSanitizePath(rootfsDir, targetPath string) error {
	// Clean and resolve paths to prevent directory traversal
	cleanRootfs := filepath.Clean(rootfsDir)
	cleanTarget := filepath.Clean(targetPath)

	// Ensure rootfs directory exists and is a directory
	if info, err := os.Stat(cleanRootfs); err != nil {
		return fmt.Errorf("rootfs directory does not exist: %w", err)
	} else if !info.IsDir() {
		return fmt.Errorf("rootfs path is not a directory: %s", cleanRootfs)
	}

	// Check that target path is within rootfs directory (prevent path traversal)
	relPath, err := filepath.Rel(cleanRootfs, cleanTarget)
	if err != nil {
		return fmt.Errorf("invalid path relationship: %w", err)
	}

	// Ensure the relative path doesn't start with ".." (path traversal attempt)
	if strings.HasPrefix(relPath, "..") || strings.Contains(relPath, "../") {
		return fmt.Errorf("path traversal attempt detected: %s", relPath)
	}

	// Additional security: check for dangerous characters and sequences
	dangerousPattern := regexp.MustCompile(`[;&|$\x60\\]|&&|\|\||>>|<<`)
	if dangerousPattern.MatchString(cleanTarget) {
		return fmt.Errorf("dangerous characters detected in path: %s", cleanTarget)
	}

	// Ensure path length is reasonable (prevent buffer overflow attacks)
	if len(cleanTarget) > 4096 {
		return fmt.Errorf("path too long: %d characters", len(cleanTarget))
	}

	return nil
}

// extractContainerMetadata extracts runtime configuration from a Docker image
func (d *DockerExecutor) extractContainerMetadata(ctx context.Context, logger *slog.Logger, imageName string) (*builderv1.ImageMetadata, error) {
	// AIDEV-NOTE: Extract container metadata for microvm execution
	logger.InfoContext(ctx, "extracting container metadata", slog.String("image", imageName))

	// Use docker inspect to get image configuration
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--type=image", imageName)
	output, err := cmd.Output()
	if err != nil {
		logger.ErrorContext(ctx, "docker inspect failed",
			slog.String("error", err.Error()),
			slog.String("image", imageName),
		)
		return nil, fmt.Errorf("docker inspect failed: %w", err)
	}

	// Parse the JSON output
	var inspectResults []map[string]interface{}
	if err := json.Unmarshal(output, &inspectResults); err != nil {
		logger.ErrorContext(ctx, "failed to parse docker inspect output",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to parse docker inspect output: %w", err)
	}

	if len(inspectResults) == 0 {
		return nil, fmt.Errorf("no image data returned from docker inspect")
	}

	imageData := inspectResults[0]
	config, ok := imageData["Config"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing Config in docker inspect output")
	}

	// Extract runtime configuration
	metadata := &builderv1.ImageMetadata{
		OriginalImage: imageName,
	}

	// Extract entrypoint
	if entrypoint, ok := config["Entrypoint"].([]interface{}); ok {
		for _, e := range entrypoint {
			if str, ok := e.(string); ok {
				metadata.Entrypoint = append(metadata.Entrypoint, str)
			}
		}
	}

	// Extract command
	if cmd, ok := config["Cmd"].([]interface{}); ok {
		for _, c := range cmd {
			if str, ok := c.(string); ok {
				metadata.Command = append(metadata.Command, str)
			}
		}
	}

	// Extract working directory
	if workingDir, ok := config["WorkingDir"].(string); ok {
		metadata.WorkingDir = workingDir
	}

	// Extract environment variables
	if env, ok := config["Env"].([]interface{}); ok {
		metadata.Env = make(map[string]string)
		for _, e := range env {
			if str, ok := e.(string); ok {
				parts := strings.SplitN(str, "=", 2)
				if len(parts) == 2 {
					metadata.Env[parts[0]] = parts[1]
				}
			}
		}
	}

	// Extract exposed ports
	if exposedPorts, ok := config["ExposedPorts"].(map[string]interface{}); ok {
		for port := range exposedPorts {
			// Docker format is "port/protocol", extract just the port number
			parts := strings.Split(port, "/")
			if len(parts) > 0 {
				metadata.ExposedPorts = append(metadata.ExposedPorts, parts[0])
			}
		}
	}

	logger.InfoContext(ctx, "extracted container metadata",
		slog.Int("entrypoint_len", len(metadata.Entrypoint)),
		slog.Int("cmd_len", len(metadata.Command)),
		slog.String("working_dir", metadata.WorkingDir),
		slog.Int("env_vars", len(metadata.Env)),
		slog.Int("exposed_ports", len(metadata.ExposedPorts)),
	)

	return metadata, nil
}

// saveContainerMetadata saves the container metadata to a JSON file
func (d *DockerExecutor) saveContainerMetadata(ctx context.Context, logger *slog.Logger, metadata *builderv1.ImageMetadata, path string) error {
	// AIDEV-NOTE: Save metadata for metald to use when configuring microvm
	logger.InfoContext(ctx, "saving container metadata", slog.String("path", path))

	// Marshal metadata to JSON
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	logger.InfoContext(ctx, "container metadata saved",
		slog.String("path", path),
		slog.Int("size", len(data)),
	)

	return nil
}

// injectMetaldInit copies the metald-init binary into the rootfs
// AIDEV-NOTE: This function ensures every container rootfs has metald-init available at /usr/bin/metald-init
// This is critical for VM boot as the kernel expects init=/usr/bin/metald-init
func (d *DockerExecutor) injectMetaldInit(ctx context.Context, logger *slog.Logger, rootfsDir string) error {
	logger.InfoContext(ctx, "injecting metald-init into rootfs",
		slog.String("rootfs_dir", rootfsDir),
	)

	// Source path for metald-init binary
	// Try multiple possible locations for metald-init
	var srcPaths = []string{
		"/usr/bin/metald-init",                  // Standard installation location
		"./cmd/metald-init/metald-init",         // Local build
		"../metald/cmd/metald-init/metald-init", // Relative from builderd
		"/usr/local/bin/metald-init",            // Legacy location (fallback)
		"/opt/metald/bin/metald-init",           // Custom location
	}

	var srcPath string
	for _, path := range srcPaths {
		if _, err := os.Stat(path); err == nil {
			srcPath = path
			break
		}
	}

	if srcPath == "" {
		return fmt.Errorf("metald-init binary not found in any expected location: %v", srcPaths)
	}

	// Destination paths in rootfs
	usrBinDir := filepath.Join(rootfsDir, "usr", "bin")
	dstPath := filepath.Join(usrBinDir, "metald-init")

	// Create /usr/bin directory if it doesn't exist
	if err := os.MkdirAll(usrBinDir, 0755); err != nil {
		return fmt.Errorf("failed to create /usr/bin directory: %w", err)
	}

	// Copy metald-init binary
	srcData, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read metald-init source: %w", err)
	}

	if err := os.WriteFile(dstPath, srcData, 0755); err != nil {
		return fmt.Errorf("failed to write metald-init to rootfs: %w", err)
	}

	logger.InfoContext(ctx, "metald-init injection completed",
		slog.String("src_path", srcPath),
		slog.String("dst_path", dstPath),
		slog.Int("size_bytes", len(srcData)),
	)

	return nil
}

// createContainerCmd creates /container.cmd file with the container's command for metald-init
// AIDEV-NOTE: This function creates the command file that metald-init reads to know what to execute
// The file contains a JSON array of the full command (entrypoint + command)
func (d *DockerExecutor) createContainerCmd(ctx context.Context, logger *slog.Logger, rootfsDir string, metadata *builderv1.ImageMetadata) error {
	logger.InfoContext(ctx, "creating container command file",
		slog.String("rootfs_dir", rootfsDir),
	)

	// Build the full command from entrypoint + command
	var fullCmd []string

	// Add entrypoint if present
	if len(metadata.Entrypoint) > 0 {
		fullCmd = append(fullCmd, metadata.Entrypoint...)
	}

	// Add command if present
	if len(metadata.Command) > 0 {
		fullCmd = append(fullCmd, metadata.Command...)
	}

	// If no command specified, provide a default
	if len(fullCmd) == 0 {
		logger.WarnContext(ctx, "no entrypoint or command found, using default shell")
		fullCmd = []string{"/bin/sh"}
	}

	// Create the command file path
	cmdPath := filepath.Join(rootfsDir, "container.cmd")

	// Marshal command to JSON
	cmdData, err := json.Marshal(fullCmd)
	if err != nil {
		return fmt.Errorf("failed to marshal container command: %w", err)
	}

	// Write to file
	if err := os.WriteFile(cmdPath, cmdData, 0644); err != nil {
		return fmt.Errorf("failed to write container.cmd file: %w", err)
	}

	logger.InfoContext(ctx, "container command file created",
		slog.String("path", cmdPath),
		slog.Any("command", fullCmd),
		slog.Int("size", len(cmdData)),
	)

	return nil
}

// createContainerEnv creates environment configuration file for complete container runtime replication
// AIDEV-NOTE: This function creates a comprehensive environment setup that metald-init reads
// to replicate the exact container runtime environment including working directory, env vars, etc.
func (d *DockerExecutor) createContainerEnv(ctx context.Context, logger *slog.Logger, rootfsDir string, metadata *builderv1.ImageMetadata) error {
	logger.InfoContext(ctx, "creating container environment file",
		slog.String("rootfs_dir", rootfsDir),
	)

	// Create comprehensive environment configuration
	envConfig := struct {
		WorkingDir   string            `json:"working_dir,omitempty"`
		Env          map[string]string `json:"env,omitempty"`
		ExposedPorts []string          `json:"exposed_ports,omitempty"`
	}{
		WorkingDir:   metadata.WorkingDir,
		Env:          metadata.Env,
		ExposedPorts: metadata.ExposedPorts,
	}

	// Set default working directory if not specified
	if envConfig.WorkingDir == "" {
		envConfig.WorkingDir = "/"
	}

	// Ensure essential environment variables are set
	if envConfig.Env == nil {
		envConfig.Env = make(map[string]string)
	}

	// Set default PATH if not present
	if _, hasPath := envConfig.Env["PATH"]; !hasPath {
		envConfig.Env["PATH"] = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	}

	// Create the environment file path
	envPath := filepath.Join(rootfsDir, "container.env")

	// Marshal environment config to JSON
	envData, err := json.MarshalIndent(envConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal container environment: %w", err)
	}

	// Write to file
	if err := os.WriteFile(envPath, envData, 0644); err != nil {
		return fmt.Errorf("failed to write container.env file: %w", err)
	}

	logger.InfoContext(ctx, "container environment file created",
		slog.String("path", envPath),
		slog.String("working_dir", envConfig.WorkingDir),
		slog.Int("env_vars", len(envConfig.Env)),
		slog.Int("size", len(envData)),
	)

	return nil
}
