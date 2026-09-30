package projects

import (
	"context"
	"database/sql"
	"time"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/projectgate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
)

// EnsureDefaultProject returns the exact default project owned by workspaceID,
// creating it in the caller's transaction when it does not exist.
func EnsureDefaultProject(ctx context.Context, tx db.DBTX, workspaceID string) (string, error) {
	projectID, found, err := FindDefaultProject(ctx, tx, workspaceID)
	if err != nil {
		return "", err
	}
	if found {
		return projectID, nil
	}

	projectID = uid.New(uid.ProjectPrefix)
	err = db.Query.InsertProject(ctx, tx, db.InsertProjectParams{
		ID:               projectID,
		WorkspaceID:      workspaceID,
		Name:             "Default",
		Slug:             projectgate.DefaultSlug,
		DeleteProtection: sql.NullBool{Bool: true, Valid: true},
		CreatedAt:        time.Now().UnixMilli(),
		UpdatedAt:        sql.NullInt64{},
	})
	if err == nil {
		return projectID, nil
	}
	if !db.IsDuplicateKeyError(err) {
		return "", fault.Wrap(err,
			fault.Internal("unable to create default project"),
			fault.Public("We're unable to resolve the workspace's default project."),
		)
	}

	// The first lookup may have established a repeatable-read snapshot before
	// another transaction inserted the project. A locking read sees that commit.
	projectID, err = db.Query.LockDefaultProjectByWorkspaceID(ctx, tx, workspaceID)
	if err != nil {
		return "", fault.Wrap(err,
			fault.Internal("unable to resolve default project after concurrent creation"),
			fault.Public("We're unable to resolve the workspace's default project."),
		)
	}

	return projectID, nil
}

// FindDefaultProject returns the exact default project owned by workspaceID.
// The found result is false when the workspace has no default project.
func FindDefaultProject(ctx context.Context, tx db.DBTX, workspaceID string) (projectID string, found bool, err error) {
	projectID, err = db.Query.FindDefaultProjectByWorkspaceID(ctx, tx, workspaceID)
	if err == nil {
		return projectID, true, nil
	}
	if db.IsNotFound(err) {
		return "", false, nil
	}

	return "", false, fault.Wrap(err,
		fault.Internal("unable to resolve default project"),
		fault.Public("We're unable to resolve the workspace's default project."),
	)
}
