package version

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/charmbracelet/lipgloss"
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
			Name:     "workspace",
			Usage:    "Workspace ID",
			Value:    "acme",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "project",
			Usage:    "Project ID",
			Value:    "my-api",
			Required: false,
		},
	},
	Action: createAction,
}

func createAction(ctx context.Context, cmd *cli.Command) error {
	logger := logging.New()

	// Hardcoded for demo
	workspace := "acme"
	project := "my-api"

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

	return runDeploymentSteps(ctx, cmd, workspace, project, branch, dockerImage, dockerfile, buildContext, commit, logger)
}

// Styles for clean output
var (
	sectionName = lipgloss.NewStyle().Bold(true)
	metaText    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errorText   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

// Fun loading messages for different deployment phases
var (
	buildingMessages = []string{
		"Transforming containers into pure magic...",
		"Teaching Docker images to fly...",
		"Compressing pixels and dreams...",
		"Turning containers inside out...",
		"Extracting the essence of your code...",
		"Squishing containers flat like pancakes...",
		"Converting Docker to VM-speak...",
	}

	deployingMessages = []string{
		"Waking up sleepy virtual machines...",
		"Teaching VMs to dance with your code...",
		"Summoning compute spirits from the cloud...",
		"Bribing CPUs with electricity...",
		"Convincing VMs to get out of bed...",
		"Herding virtual cats into formation...",
		"Rolling out the red carpet for your app...",
	}

	buildQueuedMessages = []string{
		"Waiting in line behind the other builds...",
		"Taking a number at the build deli...",
		"Patience, young padawan...",
		"Good things come to those who wait...",
		"Counting sheep until build starts...",
		"Build is doing pre-flight checks...",
	}
)

func printDeploymentComplete(versionID, workspace, branch, commit string) {
	fmt.Println()
	fmt.Printf("%s\n", sectionName.Render("Deployment Complete"))
	fmt.Printf("  Version ID: %s\n", metaText.Render(versionID))
	fmt.Printf("  Status: Ready\n")
	fmt.Printf("  Environment: Production\n")

	fmt.Println()
	fmt.Printf("%s\n", sectionName.Render("Domains"))

	// Use actual Git info for hostname generation
	gitInfo := git.GetInfo()
	shortSHA := "unknown"
	if gitInfo.ShortSHA != "" {
		shortSHA = gitInfo.ShortSHA
	} else if commit != "" && len(commit) >= 7 {
		shortSHA = commit[:7]
	}

	fmt.Printf("  %s\n", metaText.Render(fmt.Sprintf("https://%s-%s-%s.unkey.app", branch, shortSHA, workspace)))
	fmt.Printf("  %s\n", metaText.Render("https://api.acme.com"))

	fmt.Println()
	fmt.Printf("%s\n", sectionName.Render("Source"))
	fmt.Printf("  Branch: %s\n", branch)
	if gitInfo.ShortSHA != "" {
		fmt.Printf("  Commit: %s\n", gitInfo.ShortSHA)
		if gitInfo.IsDirty {
			fmt.Printf("  Status: %s\n", metaText.Render("Working directory has uncommitted changes"))
		}
	}
}

func runDeploymentSteps(ctx context.Context, cmd *cli.Command, workspace, project, branch, dockerImage, dockerfile, buildContext, commit string, logger logging.Logger) error {

	// Get Git info for better image tagging
	gitInfo := git.GetInfo()

	// If no docker image provided, build one
	if dockerImage == "" {
		fmt.Printf("%s\n", sectionName.Render("Building"))

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

		fmt.Printf("  Image tag: %s\n", metaText.Render(dockerImage))
		fmt.Printf("  Build context: %s\n", metaText.Render(buildContext))

		// Docker build steps
		fmt.Printf("  Building Docker image...\n")

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
			fmt.Printf("    %s\n", metaText.Render(line))
		}

		// Wait for the build to complete
		err = buildCmd.Wait()

		if err != nil {
			fmt.Printf("  %s: Docker build failed\n", errorText.Render("Error"))
			// Show the full build output on failure
			for _, line := range strings.Split(allOutput.String(), "\n") {
				if strings.TrimSpace(line) != "" {
					fmt.Printf("    %s\n", line)
				}
			}
			return fmt.Errorf("docker build failed: %w", err)
		}

		// Publishing section
		fmt.Printf("%s\n", sectionName.Render("Publishing"))

		pushCmd := exec.CommandContext(ctx, "docker", "push", dockerImage)

		// Capture output for error reporting
		var pushOutput strings.Builder
		pushCmd.Stdout = &pushOutput
		pushCmd.Stderr = &pushOutput

		// Run the push
		if err := pushCmd.Run(); err != nil {
			fmt.Printf("  %s: Docker push failed\n", errorText.Render("Error"))
			// Show the push output on failure
			for _, line := range strings.Split(pushOutput.String(), "\n") {
				if strings.TrimSpace(line) != "" {
					fmt.Printf("    %s\n", line)
				}
			}
			return fmt.Errorf("docker push failed: %w", err)
		}
	}

	// Creating Version section
	fmt.Printf("%s\n", sectionName.Render("Creating Version"))

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
	fmt.Printf("  Version ID: %s\n", metaText.Render(versionID))
	fmt.Printf("  Image: %s\n", metaText.Render(dockerImage))
	fmt.Printf("  Branch: %s\n", metaText.Render(branch))

	// Poll for version status updates
	if err := pollVersionStatus(ctx, logger, client, versionID); err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	printDeploymentComplete(versionID, workspace, branch, commit)

	return nil
}

// pollVersionStatus polls the control plane API for version status updates
func pollVersionStatus(ctx context.Context, logger logging.Logger, client ctrlv1connect.VersionServiceClient, versionID string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(30 * time.Second) // 30 second timeout
	defer timeout.Stop()

	lastVersionStatus := ""

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			fmt.Printf("Error: Deployment timeout after 30 seconds\n")
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
			currentVersionStatus := version.GetStatus().String()

			// Show version status updates with section headers
			if currentVersionStatus != lastVersionStatus {
				switch version.GetStatus() {
				case ctrlv1.VersionStatus_VERSION_STATUS_UNSPECIFIED:
					// Skip unspecified status, no display needed
				case ctrlv1.VersionStatus_VERSION_STATUS_PENDING:
					fmt.Printf("%s\n", sectionName.Render("Pending"))
					message := buildQueuedMessages[rand.Intn(len(buildQueuedMessages))] // nolint:gosec // Weak random is acceptable for UI messages
					fmt.Printf("  %s\n", metaText.Render(message))
				case ctrlv1.VersionStatus_VERSION_STATUS_BUILDING:
					fmt.Printf("%s\n", sectionName.Render("Building"))
					message := buildingMessages[rand.Intn(len(buildingMessages))] // nolint:gosec // Weak random is acceptable for UI messages
					fmt.Printf("  %s\n", metaText.Render(message))
				case ctrlv1.VersionStatus_VERSION_STATUS_DEPLOYING:
					fmt.Printf("%s\n", sectionName.Render("Deploying"))
					message := deployingMessages[rand.Intn(len(deployingMessages))] // nolint:gosec // Weak random is acceptable for UI messages
					fmt.Printf("  %s\n", metaText.Render(message))
				case ctrlv1.VersionStatus_VERSION_STATUS_ACTIVE:
					// Will be handled after the polling loop
				case ctrlv1.VersionStatus_VERSION_STATUS_FAILED:
					fmt.Printf("  %s: Deployment failed\n", errorText.Render("Error"))
				case ctrlv1.VersionStatus_VERSION_STATUS_ARCHIVED:
					fmt.Printf("  %s: Version archived\n", metaText.Render("Info"))
				}
				lastVersionStatus = currentVersionStatus
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
