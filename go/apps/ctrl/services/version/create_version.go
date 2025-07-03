package version

import (
	"context"
	"database/sql"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
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

	// Start the build process by calling the build service directly
	buildReq := connect.NewRequest(&ctrlv1.CreateBuildRequest{
		WorkspaceId: req.Msg.GetWorkspaceId(),
		ProjectId:   req.Msg.GetProjectId(),
		VersionId:   versionID,
		DockerImage: req.Msg.GetDockerImageTag(),
	})

	_, err = s.buildService.CreateBuild(ctx, buildReq)
	if err != nil {
		// Log error but don't fail the version creation
		// The version will remain in pending state
		// TODO: add proper error handling and version status update
	}

	// Build service will create the build with this version_id
	// No need to store build_id in version - we can query by version_id

	res := connect.NewResponse(&ctrlv1.CreateVersionResponse{
		VersionId: versionID,
		Status:    ctrlv1.VersionStatus_VERSION_STATUS_PENDING,
	})

	return res, nil
}
