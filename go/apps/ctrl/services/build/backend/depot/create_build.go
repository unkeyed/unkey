package depot

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"buf.build/gen/go/depot/api/connectrpc/go/depot/build/v1/buildv1connect"
	"buf.build/gen/go/depot/api/connectrpc/go/depot/core/v1/corev1connect"
	buildv1 "buf.build/gen/go/depot/api/protocolbuffers/go/depot/build/v1"
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

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
)

const (
	// Cache policy constants for Depot projects
	defaultCacheKeepGB   = 50
	defaultCacheKeepDays = 14
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
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to create build: %w", err))
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
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to acquire machine: %w", buildErr))
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
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to connect to buildkit: %w", buildErr))
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

	//nolint: exhaustruct
	solverOptions := client.SolveOpt{
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
	buildStatusCh := make(chan *client.SolveStatus, 10)
	go func() {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		for status := range buildStatusCh {
			_ = enc.Encode(status)
		}
	}()
	_, buildErr = buildkitClient.Solve(ctx, nil, solverOptions, buildStatusCh)
	if buildErr != nil {
		s.logger.Error("Build failed",
			"error", buildErr,
			"image_name", imageName,
			"build_id", buildResp.ID,
			"depot_project_id", depotProjectID,
			"unkey_project_id", unkeyProjectID)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("build failed: %w", buildErr))
	}

	s.logger.Info("Build completed successfully",
		"image_name", imageName,
		"build_id", buildResp.ID,
		"build_token", buildResp.Token,
		"depot_project_id", depotProjectID,
		"platform", platform,
		"architecture", architecture,
		"unkey_project_id", unkeyProjectID)

	// Fetch and print build steps and logs from Depot API
	if err := s.printBuildLogs(ctx, buildResp.ID, buildResp.Token, depotProjectID); err != nil {
		s.logger.Error("Failed to fetch build logs from Depot",
			"error", err,
			"build_id", buildResp.ID)
		// Don't fail the request - logs are optional
	}

	return connect.NewResponse(&ctrlv1.CreateBuildResponse{
		ImageName:      imageName,
		BuildId:        buildResp.ID,
		DepotProjectId: depotProjectID,
	}), nil
}

// printBuildLogs fetches build steps and their logs from Depot API and prints them
func (s *Depot) printBuildLogs(ctx context.Context, buildID, buildToken, projectID string) error {
	httpClient := &http.Client{}
	authInterceptor := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+s.registryConfig.Password) // Depot org token
			return next(ctx, req)
		}
	})

	buildClient := buildv1connect.NewBuildServiceClient(
		httpClient,
		s.depotConfig.APIUrl,
		connect.WithInterceptors(authInterceptor),
	)

	// Get build steps
	stepsResp, err := buildClient.GetBuildSteps(ctx, connect.NewRequest(&buildv1.GetBuildStepsRequest{
		BuildId:   buildID,
		ProjectId: projectID,
	}))
	if err != nil {
		return fmt.Errorf("failed to get build steps: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	fmt.Println("\n=== BUILD STEPS ===")
	if err := enc.Encode(stepsResp.Msg); err != nil {
		return fmt.Errorf("failed to encode steps: %w", err)
	}

	for _, step := range stepsResp.Msg.GetBuildSteps() {
		logsResp, err := buildClient.GetBuildStepLogs(ctx, connect.NewRequest(&buildv1.GetBuildStepLogsRequest{
			BuildId:         buildID,
			ProjectId:       projectID,
			BuildStepDigest: step.GetDigest(),
		}))
		if err != nil {
			s.logger.Error("Failed to get logs for step",
				"error", err,
				"build_id", buildID,
				"digest", step.GetDigest())
			continue
		}

		fmt.Printf("\n=== LOGS FOR STEP %s ===\n", step.GetDigest())
		if err := enc.Encode(logsResp.Msg.GetLogs()); err != nil {
			s.logger.Error("Failed to encode logs for step",
				"error", err)
		}
	}

	return nil
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
