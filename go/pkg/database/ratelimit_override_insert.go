package database

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/database/transform"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) InsertRatelimitOverride(ctx context.Context, override entities.RatelimitOverride) error {

	p, err := transform.RatelimitOverrideEntityToInsertParams(override)
	if err != nil {

		return fault.Wrap(err,
			fault.WithDesc(
				"failed transforming",
				"The override configuration can not get serialized.",
			),
		)
	}

	err = db.write().InsertOverride(ctx, p)
	if err != nil {

		return fault.Wrap(err,
			fault.WithDesc("failed inserting", ""),
		)
	}
	return nil
}
