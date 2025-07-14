package deploy

import (
	"context"
	"errors"
	"fmt"

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

	// Print source information
	printDeploymentSource(gitInfo, opts)

	// Build or use existing Docker image
	dockerImage := opts.DockerImage
	if dockerImage == "" {
		var err error
		dockerImage, err = buildDockerImage(ctx, opts, gitInfo)
		if err != nil {
			return fmt.Errorf("docker build failed: %w", err)
		}
	}

	if err := notifyControlPlane(ctx, logger, opts, dockerImage); err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	return nil
}

func printDeploymentSource(gitInfo git.Info, opts *DeployOptions) {
	fmt.Println("Source")
	fmt.Printf("  Branch: %s\n", opts.Branch)

	if gitInfo.IsRepo && gitInfo.CommitSHA != "" {
		shortSHA := gitInfo.CommitSHA
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}
		fmt.Printf("  Commit: %s\n", shortSHA)

		if gitInfo.IsDirty {
			fmt.Printf("  Status: Working directory has uncommitted changes\n")
		}
	} else if !gitInfo.IsRepo {
		fmt.Printf("  Status: Not a git repository\n")
	}

	fmt.Printf("  Context: %s\n", opts.Context)
	if opts.DockerImage != "" {
		fmt.Printf("  Docker Image: %s\n", opts.DockerImage)
	}
	fmt.Println()
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
