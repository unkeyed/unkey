package deploy

import (
	"context"
	"errors"
	"fmt"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/git"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

const DEBUG_DELAY = 250

var (
	ErrDockerNotFound    = errors.New("docker command not found - please install Docker")
	ErrDockerBuildFailed = errors.New("docker build failed")
	ErrDockerPushFailed  = errors.New("docker push failed")
	ErrInvalidImageTag   = errors.New("invalid image tag generated")
)

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

func Deploy(ctx context.Context, args []string, env map[string]string) error {
	if len(args) < 1 {
		PrintDeployHelp()
		return fmt.Errorf("deploy command requires arguments")
	}

	opts, err := parseDeployFlags(args, env)
	if err != nil {
		return err
	}

	return executeDeploy(ctx, opts)
}

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
			return ErrDockerNotFound
		}
		imageTag := generateImageTag(opts, gitInfo)
		dockerImage = fmt.Sprintf("%s:%s", opts.Registry, imageTag)
	} else {
		dockerImage = opts.DockerImage
	}

	if opts.DockerImage == "" {
		ui.Print(fmt.Sprintf("Building image: %s", dockerImage))
		if err := buildImage(ctx, opts, dockerImage); err != nil {
			ui.PrintError("Docker build failed")
			return err
		}
		ui.PrintSuccess("Image built successfully")
	} else {
		ui.Print("Using pre-built Docker image")
	}

	if !opts.SkipPush && opts.DockerImage == "" {
		ui.Print("Pushing to registry")
		if err := pushImage(ctx, dockerImage, opts.Registry); err != nil {
			ui.PrintError("Push failed but continuing deployment")
			ui.PrintErrorDetails(err.Error())
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
		return err
	}

	ui.PrintSuccess(fmt.Sprintf("Version created: %s", versionId))

	ui.StartSpinner("Deploying to Unkey...")

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
		ui.StopSpinner("Deployment failed", false)
		return err
	}

	ui.StopSpinner("Deployment completed successfully", true)

	fmt.Printf("\n")
	printCompletionInfo(opts, gitInfo, versionId)
	fmt.Printf("\n")

	return nil
}

func handleVersionFailure(controlPlane *ControlPlaneClient, version *ctrlv1.Version, ui *UI) error {
	errorMsg := controlPlane.getFailureMessage(version)
	ui.PrintError("Deployment failed")
	ui.PrintErrorDetails(errorMsg)
	return fmt.Errorf("deployment failed: %s", errorMsg)
}

func handleStepUpdate(event VersionStepEvent, ui *UI) error {
	ui.mu.Lock()
	if ui.spinning {
		ui.spinning = false
		fmt.Print("\r\033[K")
	}
	ui.mu.Unlock()

	step := event.Step

	if step.GetErrorMessage() != "" {
		ui.PrintStepError(step.GetMessage())
		ui.PrintErrorDetails(step.GetErrorMessage())
		return fmt.Errorf("deployment failed: %s", step.GetErrorMessage())
	}

	if step.GetMessage() != "" {
		ui.PrintStepSuccess(step.GetMessage())

		if DEBUG_DELAY > 0 {
			time.Sleep(DEBUG_DELAY * time.Millisecond)
		}
	}

	return nil
}

func printSourceInfo(opts *DeployOptions, gitInfo git.Info) {
	fmt.Printf("Source Information:\n")
	fmt.Printf("    Branch: %s\n", opts.Branch)

	if gitInfo.IsRepo && gitInfo.CommitSHA != "" {
		shortSHA := gitInfo.CommitSHA
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}
		commitInfo := shortSHA
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
