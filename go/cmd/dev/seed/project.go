package seed

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

var projectCmd = &cli.Command{
	Name:  "project",
	Usage: "Create a project and deployment using the control plane API",
	Flags: []cli.Flag{
		cli.String("ctrl-url", "Control plane API URL", cli.Default("http://localhost:7091"), cli.EnvVar("UNKEY_CTRL_URL")),
		cli.String("api-key", "API key for control plane authentication", cli.Default("your-local-dev-key"), cli.EnvVar("UNKEY_API_KEY")),
		cli.String("workspace-id", "Workspace ID (defaults to ws_local from local seed)", cli.Default("ws_local")),
		cli.String("project-name", "Name for the project", cli.Default("Test Project")),
		cli.String("project-slug", "Slug for the project (auto-generated if empty)", cli.Default("")),
		cli.String("environment-slug", "Slug for the environment to deploy to", cli.Default("preview")),
		cli.String("git-repo", "Git repository URL", cli.Default("")),
		cli.String("git-branch", "Default git branch", cli.Default("main")),
		cli.String("docker-image", "Docker image to deploy", cli.Default("nginx:latest")),
		cli.Bool("create-deployment", "Also create a deployment after creating the project", cli.Default(true)),
	},
	Action: seedProject,
}

func seedProject(ctx context.Context, cmd *cli.Command) error {
	logger := logging.New()

	ctrlURL := cmd.String("ctrl-url")
	apiKey := cmd.String("api-key")
	workspaceID := cmd.String("workspace-id")
	projectName := cmd.String("project-name")
	projectSlug := cmd.String("project-slug")
	environmentSlug := cmd.String("environment-slug")
	gitRepo := cmd.String("git-repo")
	gitBranch := cmd.String("git-branch")
	dockerImage := cmd.String("docker-image")
	createDeployment := cmd.Bool("create-deployment")

	// Generate project slug if not provided
	if projectSlug == "" {
		projectSlug = fmt.Sprintf("proj-%d", time.Now().Unix())
	}

	// Create HTTP client with API key authentication
	httpClient := &http.Client{
		Transport: &apiKeyTransport{
			apiKey: apiKey,
			base:   http.DefaultTransport,
		},
	}

	// Create control plane client
	projectClient := ctrlv1connect.NewProjectServiceClient(
		httpClient,
		ctrlURL,
	)

	// Create project
	logger.Info("creating project via control plane API",
		"workspace", workspaceID,
		"name", projectName,
		"slug", projectSlug,
	)

	createProjReq := connect.NewRequest(&ctrlv1.CreateProjectRequest{
		WorkspaceId:   workspaceID,
		Name:          projectName,
		Slug:          projectSlug,
		GitRepository: gitRepo,
	})

	projResp, err := projectClient.CreateProject(ctx, createProjReq)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	projectID := projResp.Msg.GetId()
	logger.Info("project created successfully", "id", projectID)

	// Create deployment if requested
	if createDeployment {
		deploymentClient := ctrlv1connect.NewDeploymentServiceClient(
			httpClient,
			ctrlURL,
		)

		logger.Info("creating deployment via control plane API",
			"project", projectID,
			"environment", environmentSlug,
			"image", dockerImage,
		)

		createDepReq := connect.NewRequest(&ctrlv1.CreateDeploymentRequest{
			ProjectId:       projectID,
			EnvironmentSlug: environmentSlug,
			Branch:          gitBranch,
			Source: &ctrlv1.CreateDeploymentRequest_DockerImage{
				DockerImage: dockerImage,
			},
			KeyspaceId: nil,
			GitCommit: &ctrlv1.GitCommitInfo{
				CommitSha:       "abc123def456",
				CommitMessage:   "Initial deployment via seed",
				AuthorHandle:    "seed-script",
				AuthorAvatarUrl: "",
				Timestamp:       time.Now().UnixMilli(),
			},
		})

		depResp, err := deploymentClient.CreateDeployment(ctx, createDepReq)
		if err != nil {
			return fmt.Errorf("failed to create deployment: %w", err)
		}

		deploymentID := depResp.Msg.GetDeploymentId()
		logger.Info("deployment created successfully",
			"id", deploymentID,
			"status", depResp.Msg.GetStatus().String(),
		)

		// Optionally wait for deployment to be ready
		if depResp.Msg.GetStatus() == ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING ||
			depResp.Msg.GetStatus() == ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_BUILDING {
			logger.Info("deployment is being processed, check status with control plane API")
		}
	}

	logger.Info("seed completed successfully",
		"workspace", workspaceID,
		"project", projectID,
		"projectSlug", projectSlug,
	)

	return nil
}

// apiKeyTransport adds API key authentication to HTTP requests
type apiKeyTransport struct {
	apiKey string
	base   http.RoundTripper
}

func (t *apiKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.apiKey))
	return t.base.RoundTrip(req)
}
