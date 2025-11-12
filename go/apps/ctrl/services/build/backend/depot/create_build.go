package depot

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"slices"
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

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
)

const (
	// Cache policy constants for Depot projects
	defaultCacheKeepGB   = 50
	defaultCacheKeepDays = 14

	// Buffer size for BuildKit status updates channel.
	// If set too low, it will drop some of the buildStepLogs in big dockerfiles.
	buildStatusChannelBuffer = 1000
)

// CreateBuild orchestrates the container image build process using Depot.
//
// Steps:
//
//	Get or create Depot project
//	Register a new build with Depot
//	Acquire a build machine
//	Connect to the buildkit instance
//	Prepare build context and configuration
//	Execute the build with status logging
//	Return build metadata
func (s *Depot) CreateBuild(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateBuildRequest],
) (*connect.Response[ctrlv1.CreateBuildResponse], error) {
	buildContextPath := req.Msg.GetBuildContextPath()
	unkeyProjectID := req.Msg.GetUnkeyProjectId()
	deploymentID := req.Msg.GetDeploymentId()

	if err := assert.All(
		assert.NotEmpty(buildContextPath, "build_context_path is required"),
		assert.NotEmpty(unkeyProjectID, "unkey_project_id is required"),
		assert.NotEmpty(deploymentID, "deploymentID is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	platform := s.buildPlatform.Platform
	architecture := s.buildPlatform.Architecture

	s.logger.Info("Starting build process - getting presigned URL for build context",
		"build_context_path", buildContextPath,
		"unkey_project_id", unkeyProjectID,
		"platform", platform,
		"architecture", architecture)

	contextURL, err := s.storage.GenerateDownloadURL(ctx, buildContextPath, 15*time.Minute)
	if err != nil {
		s.logger.Error("Failed to get presigned URL",
			"error", err,
			"build_context_path", buildContextPath,
			"unkey_project_id", unkeyProjectID)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to get presigned URL: %w", err))
	}

	depotProjectID, err := s.getOrCreateDepotProject(ctx, unkeyProjectID)
	if err != nil {
		s.logger.Error("Failed to get/create depot project",
			"error", err,
			"unkey_project_id", unkeyProjectID)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to get/create depot project: %w", err))
	}

	s.logger.Info("Creating depot build",
		"depot_project_id", depotProjectID,
		"unkey_project_id", unkeyProjectID)

	buildResp, err := build.NewBuild(ctx, &cliv1.CreateBuildRequest{
		Options:   nil,
		ProjectId: depotProjectID,
	}, s.registryConfig.Password)
	if err != nil {
		s.logger.Error("Creating depot build failed",
			"error", err,
			"depot_project_id", depotProjectID,
			"unkey_project_id", unkeyProjectID)
		return nil, wrapBuildError(err, connect.CodeInternal, "failed to create build")
	}

	s.logger.Info("Depot build created",
		"build_id", buildResp.ID,
		"depot_project_id", depotProjectID,
		"unkey_project_id", unkeyProjectID)

	var buildErr error
	defer buildResp.Finish(buildErr)

	s.logger.Info("Acquiring build machine",
		"build_id", buildResp.ID,
		"architecture", architecture,
		"unkey_project_id", unkeyProjectID)

	var buildkit *machine.Machine
	buildkit, buildErr = machine.Acquire(ctx, buildResp.ID, buildResp.Token, architecture)
	if buildErr != nil {
		s.logger.Error("Acquiring depot build failed",
			"error", buildErr,
			"build_id", buildResp.ID,
			"depot_project_id", depotProjectID,
			"unkey_project_id", unkeyProjectID)
		return nil, wrapBuildError(buildErr, connect.CodeInternal, "failed to acquire machine")
	}
	//nolint: all
	defer buildkit.Release()

	s.logger.Info("Build machine acquired, connecting to buildkit",
		"build_id", buildResp.ID,
		"unkey_project_id", unkeyProjectID)

	var buildkitClient *client.Client
	buildkitClient, buildErr = buildkit.Connect(ctx)
	if buildErr != nil {
		s.logger.Error("Connection to depot build failed",
			"error", buildErr,
			"build_id", buildResp.ID,
			"depot_project_id", depotProjectID,
			"unkey_project_id", unkeyProjectID)
		return nil, wrapBuildError(buildErr, connect.CodeInternal, "failed to connect to buildkit")
	}
	defer buildkitClient.Close()

	imageName := fmt.Sprintf("%s/%s:%s-%s", s.registryConfig.URL, depotProjectID, unkeyProjectID, deploymentID)

	dockerfilePath := req.Msg.GetDockerfilePath()
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}

	s.logger.Info("Starting build execution",
		"image_name", imageName,
		"dockerfile", dockerfilePath,
		"platform", platform,
		"architecture", architecture,
		"build_id", buildResp.ID,
		"unkey_project_id", unkeyProjectID)

	buildStatusCh := make(chan *client.SolveStatus, buildStatusChannelBuffer)
	go s.processBuildStatusLogs(buildStatusCh, req.Msg.GetWorkspaceId(), unkeyProjectID, deploymentID)

	solverOptions := s.buildSolverOptions(platform, contextURL, dockerfilePath, imageName)
	_, buildErr = buildkitClient.Solve(ctx, nil, solverOptions, buildStatusCh)
	if buildErr != nil {
		s.logger.Error("Build failed",
			"error", buildErr,
			"image_name", imageName,
			"build_id", buildResp.ID,
			"depot_project_id", depotProjectID,
			"unkey_project_id", unkeyProjectID)
		return nil, wrapBuildError(buildErr, connect.CodeInternal, "build failed")
	}

	s.logger.Info("Build completed successfully")

	return connect.NewResponse(&ctrlv1.CreateBuildResponse{
		ImageName:      imageName,
		BuildId:        buildResp.ID,
		DepotProjectId: depotProjectID,
	}), nil
}

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

// getOrCreateDepotProject retrieves or creates a Depot project for the given Unkey project.
//
// Steps:
//
//	Check database for existing project mapping
//	Create new Depot project if not found
//	Store project mapping in database
func (s *Depot) getOrCreateDepotProject(ctx context.Context, unkeyProjectID string) (string, error) {
	project, err := db.WithRetryContext(ctx, func() (db.FindProjectByIdRow, error) {
		return db.Query.FindProjectById(ctx, s.db.RO(), unkeyProjectID)
	})
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
	_, err = db.WithRetryContext(ctx, func() (struct{}, error) {
		err := db.Query.UpdateProjectDepotID(ctx, s.db.RW(), db.UpdateProjectDepotIDParams{
			DepotProjectID: sql.NullString{
				String: depotProjectID,
				Valid:  true,
			},
			UpdatedAt: sql.NullInt64{Int64: now, Valid: true},
			ID:        unkeyProjectID,
		})
		return struct{}{}, err
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

type event struct {
	timestamp time.Time
	vertex    *client.Vertex
	log       *client.VertexLog
}

func (s *Depot) processBuildStatusLogs(
	statusCh <-chan *client.SolveStatus,
	workspaceID, projectID, deploymentID string,
) {
	// Collect all build events from BuildKit before processing
	// We buffer everything first because BuildKit sends events out-of-order,
	// and we need to sort them chronologically before inserting into CH
	var events []event

	// Track which build steps produced log output
	// This flag gets stored in the build_step table so we know which steps have associated logs
	verticesWithLogs := map[digest.Digest]bool{}

	// Stream all status updates from BuildKit until the build completes
	// Each status update contains logs and vertices
	for status := range statusCh {
		for _, log := range status.Logs {
			verticesWithLogs[log.Vertex] = true
			events = append(events, event{
				timestamp: log.Timestamp,
				log:       log,
			})
		}

		for _, vertex := range status.Vertexes {
			if vertex == nil {
				s.logger.Warn("vertex is nil")
				continue
			}
			if vertex.Completed != nil {
				// Use completion time for sorting, but fallback to start time if completion is earlier
				// happens with cached steps that complete instantly
				ts := *vertex.Completed
				if vertex.Started != nil && vertex.Started.After(ts) {
					ts = *vertex.Started
				}
				events = append(events, event{
					timestamp: ts,
					vertex:    vertex,
				})
			}
		}
	}

	// Sort events by timestamp to ensure chronological order in CH
	// BuildKit sends events asynchronously, so we must sort before inserting
	// Use stable sort because cached steps often complete at the exact same millisecond
	// and we need deterministic ordering when timestamps collide
	slices.SortStableFunc(events, func(left, right event) int {
		if left.timestamp.Before(right.timestamp) {
			return -1
		}
		if left.timestamp.After(right.timestamp) {
			return 1
		}
		return 0
	})

	completed := map[digest.Digest]bool{}
	for _, e := range events {
		if e.vertex != nil && !completed[e.vertex.Digest] {
			completed[e.vertex.Digest] = true
			s.clickhouse.BufferBuildStep(schema.BuildStepV1{
				Error:        e.vertex.Error,
				StartedAt:    ptr.SafeDeref(e.vertex.Started).UnixMilli(),
				CompletedAt:  ptr.SafeDeref(e.vertex.Completed).UnixMilli(),
				WorkspaceID:  workspaceID,
				ProjectID:    projectID,
				DeploymentID: deploymentID,
				StepID:       e.vertex.Digest.String(),
				Name:         e.vertex.Name,
				Cached:       e.vertex.Cached,
				HasLogs:      verticesWithLogs[e.vertex.Digest],
			})
		}

		if e.log != nil {
			s.clickhouse.BufferBuildStepLog(schema.BuildStepLogV1{
				WorkspaceID:  workspaceID,
				ProjectID:    projectID,
				DeploymentID: deploymentID,
				StepID:       e.log.Vertex.String(),
				Time:         e.log.Timestamp.UnixMilli(),
				Message:      string(e.log.Data),
			})
		}
	}
}
