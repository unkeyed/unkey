package executor

import (
	"context"
	"log/slog"

	builderv1 "github.com/unkeyed/unkey/go/gen/proto/builderd/v1"
)

// StepInput contains the input data for a build step
type StepInput struct {
	BuildID      string
	Config       *builderv1.BuildConfig
	WorkspaceDir string
	RootfsDir    string
	Logger       *slog.Logger

	// Output from previous steps
	ImageName   string
	ContainerID string
	Metadata    *builderv1.ImageMetadata
}

// StepOutput contains the output data from a build step
type StepOutput struct {
	// Data to pass to next step
	ImageName   string
	ContainerID string
	Metadata    *builderv1.ImageMetadata
	RootfsPath  string

	// Step completion info
	Success bool
	Error   error
}

// StepExecutor represents a single build step
type StepExecutor interface {
	Execute(ctx context.Context, input StepInput) (StepOutput, error)
	Name() string
}

// BuildPipeline represents a sequence of build steps
type BuildPipeline struct {
	steps []StepExecutor
}

// NewDockerBuildPipeline creates a pipeline for Docker image builds
func NewDockerBuildPipeline(dockerExecutor *DockerExecutor) *BuildPipeline {
	return &BuildPipeline{
		steps: []StepExecutor{
			&PullImageStep{executor: dockerExecutor},
			&CreateContainerStep{executor: dockerExecutor},
			&ExtractMetadataStep{executor: dockerExecutor},
			&ExtractFilesystemStep{executor: dockerExecutor},
			&OptimizeRootfsStep{executor: dockerExecutor},
			&CleanupStep{executor: dockerExecutor},
		},
	}
}

// Execute runs the entire pipeline
func (p *BuildPipeline) Execute(ctx context.Context, initialInput StepInput) (*BuildResult, error) {
	input := initialInput

	for i, step := range p.steps {
		initialInput.Logger.InfoContext(ctx, "executing build step",
			slog.String("step", step.Name()),
			slog.Int("step_index", i),
			slog.Int("total_steps", len(p.steps)),
		)

		output, err := step.Execute(ctx, input)
		if err != nil {
			return nil, err
		}

		// Prepare input for next step
		input.ImageName = output.ImageName
		input.ContainerID = output.ContainerID
		input.Metadata = output.Metadata
	}

	// Return final build result
	return &BuildResult{
		RootfsPath:    input.RootfsDir,
		ImageMetadata: input.Metadata,
	}, nil
}

// Resume executes the pipeline starting from a specific step
func (p *BuildPipeline) Resume(ctx context.Context, input StepInput, startStepIndex int) (*BuildResult, error) {
	input.Logger.InfoContext(ctx, "resuming build pipeline",
		slog.Int("start_step", startStepIndex),
		slog.Int("total_steps", len(p.steps)),
	)

	for i := startStepIndex; i < len(p.steps); i++ {
		step := p.steps[i]

		input.Logger.InfoContext(ctx, "executing build step (resumed)",
			slog.String("step", step.Name()),
			slog.Int("step_index", i),
		)

		output, err := step.Execute(ctx, input)
		if err != nil {
			return nil, err
		}

		// Prepare input for next step
		input.ImageName = output.ImageName
		input.ContainerID = output.ContainerID
		input.Metadata = output.Metadata
	}

	return &BuildResult{
		RootfsPath:    input.RootfsDir,
		ImageMetadata: input.Metadata,
	}, nil
}
