package projects

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
)

// DefaultSlug identifies the workspace's internal ownership project.
const DefaultSlug = "default"

// IsReservedSlug reports whether slug would collide with the internal default
// project under MySQL's case-insensitive slug collation.
func IsReservedSlug(slug string) bool {
	return strings.EqualFold(slug, DefaultSlug)
}

// EnsureDefaultProject returns the exact default project owned by workspaceID,
// creating it in the caller's transaction when it does not exist.
func EnsureDefaultProject(ctx context.Context, tx db.DBTX, workspaceID string) (string, error) {
	projectID, err := db.Query.FindDefaultProjectByWorkspaceID(ctx, tx, workspaceID)
	if err == nil {
		return projectID, nil
	}
	if !db.IsNotFound(err) {
		return "", fault.Wrap(err,
			fault.Internal("unable to resolve default project"),
			fault.Public("We're unable to resolve the workspace's default project."),
		)
	}

	projectID = uid.New(uid.ProjectPrefix)
	err = db.Query.InsertProject(ctx, tx, db.InsertProjectParams{
		ID:               projectID,
		WorkspaceID:      workspaceID,
		Name:             "Default",
		Slug:             DefaultSlug,
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
