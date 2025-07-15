package deploy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/go/cmd/cli/orchestrator"
	"github.com/unkeyed/unkey/go/pkg/git"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

var (
	ErrDockerNotFound    = errors.New("docker command not found - please install Docker")
	ErrDockerBuildFailed = errors.New("docker build failed")
	ErrDockerPushFailed  = errors.New("docker push failed")
	ErrInvalidImageTag   = errors.New("invalid image tag generated")
)

// DeployOptions holds all deployment configuration
type DeployOptions struct {
	WorkspaceID     string
	ProjectID       string
	Context         string
	Branch          string
	DockerImage     string
	Dockerfile      string
	Commit          string
	Registry        string
	SkipPush        bool
	ControlPlaneURL string
	AuthToken       string
}

// Deploy handles the deploy command
func Deploy(ctx context.Context, args []string, env map[string]string) error {
	opts, err := parseDeployFlags(args, env)
	if err != nil {
		return err
	}
	return executeDeploy(ctx, opts)
}

// executeDeploy - clean and focused using the orchestrator pattern
func executeDeploy(ctx context.Context, opts *DeployOptions) error {
	logger := logging.New()

	// Create and execute deployment orchestrator
	orchestrator := NewDeploymentOrchestrator(ctx, opts, logger)
	return orchestrator.Execute()
}

// DeploymentOrchestrator wraps the generic orchestrator with deployment-specific logic
type DeploymentOrchestrator struct {
	*orchestrator.Orchestrator
	opts         *DeployOptions
	logger       logging.Logger
	controlPlane *ControlPlaneClient
}

// NewDeploymentOrchestrator creates a new deployment orchestrator
func NewDeploymentOrchestrator(ctx context.Context, opts *DeployOptions, logger logging.Logger) *DeploymentOrchestrator {
	orch := orchestrator.New(ctx, "Unkey Deploy Progress")

	do := &DeploymentOrchestrator{
		Orchestrator: orch,
		opts:         opts,
		logger:       logger,
		controlPlane: NewControlPlaneClient(opts),
	}

	// Build the deployment pipeline
	do.buildPipeline()

	return do
}

// buildPipeline constructs the deployment steps
func (do *DeploymentOrchestrator) buildPipeline() {
	gitInfo := git.GetInfo()

	// Auto-detect Git values if not provided
	if do.opts.Branch == "main" && gitInfo.IsRepo && gitInfo.Branch != "" {
		do.opts.Branch = gitInfo.Branch
	}
	if do.opts.Commit == "" && gitInfo.CommitSHA != "" {
		do.opts.Commit = gitInfo.CommitSHA
	}

	do.AddSteps(
		// Step 1: Gather source information
		orchestrator.NewStep("source", "Source information").
			Execute(func(ctx context.Context) error {
				return nil // Just gathering info
			}).
			OnSuccess(func() string {
				return do.buildSourceInfo(gitInfo)
			}).
			Build(),

		// Step 2: Prepare deployment environment
		orchestrator.NewStep("prepare", "Preparing deployment").
			Execute(do.prepareDeployment).
			OnSuccess(func() string {
				return "Environment validated"
			}).
			OnError(func(err error) string {
				if err == ErrDockerNotFound {
					return "docker command not found - please install Docker"
				}
				return fmt.Sprintf("Preparation failed: %v", err)
			}).
			Build(),

		// Step 3: Build Docker image (conditional)
		orchestrator.ConditionalStep(
			"build",
			"Building Docker image",
			do.buildImage,
			func() bool { return do.opts.DockerImage != "" }, // Skip if using pre-built image
			func() string { return "Using pre-built Docker image" },
		),

		// Step 4: Push to registry (conditional)
		orchestrator.ConditionalStep(
			"push",
			"Publishing to registry",
			do.pushImage,
			func() bool { return do.opts.SkipPush || do.opts.DockerImage != "" },
			func() string {
				if do.opts.SkipPush {
					return "Push skipped (--skip-push enabled)"
				}
				return "Using external Docker image"
			},
		),

		// Step 5: Deploy to Unkey
		orchestrator.NewStep("deploy", "Deploying to Unkey").
			Execute(do.deployToUnkey).
			OnError(func(err error) string {
				return fmt.Sprintf("Deployment failed: %v", err)
			}).
			Build(),

		// Step 6: Activate version (managed by polling)
		orchestrator.NewStep("activate", "Activating version").
			Execute(func(ctx context.Context) error {
				return nil // Managed by polling in deployToUnkey
			}).
			Build(),

		// Step 7: Generate completion summary
		orchestrator.NewStep("complete", "Deployment summary").
			Execute(do.generateSummary).
			OnSuccess(func() string {
				return do.buildCompletionInfo(gitInfo)
			}).
			OnError(func(err error) string {
				return "Failed to generate deployment summary"
			}).
			Build(),
	)
}

// Step implementations
func (do *DeploymentOrchestrator) prepareDeployment(ctx context.Context) error {
	if do.opts.DockerImage == "" {
		if !isDockerAvailable() {
			return ErrDockerNotFound
		}

		gitInfo := git.GetInfo()
		imageTag := generateImageTag(do.opts, gitInfo)
		dockerImage := fmt.Sprintf("%s:%s", do.opts.Registry, imageTag)
		do.SetState("dockerImage", dockerImage)
	} else {
		do.SetState("dockerImage", do.opts.DockerImage)
	}

	return nil
}

func (do *DeploymentOrchestrator) buildImage(ctx context.Context) error {
	dockerImage := orchestrator.MustStateAs[string](do.Orchestrator, "dockerImage")

	do.UpdateStepMessage("build", fmt.Sprintf("Building %s", dockerImage))

	if err := buildImage(ctx, do.opts, dockerImage); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	return nil
}

func (do *DeploymentOrchestrator) pushImage(ctx context.Context) error {
	dockerImage := orchestrator.MustStateAs[string](do.Orchestrator, "dockerImage")

	do.UpdateStepMessage("push", "Publishing to registry")

	if err := pushImage(ctx, dockerImage, do.opts.Registry); err != nil {
		// For push failures, we continue deployment but log the error
		fmt.Printf("Push failed but continuing with deployment\n")
		return nil // Don't fail the step
	}

	return nil
}

func (do *DeploymentOrchestrator) deployToUnkey(ctx context.Context) error {
	dockerImage := orchestrator.MustStateAs[string](do.Orchestrator, "dockerImage")

	do.UpdateStepMessage("deploy", "Starting deployment")

	// Create version
	versionId, err := do.controlPlane.CreateVersion(ctx, dockerImage)
	if err != nil {
		return fmt.Errorf("failed to create version: %w", err)
	}

	do.SetState("versionId", versionId)
	do.UpdateStepMessage("deploy", fmt.Sprintf("Version created: %s", versionId))

	// Poll for completion - this will update the activate step
	if err := do.controlPlane.PollVersionStatus(ctx, do.logger, versionId, do.GetTracker()); err != nil {
		return fmt.Errorf("deployment polling failed: %w", err)
	}

	return nil
}

func (do *DeploymentOrchestrator) generateSummary(ctx context.Context) error {
	versionId, ok := orchestrator.StateAs[string](do.Orchestrator, "versionId")
	if !ok {
		return fmt.Errorf("no version ID available for summary")
	}
	if versionId == "" {
		return fmt.Errorf("empty version ID")
	}
	return nil
}

// Helper methods for building display information
func (do *DeploymentOrchestrator) buildSourceInfo(gitInfo git.Info) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Branch: %s", do.opts.Branch))

	if gitInfo.IsRepo && gitInfo.CommitSHA != "" {
		shortSHA := gitInfo.CommitSHA
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}
		commitInfo := fmt.Sprintf("Commit: %s", shortSHA)
		if gitInfo.IsDirty {
			commitInfo += " (dirty)"
		}
		parts = append(parts, commitInfo)
	} else if !gitInfo.IsRepo {
		parts = append(parts, "Not a git repository")
	}

	parts = append(parts, fmt.Sprintf("Context: %s", do.opts.Context))

	if do.opts.DockerImage != "" {
		parts = append(parts, fmt.Sprintf("Image: %s", do.opts.DockerImage))
	}

	return strings.Join(parts, " | ")
}

func (do *DeploymentOrchestrator) buildCompletionInfo(gitInfo git.Info) string {
	versionId := orchestrator.MustStateAs[string](do.Orchestrator, "versionId")

	if versionId == "" || do.opts.WorkspaceID == "" || do.opts.Branch == "" {
		return ""
	}

	var parts []string

	parts = append(parts, fmt.Sprintf("Version: %s", versionId))
	parts = append(parts, "Status: Ready")
	parts = append(parts, "Env: Production")

	identifier := versionId
	if gitInfo.ShortSHA != "" {
		identifier = gitInfo.ShortSHA
	}

	domain := fmt.Sprintf("https://%s-%s-%s.unkey.app", do.opts.Branch, identifier, do.opts.WorkspaceID)
	parts = append(parts, fmt.Sprintf("URL: %s", domain))

	return strings.Join(parts, " | ")
}
