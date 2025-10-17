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

type authTransport struct {
	token string
	base  http.RoundTripper
}

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
		assert.NotEmpty(req.Msg.ContextKey, "contextKey is required"),
		assert.NotEmpty(req.Msg.UnkeyProjectId, "unkeyProjectID is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	const architecture = "arm64"
	platform := fmt.Sprintf("linux/%s", architecture)

	s.logger.Info("Starting build process",
		"context_key", req.Msg.ContextKey,
		"unkey_project_id", req.Msg.UnkeyProjectId)

	s.logger.Info("Getting presigned URL for build context",
		"context_key", req.Msg.ContextKey,
		"unkey_project_id", req.Msg.UnkeyProjectId)

	contextURL, err := s.storage.GetPresignedURL(ctx, req.Msg.ContextKey, 15*time.Minute)
	if err != nil {
		s.logger.Error("Failed to get presigned URL",
			"error", err,
			"context_key", req.Msg.ContextKey,
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
	}, s.accessToken)
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
		"platform", architecture,
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

	// INFO: "s.registryUrl", "depotProjectID" order of these two arg must never change, otherwise depot will decline the registry upload.
	imageName := fmt.Sprintf("%s/%s:%s-%s", s.registryUrl, depotProjectID, req.Msg.UnkeyProjectId, req.Msg.DeploymentId)

	dockerfilePath := req.Msg.GetDockerfilePath()
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}

	s.logger.Info("Starting build execution",
		"image_name", imageName,
		"dockerfile", dockerfilePath,
		"platform", platform,
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
						s.registryUrl: {
							Username: s.username,
							Password: s.accessToken,
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
			"depot_project_id",
			project.DepotProjectID,
			"unkey_project_id",
			"project_name",
			projectName,
		)
		return project.DepotProjectID.String, nil
	}

	httpClient := &http.Client{
		Transport: &authTransport{
			token: s.accessToken,
			base:  http.DefaultTransport,
		},
	}

	projectClient := corev1connect.NewProjectServiceClient(httpClient, s.apiUrl)
	createResp, err := projectClient.CreateProject(ctx, connect.NewRequest(&corev1.CreateProjectRequest{
		Name:     projectName,
		RegionId: "us-east-1",
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

	s.logger.Info("Created new Depot project", "depot_project_id", createResp.Msg.Project.ProjectId, "unkey_project_id", unkeyProjectID, "project_name", projectName)

	return createResp.Msg.Project.ProjectId, nil
}

// RoundTrip implements http.RoundTripper by adding Bearer token authentication to requests.
// This is required for authenticating with the Depot API when creating projects programmatically.
func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req)
}
