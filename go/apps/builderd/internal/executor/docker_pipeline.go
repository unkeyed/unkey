package executor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/unkeyed/unkey/go/apps/builderd/internal/config"
	"github.com/unkeyed/unkey/go/apps/builderd/internal/observability"
	builderv1 "github.com/unkeyed/unkey/go/gen/proto/builderd/v1"
)

// DockerPipelineExecutor wraps the existing DockerExecutor with step-based execution
type DockerPipelineExecutor struct {
	dockerExecutor *DockerExecutor
	pipeline       *BuildPipeline
	logger         *slog.Logger
	config         *config.Config
	buildMetrics   *observability.BuildMetrics
}

// NewDockerPipelineExecutor creates a new pipeline-based Docker executor
func NewDockerPipelineExecutor(logger *slog.Logger, cfg *config.Config, metrics *observability.BuildMetrics) *DockerPipelineExecutor {
	dockerExecutor := NewDockerExecutor(logger, cfg, metrics)
	pipeline := NewDockerBuildPipeline(dockerExecutor)

	return &DockerPipelineExecutor{
		dockerExecutor: dockerExecutor,
		pipeline:       pipeline,
		logger:         logger,
		config:         cfg,
		buildMetrics:   metrics,
	}
}

// ExtractDockerImageWithID executes the full Docker build pipeline
func (d *DockerPipelineExecutor) ExtractDockerImageWithID(ctx context.Context, request *builderv1.CreateBuildRequest, buildID string) (*BuildResult, error) {
	start := time.Now()

	logger := d.logger.With(
		slog.String("build_id", buildID),
		slog.String("image_uri", request.GetConfig().GetSource().GetDockerImage().GetImageUri()),
	)

	logger.InfoContext(ctx, "starting Docker pipeline build")

	// Record build start metrics
	if d.buildMetrics != nil {
		d.buildMetrics.RecordBuildStart(ctx, "docker", "docker")
	}

	defer func() {
		duration := time.Since(start)
		logger.InfoContext(ctx, "Docker pipeline build completed", slog.Duration("duration", duration))
	}()

	dockerSource := request.GetConfig().GetSource().GetDockerImage()
	if dockerSource == nil {
		return nil, fmt.Errorf("docker image source is required")
	}

	// Setup directories
	workspaceDir := filepath.Join(d.config.Builder.WorkspaceDir, buildID)
	rootfsDir := filepath.Join(d.config.Builder.RootfsOutputDir, buildID)

	logger = logger.With(
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

	// Prepare initial step input
	input := StepInput{
		BuildID:      buildID,
		Config:       request.GetConfig(),
		WorkspaceDir: workspaceDir,
		RootfsDir:    rootfsDir,
		Logger:       logger,
	}

	// Execute the pipeline
	result, err := d.pipeline.Execute(ctx, input)
	if err != nil {
		logger.ErrorContext(ctx, "pipeline execution failed", slog.String("error", err.Error()))
		if d.buildMetrics != nil {
			d.buildMetrics.RecordBuildComplete(ctx, "docker", "docker", time.Since(start), false)
		}
		return nil, err
	}

	// Record success metrics
	if d.buildMetrics != nil {
		d.buildMetrics.RecordBuildComplete(ctx, "docker", "docker", time.Since(start), true)
	}

	return result, nil
}

// ResumeBuild resumes a build from a specific step
func (d *DockerPipelineExecutor) ResumeBuild(ctx context.Context, request *builderv1.CreateBuildRequest, buildID string, lastCompletedStep int) (*BuildResult, error) {
	start := time.Now()

	logger := d.logger.With(
		slog.String("build_id", buildID),
		slog.String("image_uri", request.GetConfig().GetSource().GetDockerImage().GetImageUri()),
		slog.Int("resume_from_step", lastCompletedStep+1),
	)

	logger.InfoContext(ctx, "resuming Docker pipeline build")

	// Setup directories (should already exist)
	workspaceDir := filepath.Join(d.config.Builder.WorkspaceDir, buildID)
	rootfsDir := filepath.Join(d.config.Builder.RootfsOutputDir, buildID)

	// Prepare step input - we'd need to reconstruct state from previous steps here
	// For simplicity, we're starting fresh but skipping completed steps
	input := StepInput{
		BuildID:      buildID,
		Config:       request.GetConfig(),
		WorkspaceDir: workspaceDir,
		RootfsDir:    rootfsDir,
		Logger:       logger,
		// TODO: Restore ImageName, ContainerID, Metadata from previous execution
	}

	// Resume from the next step after the last completed one
	result, err := d.pipeline.Resume(ctx, input, lastCompletedStep+1)
	if err != nil {
		logger.ErrorContext(ctx, "pipeline resumption failed", slog.String("error", err.Error()))
		if d.buildMetrics != nil {
			d.buildMetrics.RecordBuildComplete(ctx, "docker", "docker", time.Since(start), false)
		}
		return nil, err
	}

	// Record success metrics
	if d.buildMetrics != nil {
		d.buildMetrics.RecordBuildComplete(ctx, "docker", "docker", time.Since(start), true)
	}

	return result, nil
}

// GetStepNames returns the names of all steps in the pipeline
func (d *DockerPipelineExecutor) GetStepNames() []string {
	steps := make([]string, len(d.pipeline.steps))
	for i, step := range d.pipeline.steps {
		steps[i] = step.Name()
	}
	return steps
}

// Execute implements the Executor interface (generates build ID)
func (d *DockerPipelineExecutor) Execute(ctx context.Context, request *builderv1.CreateBuildRequest) (*BuildResult, error) {
	// Generate build ID for backward compatibility
	return d.ExtractDockerImageWithID(ctx, request, generateBuildID())
}

// ExecuteWithID implements the Executor interface (uses provided build ID)
func (d *DockerPipelineExecutor) ExecuteWithID(ctx context.Context, request *builderv1.CreateBuildRequest, buildID string) (*BuildResult, error) {
	return d.ExtractDockerImageWithID(ctx, request, buildID)
}

// GetSupportedSources implements the Executor interface
func (d *DockerPipelineExecutor) GetSupportedSources() []string {
	return []string{"docker"}
}

// Cleanup implements the Executor interface - delegates to underlying DockerExecutor
func (d *DockerPipelineExecutor) Cleanup(ctx context.Context, buildID string) error {
	return d.dockerExecutor.Cleanup(ctx, buildID)
}
