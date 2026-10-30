package deploy

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/cmd/deploy/internal/errors"
	"github.com/unkeyed/unkey/cmd/deploy/internal/ui"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/git"
)

const (
	// Default configuration values
	DefaultBranch      = "main"
	DefaultEnvironment = "preview"
)

// DeployOptions contains all configuration for deployment
type DeployOptions struct {
	Project     string
	App         string
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
	cli.String("project", "Project slug", cli.Required(), cli.EnvVar("UNKEY_PROJECT")),
	cli.String("app", "App slug within the project", cli.EnvVar("UNKEY_APP"), cli.Default("default")),
	// Optional flags with defaults
	cli.String("keyspace-id", "Keyspace ID for API key authentication", cli.EnvVar("UNKEY_KEYSPACE_ID")),
	cli.String("branch", "Git branch", cli.Default(DefaultBranch)),
	cli.String("commit", "Git commit SHA"),
	cli.String("env", "Environment slug to deploy to", cli.Default(DefaultEnvironment)),
	// Authentication flag
	cli.String("root-key", "Root key for authentication. Falls back to UNKEY_ROOT_KEY, then unkey login.", cli.EnvVar("UNKEY_ROOT_KEY")),
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
unkey deploy ghcr.io/user/app:v1.0.0 --project=my-project
unkey deploy myregistry.io/app:latest --project=my-project --app=api --env=production`,
	Flags:  DeployFlags,
	Action: DeployAction,
}

func DeployAction(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args()
	if len(args) == 0 {
		return fmt.Errorf("docker image is required\n\nUsage: unkey deploy <docker-image> [flags]")
	}
	dockerImage := args[0]

	credential, err := util.Credential(ctx, cmd)
	if err != nil {
		return fmt.Errorf("%s", ui.Text(err.Error()))
	}
	apiURL := credential.APIURL
	if apiURL == "" {
		apiURL = cmd.String("api-base-url")
	}
	opts := DeployOptions{
		Project:     cmd.String("project"),
		App:         cmd.String("app"),
		KeyspaceID:  cmd.String("keyspace-id"),
		DockerImage: dockerImage,
		Branch:      cmd.String("branch"),
		Commit:      cmd.String("commit"),
		Environment: cmd.String("env"),
		RootKey:     credential.Key,
		APIBaseURL:  apiURL,
	}

	return executeDeploy(ctx, opts)
}

func executeDeploy(ctx context.Context, opts DeployOptions) error {
	terminal := ui.NewUI()
	defer terminal.Close()
	gitInfo := git.GetInfo()

	if opts.Branch == DefaultBranch && gitInfo.IsRepo && gitInfo.Branch != "" {
		opts.Branch = gitInfo.Branch
	}
	if opts.Commit == "" && gitInfo.CommitSHA != "" {
		opts.Commit = gitInfo.CommitSHA
	}

	printContext(opts)

	controlPlane := NewControlPlaneClient(opts)
	terminal.StartSpinner("Creating deployment")
	deploymentID, err := controlPlane.CreateDeployment(ctx, opts.DockerImage)
	if err != nil {
		return fmt.Errorf("failed to create deployment: %s", errors.FormatError(err))
	}
	terminal.StopSpinner("Deployment created")

	var finalDeployment *components.V2DeployGetDeploymentResponseData
	terminal.StartSpinner("Waiting for deployment")
	onStatusChange := func(event DeploymentStatusEvent) error {
		if event.CurrentStatus == components.V2DeployGetDeploymentResponseDataStatusFailed {
			return fmt.Errorf("deployment failed: %s", ui.Text(controlPlane.getFailureMessage(event.Deployment)))
		}
		if event.CurrentStatus == components.V2DeployGetDeploymentResponseDataStatusReady {
			finalDeployment = event.Deployment
		}
		return nil
	}

	err = controlPlane.PollDeploymentStatus(ctx, deploymentID, onStatusChange)
	if err != nil {
		return err
	}

	if finalDeployment != nil {
		terminal.StopSpinner("Deployment ready.")
		printCompletionInfo(finalDeployment, opts.Environment)
	}

	return nil
}

func printContext(opts DeployOptions) {
	fmt.Fprintf(os.Stderr, "Project: %s  App: %s  Environment: %s\n", ui.Text(opts.Project), ui.Text(opts.App), ui.Text(opts.Environment))
	fmt.Fprintf(os.Stderr, "Image: %s\n", ui.Text(opts.DockerImage))
}

func printCompletionInfo(deployment *components.V2DeployGetDeploymentResponseData, env string) {
	fmt.Printf("ID: %s\n", ui.Text(deployment.GetID()))
	fmt.Printf("Status: Ready\n")
	fmt.Printf("Environment: %s\n", ui.Text(env))

	hostnames := deployment.GetHostnames()
	if len(hostnames) > 0 {
		for _, hostname := range hostnames {
			if strings.HasPrefix(hostname, "localhost:") {
				fmt.Printf("URL: http://%s\n", ui.Text(hostname))
			} else {
				fmt.Printf("URL: https://%s\n", ui.Text(hostname))
			}
		}
	} else {
		fmt.Println("URLs: none assigned")
	}
}
