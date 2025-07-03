package version

import (
	"context"
	"database/sql"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func (s *Service) CreateVersion(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateVersionRequest],
) (*connect.Response[ctrlv1.CreateVersionResponse], error) {
	// Generate version ID
	versionID := uid.New(uid.VersionPrefix, 4)
	now := time.Now().UnixMilli()

	// Insert version into database
	err := db.Query.InsertVersion(ctx, s.db.RW(), db.InsertVersionParams{
		ID:             versionID,
		WorkspaceID:    req.Msg.GetWorkspaceId(),
		ProjectID:      req.Msg.GetProjectId(),
		EnvironmentID:  req.Msg.GetEnvironmentId(),
		BranchID:       sql.NullString{String: "", Valid: false}, // TODO: resolve branch
		BuildID:        sql.NullString{String: "", Valid: false}, // TODO: create build if needed
		RootfsImageID:  "",                                       // TODO: handle image
		GitCommitSha:   sql.NullString{String: req.Msg.GetGitCommitSha(), Valid: req.Msg.GetGitCommitSha() != ""},
		GitBranch:      sql.NullString{String: req.Msg.GetBranch(), Valid: req.Msg.GetBranch() != ""},
		ConfigSnapshot: []byte("{}"), // TODO: build config
		TopologyConfig: []byte("{}"), // TODO: build topology
		Status:         db.VersionsStatusPending,
		CreatedAtM:     now,
		UpdatedAtM:     sql.NullInt64{Int64: now, Valid: true},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	s.logger.Info("starting deployment workflow for version",
		"version_id", versionID,
		"workspace_id", req.Msg.GetWorkspaceId(),
		"project_id", req.Msg.GetProjectId(),
		"docker_image", req.Msg.GetDockerImageTag())

	// Start the deployment workflow directly
	deployReq := &DeployRequest{
		WorkspaceID: req.Msg.GetWorkspaceId(),
		ProjectID:   req.Msg.GetProjectId(),
		VersionID:   versionID,
		DockerImage: req.Msg.GetDockerImageTag(),
	}

	executionID, err := s.hydraEngine.StartWorkflow(ctx, "deployment", deployReq,
		hydra.WithMaxAttempts(3),
		hydra.WithTimeout(25*time.Minute),
		hydra.WithRetryBackoff(1*time.Minute),
	)
	if err != nil {
		s.logger.Error("failed to start deployment workflow",
			"version_id", versionID,
			"error", err)
		// Don't fail version creation - workflow can be retried
	} else {
		s.logger.Info("deployment workflow started",
			"version_id", versionID,
			"execution_id", executionID)
	}

	res := connect.NewResponse(&ctrlv1.CreateVersionResponse{
		VersionId: versionID,
		Status:    ctrlv1.VersionStatus_VERSION_STATUS_PENDING,
	})

	return res, nil
}
