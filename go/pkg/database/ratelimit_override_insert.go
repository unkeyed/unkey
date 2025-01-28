package database

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) InsertRatelimitOverride(ctx context.Context, override entities.RatelimitOverride) error {

	err := db.write().InsertOverride(ctx, gen.InsertOverrideParams{
		ID:          override.ID,
		WorkspaceID: override.WorkspaceID,
		NamespaceID: override.NamespaceID,
		Identifier:  override.Identifier,
		Limit:       override.Limit,
		Duration:    int32(override.Duration.Milliseconds()), // nolint:gosec
	})
	if err != nil {

		return fault.Wrap(err,
			fault.WithDesc("failed inserting", ""),
		)
	}
	return nil
}
