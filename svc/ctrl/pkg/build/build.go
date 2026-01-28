package build

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
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
	"github.com/opencontainers/go-digest"
	restate "github.com/restatedev/sdk-go"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
)

const (
	// defaultCacheKeepGB is the maximum cache size in gigabytes for new Depot
	// projects. Depot evicts least-recently-used cache entries when exceeded.
	defaultCacheKeepGB = 50

	// defaultCacheKeepDays is the maximum age in days for cached build layers.
	// Layers older than this are evicted regardless of cache size.
	defaultCacheKeepDays = 14
)

// BuildDockerImage builds a container image using Depot and pushes it to the
// configured registry.
//
// The method retrieves or creates a Depot project for the Unkey project,
// acquires a remote build machine, and executes the build. Build progress
// is streamed to ClickHouse for observability. On success, returns the
// fully-qualified image name and Depot metadata.
//
// Required request fields: S3Url (build context), BuildContextPath, ProjectId,
// DeploymentId, and DockerfilePath. All fields are validated; missing fields
// result in a terminal error.
//
// Returns a terminal error for validation failures. Other errors may be
// retried by Restate.
func (s *Depot) BuildDockerImage(
	ctx restate.Context,
	req *hydrav1.BuildDockerImageRequest,
) (*hydrav1.BuildDockerImageResponse, error) {

	unkeyProjectID := req.GetProjectId()

	if err := assert.All(
		assert.NotEmpty(req.GetS3Url(), "s3_url is required"),
		assert.NotEmpty(req.GetBuildContextPath(), "build_context_path is required"),
		assert.NotEmpty(unkeyProjectID, "project_id is required"),
		assert.NotEmpty(req.GetDeploymentId(), "deployment_id is required"),
		assert.NotEmpty(req.GetDockerfilePath(), "dockerfile_path is required"),
	); err != nil {
		return nil, restate.TerminalError(err)
	}

	platform := s.buildPlatform.Platform
	architecture := s.buildPlatform.Architecture

	s.logger.Info("Starting build process - getting presigned URL for build context",
		"build_context_path", req.GetBuildContextPath(),
		"unkey_project_id", unkeyProjectID,
		"platform", platform,
		"architecture", architecture)

	depotProjectID, err := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
		return s.getOrCreateDepotProject(runCtx, unkeyProjectID)
	}, restate.WithName("get or create depot project"))
	if err != nil {
		return nil, fmt.Errorf("failed to get/create depot project: %w", err)
	}

	s.logger.Info("Creating depot build",
		"depot_project_id", depotProjectID,
		"unkey_project_id", unkeyProjectID)

	return restate.Run(ctx, func(runCtx restate.RunContext) (*hydrav1.BuildDockerImageResponse, error) {

		depotBuild, err := build.NewBuild(runCtx, &cliv1.CreateBuildRequest{
			Options:   nil,
			ProjectId: depotProjectID,
		}, s.registryConfig.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to create build: %w", err)
		}
		defer depotBuild.Finish(err)

		s.logger.Info("Depot build created",
			"build_id", depotBuild.ID,
			"depot_project_id", depotProjectID,
			"unkey_project_id", unkeyProjectID)

		s.logger.Info("Acquiring build machine",
			"build_id", depotBuild.ID,
			"architecture", architecture,
			"unkey_project_id", unkeyProjectID)

		buildkit, err := machine.Acquire(runCtx, depotBuild.ID, depotBuild.Token, architecture)
		if err != nil {
			return nil, fmt.Errorf("failed to acquire machine: %w", err)
		}
		defer func() {
			if releaseErr := buildkit.Release(); releaseErr != nil {
				s.logger.Error("unable to release buildkit", "error", releaseErr)
			}
		}()

		s.logger.Info("Build machine acquired, connecting to buildkit",
			"build_id", depotBuild.ID,
			"unkey_project_id", unkeyProjectID)

		buildClient, err := buildkit.Connect(runCtx)
		if err != nil {
			return nil, fmt.Errorf("unable to create build client: %w", err)
		}
		defer func() {
			if closeErr := buildClient.Close(); closeErr != nil {
				s.logger.Error("unable to close client", "error", closeErr)
			}
		}()

		imageName := fmt.Sprintf("%s/%s:%s-%s", s.registryConfig.URL, depotProjectID, unkeyProjectID, req.GetDeploymentId())

		dockerfilePath := req.GetDockerfilePath()
		if dockerfilePath == "" {
			dockerfilePath = "Dockerfile"
		}

		s.logger.Info("Starting build execution",
			"image_name", imageName,
			"dockerfile", dockerfilePath,
			"platform", platform,
			"architecture", architecture,
			"build_id", depotBuild.ID,
			"unkey_project_id", unkeyProjectID)

		buildStatusCh := make(chan *client.SolveStatus, 100)
		go s.processBuildStatus(buildStatusCh, req.GetWorkspaceId(), unkeyProjectID, req.GetDeploymentId())

		solverOptions := s.buildSolverOptions(platform, req.GetS3Url(), dockerfilePath, imageName)

		_, err = buildClient.Solve(runCtx, nil, solverOptions, buildStatusCh)
		if err != nil {
			return nil, fmt.Errorf("build failed: %w", err)
		}

		s.logger.Info("Build completed successfully")

		return &hydrav1.BuildDockerImageResponse{
			ImageName:      imageName,
			DepotBuildId:   depotBuild.ID,
			DepotProjectId: depotProjectID,
		}, nil
	})
}

// buildSolverOptions constructs the buildkit solver configuration for a build.
// It configures the dockerfile frontend, sets the platform and context URL,
// attaches registry authentication, and configures image export with push.
func (s *Depot) buildSolverOptions(
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
						s.registryConfig.URL: {
							Username: s.registryConfig.Username,
							Password: s.registryConfig.Password,
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

// getOrCreateDepotProject retrieves the Depot project ID for an Unkey project,
// creating one if it doesn't exist. The mapping is persisted to the database
// so subsequent builds reuse the same Depot project and its cache.
//
// New projects are named "unkey-{projectID}" and created in the region
// specified by [DepotConfig.ProjectRegion] with the default cache policy.
func (s *Depot) getOrCreateDepotProject(ctx context.Context, unkeyProjectID string) (string, error) {
	project, err := db.Query.FindProjectById(ctx, s.db.RO(), unkeyProjectID)
	if err != nil {
		return "", fmt.Errorf("failed to query project: %w", err)
	}

	projectName := fmt.Sprintf("unkey-%s", unkeyProjectID)
	if project.DepotProjectID.Valid && project.DepotProjectID.String != "" {
		s.logger.Info(
			"Returning existing depot project",
			"depot_project_id", project.DepotProjectID,
			"unkey_project_id", unkeyProjectID,
			"project_name", projectName,
		)
		return project.DepotProjectID.String, nil
	}

	httpClient := &http.Client{}
	authInterceptor := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+s.registryConfig.Password)
			return next(ctx, req)
		}
	})

	projectClient := corev1connect.NewProjectServiceClient(httpClient, s.depotConfig.APIUrl, connect.WithInterceptors(authInterceptor))
	//nolint: exhaustruct // optional fields
	createResp, err := projectClient.CreateProject(ctx, connect.NewRequest(&corev1.CreateProjectRequest{
		Name:     projectName,
		RegionId: s.depotConfig.ProjectRegion,
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
	err = db.Query.UpdateProjectDepotID(ctx, s.db.RW(), db.UpdateProjectDepotIDParams{
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

	s.logger.Info("Created new Depot project",
		"depot_project_id", depotProjectID,
		"unkey_project_id", unkeyProjectID,
		"project_name", projectName)

	return depotProjectID, nil
}

// processBuildStatus consumes build status events from buildkit and writes
// telemetry to ClickHouse. It tracks completed vertices (build steps) and
// their logs, buffering them for batch insertion.
//
// This method runs in its own goroutine and exits when statusCh is closed.
func (s *Depot) processBuildStatus(
	statusCh <-chan *client.SolveStatus,
	workspaceID, projectID, deploymentID string,
) {
	completed := map[digest.Digest]bool{}
	verticesWithLogs := map[digest.Digest]bool{}

	for status := range statusCh {
		// Mark vertices that have logs
		for _, log := range status.Logs {
			verticesWithLogs[log.Vertex] = true
		}

		// Process completed vertices
		for _, vertex := range status.Vertexes {
			if vertex == nil {
				s.logger.Warn("vertex is nil")
				continue
			}
			if vertex.Completed != nil && !completed[vertex.Digest] {
				completed[vertex.Digest] = true

				s.clickhouse.BufferBuildStep(schema.BuildStepV1{
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

		// Process logs
		for _, log := range status.Logs {
			s.clickhouse.BufferBuildStepLog(schema.BuildStepLogV1{
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
