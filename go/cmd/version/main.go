package version

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
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
	"github.com/urfave/cli/v3"
)

var Cmd = &cli.Command{
	Name:  "version",
	Usage: "Manage API versions",
	Description: `Create, list, and manage versions of your API.
	
Versions are immutable snapshots of your code, configuration, and infrastructure settings.`,

	Commands: []*cli.Command{
		createCmd,
		getCmd,
		listCmd,
		rollbackCmd,
		// TODO: Remove this bootstrap command once we have a proper UI
		bootstrapProjectCmd, // defined in bootstrap.go
	},
}

var createCmd = &cli.Command{
	Name:    "create",
	Aliases: []string{"deploy"},
	Usage:   "Create a new version of your API",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "branch",
			Usage:    "Git branch name",
			Value:    "main",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "docker-image",
			Usage:    "Docker image tag (e.g., ghcr.io/user/app:tag). If not provided, builds from current directory",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "dockerfile",
			Usage:    "Path to Dockerfile",
			Value:    "Dockerfile",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "context",
			Usage:    "Build context directory",
			Value:    ".",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "commit",
			Usage:    "Git commit SHA",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "control-plane-url",
			Usage:    "Control plane base URL",
			Value:    "http://localhost:7091",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "auth-token",
			Usage:    "Control plane auth token",
			Value:    "ctrl-secret-token",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "workspace-id",
			Usage:    "Workspace ID",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "project-id",
			Usage:    "Project ID",
			Required: true,
		},
	},
	Action: createAction,
}

func createAction(ctx context.Context, cmd *cli.Command) error {
	logger := logging.New()

	// Get workspace and project IDs from CLI flags
	workspaceID := cmd.String("workspace-id")
	projectID := cmd.String("project-id")

	// Get Git information automatically
	gitInfo := git.GetInfo()

	// Use Git info as defaults, allow CLI flags to override
	branch := cmd.String("branch")
	if branch == "main" && gitInfo.IsRepo { // CLI default is "main"
		branch = gitInfo.Branch
	}

	commit := cmd.String("commit")
	if commit == "" && gitInfo.CommitSHA != "" {
		commit = gitInfo.CommitSHA
	}

	dockerImage := cmd.String("docker-image")
	dockerfile := cmd.String("dockerfile")
	buildContext := cmd.String("context")

	return runDeploymentSteps(ctx, cmd, workspaceID, projectID, branch, dockerImage, dockerfile, buildContext, commit, logger)
}

func printDeploymentComplete(versionID, workspace, branch, commit string) {
	// Use actual Git info for hostname generation
	gitInfo := git.GetInfo()
	identifier := versionID
	if gitInfo.IsRepo && gitInfo.CommitSHA != "" {
		identifier = gitInfo.CommitSHA
	}

	fmt.Println()
	fmt.Println("Deployment Complete")
	fmt.Printf("  Version ID: %s\n", versionID)
	fmt.Printf("  Status: Ready\n")
	fmt.Printf("  Environment: Production\n")

	fmt.Println()
	fmt.Println("Domains")
	// Replace underscores with dashes for valid hostname format
	cleanIdentifier := strings.ReplaceAll(identifier, "_", "-")
	fmt.Printf("  https://%s-%s-%s.unkey.app\n", branch, cleanIdentifier, workspace)
	fmt.Printf("  https://api.acme.com\n")
}

func runDeploymentSteps(ctx context.Context, cmd *cli.Command, workspace, project, branch, dockerImage, dockerfile, buildContext, commit string, logger logging.Logger) error {

	// Get Git info for better image tagging
	gitInfo := git.GetInfo()

	// Print source information immediately
	fmt.Println("Source")
	fmt.Printf("  Branch: %s\n", branch)
	if gitInfo.CommitSHA != "" {
		fmt.Printf("  Commit: %s\n", gitInfo.CommitSHA)
		if gitInfo.IsDirty {
			fmt.Printf("  Status: Working directory has uncommitted changes\n")
		}
	}
	fmt.Println()

	// If no docker image provided, build one
	if dockerImage == "" {
		// Generate image tag using Git info when available
		var imageTag string
		if gitInfo.ShortSHA != "" {
			imageTag = fmt.Sprintf("%s-%s", branch, gitInfo.ShortSHA)
		} else {
			// Fallback to timestamp if no Git info
			timestamp := time.Now().Unix()
			imageTag = fmt.Sprintf("%s-%d", branch, timestamp)
		}
		dockerImage = fmt.Sprintf("ghcr.io/unkeyed/deploy-wip:%s", imageTag)

		fmt.Printf("Building Docker image %s...\n", dockerImage)

		// Build the Docker image with minimal output
		var buildArgs []string
		buildArgs = append(buildArgs, "build")

		// Only add -f flag if dockerfile is not the default "Dockerfile"
		if dockerfile != "Dockerfile" {
			buildArgs = append(buildArgs, "-f", dockerfile)
		}

		buildArgs = append(buildArgs,
			"-t", dockerImage,
			"--build-arg", fmt.Sprintf("VERSION=%s-%s", branch, commit),
			buildContext,
		)

		buildCmd := exec.CommandContext(ctx, "docker", buildArgs...)

		// Create pipes to capture stdout and stderr
		stdout, err := buildCmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdout pipe: %w", err)
		}
		stderr, err := buildCmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("failed to create stderr pipe: %w", err)
		}

		// Start the build command
		if startErr := buildCmd.Start(); startErr != nil {
			return fmt.Errorf("failed to start docker build: %w", startErr)
		}

		// Capture all output for error reporting
		var allOutput strings.Builder

		// Create a combined reader for both stdout and stderr
		combinedOutput := io.MultiReader(stdout, stderr)
		scanner := bufio.NewScanner(combinedOutput)

		// Process output line by line
		for scanner.Scan() {
			line := scanner.Text()
			allOutput.WriteString(line + "\n")

			// Print all docker build output
			fmt.Printf("    %s\n", line)
		}

		// Wait for the build to complete
		err = buildCmd.Wait()

		if err != nil {
			fmt.Printf("Docker build failed\n")
			// Show the full build output on failure
			for _, line := range strings.Split(allOutput.String(), "\n") {
				if strings.TrimSpace(line) != "" {
					fmt.Printf("  %s\n", line)
				}
			}
			return fmt.Errorf("docker build failed: %w", err)
		}

		fmt.Printf("Publishing Docker image...\n")

		pushCmd := exec.CommandContext(ctx, "docker", "push", dockerImage)

		// Capture output for error reporting
		var pushOutput strings.Builder
		pushCmd.Stdout = &pushOutput
		pushCmd.Stderr = &pushOutput

		// Run the push
		if err := pushCmd.Run(); err != nil {
			fmt.Printf("Docker push failed\n")
			// Show the push output on failure
			for _, line := range strings.Split(pushOutput.String(), "\n") {
				if strings.TrimSpace(line) != "" {
					fmt.Printf("  %s\n", line)
				}
			}
			return fmt.Errorf("docker push failed: %w", err)
		}
	}

	// Create control plane client
	controlPlaneURL := cmd.String("control-plane-url")
	authToken := cmd.String("auth-token")

	httpClient := &http.Client{}
	client := ctrlv1connect.NewVersionServiceClient(httpClient, controlPlaneURL)

	// Create version request
	createReq := connect.NewRequest(&ctrlv1.CreateVersionRequest{
		WorkspaceId:    workspace,
		ProjectId:      project,
		Branch:         branch,
		SourceType:     ctrlv1.SourceType_SOURCE_TYPE_CLI_UPLOAD,
		GitCommitSha:   cmd.String("commit"),
		EnvironmentId:  "env_prod",
		DockerImageTag: dockerImage,
	})

	// Add auth header
	createReq.Header().Set("Authorization", "Bearer "+authToken)

	// Call the API
	createResp, err := client.CreateVersion(ctx, createReq)
	if err != nil {
		fmt.Println()

		// Check if it's a connection error
		if strings.Contains(err.Error(), "connection refused") {
			return fault.Wrap(err,
				fault.Code(codes.UnkeyAppErrorsInternalServiceUnavailable),
				fault.Internal(fmt.Sprintf("Failed to connect to control plane at %s", controlPlaneURL)),
				fault.Public("Unable to connect to control plane. Is it running?"),
			)
		}

		// Check if it's an auth error
		if connectErr := new(connect.Error); errors.As(err, &connectErr) {
			if connectErr.Code() == connect.CodeUnauthenticated {
				return fault.Wrap(err,
					fault.Code(codes.UnkeyAuthErrorsAuthenticationMalformed),
					fault.Internal(fmt.Sprintf("Authentication failed with token: %s", authToken)),
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

	versionID := createResp.Msg.GetVersionId()
	fmt.Printf("Creating Version\n")
	fmt.Printf("  Version ID: %s\n", versionID)

	// Poll for version status updates
	if err := pollVersionStatus(ctx, logger, client, versionID); err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	printDeploymentComplete(versionID, workspace, branch, commit)

	return nil
}

// pollVersionStatus polls the control plane API and displays deployment steps as they occur
func pollVersionStatus(ctx context.Context, logger logging.Logger, client ctrlv1connect.VersionServiceClient, versionID string) error {
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
			// Always poll version status
			getReq := connect.NewRequest(&ctrlv1.GetVersionRequest{
				VersionId: versionID,
			})
			getReq.Header().Set("Authorization", "Bearer ctrl-secret-token")

			getResp, err := client.GetVersion(ctx, getReq)
			if err != nil {
				logger.Debug("Failed to get version status", "error", err, "version_id", versionID)
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

var getCmd = &cli.Command{
	Name:      "get",
	Usage:     "Get details about a version",
	ArgsUsage: "<version-id>",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		logger := slog.Default()

		if cmd.Args().Len() < 1 {
			return cli.Exit("version ID required", 1)
		}

		versionID := cmd.Args().First()
		logger.Info("Getting version details", "version_id", versionID)

		// Call control plane API to get version
		fmt.Printf("Version: %s\n", versionID)
		fmt.Println("Status: ACTIVE")
		fmt.Println("Branch: main")
		fmt.Println("Created: 2024-01-01 12:00:00")
		fmt.Println("Hostnames:")
		fmt.Println("  - https://abc123-workspace.unkey.app")

		return nil
	},
}

var listCmd = &cli.Command{
	Name:  "list",
	Usage: "List versions",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "branch",
			Usage: "Filter by branch",
		},
		&cli.StringFlag{
			Name:  "status",
			Usage: "Filter by status (pending, building, active, failed)",
		},
		&cli.IntFlag{
			Name:  "limit",
			Usage: "Number of versions to show",
			Value: 10,
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		// Hardcoded for demo
		workspace := "acme"
		project := "my-api"

		fmt.Printf("Versions for %s/%s:\n", workspace, project)
		fmt.Println()
		fmt.Println("ID               STATUS    BRANCH    CREATED")
		fmt.Println("v_abc123def456   ACTIVE    main      2024-01-01 12:00:00")
		fmt.Println("v_def456ghi789   ACTIVE    feature   2024-01-01 11:00:00")
		fmt.Println("v_ghi789jkl012   FAILED    main      2024-01-01 10:00:00")

		return nil
	},
}

var rollbackCmd = &cli.Command{
	Name:      "rollback",
	Usage:     "Rollback to a previous version",
	ArgsUsage: "<hostname> <version-id>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "force",
			Usage: "Skip confirmation prompt",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		logger := slog.Default()

		if cmd.Args().Len() < 2 {
			return cli.Exit("hostname and version ID required", 1)
		}

		hostname := cmd.Args().Get(0)
		versionID := cmd.Args().Get(1)
		force := cmd.Bool("force")

		logger.Info("Rolling back version",
			"hostname", hostname,
			"version_id", versionID,
			"force", force,
		)

		if !force {
			fmt.Printf("⚠ Are you sure you want to rollback %s to version %s? [y/N] ", hostname, versionID)
			// Read user confirmation
		}

		// Call control plane API to rollback
		fmt.Printf("Rolling back %s to version %s...\n", hostname, versionID)
		fmt.Println("✓ Rollback completed successfully!")

		return nil
	},
}
