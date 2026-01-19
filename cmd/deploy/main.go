package deploy

import (
	"context"
	"fmt"
	"strings"

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
	// Config directory flag (highest priority)
	cli.String("config", "Directory containing unkey.json config file"),
	// Init flag
	cli.Bool("init", "Initialize configuration file in the specified directory"),
	cli.Bool("force", "Force overwrite existing configuration file when using --init"),
	// Required flags (can be provided via config file)
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
	Usage:    "Deploy a new version or initialize configuration",
	Description: `Build and deploy a new version of your application, or initialize configuration.

The deploy command handles the complete deployment lifecycle: from building Docker images to deploying them on Unkey's infrastructure. It automatically detects your Git context, builds containers, and manages the deployment process with real-time status updates.

INITIALIZATION MODE:
Use --init to create a configuration template file. This generates an unkey.json file with your project settings, making future deployments simpler and more consistent across environments.

DEPLOYMENT PROCESS:
1. Load configuration from unkey.json or flags
2. Build Docker image from your application
3. Push image to container registry
4. Create deployment version on Unkey platform
5. Monitor deployment status until active

EXAMPLES:
unkey deploy --init                           # Initialize new project configuration
unkey deploy --init --config=./my-project    # Initialize with custom location
unkey deploy --init --force                  # Force overwrite existing configuration
unkey deploy                                 # Standard deployment (uses ./unkey.json)
unkey deploy --config=./production           # Deploy from specific config directory
unkey deploy --context=./api                 # Deploy with custom build context
unkey deploy --docker-image=ghcr.io/user/app:v1.0.0 # Deploy pre-built image`,
	Flags:  DeployFlags,
	Action: DeployAction,
}

func DeployAction(ctx context.Context, cmd *cli.Command) error {
	// Handle --init flag
	if cmd.Bool("init") {
		ui := NewUI()
		return handleInit(cmd, ui)
	}

	// Load configuration file
	configPath, err := getConfigPath(cmd.String("config"))
	if err != nil {
		return err
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Merge config with command flags (flags take precedence)
	finalConfig := cfg.mergeWithFlags(
		cmd.String("project-id"),
		cmd.String("keyspace-id"),
		cmd.String("context"),
	)

	// Validate that we have required fields
	if err := finalConfig.validate(); err != nil {
		return err // Clean error message already
	}

	opts := DeployOptions{
		KeyspaceID:  finalConfig.KeyspaceID,
		ProjectID:   finalConfig.ProjectID,
		Context:     finalConfig.Context,
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
	ui := NewUI()
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
		ui.StartSpinner("Creating deployment")
		deploymentID, err = controlPlane.CreateDeployment(ctx, "", opts.DockerImage)
		if err != nil {
			ui.StopSpinner("Failed to create deployment", false)
			ui.PrintErrorDetails(err.Error())
			return err
		}
		ui.StopSpinner(fmt.Sprintf("Deployment created: %s", deploymentID), true)
	} else {
		// Build from context
		ui.StartSpinner("Uploading build context")
		var buildContextPath string
		buildContextPath, err = controlPlane.UploadBuildContext(ctx, opts.Context)
		if err != nil {
			ui.StopSpinner("Failed to upload build context", false)
			ui.PrintErrorDetails(err.Error())
			return err
		}
		ui.StopSpinner(fmt.Sprintf("Build context uploaded: %s", buildContextPath), true)

		ui.StartSpinner("Creating deployment")
		deploymentID, err = controlPlane.CreateDeployment(ctx, buildContextPath, "")
		if err != nil {
			ui.StopSpinner("Failed to create deployment", false)
			ui.PrintErrorDetails(err.Error())
			return err
		}
		ui.StopSpinner(fmt.Sprintf("Deployment created: %s", deploymentID), true)
	}

	// Track final deployment for completion info
	var finalDeployment *ctrlv1.Deployment

	// Start monitoring spinner
	ui.StartSpinner("Deployment in progress")

	// Handle deployment status changes
	onStatusChange := func(event DeploymentStatusEvent) error {
		// nolint: exhaustive // We just need those two for now
		switch event.CurrentStatus {
		case ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_FAILED:
			return handleDeploymentFailure(controlPlane, event.Deployment, ui)
		case ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_READY:
			// Store deployment but don't print success, wait for polling to complete
			finalDeployment = event.Deployment
		}
		return nil
	}

	// Poll for deployment completion
	err = controlPlane.PollDeploymentStatus(ctx, logger, deploymentID, onStatusChange)
	if err != nil {
		ui.StopSpinner("Deployment failed", false)
		return err
	}

	// Print final success message only after all polling is complete
	if finalDeployment != nil {
		ui.StopSpinner("Deployment completed successfully", true)
		fmt.Printf("\n")
		printCompletionInfo(finalDeployment, opts.Environment)
		fmt.Printf("\n")
	}

	return nil
}

func handleDeploymentFailure(controlPlane *ControlPlaneClient, deployment *ctrlv1.Deployment, ui *UI) error {
	errorMsg := controlPlane.getFailureMessage(deployment)
	ui.StopSpinner("Deployment failed", false)
	ui.PrintErrorDetails(errorMsg)
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
