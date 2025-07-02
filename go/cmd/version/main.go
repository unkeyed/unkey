package version

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
		&cli.BoolFlag{
			Name:  "watch",
			Usage: "Watch the deployment progress",
			Value: true,
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

	branch := cmd.String("branch")
	dockerImage := cmd.String("docker-image")
	dockerfile := cmd.String("dockerfile")
	buildContext := cmd.String("context")
	commit := cmd.String("commit")
	watch := cmd.Bool("watch")

	return runDeploymentSteps(ctx, cmd, workspace, project, branch, dockerImage, dockerfile, buildContext, commit, watch, logger)
}

// Styles for clean output - Vitest-inspired hierarchical display
var (
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).MarginTop(1).MarginBottom(1)
	successIcon  = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✓")
	errorIcon    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("✗")
	pendingIcon  = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("○")
	sectionName  = lipgloss.NewStyle().Bold(true)
	subStepName  = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	detailsText  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	durationText = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func printDeploymentComplete(versionID, workspace, branch, commit string) {
	fmt.Println()
	fmt.Printf("%s %s\n", successIcon, sectionName.Render("Deployment Complete"))
	fmt.Printf("  %s %s: %s\n", successIcon, subStepName.Render("Version ID"), detailsText.Render(versionID))
	fmt.Printf("  %s %s: %s\n", successIcon, subStepName.Render("Status"), detailsText.Render("Ready"))
	fmt.Printf("  %s %s: %s\n", successIcon, subStepName.Render("Environment"), detailsText.Render("Production"))

	fmt.Println()
	fmt.Printf("%s %s\n", successIcon, sectionName.Render("Domains"))

	// Generate hostnames based on patterns
	gitSha := "abc123d"
	if commit != "" && len(commit) >= 7 {
		gitSha = commit[:7]
	}

	fmt.Printf("  %s\n", detailsText.Render(fmt.Sprintf("https://%s-%s-%s.unkey.app", branch, gitSha, workspace)))
	fmt.Printf("  %s\n", detailsText.Render("https://api.acme.com"))

	fmt.Println()
	fmt.Printf("%s %s\n", successIcon, sectionName.Render("Source"))
	fmt.Printf("  %s %s: %s\n", successIcon, subStepName.Render("Branch"), detailsText.Render(branch))
	if commit != "" {
		fmt.Printf("  %s %s: %s\n", successIcon, subStepName.Render("Commit"), detailsText.Render(commit[:7]))
	}
}

func runDeploymentSteps(ctx context.Context, cmd *cli.Command, workspace, project, branch, dockerImage, dockerfile, buildContext, commit string, watch bool, logger logging.Logger) error {

	// Simple header
	version := "v1.0.0"
	fmt.Println(headerStyle.Render(fmt.Sprintf("DEPLOY  %s  %s", version, buildContext)))
	fmt.Println()

	// If no docker image provided, build one
	if dockerImage == "" {
		// Start Building section
		fmt.Print("○ Building")
		buildStart := time.Now()

		// Generate image tag
		timestamp := time.Now().Unix()
		dockerImage = fmt.Sprintf("ghcr.io/unkeyed/deploy-wip:%s-%d", branch, timestamp)

		fmt.Printf("\r%s %s %s\n", successIcon, sectionName.Render("Building"), durationText.Render(fmt.Sprintf("(%.1fs)", time.Since(buildStart).Seconds())))
		fmt.Printf("  %s %s: %s\n", successIcon, subStepName.Render("Generate image tag"), detailsText.Render(dockerImage))
		fmt.Printf("  %s %s: %s\n", successIcon, subStepName.Render("Docker build context"), detailsText.Render(buildContext))

		// Docker build steps
		fmt.Printf("  %s %s", pendingIcon, subStepName.Render("Build steps"))
		dockerBuildStart := time.Now()

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
			"--quiet", // Suppress build output unless there's an error
			buildContext,
		)

		buildCmd := exec.CommandContext(ctx, "docker", buildArgs...)

		// Capture all output for error reporting
		var allOutput strings.Builder
		buildCmd.Stdout = &allOutput
		buildCmd.Stderr = &allOutput

		// Run the build
		err := buildCmd.Run()

		if err != nil {
			fmt.Printf("\r  %s %s %s\n", errorIcon, subStepName.Render("Build steps"), durationText.Render(fmt.Sprintf("(%.1fs)", time.Since(dockerBuildStart).Seconds())))
			// Show the full build output on failure
			fmt.Printf("    Build output:\n")
			for _, line := range strings.Split(allOutput.String(), "\n") {
				if strings.TrimSpace(line) != "" {
					fmt.Printf("    %s\n", detailsText.Render(line))
				}
			}
			return fmt.Errorf("docker build failed: %w", err)
		}

		fmt.Printf("\r  %s %s %s\n", successIcon, subStepName.Render("Build steps"), durationText.Render(fmt.Sprintf("(%.1fs)", time.Since(dockerBuildStart).Seconds())))

		// Publishing section
		fmt.Printf("%s %s", pendingIcon, sectionName.Render("Publishing"))
		publishStart := time.Now()

		pushCmd := exec.CommandContext(ctx, "docker", "push", dockerImage)

		// Capture output for error reporting
		var pushOutput strings.Builder
		pushCmd.Stdout = &pushOutput
		pushCmd.Stderr = &pushOutput

		// Run the push
		if err := pushCmd.Run(); err != nil {
			fmt.Printf("\r%s %s %s\n", errorIcon, sectionName.Render("Publishing"), durationText.Render(fmt.Sprintf("(%.1fs)", time.Since(publishStart).Seconds())))
			// Show the push output on failure
			fmt.Printf("    Push output:\n")
			for _, line := range strings.Split(pushOutput.String(), "\n") {
				if strings.TrimSpace(line) != "" {
					fmt.Printf("    %s\n", detailsText.Render(line))
				}
			}
			return fmt.Errorf("docker push failed: %w", err)
		}

		fmt.Printf("\r%s %s %s\n", successIcon, sectionName.Render("Publishing"), durationText.Render(fmt.Sprintf("(%.1fs)", time.Since(publishStart).Seconds())))
	}

	// Creating Version section
	fmt.Printf("%s %s", pendingIcon, sectionName.Render("Creating Version"))
	versionStart := time.Now()

	// Create control plane client
	controlPlaneURL := cmd.String("control-plane-url")
	authToken := cmd.String("auth-token")

	httpClient := &http.Client{}
	client := ctrlv1connect.NewVersionServiceClient(httpClient, controlPlaneURL)

	// Create version request
	createReq := connect.NewRequest(&ctrlv1.CreateVersionRequest{
		WorkspaceId:   workspace,
		ProjectId:     project,
		Branch:        branch,
		SourceType:    ctrlv1.SourceType_SOURCE_TYPE_CLI_UPLOAD,
		GitCommitSha:  cmd.String("commit"),
		EnvironmentId: "env_prod",
		UploadUrl:     "", // Not used for this flow
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
	fmt.Printf("\r%s %s %s\n", successIcon, sectionName.Render("Creating Version"), durationText.Render(fmt.Sprintf("(%.1fs)", time.Since(versionStart).Seconds())))
	fmt.Printf("  %s %s: %s\n", successIcon, subStepName.Render("Version ID"), detailsText.Render(versionID))
	fmt.Printf("  %s %s: %s\n", successIcon, subStepName.Render("Image"), detailsText.Render(dockerImage))
	fmt.Printf("  %s %s: %s\n", successIcon, subStepName.Render("Branch"), detailsText.Render(branch))

	if watch {
		// Deploying section
		fmt.Printf("%s %s", pendingIcon, sectionName.Render("Deploying"))
		deployStart := time.Now()

		// Poll for version status updates
		if err := pollVersionStatus(ctx, logger, client, versionID); err != nil {
			fmt.Printf("\r%s %s %s\n", errorIcon, sectionName.Render("Deploying"), durationText.Render(fmt.Sprintf("(%.1fs)", time.Since(deployStart).Seconds())))
			return fmt.Errorf("deployment failed: %w", err)
		}

		fmt.Printf("\r%s %s %s\n", successIcon, sectionName.Render("Deploying"), durationText.Render(fmt.Sprintf("(%.1fs)", time.Since(deployStart).Seconds())))
		printDeploymentComplete(versionID, workspace, branch, commit)
	}

	return nil
}

// pollVersionStatus polls the control plane API for version status updates
func pollVersionStatus(ctx context.Context, logger logging.Logger, client ctrlv1connect.VersionServiceClient, versionID string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(30 * time.Second) // 30 second timeout
	defer timeout.Stop()

	lastStatus := ""

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			fmt.Printf("  %s %s\n", errorIcon, subStepName.Render("Deployment timeout after 30 seconds"))
			return fault.Wrap(errors.New("deployment timeout"),
				fault.Code(codes.UnkeyAppErrorsInternalServiceUnavailable),
				fault.Internal("Version status polling timed out after 30 seconds"),
				fault.Public("Deployment is taking longer than expected. Check status manually."),
			)
		case <-ticker.C:
			// Call the GetVersion API
			getReq := connect.NewRequest(&ctrlv1.GetVersionRequest{
				VersionId: versionID,
			})
			getReq.Header().Set("Authorization", "Bearer ctrl-secret-token")

			getResp, err := client.GetVersion(ctx, getReq)
			if err != nil {
				logger.Debug("Failed to get version status",
					"error", err,
					"version_id", versionID,
				)
				continue
			}

			version := getResp.Msg.GetVersion()
			currentStatus := version.GetStatus().String()

			// Only print status updates when they change
			if currentStatus != lastStatus {
				statusName := getStatusDisplayName(version.GetStatus())
				fmt.Printf("  %s %s\n", pendingIcon, subStepName.Render(statusName))
				lastStatus = currentStatus
			}

			// Check if deployment is complete
			if version.GetStatus() == ctrlv1.VersionStatus_VERSION_STATUS_ACTIVE {
				return nil
			}

			// Check if deployment failed
			if version.GetStatus() == ctrlv1.VersionStatus_VERSION_STATUS_FAILED {
				return fmt.Errorf("deployment failed")
			}

			logger.Debug("Polling version status",
				"version_id", versionID,
				"status", currentStatus,
			)
		}
	}
}

// getStatusDisplayName converts a version status to a human-readable display name
func getStatusDisplayName(status ctrlv1.VersionStatus) string {
	switch status {
	case ctrlv1.VersionStatus_VERSION_STATUS_UNSPECIFIED:
		return "Status unknown"
	case ctrlv1.VersionStatus_VERSION_STATUS_PENDING:
		return "Queuing deployment"
	case ctrlv1.VersionStatus_VERSION_STATUS_BUILDING:
		return "Converting Docker image to Firecracker rootfs"
	case ctrlv1.VersionStatus_VERSION_STATUS_DEPLOYING:
		return "Starting instances"
	case ctrlv1.VersionStatus_VERSION_STATUS_ACTIVE:
		return "Ready and serving traffic"
	case ctrlv1.VersionStatus_VERSION_STATUS_FAILED:
		return "Deployment failed"
	case ctrlv1.VersionStatus_VERSION_STATUS_ARCHIVED:
		return "Deployment archived"
	default:
		return "Unknown status"
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
