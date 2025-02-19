package database

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/database/transform"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) InsertRatelimitNamespace(ctx context.Context, namespace entities.RatelimitNamespace) error {
	params := transform.RatelimitNamespaceEntityToInsertParams(namespace)

	err := db.write().InsertRatelimitNamespace(ctx, params)
	if err != nil {
		return fault.Wrap(err,
			fault.WithDesc("failed to insert ratelimit namespace", ""),
		)
	}

	return nil
}
