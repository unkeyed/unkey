package projects

import (
	"context"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
)

// ResolveDefaultID returns the exact default project owned by workspaceID.
func ResolveDefaultID(ctx context.Context, database db.DBTX, workspaceID string) (string, error) {
	projectID, err := db.Query.FindDefaultProjectByWorkspaceID(ctx, database, workspaceID)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("unable to resolve default project"))
	}

	return projectID, nil
}
