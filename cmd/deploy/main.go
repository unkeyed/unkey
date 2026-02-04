package deploy

import (
	"context"
	"fmt"
	"strings"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/deploy/internal/errors"
	"github.com/unkeyed/unkey/cmd/deploy/internal/ui"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/git"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	// Default configuration values
	DefaultBranch      = "main"
	DefaultEnvironment = "preview"
)

// DeployOptions contains all configuration for deployment
type DeployOptions struct {
	ProjectID   string
	KeyspaceID  string
	DockerImage string
	Branch      string
	Commit      string
	Environment string
	RootKey     string
	APIBaseURL  string
}

var DeployFlags = []cli.Flag{
	// Required flags
	cli.String("project-id", "Project ID", cli.EnvVar("UNKEY_PROJECT_ID")),
	// Optional flags with defaults
	cli.String("keyspace-id", "Keyspace ID for API key authentication", cli.EnvVar("UNKEY_KEYSPACE_ID")),
	cli.String("branch", "Git branch", cli.Default(DefaultBranch)),
	cli.String("commit", "Git commit SHA"),
	cli.String("env", "Environment slug to deploy to", cli.Default(DefaultEnvironment)),
	// Authentication flag
	cli.String("root-key", "Root key for authentication", cli.EnvVar("UNKEY_ROOT_KEY")),
	// API configuration
	cli.String("api-base-url", "API base URL for local testing", cli.EnvVar("UNKEY_API_BASE_URL")),
}

// Cmd is the deploy command that deploys pre-built Docker images to Unkey infrastructure.
// It handles deployment lifecycle management with real-time status updates.
var Cmd = &cli.Command{
	Version:     "",
	Commands:    []*cli.Command{},
	Aliases:     []string{},
	Name:        "deploy",
	Usage:       "Deploy a pre-built Docker image",
	AcceptsArgs: true,
	Description: `Deploy a pre-built Docker image to Unkey infrastructure.

The deploy command handles the deployment lifecycle: from creating a deployment
to monitoring its status until it's ready. It automatically detects your Git
context for metadata.

USAGE:
  unkey deploy <docker-image> [flags]

DEPLOYMENT PROCESS:
1. Create deployment with pre-built Docker image
2. Monitor deployment status until active

EXAMPLES:
unkey deploy ghcr.io/user/app:v1.0.0 --project-id=proj_123
unkey deploy myregistry.io/app:latest --project-id=proj_123 --env=production`,
	Flags:  DeployFlags,
	Action: DeployAction,
}

func DeployAction(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args()
	if len(args) == 0 {
		return fmt.Errorf("docker image is required\n\nUsage: unkey deploy <docker-image> [flags]")
	}
	dockerImage := args[0]

	projectID := cmd.String("project-id")
	if projectID == "" {
		return fmt.Errorf("project ID is required (use --project-id flag or UNKEY_PROJECT_ID env var)")
	}

	opts := DeployOptions{
		ProjectID:   projectID,
		KeyspaceID:  cmd.String("keyspace-id"),
		DockerImage: dockerImage,
		Branch:      cmd.String("branch"),
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

	// Create deployment with pre-built Docker image
	terminal.StartSpinner("Creating deployment")
	deploymentID, err := controlPlane.CreateDeployment(ctx, opts.DockerImage)
	if err != nil {
		terminal.StopSpinner(errors.FormatError(err), false)
		// Don't return error it will just double print the error without formatting
		return nil
	}
	terminal.StopSpinner(fmt.Sprintf("Deployment created: %s", deploymentID), true)

	// Track final deployment for completion info
	var finalDeployment *components.V2DeployGetDeploymentResponseData

	// Start monitoring spinner
	terminal.StartSpinner("Deployment in progress")

	// Handle deployment status changes
	onStatusChange := func(event DeploymentStatusEvent) error {
		// nolint: exhaustive // We just need those two for now
		switch event.CurrentStatus {
		case components.StatusFailed:
			return handleDeploymentFailure(controlPlane, event.Deployment, terminal)
		case components.StatusReady:
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

func handleDeploymentFailure(controlPlane *ControlPlaneClient, deployment *components.V2DeployGetDeploymentResponseData, terminal *ui.UI) error {
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

	fmt.Printf("    Image: %s\n", opts.DockerImage)
	fmt.Printf("\n")
}

func printCompletionInfo(deployment *components.V2DeployGetDeploymentResponseData, env string) {
	if deployment == nil || deployment.GetID() == "" {
		fmt.Printf("✓ Deployment completed\n")
		return
	}

	caser := cases.Title(language.English)

	fmt.Println()
	fmt.Println("Deployment Complete")
	fmt.Printf("  Deployment ID: %s\n", deployment.GetID())
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
