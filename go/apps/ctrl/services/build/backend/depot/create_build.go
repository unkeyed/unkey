package depot

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"buf.build/gen/go/depot/api/connectrpc/go/depot/core/v1/corev1connect"
	"connectrpc.com/connect"
	"github.com/depot/depot-go/build"
	"github.com/depot/depot-go/machine"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"

	corev1 "buf.build/gen/go/depot/api/protocolbuffers/go/depot/core/v1"
	cliv1 "github.com/depot/depot-go/proto/depot/cli/v1"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
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
	if err := assert.All(
		assert.NotEmpty(req.Msg.BuildContextPath, "build_context_path is required"),
		assert.NotEmpty(req.Msg.UnkeyProjectId, "unkey_project_id is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	platform := s.buildPlatform.Platform
	architecture := s.buildPlatform.Architecture

	s.logger.Info("Starting build process - getting presigned URL for build context",
		"build_context_path", req.Msg.BuildContextPath,
		"unkey_project_id", req.Msg.UnkeyProjectId,
		"platform", platform,
		"architecture", architecture)

	contextURL, err := s.storage.GenerateDownloadURL(ctx, req.Msg.BuildContextPath, 15*time.Minute)
	if err != nil {
		s.logger.Error("Failed to get presigned URL",
			"error", err,
			"build_context_path", req.Msg.BuildContextPath,
			"unkey_project_id", req.Msg.UnkeyProjectId)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to get presigned URL: %w", err))
	}

	depotProjectID, err := s.getOrCreateDepotProject(ctx, req.Msg.UnkeyProjectId)
	if err != nil {
		s.logger.Error("Failed to get/create depot project",
			"error", err,
			"unkey_project_id", req.Msg.UnkeyProjectId)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to get/create depot project: %w", err))
	}

	s.logger.Info("Creating depot build",
		"depot_project_id", depotProjectID,
		"unkey_project_id", req.Msg.UnkeyProjectId)

	buildResp, err := build.NewBuild(ctx, &cliv1.CreateBuildRequest{
		ProjectId: depotProjectID,
	}, s.registryConfig.Password)
	if err != nil {
		s.logger.Error("Creating depot build failed",
			"error", err,
			"depot_project_id", depotProjectID,
			"unkey_project_id", req.Msg.UnkeyProjectId)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to create build: %w", err))
	}

	s.logger.Info("Depot build created",
		"build_id", buildResp.ID,
		"depot_project_id", depotProjectID,
		"unkey_project_id", req.Msg.UnkeyProjectId)

	var buildErr error
	defer buildResp.Finish(buildErr)

	s.logger.Info("Acquiring build machine",
		"build_id", buildResp.ID,
		"architecture", architecture,
		"unkey_project_id", req.Msg.UnkeyProjectId)

	buildkit, buildErr := machine.Acquire(ctx, buildResp.ID, buildResp.Token, architecture)
	if buildErr != nil {
		s.logger.Error("Acquiring depot build failed",
			"error", buildErr,
			"build_id", buildResp.ID,
			"depot_project_id", depotProjectID,
			"unkey_project_id", req.Msg.UnkeyProjectId)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to acquire machine: %w", buildErr))
	}
	defer buildkit.Release()

	s.logger.Info("Build machine acquired, connecting to buildkit",
		"build_id", buildResp.ID,
		"unkey_project_id", req.Msg.UnkeyProjectId)

	buildkitClient, buildErr := buildkit.Connect(ctx)
	if buildErr != nil {
		s.logger.Error("Connection to depot build failed",
			"error", buildErr,
			"build_id", buildResp.ID,
			"depot_project_id", depotProjectID,
			"unkey_project_id", req.Msg.UnkeyProjectId)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to connect to buildkit: %w", buildErr))
	}
	defer buildkitClient.Close()

	// INFO: "s.registryConfig.URL", "depotProjectID" order of these two arg must never change, otherwise depot will decline the registry upload.
	imageName := fmt.Sprintf("%s/%s:%s-%s", s.registryConfig.URL, depotProjectID, req.Msg.GetUnkeyProjectId(), req.Msg.GetDeploymentId())
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
		"unkey_project_id", req.Msg.UnkeyProjectId)

	solverOptions := client.SolveOpt{
		Frontend: "dockerfile.v0",
		FrontendAttrs: map[string]string{
			"platform": platform,
			"context":  contextURL,
			"filename": dockerfilePath,
		},
		Session: []session.Attachable{
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

	_, buildErr = buildkitClient.Solve(ctx, nil, solverOptions, nil)
	if buildErr != nil {
		s.logger.Error("Build failed",
			"error", buildErr,
			"image_name", imageName,
			"build_id", buildResp.ID,
			"depot_project_id", depotProjectID,
			"unkey_project_id", req.Msg.UnkeyProjectId)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("build failed: %w", buildErr))
	}

	s.logger.Info("Build completed successfully",
		"image_name", imageName,
		"build_id", buildResp.ID,
		"depot_project_id", depotProjectID,
		"platform", platform,
		"architecture", architecture,
		"unkey_project_id", req.Msg.UnkeyProjectId)

	return connect.NewResponse(&ctrlv1.CreateBuildResponse{
		ImageName:      imageName,
		BuildId:        buildResp.ID,
		DepotProjectId: depotProjectID,
	}), nil
}

// getOrCreateDepotProject retrieves or creates a Depot project for the given Unkey project.
//
// Steps:
//
//	Check database for existing project mapping
//	Create new Depot project if not found
//	Store project mapping in database
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
	createResp, err := projectClient.CreateProject(ctx, connect.NewRequest(&corev1.CreateProjectRequest{
		Name:     projectName,
		RegionId: s.depotConfig.ProjectRegion,
		CachePolicy: &corev1.CachePolicy{
			KeepGb:   50,
			KeepDays: 14,
		},
	}))
	if err != nil {
		return "", fmt.Errorf("failed to create project: %w", err)
	}

	now := time.Now().UnixMilli()
	err = db.Query.UpdateProjectDepotID(ctx, s.db.RW(), db.UpdateProjectDepotIDParams{
		DepotProjectID: sql.NullString{
			String: createResp.Msg.Project.ProjectId,
			Valid:  true,
		},
		UpdatedAt: sql.NullInt64{Int64: now, Valid: true},
		ID:        unkeyProjectID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to update depot_project_id: %w", err)
	}

	s.logger.Info("Created new Depot project",
		"depot_project_id", createResp.Msg.Project.ProjectId,
		"unkey_project_id", unkeyProjectID,
		"project_name", projectName)

	return createResp.Msg.Project.ProjectId, nil
}
