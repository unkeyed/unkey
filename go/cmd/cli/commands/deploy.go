package commands

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/git"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

var (
	ErrDockerNotFound    = errors.New("docker command not found - please install Docker")
	ErrDockerBuildFailed = errors.New("docker build failed")
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
	opts, err := parseDeployFlags("deploy", args, env)
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

	// Create control plane client and deploy
	if err := deployToControlPlane(ctx, logger, opts, dockerImage); err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	return nil
}

func deployToControlPlane(ctx context.Context, logger logging.Logger, opts *DeployOptions, dockerImage string) error {
	// Create control plane client
	httpClient := &http.Client{}
	client := ctrlv1connect.NewVersionServiceClient(httpClient, opts.ControlPlaneURL)

	// Create version request
	createReq := connect.NewRequest(&ctrlv1.CreateVersionRequest{
		WorkspaceId:    opts.WorkspaceID,
		ProjectId:      opts.ProjectID,
		Branch:         opts.Branch,
		SourceType:     ctrlv1.SourceType_SOURCE_TYPE_CLI_UPLOAD,
		GitCommitSha:   opts.Commit,
		EnvironmentId:  "env_prod",
		DockerImageTag: dockerImage,
	})

	// Add auth header
	createReq.Header().Set("Authorization", "Bearer "+opts.AuthToken)

	// Call the API
	createResp, err := client.CreateVersion(ctx, createReq)
	if err != nil {
		// Check if it's a connection error
		if strings.Contains(err.Error(), "connection refused") {
			return fault.Wrap(err,
				fault.Code(codes.UnkeyAppErrorsInternalServiceUnavailable),
				fault.Internal(fmt.Sprintf("Failed to connect to control plane at %s", opts.ControlPlaneURL)),
				fault.Public("Unable to connect to control plane. Is it running?"),
			)
		}

		// Check if it's an auth error
		if connectErr := new(connect.Error); errors.As(err, &connectErr) {
			if connectErr.Code() == connect.CodeUnauthenticated {
				return fault.Wrap(err,
					fault.Code(codes.UnkeyAuthErrorsAuthenticationMalformed),
					fault.Internal(fmt.Sprintf("Authentication failed with token: %s", opts.AuthToken)),
					fault.Public("Authentication failed. Check your auth token."),
				)
			}
		}

		// Generic API error
		return fault.Wrap(err,
			fault.Code(codes.UnkeyAppErrorsInternalUnexpectedError),
			fault.Internal(fmt.Sprintf("CreateVersion API call failed: %v", err)),
			fault.Public("Failed to create version. Please try again."),
		)
	}

	versionId := createResp.Msg.GetVersionId()
	if versionId != "" {
		fmt.Printf("  Version ID: %s\n", versionId)
	}

	// Poll for version status updates
	if err := pollVersionStatus(ctx, logger, client, opts.AuthToken, versionId); err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	printDeploymentComplete(versionId, opts.WorkspaceID, opts.Branch)
	return nil
}

// pollVersionStatus polls the control plane API and displays deployment steps as they occur
func pollVersionStatus(ctx context.Context, logger logging.Logger, client ctrlv1connect.VersionServiceClient, authToken, versionId string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	timeout := time.NewTimer(300 * time.Second) // 5 minute timeout for full deployment
	defer timeout.Stop()

	displayedSteps := make(map[string]bool)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			fmt.Printf("Error: Deployment timeout after 5 minutes\n")
			return fmt.Errorf("deployment timeout")
		case <-ticker.C:
			// Poll version status
			getReq := connect.NewRequest(&ctrlv1.GetVersionRequest{
				VersionId: versionId,
			})
			getReq.Header().Set("Authorization", "Bearer "+authToken)

			getResp, err := client.GetVersion(ctx, getReq)
			if err != nil {
				logger.Debug("Failed to get version status", "error", err, "version_id", versionId)
				continue
			}

			version := getResp.Msg.GetVersion()

			// Display version steps in real-time
			steps := version.GetSteps()
			for _, step := range steps {
				stepKey := step.GetStatus()
				if !displayedSteps[stepKey] {
					displayVersionStep(step)
					displayedSteps[stepKey] = true
				}
			}

			// Check if deployment is complete
			if version.GetStatus() == ctrlv1.VersionStatus_VERSION_STATUS_ACTIVE {
				return nil
			}

			// Check if deployment failed
			if version.GetStatus() == ctrlv1.VersionStatus_VERSION_STATUS_FAILED {
				return fmt.Errorf("deployment failed")
			}
		}
	}
}

// displayVersionStep shows a version step with appropriate formatting
func displayVersionStep(step *ctrlv1.VersionStep) {
	message := step.GetMessage()
	// Display only the actual message from the database, indented under "Creating Version"
	if message != "" {
		fmt.Printf("  %s\n", message)
	}

	// Show error message if present
	if step.GetErrorMessage() != "" {
		fmt.Printf("  Error: %s\n", step.GetErrorMessage())
	}
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

func buildDockerImage(ctx context.Context, opts *DeployOptions, gitInfo git.Info) (string, error) {
	// Check if Docker is available
	if !isDockerAvailable() {
		return "", ErrDockerNotFound
	}

	// Generate image tag using Git info when available
	var imageTag string
	if gitInfo.ShortSHA != "" {
		imageTag = fmt.Sprintf("%s-%s", opts.Branch, gitInfo.ShortSHA)
	} else {
		// Fallback to timestamp if no Git info
		timestamp := time.Now().Unix()
		imageTag = fmt.Sprintf("%s-%d", opts.Branch, timestamp)
	}

	dockerImage := fmt.Sprintf("%s:%s", opts.Registry, imageTag)

	fmt.Printf("Building Docker image %s...\n", dockerImage)

	// Build the Docker image
	var buildArgs []string
	buildArgs = append(buildArgs, "build")

	// Only add -f flag if dockerfile is not the default "Dockerfile"
	if opts.Dockerfile != "Dockerfile" {
		buildArgs = append(buildArgs, "-f", opts.Dockerfile)
	}

	buildArgs = append(buildArgs,
		"-t", dockerImage,
		"--build-arg", fmt.Sprintf("VERSION=%s", opts.Commit),
		opts.Context,
	)

	buildCmd := exec.CommandContext(ctx, "docker", buildArgs...)

	// Create pipes to capture output
	stdout, err := buildCmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := buildCmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the build command
	if err := buildCmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start docker build: %w", err)
	}

	// Capture all output for error reporting
	var allOutput strings.Builder
	combinedOutput := io.MultiReader(stdout, stderr)
	scanner := bufio.NewScanner(combinedOutput)

	// Process output line by line
	for scanner.Scan() {
		line := scanner.Text()
		allOutput.WriteString(line + "\n")
		fmt.Printf("    %s\n", line)
	}

	// Wait for the build to complete
	if err := buildCmd.Wait(); err != nil {
		fmt.Printf("Docker build failed\n")
		// Show build output on failure
		for line := range strings.SplitSeq(allOutput.String(), "\n") {
			if strings.TrimSpace(line) != "" {
				fmt.Printf("  %s\n", line)
			}
		}
		return "", ErrDockerBuildFailed
	}

	// Skip push if requested
	if opts.SkipPush {
		fmt.Printf("Skipping Docker push (--skip-push enabled)\n")
		return dockerImage, nil
	}

	fmt.Printf("\nPublishing Docker image...\n")

	// Push the image
	pushCmd := exec.CommandContext(ctx, "docker", "push", dockerImage)
	var pushOutput strings.Builder
	pushCmd.Stdout = &pushOutput
	pushCmd.Stderr = &pushOutput

	if err := pushCmd.Run(); err != nil {
		// Parse the error output to provide better messages
		outputStr := strings.TrimSpace(pushOutput.String())

		if strings.Contains(outputStr, "denied") {
			fmt.Printf("Docker push failed: Registry access denied\n")
			fmt.Printf("  Registry: %s\n", opts.Registry)
			fmt.Printf("  \n")
			fmt.Printf("  This usually means:\n")
			fmt.Printf("  • You're not logged into the registry: docker login %s\n", getRegistryHost(opts.Registry))
			fmt.Printf("  • You don't have push permissions to this repository\n")
			fmt.Printf("  • The repository doesn't exist or is private\n")
			fmt.Printf("  \n")
			fmt.Printf("  For development, you can:\n")
			fmt.Printf("  • Use your own registry: --registry=your-registry/your-app\n")
			fmt.Printf("  • Use a pre-built image: --docker-image=nginx:alpine\n")
			fmt.Printf("  • Skip the push: --skip-push\n\n")
			return dockerImage, nil // Continue for development
		}

		if strings.Contains(outputStr, "not found") || strings.Contains(outputStr, "404") {
			fmt.Printf("Docker push failed: Registry not found\n")
			fmt.Printf("  Registry: %s\n", opts.Registry)
			fmt.Printf("  \n")
			fmt.Printf("  The repository may not exist. Try:\n")
			fmt.Printf("  • Creating the repository first\n")
			fmt.Printf("  • Using a different registry: --registry=your-registry/your-app\n")
			fmt.Printf("  • Skip the push: --skip-push\n\n")
			return dockerImage, nil // Continue for development
		}

		if strings.Contains(outputStr, "unauthorized") {
			fmt.Printf("Docker push failed: Authentication required\n")
			fmt.Printf("  Run: docker login %s\n", getRegistryHost(opts.Registry))
			fmt.Printf("  Or skip the push: --skip-push\n\n")
			return dockerImage, nil // Continue for development
		}

		// Generic push error
		fmt.Printf("Docker push failed (continuing anyway for development)\n")
		fmt.Printf("  %s\n", outputStr)
		return dockerImage, nil
	}

	return dockerImage, nil
}

func isDockerAvailable() bool {
	cmd := exec.Command("docker", "--version")
	return cmd.Run() == nil
}

func getRegistryHost(registry string) string {
	parts := strings.Split(registry, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return registry
}

func printDeploymentComplete(versionId, workspace, branch string) {
	// Use Git info for hostname generation
	gitInfo := git.GetInfo()
	identifier := versionId
	if gitInfo.ShortSHA != "" {
		identifier = gitInfo.ShortSHA
	}

	fmt.Println()
	fmt.Println("Deployment Complete")
	fmt.Printf("  Version ID: %s\n", versionId)
	fmt.Printf("  Status: Ready\n")
	fmt.Printf("  Environment: Production\n")
	fmt.Println()
	fmt.Println("Domains")

	// Replace underscores with dashes for valid hostname format
	cleanIdentifier := strings.ReplaceAll(identifier, "_", "-")
	fmt.Printf("  https://%s-%s-%s.unkey.app\n", branch, cleanIdentifier, workspace)
	fmt.Printf("  https://api.acme.com\n")
}

// parseDeployFlags parses flags for deploy/version create commands
func parseDeployFlags(commandName string, args []string, env map[string]string) (*DeployOptions, error) {
	fs := flag.NewFlagSet(commandName, flag.ExitOnError)
	opts := &DeployOptions{}

	defaultWorkspaceID := env["UNKEY_WORKSPACE_ID"]
	defaultProjectID := env["UNKEY_PROJECT_ID"]
	defaultRegistry := env["UNKEY_DOCKER_REGISTRY"]
	if defaultRegistry == "" {
		defaultRegistry = "ghcr.io/unkeyed/deploy"
	}

	// Required flags
	fs.StringVar(&opts.WorkspaceID, "workspace-id", defaultWorkspaceID, "Workspace ID (required)")
	fs.StringVar(&opts.ProjectID, "project-id", defaultProjectID, "Project ID (required)")

	// Optional flags with defaults
	fs.StringVar(&opts.Context, "context", ".", "Docker context path")
	fs.StringVar(&opts.Branch, "branch", "main", "Git branch")
	fs.StringVar(&opts.DockerImage, "docker-image", "", "Pre-built docker image")
	fs.StringVar(&opts.Dockerfile, "dockerfile", "Dockerfile", "Path to Dockerfile")
	fs.StringVar(&opts.Commit, "commit", "", "Git commit SHA")
	fs.StringVar(&opts.Registry, "registry", defaultRegistry, "Docker registry")
	fs.BoolVar(&opts.SkipPush, "skip-push", false, "Skip pushing to registry (for local testing)")

	// Control plane flags (internal)
	fs.StringVar(&opts.ControlPlaneURL, "control-plane-url", "http://localhost:7091", "Control plane URL")
	fs.StringVar(&opts.AuthToken, "auth-token", "ctrl-secret-token", "Control plane auth token")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("failed to parse %s flags: %w", commandName, err)
	}

	// Validate required fields
	if opts.WorkspaceID == "" {
		return nil, fmt.Errorf("--workspace-id is required (or set UNKEY_WORKSPACE_ID)")
	}
	if opts.ProjectID == "" {
		return nil, fmt.Errorf("--project-id is required (or set UNKEY_PROJECT_ID)")
	}

	return opts, nil
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
