package deploy

import (
	"context"
	"fmt"
	"strings"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/git"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	// Default values
	DefaultBranch          = "main"
	DefaultDockerfile      = "Dockerfile"
	DefaultRegistry        = "ghcr.io/unkeyed/deploy"
	DefaultControlPlaneURL = "https://ctrl.unkey.cloud"
	DefaultAuthToken       = "ctrl-secret-token"
	DefaultEnvironment     = "preview"

	// Environment variables
	EnvKeyspaceID = "UNKEY_KEYSPACE_ID"
	EnvRegistry   = "UNKEY_REGISTRY"

	// URL prefixes
	HTTPSPrefix     = "https://"
	HTTPPrefix      = "http://"
	LocalhostPrefix = "localhost:"

	// UI Messages
	HeaderTitle     = "Unkey Deploy Progress"
	HeaderSeparator = "──────────────────────────────────────────────────"

	// Step messages
	MsgPreparingDeployment      = "Preparing deployment"
	MsgUploadingBuildContext    = "Uploading build context"
	MsgBuildContextUploaded     = "Build context uploaded"
	MsgCreatingDeployment       = "Creating deployment"
	MsgDeploymentCreated        = "Deployment created"
	MsgFailedToUploadContext    = "Failed to upload build context"
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

// DeployOptions contains all configuration for deployment
type DeployOptions struct {
	ProjectID       string
	KeyspaceID      string
	Context         string
	DockerImage     string
	Branch          string
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
	cli.String("project-id", "Project ID", cli.EnvVar("UNKEY_PROJECT_ID")),
	cli.String("keyspace-id", "Keyspace ID for API key authentication", cli.EnvVar(EnvKeyspaceID)),
	// Optional flags with defaults
	cli.String("context", "Build context path", cli.Default(".")),
	cli.String("branch", "Git branch", cli.Default(DefaultBranch)),
	cli.String("docker-image", "Pre-built docker image"),
	cli.String("dockerfile", "Path to Dockerfile", cli.Default(DefaultDockerfile)),
	cli.String("commit", "Git commit SHA"),
	cli.String("registry", "Container registry",
		cli.Default(DefaultRegistry),
		cli.EnvVar(EnvRegistry)),
	cli.String("env", "Environment slug to deploy to", cli.Default(DefaultEnvironment)),
	cli.Bool("skip-push", "Skip pushing to registry (for local testing)"),
	cli.Bool("verbose", "Show detailed output for build and deployment operations"),
	cli.Bool("linux", "Build Docker image for linux/amd64 platform (for deployment to cloud clusters)", cli.Default(true)),
	// Control plane flags (internal)
	cli.String("control-plane-url", "Control plane URL", cli.Default(DefaultControlPlaneURL)),
	cli.String("auth-token", "Control plane auth token", cli.Default(DefaultAuthToken)),
	cli.String("api-key", "API key for ctrl service authentication", cli.EnvVar("API_KEY")),
}

// WARNING: Changing the "Description" part will also affect generated MDX.
// Cmd defines the deploy CLI command
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
		cmd.String("project-id"),
		cmd.String("keyspace-id"),
		cmd.String("context"),
	)

	// Validate that we have required fields
	if err := finalConfig.validate(); err != nil {
		return err // Clean error message already
	}

	opts := DeployOptions{
		KeyspaceID:      finalConfig.KeyspaceID,
		ProjectID:       finalConfig.ProjectID,
		Context:         finalConfig.Context,
		DockerImage:     cmd.String("docker-image"),
		Branch:          cmd.String("branch"),
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

	controlPlane := NewControlPlaneClient(opts)

	var deploymentID string
	var err error

	// Determine deployment source: prebuilt image or build from context
	if opts.DockerImage != "" {
		// Use prebuilt Docker image
		ui.Print(MsgCreatingDeployment)
		deploymentID, err = controlPlane.CreateDeployment(ctx, "", opts.DockerImage)
		if err != nil {
			ui.PrintError(MsgFailedToCreateDeployment)
			ui.PrintErrorDetails(err.Error())
			return err
		}
		ui.PrintSuccess(fmt.Sprintf("%s: %s", MsgDeploymentCreated, deploymentID))
	} else {
		// Build from context
		ui.Print(MsgUploadingBuildContext)
		buildContextPath, err := controlPlane.UploadBuildContext(ctx, opts.Context)
		if err != nil {
			ui.PrintError(MsgFailedToUploadContext)
			ui.PrintErrorDetails(err.Error())
			return err
		}
		ui.PrintSuccess(fmt.Sprintf("%s: %s", MsgBuildContextUploaded, buildContextPath))

		ui.Print(MsgCreatingDeployment)
		deploymentID, err = controlPlane.CreateDeployment(ctx, buildContextPath, "")
		if err != nil {
			ui.PrintError(MsgFailedToCreateDeployment)
			ui.PrintErrorDetails(err.Error())
			return err
		}
		ui.PrintSuccess(fmt.Sprintf("%s: %s", MsgDeploymentCreated, deploymentID))
	}

	// Track final deployment for completion info
	var finalDeployment *ctrlv1.Deployment

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
		ui.CompleteCurrentStep(MsgDeploymentFailed, false)
		return err
	}

	// Print final success message only after all polling is complete
	if finalDeployment != nil {
		ui.CompleteCurrentStep(MsgDeploymentStepCompleted, true)
		ui.PrintSuccess(MsgDeploymentCompleted)
		fmt.Printf("\n")
		printCompletionInfo(finalDeployment, opts.Environment)
		fmt.Printf("\n")
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

	if opts.DockerImage != "" {
		fmt.Printf("    %s: %s\n", LabelImage, opts.DockerImage)
	} else {
		fmt.Printf("    %s: %s\n", LabelContext, opts.Context)
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
	fmt.Println(CompletionTitle)
	fmt.Printf("  %s: %s\n", CompletionDeploymentID, deployment.GetId())
	fmt.Printf("  %s: %s\n", CompletionStatus, CompletionReady)
	fmt.Printf("  %s: %s\n", CompletionEnvironment, caser.String(env))

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
