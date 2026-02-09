package deploy

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"buf.build/gen/go/depot/api/connectrpc/go/depot/core/v1/corev1connect"
	corev1 "buf.build/gen/go/depot/api/protocolbuffers/go/depot/core/v1"
	"connectrpc.com/connect"
	"github.com/depot/depot-go/build"
	"github.com/depot/depot-go/machine"
	cliv1 "github.com/depot/depot-go/proto/depot/cli/v1"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/session/secrets/secretsprovider"
	"github.com/opencontainers/go-digest"
	restate "github.com/restatedev/sdk-go"

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ptr"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

const (
	// defaultCacheKeepGB is the maximum cache size in gigabytes for new Depot
	// projects. Depot evicts least-recently-used cache entries when exceeded.
	defaultCacheKeepGB = 50

	// defaultCacheKeepDays is the maximum age in days for cached build layers.
	// Layers older than this are evicted regardless of cache size.
	defaultCacheKeepDays = 14
)

// buildResult contains the output of a Docker image build, including the image
// name and identifiers needed to trace builds in Depot.
type buildResult struct {
	ImageName      string
	DepotBuildID   string
	DepotProjectID string
}

// gitBuildParams holds the inputs for building a container image from a Git
// repository, including the exact commit and the build context location.
type gitBuildParams struct {
	InstallationID int64
	Repository     string
	CommitSHA      string
	ContextPath    string
	DockerfilePath string
	ProjectID      string
	DeploymentID   string
	WorkspaceID    string
}

// buildDockerImageFromGit builds a container image from a GitHub repository using Depot.
//
// The method retrieves or creates a Depot project for the Unkey project,
// acquires a remote build machine, and executes the build. BuildKit fetches
// the repository directly from GitHub using the provided installation token.
// Build progress is streamed to ClickHouse for observability.
func (w *Workflow) buildDockerImageFromGit(
	ctx restate.Context,
	params gitBuildParams,
) (*buildResult, error) {
	platform := w.buildPlatform.Platform
	architecture := w.buildPlatform.Architecture

	logger.Info("Starting git build process",
		"repository", params.Repository,
		"commit_sha", params.CommitSHA,
		"project_id", params.ProjectID,
		"platform", platform,
		"architecture", architecture)

	depotProjectID, err := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
		return w.getOrCreateDepotProject(runCtx, params.ProjectID)
	}, restate.WithName("get or create depot project"))
	if err != nil {
		return nil, fmt.Errorf("failed to get/create depot project: %w", err)
	}

	logger.Info("Creating depot build",
		"depot_project_id", depotProjectID,
		"project_id", params.ProjectID)

	return restate.Run(ctx, func(runCtx restate.RunContext) (*buildResult, error) {
		// Get GitHub installation token for BuildKit to fetch the repo
		var ghToken githubclient.InstallationToken
		if w.allowUnauthenticatedDeployments {
			// Unauthenticated mode - skip GitHub auth for public repos (local dev only)
			logger.Info("Unauthenticated mode: skipping GitHub authentication for public repo",
				"repository", params.Repository)
		} else {
			token, err := w.github.GetInstallationToken(params.InstallationID)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitHub installation token: %w", err)
			}
			ghToken = token
		}

		depotBuild, err := build.NewBuild(runCtx, &cliv1.CreateBuildRequest{
			Options:   nil,
			ProjectId: depotProjectID,
		}, w.registryConfig.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to create build: %w", err)
		}
		defer func() { depotBuild.Finish(err) }()

		logger.Info("Depot build created",
			"build_id", depotBuild.ID,
			"depot_project_id", depotProjectID,
			"project_id", params.ProjectID)

		logger.Info("Acquiring build machine",
			"build_id", depotBuild.ID,
			"architecture", architecture,
			"project_id", params.ProjectID)

		buildkit, err := machine.Acquire(runCtx, depotBuild.ID, depotBuild.Token, architecture)
		if err != nil {
			return nil, fmt.Errorf("failed to acquire machine: %w", err)
		}
		defer func() {
			if releaseErr := buildkit.Release(); releaseErr != nil {
				logger.Error("unable to release buildkit", "error", releaseErr)
			}
		}()

		logger.Info("Build machine acquired, connecting to buildkit",
			"build_id", depotBuild.ID,
			"project_id", params.ProjectID)

		buildClient, err := buildkit.Connect(runCtx)
		if err != nil {
			return nil, fmt.Errorf("unable to create build client: %w", err)
		}
		defer func() {
			if closeErr := buildClient.Close(); closeErr != nil {
				logger.Error("unable to close client", "error", closeErr)
			}
		}()

		imageName := fmt.Sprintf("%s/%s:%s-%s", w.registryConfig.URL, depotProjectID, params.ProjectID, params.DeploymentID)

		dockerfilePath := params.DockerfilePath
		if dockerfilePath == "" {
			dockerfilePath = "Dockerfile"
		}

		// Normalize context path: trim whitespace and leading slashes, treat "." as root
		contextPath := strings.TrimSpace(params.ContextPath)
		contextPath = strings.TrimPrefix(contextPath, "/")
		if contextPath == "." {
			contextPath = ""
		}

		// Build git context URL with commit SHA
		// Format: https://github.com/owner/repo.git#<ref>:<subdir>
		// Note: BuildKit requires full 40-char SHA for reliable builds
		gitContextURL := fmt.Sprintf("https://github.com/%s.git#%s", params.Repository, params.CommitSHA)
		if contextPath != "" {
			gitContextURL = fmt.Sprintf("https://github.com/%s.git#%s:%s", params.Repository, params.CommitSHA, contextPath)
		}

		logger.Info("Starting build execution",
			"image_name", imageName,
			"dockerfile", dockerfilePath,
			"platform", platform,
			"architecture", architecture,
			"build_id", depotBuild.ID,
			"project_id", params.ProjectID,
			"git_context_url", gitContextURL,
		)

		buildStatusCh := make(chan *client.SolveStatus, 100)
		go w.processBuildStatus(buildStatusCh, params.WorkspaceID, params.ProjectID, params.DeploymentID)

		// Choose solver options based on authentication mode
		var solverOptions client.SolveOpt
		if w.allowUnauthenticatedDeployments {
			solverOptions = w.buildSolverOptions(platform, gitContextURL, dockerfilePath, imageName)
		} else {
			solverOptions = w.buildGitSolverOptions(platform, gitContextURL, dockerfilePath, imageName, ghToken.Token)
		}

		_, err = buildClient.Solve(runCtx, nil, solverOptions, buildStatusCh)
		if err != nil {
			// Build failures (bad Dockerfile, compilation errors, etc.) won't fix
			// themselves on retry — mark as terminal to stop Restate from retrying.
			return nil, restate.TerminalError(fmt.Errorf("build failed: %w", err))
		}

		logger.Info("Build completed successfully")

		return &buildResult{
			ImageName:      imageName,
			DepotBuildID:   depotBuild.ID,
			DepotProjectID: depotProjectID,
		}, nil
	}, restate.WithName("build docker image from git"))
}

// buildSolverOptions constructs the BuildKit solver configuration for URL-based
// contexts, including registry auth and image export settings. Use
// [Workflow.buildGitSolverOptions] when the context requires GitHub credentials.
func (w *Workflow) buildSolverOptions(
	platform, contextURL, dockerfilePath, imageName string,
) client.SolveOpt {
	return client.SolveOpt{
		Frontend: "dockerfile.v0",
		FrontendAttrs: map[string]string{
			"platform": platform,
			"context":  contextURL,
			"filename": dockerfilePath,
		},

		Session: []session.Attachable{
			//nolint: exhaustruct
			authprovider.NewDockerAuthProvider(authprovider.DockerAuthProviderConfig{
				ConfigFile: &configfile.ConfigFile{
					AuthConfigs: map[string]types.AuthConfig{
						w.registryConfig.URL: {
							Username: w.registryConfig.Username,
							Password: w.registryConfig.Password,
						},
					},
				},
			}),
		},
		//nolint: exhaustruct
		Exports: []client.ExportEntry{
			{
				Type: "image",
				Attrs: map[string]string{
					"name":           imageName,
					"oci-mediatypes": "true",
					"push":           "true",
				},
			},
		},
	}
}

// buildGitSolverOptions constructs the buildkit solver configuration for a git context build.
// It includes GitHub token authentication via the secrets provider.
func (w *Workflow) buildGitSolverOptions(
	platform, gitContextURL, dockerfilePath, imageName, githubToken string,
) client.SolveOpt {
	return client.SolveOpt{
		Frontend: "dockerfile.v0",
		FrontendAttrs: map[string]string{
			"platform": platform,
			"context":  gitContextURL,
			"filename": dockerfilePath,
		},

		Session: []session.Attachable{
			//nolint: exhaustruct
			authprovider.NewDockerAuthProvider(authprovider.DockerAuthProviderConfig{
				ConfigFile: &configfile.ConfigFile{
					AuthConfigs: map[string]types.AuthConfig{
						w.registryConfig.URL: {
							Username: w.registryConfig.Username,
							Password: w.registryConfig.Password,
						},
					},
				},
			}),
			// Provide GitHub token for BuildKit to authenticate when fetching the git repo
			secretsprovider.FromMap(map[string][]byte{
				"GIT_AUTH_TOKEN.github.com": []byte(githubToken),
			}),
		},
		//nolint: exhaustruct
		Exports: []client.ExportEntry{
			{
				Type: "image",
				Attrs: map[string]string{
					"name":           imageName,
					"oci-mediatypes": "true",
					"push":           "true",
				},
			},
		},
	}
}

// getOrCreateDepotProject retrieves the Depot project ID for an Unkey project,
// creating one if it doesn't exist.
func (w *Workflow) getOrCreateDepotProject(ctx context.Context, unkeyProjectID string) (string, error) {
	project, err := db.Query.FindProjectById(ctx, w.db.RO(), unkeyProjectID)
	if err != nil {
		return "", fmt.Errorf("failed to query project: %w", err)
	}

	projectName := fmt.Sprintf("unkey-%s", unkeyProjectID)
	if project.DepotProjectID.Valid && project.DepotProjectID.String != "" {
		logger.Info(
			"Returning existing depot project",
			"depot_project_id", project.DepotProjectID,
			"project_id", unkeyProjectID,
			"project_name", projectName,
		)
		return project.DepotProjectID.String, nil
	}

	httpClient := &http.Client{}
	authInterceptor := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+w.registryConfig.Password)
			return next(ctx, req)
		}
	})

	projectClient := corev1connect.NewProjectServiceClient(httpClient, w.depotConfig.APIUrl, connect.WithInterceptors(authInterceptor))
	//nolint: exhaustruct // optional fields
	createResp, err := projectClient.CreateProject(ctx, connect.NewRequest(&corev1.CreateProjectRequest{
		Name:     projectName,
		RegionId: w.depotConfig.ProjectRegion,
		//nolint: exhaustruct // missing fields is deprecated
		CachePolicy: &corev1.CachePolicy{
			KeepGb:   defaultCacheKeepGB,
			KeepDays: defaultCacheKeepDays,
		},
	}))
	if err != nil {
		return "", fmt.Errorf("failed to create project: %w", err)
	}
	depotProjectID := createResp.Msg.GetProject().GetProjectId()

	now := time.Now().UnixMilli()
	err = db.Query.UpdateProjectDepotID(ctx, w.db.RW(), db.UpdateProjectDepotIDParams{
		DepotProjectID: sql.NullString{
			String: depotProjectID,
			Valid:  true,
		},
		UpdatedAt: sql.NullInt64{Int64: now, Valid: true},
		ID:        unkeyProjectID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to update depot_project_id: %w", err)
	}

	logger.Info("Created new Depot project",
		"depot_project_id", depotProjectID,
		"project_id", unkeyProjectID,
		"project_name", projectName)

	return depotProjectID, nil
}

// processBuildStatus consumes build status events from buildkit and writes
// telemetry to ClickHouse.
func (w *Workflow) processBuildStatus(
	statusCh <-chan *client.SolveStatus,
	workspaceID, projectID, deploymentID string,
) {
	completed := map[digest.Digest]bool{}
	verticesWithLogs := map[digest.Digest]bool{}

	for status := range statusCh {
		for _, log := range status.Logs {
			verticesWithLogs[log.Vertex] = true
		}

		for _, vertex := range status.Vertexes {
			if vertex == nil {
				logger.Warn("vertex is nil")
				continue
			}
			if vertex.Completed != nil && !completed[vertex.Digest] {
				completed[vertex.Digest] = true

				w.clickhouse.BufferBuildStep(schema.BuildStepV1{
					Error:        vertex.Error,
					StartedAt:    ptr.SafeDeref(vertex.Started).UnixMilli(),
					CompletedAt:  ptr.SafeDeref(vertex.Completed).UnixMilli(),
					WorkspaceID:  workspaceID,
					ProjectID:    projectID,
					DeploymentID: deploymentID,
					StepID:       vertex.Digest.String(),
					Name:         vertex.Name,
					Cached:       vertex.Cached,
					HasLogs:      verticesWithLogs[vertex.Digest],
				})
			}
		}

		for _, log := range status.Logs {
			w.clickhouse.BufferBuildStepLog(schema.BuildStepLogV1{
				WorkspaceID:  workspaceID,
				ProjectID:    projectID,
				DeploymentID: deploymentID,
				StepID:       log.Vertex.String(),
				Time:         log.Timestamp.UnixMilli(),
				Message:      string(log.Data),
			})
		}
	}
}
