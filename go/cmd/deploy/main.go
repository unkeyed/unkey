package deploy

import (
	"context"
	"fmt"
	"strings"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/git"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

const (
	// Default values
	DefaultBranch          = "main"
	DefaultDockerfile      = "Dockerfile"
	DefaultRegistry        = "ghcr.io/unkeyed/deploy"
	DefaultControlPlaneURL = "http://localhost:7091"
	DefaultAuthToken       = "ctrl-secret-token"
	DefaultEnvironment     = "Production"

	// Environment variables
	EnvWorkspaceID = "UNKEY_WORKSPACE_ID"
	EnvKeyspaceID  = "UNKEY_KEYSPACE_ID"
	EnvRegistry    = "UNKEY_REGISTRY"

	// URL prefixes
	HTTPSPrefix     = "https://"
	HTTPPrefix      = "http://"
	LocalhostPrefix = "localhost:"

	// UI Messages
	HeaderTitle     = "Unkey Deploy Progress"
	HeaderSeparator = "──────────────────────────────────────────────────"

	// Step messages
	MsgPreparingDeployment      = "Preparing deployment"
	MsgCreatingDeployment       = "Creating deployment"
	MsgSkippingRegistryPush     = "Skipping registry push"
	MsgUsingPreBuiltImage       = "Using pre-built Docker image"
	MsgPushingToRegistry        = "Pushing to registry"
	MsgImageBuiltSuccessfully   = "Image built successfully"
	MsgImagePushedSuccessfully  = "Image pushed successfully"
	MsgPushFailedContinuing     = "Push failed but continuing deployment"
	MsgDockerNotFound           = "Docker not found - please install Docker"
	MsgFailedToCreateDeployment = "Failed to create deployment"
	MsgDeploymentFailed         = "Deployment failed"
	MsgDeploymentCompleted      = "Deployment completed successfully"
	MsgDeploymentStepCompleted  = "Deployment step completed successfully"

	// Source info labels
	LabelBranch  = "Branch"
	LabelCommit  = "Commit"
	LabelContext = "Context"
	LabelImage   = "Image"

	// Completion info labels
	CompletionTitle        = "Deployment Complete"
	CompletionDeploymentID = "Deployment ID"
	CompletionStatus       = "Status"
	CompletionEnvironment  = "Environment"
	CompletionDomains      = "Domains"
	CompletionReady        = "Ready"
	CompletionNoHostnames  = "No hostnames assigned"

	// Git status
	GitDirtyMarker = " (dirty)"
)

// Step predictor - maps current step message patterns to next expected steps
var stepSequence = map[string]string{
	"Version queued and ready to start":  "Downloading Docker image:",
	"Downloading Docker image:":          "Building rootfs from Docker image:",
	"Building rootfs from Docker image:": "Uploading rootfs image to storage",
	"Uploading rootfs image to storage":  "Creating VM for version:",
	"Creating VM for deployment:":        "VM booted successfully:",
	"VM booted successfully:":            "Assigned hostname:",
	"Assigned hostname:":                 MsgDeploymentStepCompleted,
}

// DeployOptions contains all configuration for deployment
type DeployOptions struct {
	WorkspaceID     string
	ProjectID       string
	KeyspaceID      string
	Context         string
	Branch          string
	DockerImage     string
	Dockerfile      string
	Commit          string
	Registry        string
	Environment     string
	SkipPush        bool
	Verbose         bool
	ControlPlaneURL string
	AuthToken       string
	APIKey          string
	Linux           bool
}

var DeployFlags = []cli.Flag{
	// Config directory flag (highest priority)
	cli.String("config", "Directory containing unkey.json config file"),
	// Init flag
	cli.Bool("init", "Initialize configuration file in the specified directory"),
	cli.Bool("force", "Force overwrite existing configuration file when using --init"),
	// Required flags (can be provided via config file)
	cli.String("workspace-id", "Workspace ID", cli.EnvVar(EnvWorkspaceID)),
	cli.String("project-id", "Project ID", cli.EnvVar("UNKEY_PROJECT_ID")),
	cli.String("keyspace-id", "Keyspace ID for API key authentication", cli.EnvVar(EnvKeyspaceID)),
	// Optional flags with defaults
	cli.String("context", "Build context path"),
	cli.String("branch", "Git branch", cli.Default(DefaultBranch)),
	cli.String("docker-image", "Pre-built docker image"),
	cli.String("dockerfile", "Path to Dockerfile", cli.Default(DefaultDockerfile)),
	cli.String("commit", "Git commit SHA"),
	cli.String("registry", "Container registry",
		cli.Default(DefaultRegistry),
		cli.EnvVar(EnvRegistry)),
	cli.String("env", "Environment slug to deploy to", cli.Default("preview")),
	cli.Bool("skip-push", "Skip pushing to registry (for local testing)"),
	cli.Bool("verbose", "Show detailed output for build and deployment operations"),
	cli.Bool("linux", "Build Docker image for linux/amd64 platform (for deployment to cloud clusters)"),
	// Control plane flags (internal)
	cli.String("control-plane-url", "Control plane URL", cli.Default(DefaultControlPlaneURL)),
	cli.String("auth-token", "Control plane auth token", cli.Default(DefaultAuthToken)),
	cli.String("api-key", "API key for ctrl service authentication", cli.EnvVar("API_KEY")),
}

// WARNING: Changing the "Description" part will also affect generated MDX.
// Cmd defines the deploy CLI command
var Cmd = &cli.Command{
	Name:  "deploy",
	Usage: "Deploy a new version or initialize configuration",
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
unkey deploy --workspace-id=ws_production_123 # Override workspace from config file
unkey deploy --context=./api                 # Deploy with custom build context
unkey deploy --skip-push                     # Local development (build only, no push)
unkey deploy --docker-image=ghcr.io/user/app:v1.0.0 # Deploy pre-built image
unkey deploy --verbose                       # Verbose output for debugging`,
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
		cmd.String("workspace-id"),
		cmd.String("project-id"),
		cmd.String("keyspace-id"),
		cmd.String("context"),
	)

	// Validate that we have required fields
	if err := finalConfig.validate(); err != nil {
		return err // Clean error message already
	}

	opts := DeployOptions{
		WorkspaceID:     finalConfig.WorkspaceID,
		KeyspaceID:      finalConfig.KeyspaceID,
		ProjectID:       finalConfig.ProjectID,
		Context:         finalConfig.Context,
		Branch:          cmd.String("branch"),
		DockerImage:     cmd.String("docker-image"),
		Dockerfile:      cmd.String("dockerfile"),
		Commit:          cmd.String("commit"),
		Registry:        cmd.String("registry"),
		Environment:     cmd.String("env"),
		SkipPush:        cmd.Bool("skip-push"),
		Verbose:         cmd.Bool("verbose"),
		ControlPlaneURL: cmd.String("control-plane-url"),
		AuthToken:       cmd.String("auth-token"),
		APIKey:          cmd.String("api-key"),
		Linux:           cmd.Bool("linux"),
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
	fmt.Printf("%s\n", HeaderTitle)
	fmt.Printf("%s\n", HeaderSeparator)
	printSourceInfo(opts, gitInfo)

	ui.Print(MsgPreparingDeployment)

	var dockerImage string

	// Build or use pre-built Docker image
	if opts.DockerImage == "" {
		// Check Docker availability using updated function
		if err := isDockerAvailable(); err != nil {
			ui.PrintError(MsgDockerNotFound)
			return err
		}

		// Generate image tag and full image name
		imageTag := generateImageTag(opts, gitInfo)
		dockerImage = fmt.Sprintf("%s:%s", opts.Registry, imageTag)

		ui.Print(fmt.Sprintf("Building image: %s", dockerImage))

		if err := buildImage(ctx, opts, dockerImage, ui); err != nil {
			// Don't print additional error, buildImage already reported it with proper hierarchy
			return err
		}
		ui.PrintSuccess(MsgImageBuiltSuccessfully)
	} else {
		dockerImage = opts.DockerImage
		ui.Print(MsgUsingPreBuiltImage)
	}

	// Push to registry, unless skipped or using pre-built image
	if !opts.SkipPush && opts.DockerImage == "" {
		ui.Print(MsgPushingToRegistry)
		if err := pushImage(ctx, dockerImage, opts.Registry); err != nil {
			ui.PrintError(MsgPushFailedContinuing)
			ui.PrintErrorDetails(err.Error())
			// NOTE: Currently ignoring push failures for local development
			// For production deployment, uncomment the line below:
			// return err
		} else {
			ui.PrintSuccess(MsgImagePushedSuccessfully)
		}
	} else if opts.SkipPush {
		ui.Print(MsgSkippingRegistryPush)
	}

	// Create deployment
	ui.Print(MsgCreatingDeployment)
	controlPlane := NewControlPlaneClient(opts)
	deploymentId, err := controlPlane.CreateDeployment(ctx, dockerImage)
	if err != nil {
		ui.PrintError(MsgFailedToCreateDeployment)
		ui.PrintErrorDetails(err.Error())
		return err
	}
	ui.PrintSuccess(fmt.Sprintf("Deployment created: %s", deploymentId))

	// Track final deployment for completion info
	var finalDeployment *ctrlv1.Deployment

	// Handle deployment status changes
	onStatusChange := func(event DeploymentStatusEvent) error {
		switch event.CurrentStatus {
		case ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_FAILED:
			return handleDeploymentFailure(controlPlane, event.Deployment, ui)
		case ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_READY:
			// Store deployment but don't print success, wait for polling to complete
			finalDeployment = event.Deployment
		}
		return nil
	}

	// Handle deployment step updates
	onStepUpdate := func(event DeploymentStepEvent) error {
		return handleStepUpdate(event, ui)
	}

	// Poll for deployment completion
	err = controlPlane.PollDeploymentStatus(ctx, logger, deploymentId, onStatusChange, onStepUpdate)
	if err != nil {
		ui.CompleteCurrentStep(MsgDeploymentFailed, false)
		return err
	}

	// Print final success message only after all polling is complete
	if finalDeployment != nil {
		ui.CompleteCurrentStep(MsgDeploymentStepCompleted, true)
		ui.PrintSuccess(MsgDeploymentCompleted)
		fmt.Printf("\n")
		printCompletionInfo(finalDeployment)
		fmt.Printf("\n")
	}

	return nil
}

func getNextStepMessage(currentMessage string) string {
	// Check if current message starts with any known step pattern
	for key, next := range stepSequence {
		if len(currentMessage) >= len(key) && currentMessage[:len(key)] == key {
			return next
		}
	}
	return ""
}

func handleStepUpdate(event DeploymentStepEvent, ui *UI) error {
	step := event.Step

	if step.GetErrorMessage() != "" {
		ui.CompleteCurrentStep(step.GetMessage(), false)
		ui.PrintErrorDetails(step.GetErrorMessage())
		return fmt.Errorf("deployment failed: %s", step.GetErrorMessage())
	}

	if step.GetMessage() != "" {
		message := step.GetMessage()
		nextStep := getNextStepMessage(message)

		if !ui.stepSpinning {
			// First step - start spinner, then complete and start next
			ui.StartStepSpinner(message)
			ui.CompleteStepAndStartNext(message, nextStep)
		} else {
			// Complete current step and start next
			ui.CompleteStepAndStartNext(message, nextStep)
		}
	}

	return nil
}

func handleDeploymentFailure(controlPlane *ControlPlaneClient, deployment *ctrlv1.Deployment, ui *UI) error {
	errorMsg := controlPlane.getFailureMessage(deployment)
	ui.CompleteCurrentStep(MsgDeploymentFailed, false)
	ui.PrintError(MsgDeploymentFailed)
	ui.PrintErrorDetails(errorMsg)
	return fmt.Errorf("deployment failed: %s", errorMsg)
}

func printSourceInfo(opts DeployOptions, gitInfo git.Info) {
	fmt.Printf("Source Information:\n")
	fmt.Printf("    %s: %s\n", LabelBranch, opts.Branch)

	if gitInfo.IsRepo && gitInfo.CommitSHA != "" {
		commitInfo := gitInfo.ShortSHA
		if gitInfo.IsDirty {
			commitInfo += GitDirtyMarker
		}
		fmt.Printf("    %s: %s\n", LabelCommit, commitInfo)
	}

	fmt.Printf("    %s: %s\n", LabelContext, opts.Context)

	if opts.DockerImage != "" {
		fmt.Printf("    %s: %s\n", LabelImage, opts.DockerImage)
	}

	fmt.Printf("\n")
}

func printCompletionInfo(deployment *ctrlv1.Deployment) {
	if deployment == nil || deployment.GetId() == "" {
		fmt.Printf("✓ Deployment completed\n")
		return
	}

	fmt.Println()
	fmt.Println(CompletionTitle)
	fmt.Printf("  %s: %s\n", CompletionDeploymentID, deployment.GetId())
	fmt.Printf("  %s: %s\n", CompletionStatus, CompletionReady)
	fmt.Printf("  %s: %s\n", CompletionEnvironment, DefaultEnvironment)

	fmt.Println()
	fmt.Println(CompletionDomains)

	hostnames := deployment.GetHostnames()
	if len(hostnames) > 0 {
		for _, hostname := range hostnames {
			if strings.HasPrefix(hostname, LocalhostPrefix) {
				fmt.Printf("  %s%s\n", HTTPPrefix, hostname)
			} else {
				fmt.Printf("  %s%s\n", HTTPSPrefix, hostname)
			}
		}
	} else {
		fmt.Printf("  %s\n", CompletionNoHostnames)
	}
}
