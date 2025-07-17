package deploy

import (
	"context"
	"errors"
	"fmt"

	"github.com/unkeyed/unkey/go/cmd/cli/cli"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/git"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Step predictor - maps current step message patterns to next expected steps
// Based on the actual workflow messages from version.go
// TODO: In the future directly get those from hydra
var stepSequence = map[string]string{
	"Version queued and ready to start":  "Downloading Docker image:",
	"Downloading Docker image:":          "Building rootfs from Docker image:",
	"Building rootfs from Docker image:": "Uploading rootfs image to storage",
	"Uploading rootfs image to storage":  "Creating VM for version:",
	"Creating VM for version:":           "VM booted successfully:",
	"VM booted successfully:":            "Assigned hostname:",
	"Assigned hostname:":                 "Version deployment completed successfully",
}

var (
	ErrDockerNotFound    = errors.New("docker command not found - please install Docker")
	ErrDockerBuildFailed = errors.New("docker build failed")
)

// DeployOptions contains all configuration for deployment
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

var DeployFlags = []cli.Flag{
	// Required flags
	cli.String("workspace-id", "Workspace ID", "", "UNKEY_WORKSPACE_ID", true),
	cli.String("project-id", "Project ID", "", "UNKEY_PROJECT_ID", true),

	// Optional flags with defaults
	cli.String("context", "Docker context path", ".", "", false),
	cli.String("branch", "Git branch", "main", "", false),
	cli.String("docker-image", "Pre-built docker image", "", "", false),
	cli.String("dockerfile", "Path to Dockerfile", "Dockerfile", "", false),
	cli.String("commit", "Git commit SHA", "", "", false),
	cli.String("registry", "Docker registry", "ghcr.io/unkeyed/deploy", "UNKEY_DOCKER_REGISTRY", false),
	cli.Bool("skip-push", "Skip pushing to registry (for local testing)", "", false),

	// Control plane flags (internal)
	cli.String("control-plane-url", "Control plane URL", "http://localhost:7091", "", false),
	cli.String("auth-token", "Control plane auth token", "ctrl-secret-token", "", false),
}

// Command defines the deploy CLI command
var Command = &cli.Command{
	Name:  "deploy",
	Usage: "Deploy a new version",
	Description: `Build and deploy a new version of your application.
Builds a Docker image from the specified context and
deploys it to the Unkey platform.

EXAMPLES:
    # Basic deployment
    unkey deploy \
      --workspace-id=ws_4QgQsKsKfdm3nGeC \
      --project-id=proj_9aiaks2dzl6mcywnxjf \
      --context=./demo_api

    # Deploy with your own registry
    unkey deploy \
      --workspace-id=ws_4QgQsKsKfdm3nGeC \
      --project-id=proj_9aiaks2dzl6mcywnxjf \
      --registry=docker.io/mycompany/myapp

    # Local development (skip push)
    unkey deploy \
      --workspace-id=ws_4QgQsKsKfdm3nGeC \
      --project-id=proj_9aiaks2dzl6mcywnxjf \
      --skip-push

    # Deploy pre-built image
    unkey deploy \
      --workspace-id=ws_4QgQsKsKfdm3nGeC \
      --project-id=proj_9aiaks2dzl6mcywnxjf \
      --docker-image=ghcr.io/user/app:v1.0.0`,
	Flags:  DeployFlags,
	Action: DeployAction,
}

func DeployAction(ctx context.Context, cmd *cli.Command) error {
	opts := &DeployOptions{
		WorkspaceID:     cmd.String("workspace-id"),
		ProjectID:       cmd.String("project-id"),
		Context:         cmd.String("context"),
		Branch:          cmd.String("branch"),
		DockerImage:     cmd.String("docker-image"),
		Dockerfile:      cmd.String("dockerfile"),
		Commit:          cmd.String("commit"),
		Registry:        cmd.String("registry"),
		SkipPush:        cmd.Bool("skip-push"),
		ControlPlaneURL: cmd.String("control-plane-url"),
		AuthToken:       cmd.String("auth-token"),
	}

	return executeDeploy(ctx, opts)
}

// Updated executeDeploy function - remove global spinner for deployment steps
func executeDeploy(ctx context.Context, opts *DeployOptions) error {
	ui := NewUI()
	logger := logging.New()
	gitInfo := git.GetInfo()

	if opts.Branch == "main" && gitInfo.IsRepo && gitInfo.Branch != "" {
		opts.Branch = gitInfo.Branch
	}
	if opts.Commit == "" && gitInfo.CommitSHA != "" {
		opts.Commit = gitInfo.CommitSHA
	}

	fmt.Printf("Unkey Deploy Progress\n")
	fmt.Printf("──────────────────────────────────────────────────\n")
	printSourceInfo(opts, gitInfo)

	ui.Print("Preparing deployment")

	var dockerImage string

	if opts.DockerImage == "" {
		if !isDockerAvailable() {
			ui.PrintError("Docker not found - please install Docker")
			ui.PrintErrorDetails(ErrDockerNotFound.Error())
			return nil
		}
		imageTag := generateImageTag(opts, gitInfo)
		dockerImage = fmt.Sprintf("%s:%s", opts.Registry, imageTag)

		ui.Print(fmt.Sprintf("Building image: %s", dockerImage))
		if err := buildImage(ctx, opts, dockerImage); err != nil {
			ui.PrintError("Docker build failed")
			ui.PrintErrorDetails(err.Error())
			return nil
		}
		ui.PrintSuccess("Image built successfully")
	} else {
		dockerImage = opts.DockerImage
		ui.Print("Using pre-built Docker image")
	}

	if !opts.SkipPush && opts.DockerImage == "" {
		ui.Print("Pushing to registry")
		if err := pushImage(ctx, dockerImage, opts.Registry); err != nil {
			ui.PrintError("Push failed but continuing deployment")
			ui.PrintErrorDetails(err.Error())
			// INFO: For now we are ignoring registry push because everyone one is working locally,
			// omit this when comments and put the return nil back when going to prod
			// return nil
		} else {
			ui.PrintSuccess("Image pushed successfully")
		}
	} else if opts.SkipPush {
		ui.Print("Skipping registry push")
	}

	ui.Print("Creating deployment")

	controlPlane := NewControlPlaneClient(opts)
	versionId, err := controlPlane.CreateVersion(ctx, dockerImage)
	if err != nil {
		ui.PrintError("Failed to create version")
		ui.PrintErrorDetails(err.Error())
		return nil
	}

	ui.PrintSuccess(fmt.Sprintf("Version created: %s", versionId))

	onStatusChange := func(event VersionStatusEvent) error {
		if event.CurrentStatus == ctrlv1.VersionStatus_VERSION_STATUS_FAILED {
			return handleVersionFailure(controlPlane, event.Version, ui)
		}
		return nil
	}

	onStepUpdate := func(event VersionStepEvent) error {
		return handleStepUpdate(event, ui)
	}

	err = controlPlane.PollVersionStatus(ctx, logger, versionId, onStatusChange, onStepUpdate)
	if err != nil {
		// Complete any running step spinner on error
		ui.CompleteCurrentStep("Deployment failed", false)
		return err
	}

	// Complete final step if still spinning
	ui.CompleteCurrentStep("Version deployment completed successfully", true)
	ui.PrintSuccess("Deployment completed successfully")

	fmt.Printf("\n")
	printCompletionInfo(opts, gitInfo, versionId)
	fmt.Printf("\n")

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

func handleStepUpdate(event VersionStepEvent, ui *UI) error {
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

func handleVersionFailure(controlPlane *ControlPlaneClient, version *ctrlv1.Version, ui *UI) error {
	errorMsg := controlPlane.getFailureMessage(version)
	ui.CompleteCurrentStep("Deployment failed", false)
	ui.PrintError("Deployment failed")
	ui.PrintErrorDetails(errorMsg)
	return fmt.Errorf("deployment failed: %s", errorMsg)
}

func printSourceInfo(opts *DeployOptions, gitInfo git.Info) {
	fmt.Printf("Source Information:\n")
	fmt.Printf("    Branch: %s\n", opts.Branch)

	if gitInfo.IsRepo && gitInfo.CommitSHA != "" {

		commitInfo := gitInfo.ShortSHA
		if gitInfo.IsDirty {
			commitInfo += " (dirty)"
		}
		fmt.Printf("    Commit: %s\n", commitInfo)
	}

	fmt.Printf("    Context: %s\n", opts.Context)

	if opts.DockerImage != "" {
		fmt.Printf("    Image: %s\n", opts.DockerImage)
	}

	fmt.Printf("\n")
}

func printCompletionInfo(opts *DeployOptions, gitInfo git.Info, versionId string) {
	if versionId == "" || opts.WorkspaceID == "" || opts.Branch == "" {
		fmt.Printf("✓ Deployment completed\n")
		return
	}

	fmt.Printf("Deployment Summary:\n")
	fmt.Printf("    Version: %s\n", versionId)
	fmt.Printf("    Status: Ready\n")
	fmt.Printf("    Environment: Production\n")

	identifier := versionId
	if gitInfo.ShortSHA != "" {
		identifier = gitInfo.ShortSHA
	}

	domain := fmt.Sprintf("https://%s-%s-%s.unkey.app", opts.Branch, identifier, opts.WorkspaceID)
	fmt.Printf("    URL: %s\n", domain)
}
