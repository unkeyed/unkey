package depot

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
	"github.com/moby/buildkit/session/upload/uploadprovider"

	corev1 "buf.build/gen/go/depot/api/protocolbuffers/go/depot/core/v1"
	cliv1 "github.com/depot/depot-go/proto/depot/cli/v1"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
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
	// Validate input
	if req.Msg.ContextKey == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("contextKey is required"))
	}
	if req.Msg.UnkeyProjectID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unkeyProjectID is required"))
	}

	// Download build context from S3
	s.logger.Info("Downloading build context from S3", "context_key", req.Msg.ContextKey)
	tarData, exists, err := s.storage.GetObject(ctx, req.Msg.ContextKey)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to download build context: %w", err))
	}
	if !exists {
		return nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("build context not found: %s", req.Msg.ContextKey))
	}

	// Create temp file for the tar
	tmpFile, err := os.CreateTemp("", "build-context-*.tar.gz")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to create temp file: %w", err))
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write tar data to temp file
	if _, err := tmpFile.Write(tarData); err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to write build context: %w", err))
	}
	if err := tmpFile.Close(); err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to close temp file: %w", err))
	}

	s.logger.Info("Build context downloaded", "size_bytes", len(tarData), "temp_file", tmpFile.Name())

	depotProjectID, err := s.getOrCreateDepotProject(ctx, req.Msg.UnkeyProjectID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to get/create depot project: %w", err))
	}

	buildResp, err := build.NewBuild(ctx, &cliv1.CreateBuildRequest{
		ProjectId: depotProjectID,
	}, s.accessToken)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to create build: %w", err))
	}

	var buildErr error
	defer buildResp.Finish(buildErr)

	buildkit, buildErr := machine.Acquire(ctx, buildResp.ID, buildResp.Token, "arm64")
	if buildErr != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to acquire machine: %w", buildErr))
	}
	defer buildkit.Release()

	buildkitClient, buildErr := buildkit.Connect(ctx)
	if buildErr != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to connect to buildkit: %w", buildErr))
	}

	// Open the downloaded tar file
	dockerfileReader, buildErr := os.Open(tmpFile.Name())
	if buildErr != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to open build context: %w", buildErr))
	}
	defer dockerfileReader.Close()

	uploader := uploadprovider.New()
	contextURL := uploader.Add(dockerfileReader)

	timestamp := time.Now().UnixMilli()
	imageTag := fmt.Sprintf("%s/%s:%s-%d", s.registryUrl, depotProjectID, req.Msg.UnkeyProjectID, timestamp)

	// Use ImagePath from request, default to "Dockerfile"
	dockerfilePath := req.Msg.GetDockerfilePath()
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}

	solverOptions := client.SolveOpt{
		Frontend: "dockerfile.v0",
		FrontendAttrs: map[string]string{
			"platform": "linux/arm64",
			"context":  contextURL,
			"filename": dockerfilePath,
		},
		Session: []session.Attachable{
			uploader,
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
					"name":           imageTag,
					"oci-mediatypes": "true",
					"push":           "true",
				},
			},
		},
	}

	// TODO: Maybe we should push those to CH
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
		s.logger.Error("Build failed", "error", buildErr, "depot_project_id", depotProjectID, "unkey_project_id", req.Msg.UnkeyProjectID)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("build failed: %w", buildErr))
	}

	s.logger.Info("Build completed successfully", "image_tag", imageTag, "build_id", buildResp.ID, "depot_project_id", depotProjectID)

	return connect.NewResponse(&ctrlv1.CreateBuildResponse{
		ImageName:      imageTag,
		BuildId:        buildResp.ID,
		DepotProjectId: depotProjectID,
	}), nil
}

// getOrCreateDepotProject retrieves or creates a Depot project for the given Unkey project.
//
// Steps:
//
//	Check database for existing project mapping
//	List all Depot projects and search by name
//	Return existing project ID if found
//	Create new Depot project if not found
//	Store project mapping in database
func (s *Depot) getOrCreateDepotProject(ctx context.Context, unkeyProjectID string) (string, error) {
	project, err := db.Query.FindProjectById(ctx, s.db.RO(), unkeyProjectID)
	if err != nil {
		return "", fmt.Errorf("failed to query project: %w", err)
	}

	if project.DepotProjectID.Valid && project.DepotProjectID.String != "" {
		return project.DepotProjectID.String, nil
	}

	httpClient := &http.Client{
		Transport: &authTransport{
			token: s.accessToken,
			base:  http.DefaultTransport,
		},
	}

	projectClient := corev1connect.NewProjectServiceClient(httpClient, s.apiUrl)
	projectName := fmt.Sprintf("unkey-%s", unkeyProjectID)

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

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req)
}
