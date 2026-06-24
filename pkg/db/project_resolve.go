package db

import (
	"context"
	"strings"

	"github.com/unkeyed/unkey/pkg/uid"
)

func ResolveProject(ctx context.Context, conn DBTX, workspaceID, idOrSlug string) (FindProjectByWorkspaceAndSlugRow, error) {
	if strings.HasPrefix(idOrSlug, string(uid.ProjectPrefix)+"_") {
		row, err := Query.FindProjectByWorkspaceAndId(ctx, conn, FindProjectByWorkspaceAndIdParams{
			WorkspaceID: workspaceID,
			ID:          idOrSlug,
		})
		return FindProjectByWorkspaceAndSlugRow(row), err
	}

	return Query.FindProjectByWorkspaceAndSlug(ctx, conn, FindProjectByWorkspaceAndSlugParams{
		WorkspaceID: workspaceID,
		Slug:        idOrSlug,
	})
}
