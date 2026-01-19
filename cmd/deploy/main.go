package deploy

import (
	"context"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/cmd/deploy/internal/ui"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/git"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	// Default configuration values
	DefaultBranch      = "main"
	DefaultDockerfile  = "Dockerfile"
	DefaultEnvironment = "preview"
)

// DeployOptions contains all configuration for deployment
type DeployOptions struct {
	ProjectID   string
	KeyspaceID  string
	Context     string
	DockerImage string
	Branch      string
	Dockerfile  string
	Commit      string
	Environment string
	RootKey     string
	APIBaseURL  string
}

var DeployFlags = []cli.Flag{
	// Required flags
	cli.String("project-id", "Project ID", cli.EnvVar("UNKEY_PROJECT_ID")),
	cli.String("keyspace-id", "Keyspace ID for API key authentication", cli.EnvVar("UNKEY_KEYSPACE_ID")),
	// Optional flags with defaults
	cli.String("context", "Build context path", cli.Default(".")),
	cli.String("branch", "Git branch", cli.Default(DefaultBranch)),
	cli.String("docker-image", "Pre-built docker image"),
	cli.String("dockerfile", "Path to Dockerfile", cli.Default(DefaultDockerfile)),
	cli.String("commit", "Git commit SHA"),
	cli.String("env", "Environment slug to deploy to", cli.Default(DefaultEnvironment)),
	// Authentication flag
	cli.String("root-key", "Root key for authentication", cli.EnvVar("UNKEY_ROOT_KEY")),
	// API configuration
	cli.String("api-base-url", "API base URL for local testing", cli.EnvVar("UNKEY_API_BASE_URL")),
}

// Cmd is the deploy command that builds and deploys application versions to Unkey infrastructure.
// It handles Docker image building, registry pushing, and deployment lifecycle management.
var Cmd = &cli.Command{
	Version:  "",
	Commands: []*cli.Command{},
	Aliases:  []string{},
	Name:     "deploy",
	Usage:    "Deploy a new version",
	Description: `Build and deploy a new version of your application.

The deploy command handles the complete deployment lifecycle: from building Docker images to deploying them on Unkey's infrastructure. It automatically detects your Git context, builds containers, and manages the deployment process with real-time status updates.

DEPLOYMENT PROCESS:
1. Build Docker image from your application
2. Push image to container registry
3. Create deployment version on Unkey platform
4. Monitor deployment status until active

EXAMPLES:
unkey deploy --project-id=proj_123           # Deploy with project ID
unkey deploy --context=./api                 # Deploy with custom build context
unkey deploy --docker-image=ghcr.io/user/app:v1.0.0 # Deploy pre-built image`,
	Flags:  DeployFlags,
	Action: DeployAction,
}

func DeployAction(ctx context.Context, cmd *cli.Command) error {
	// Validate required fields
	projectID := cmd.String("project-id")
	if projectID == "" {
		return fmt.Errorf("project ID is required (use --project-id flag or UNKEY_PROJECT_ID env var)")
	}

	// Build context defaults to current directory if not specified
	context := cmd.String("context")
	if context == "" {
		context = "."
	}

	opts := DeployOptions{
		ProjectID:   projectID,
		KeyspaceID:  cmd.String("keyspace-id"),
		Context:     context,
		DockerImage: cmd.String("docker-image"),
		Branch:      cmd.String("branch"),
		Dockerfile:  cmd.String("dockerfile"),
		Commit:      cmd.String("commit"),
		Environment: cmd.String("env"),
		RootKey:     cmd.String("root-key"),
		APIBaseURL:  cmd.String("api-base-url"),
	}

	return executeDeploy(ctx, opts)
}

func executeDeploy(ctx context.Context, opts DeployOptions) error {
	terminal := ui.NewUI()
	logger := logging.New()
	gitInfo := git.GetInfo()

	// Auto-detect branch and commit from git if not specified
	if opts.Branch == DefaultBranch && gitInfo.IsRepo && gitInfo.Branch != "" {
		opts.Branch = gitInfo.Branch
	}
	if opts.Commit == "" && gitInfo.CommitSHA != "" {
		opts.Commit = gitInfo.CommitSHA
	}

	// Print header
	fmt.Printf("Unkey Deploy Progress\n")
	fmt.Printf("──────────────────────────────────────────────────\n")
	printSourceInfo(opts, gitInfo)

	controlPlane := NewControlPlaneClient(opts)

	var deploymentID string
	var err error

	// Determine deployment source: prebuilt image or build from context
	if opts.DockerImage != "" {
		// Use prebuilt Docker image
		terminal.StartSpinner("Creating deployment")
		deploymentID, err = controlPlane.CreateDeployment(ctx, "", opts.DockerImage)
		if err != nil {
			terminal.StopSpinner("Failed to create deployment", false)
			terminal.PrintErrorDetails(err.Error())
			return err
		}
		terminal.StopSpinner(fmt.Sprintf("Deployment created: %s", deploymentID), true)
	} else {
		// Build from context
		terminal.StartSpinner("Uploading build context")
		var buildContextPath string
		buildContextPath, err = controlPlane.UploadBuildContext(ctx, opts.Context)
		if err != nil {
			terminal.StopSpinner("Failed to upload build context", false)
			terminal.PrintErrorDetails(err.Error())
			return err
		}
		terminal.StopSpinner(fmt.Sprintf("Build context uploaded: %s", buildContextPath), true)

		terminal.StartSpinner("Creating deployment")
		deploymentID, err = controlPlane.CreateDeployment(ctx, buildContextPath, "")
		if err != nil {
			terminal.StopSpinner("Failed to create deployment", false)
			terminal.PrintErrorDetails(err.Error())
			return err
		}
		terminal.StopSpinner(fmt.Sprintf("Deployment created: %s", deploymentID), true)
	}

	// Track final deployment for completion info
	var finalDeployment *ctrlv1.Deployment

	// Start monitoring spinner
	terminal.StartSpinner("Deployment in progress")

	// Handle deployment status changes
	onStatusChange := func(event DeploymentStatusEvent) error {
		// nolint: exhaustive // We just need those two for now
		switch event.CurrentStatus {
		case ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_FAILED:
			return handleDeploymentFailure(controlPlane, event.Deployment, terminal)
		case ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_READY:
			// Store deployment but don't print success, wait for polling to complete
			finalDeployment = event.Deployment
		}
		return nil
	}

	// Poll for deployment completion
	err = controlPlane.PollDeploymentStatus(ctx, logger, deploymentID, onStatusChange)
	if err != nil {
		terminal.StopSpinner("Deployment failed", false)
		return err
	}

	// Print final success message only after all polling is complete
	if finalDeployment != nil {
		terminal.StopSpinner("Deployment completed successfully", true)
		fmt.Printf("\n")
		printCompletionInfo(finalDeployment, opts.Environment)
		fmt.Printf("\n")
	}

	return nil
}

func handleDeploymentFailure(controlPlane *ControlPlaneClient, deployment *ctrlv1.Deployment, terminal *ui.UI) error {
	errorMsg := controlPlane.getFailureMessage(deployment)
	terminal.StopSpinner("Deployment failed", false)
	terminal.PrintErrorDetails(errorMsg)
	return fmt.Errorf("deployment failed: %s", errorMsg)
}

func printSourceInfo(opts DeployOptions, gitInfo git.Info) {
	fmt.Printf("Source Information:\n")
	fmt.Printf("    Branch: %s\n", opts.Branch)

	if gitInfo.IsRepo && gitInfo.CommitSHA != "" {
		commitInfo := gitInfo.ShortSHA
		if gitInfo.IsDirty {
			commitInfo += " (dirty)"
		}
		fmt.Printf("    Commit: %s\n", commitInfo)
	}

	if opts.DockerImage != "" {
		fmt.Printf("    Image: %s\n", opts.DockerImage)
	} else {
		fmt.Printf("    Context: %s\n", opts.Context)
	}

	fmt.Printf("\n")
}

func printCompletionInfo(deployment *ctrlv1.Deployment, env string) {
	if deployment == nil || deployment.GetId() == "" {
		fmt.Printf("✓ Deployment completed\n")
		return
	}

	caser := cases.Title(language.English)

	fmt.Println()
	fmt.Println("Deployment Complete")
	fmt.Printf("  Deployment ID: %s\n", deployment.GetId())
	fmt.Printf("  Status: Ready\n")
	fmt.Printf("  Environment: %s\n", caser.String(env))

	fmt.Println()
	fmt.Println("Domains")

	hostnames := deployment.GetHostnames()
	if len(hostnames) > 0 {
		for _, hostname := range hostnames {
			if strings.HasPrefix(hostname, "localhost:") {
				fmt.Printf("  http://%s\n", hostname)
			} else {
				fmt.Printf("  https://%s\n", hostname)
			}
		}
	} else {
		fmt.Printf("  No hostnames assigned\n")
	}
}
