package deploy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/go/cmd/cli/progress"
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

// executeDeploy performs the actual deployment with Docker building and Git integration
func executeDeploy(ctx context.Context, opts *DeployOptions) error {
	logger := logging.New()

	// Get Git info for enhanced deployment tracking
	gitInfo := git.GetInfo()

	// Auto-detect Git values if not provided
	if opts.Branch == "main" && gitInfo.IsRepo && gitInfo.Branch != "" {
		opts.Branch = gitInfo.Branch
	}
	if opts.Commit == "" && gitInfo.CommitSHA != "" {
		opts.Commit = gitInfo.CommitSHA
	}

	tracker := progress.NewTracker("Unkey Deploy Progress")
	tracker.AddStep("source", "Source information")
	tracker.AddStep("prepare", "Preparing deployment")
	tracker.AddStep("build", "Building Docker image")
	tracker.AddStep("push", "Publishing to registry")
	tracker.AddStep("deploy", "Deploying to Unkey")
	tracker.AddStep("activate", "Activating version")
	tracker.AddStep("complete", "Deployment summary")
	tracker.Start()
	defer tracker.Stop()

	tracker.StartStep("source", "Gathering source information")
	sourceInfo := buildSourceInfo(gitInfo, opts)
	tracker.CompleteStep("source", sourceInfo)

	// Step 1: Prepare - validate environment and get Docker image
	tracker.StartStep("prepare", "Validating deployment environment")

	dockerImage := opts.DockerImage
	if dockerImage == "" {
		if !isDockerAvailable() {
			tracker.FailStep("prepare", "Docker command not found - please install Docker")
			return ErrDockerNotFound
		}

		imageTag := generateImageTag(opts, gitInfo)
		dockerImage = fmt.Sprintf("%s:%s", opts.Registry, imageTag)
	}

	tracker.CompleteStep("prepare", "Environment validated")

	// Step 2: Build - only if we need to build an image
	if opts.DockerImage == "" {
		tracker.StartStep("build", fmt.Sprintf("Building %s", dockerImage))

		if err := buildImage(ctx, opts, dockerImage); err != nil {
			tracker.FailStep("build", fmt.Sprintf("Build failed: %v", err))
			return fmt.Errorf("docker build failed: %w", err)
		}

		tracker.CompleteStep("build", "Docker image built successfully")
	} else {
		tracker.SkipStep("build", "Using pre-built Docker image")
	}

	// Step 3: Push - publish to registry
	if opts.SkipPush {
		tracker.SkipStep("push", "Push skipped (--skip-push enabled)")
	} else if opts.DockerImage == "" { // Only push if we built the image
		tracker.StartStep("push", "Publishing to registry")

		if err := pushImage(ctx, dockerImage, opts.Registry); err != nil {
			// Push failure shouldn't be fatal in development
			tracker.FailStep("push", fmt.Sprintf("push failed: %v", err))
			fmt.Printf("Push failed but continuing with deployment\n")
		} else {
			tracker.CompleteStep("push", "Image published successfully")
		}
	} else {
		tracker.SkipStep("push", "Using external Docker image")
	}

	// Step 4: Deploy - notify control plane and start deployment
	tracker.StartStep("deploy", "Starting deployment")

	if err := notifyControlPlane(ctx, logger, opts, dockerImage, tracker); err != nil {
		tracker.FailStep("deploy", fmt.Sprintf("deployment failed: %v", err))
		return fmt.Errorf("deployment failed: %w", err)
	}

	// Step 5: Activate - this will be completed by notifyControlPlane
	// when the version becomes active
	return nil
}

func buildSourceInfo(gitInfo git.Info, opts *DeployOptions) string {
	var parts []string

	// Branch
	parts = append(parts, fmt.Sprintf("Branch: %s", opts.Branch))

	// Commit info
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

	// Context
	parts = append(parts, fmt.Sprintf("Context: %s", opts.Context))

	// Docker image if pre-built
	if opts.DockerImage != "" {
		parts = append(parts, fmt.Sprintf("Image: %s", opts.DockerImage))
	}

	return strings.Join(parts, " | ")
}

// PrintDeployHelp prints detailed help for deploy command
func PrintDeployHelp() {
	fmt.Println("unkey deploy - Deploy a new version")
	fmt.Println("")
	fmt.Println("USAGE:")
	fmt.Println("    unkey deploy [FLAGS]")
	fmt.Println("")
	fmt.Println("DESCRIPTION:")
	fmt.Println("    Build and deploy a new version of your application.")
	fmt.Println("    Builds a Docker image from the specified context and")
	fmt.Println("    deploys it to the Unkey platform.")
	fmt.Println("")
	fmt.Println("REQUIRED FLAGS:")
	fmt.Println("    --workspace-id <id>    Workspace ID")
	fmt.Println("    --project-id <id>      Project ID")
	fmt.Println("")
	fmt.Println("OPTIONAL FLAGS:")
	fmt.Println("    --context <path>       Docker context path (default: .)")
	fmt.Println("    --branch <name>        Git branch (default: main)")
	fmt.Println("    --docker-image <tag>   Pre-built docker image")
	fmt.Println("    --dockerfile <path>    Path to Dockerfile (default: Dockerfile)")
	fmt.Println("    --commit <sha>         Git commit SHA")
	fmt.Println("    --registry <registry>  Docker registry (default: ghcr.io/unkeyed/deploy)")
	fmt.Println("    --skip-push            Skip pushing to registry")
	fmt.Println("")
	fmt.Println("ENVIRONMENT VARIABLES:")
	fmt.Println("    UNKEY_WORKSPACE_ID     Default workspace ID")
	fmt.Println("    UNKEY_PROJECT_ID       Default project ID")
	fmt.Println("    UNKEY_DOCKER_REGISTRY  Default Docker registry")
	fmt.Println("")
	fmt.Println("EXAMPLES:")
	fmt.Println("    # Basic deployment")
	fmt.Println("    unkey deploy \\")
	fmt.Println("      --workspace-id=ws_4QgQsKsKfdm3nGeC \\")
	fmt.Println("      --project-id=proj_9aiaks2dzl6mcywnxjf \\")
	fmt.Println("      --context=./demo_api")
	fmt.Println("")
	fmt.Println("    # Deploy with your own registry")
	fmt.Println("    unkey deploy \\")
	fmt.Println("      --workspace-id=ws_4QgQsKsKfdm3nGeC \\")
	fmt.Println("      --project-id=proj_9aiaks2dzl6mcywnxjf \\")
	fmt.Println("      --registry=docker.io/mycompany/myapp")
	fmt.Println("")
	fmt.Println("    # Local development (skip push)")
	fmt.Println("    unkey deploy \\")
	fmt.Println("      --workspace-id=ws_4QgQsKsKfdm3nGeC \\")
	fmt.Println("      --project-id=proj_9aiaks2dzl6mcywnxjf \\")
	fmt.Println("      --skip-push")
	fmt.Println("")
	fmt.Println("    # Deploy pre-built image")
	fmt.Println("    unkey deploy \\")
	fmt.Println("      --workspace-id=ws_4QgQsKsKfdm3nGeC \\")
	fmt.Println("      --project-id=proj_9aiaks2dzl6mcywnxjf \\")
	fmt.Println("      --docker-image=ghcr.io/user/app:v1.0.0")
}
