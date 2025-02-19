package transform

import (
	"database/sql"
	"time"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/entities"
)

func RatelimitOverrideModelToEntity(m gen.RatelimitOverride) (entities.RatelimitOverride, error) {
	e := entities.RatelimitOverride{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		NamespaceID: m.NamespaceID,
		Identifier:  m.Identifier,
		Limit:       m.Limit,
		Duration:    time.Duration(m.Duration) * time.Millisecond,
		Async:       m.Async.Bool,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   time.Time{},
		DeletedAt:   time.Time{},
	}

	if m.UpdatedAt.Valid {
		e.UpdatedAt = m.UpdatedAt.Time
	}

	if m.DeletedAt.Valid {
		e.DeletedAt = m.DeletedAt.Time
	}

	return e, nil
}

func RatelimitOverrideEntityToModel(e entities.RatelimitOverride) gen.RatelimitOverride {
	m := gen.RatelimitOverride{
		ID:          e.ID,
		WorkspaceID: e.WorkspaceID,
		NamespaceID: e.NamespaceID,
		Identifier:  e.Identifier,
		Limit:       e.Limit,
		Duration:    int32(e.Duration.Milliseconds()), // nolint:gosec
		Async: sql.NullBool{
			Bool:  e.Async,
			Valid: true,
		},
		CreatedAt: e.CreatedAt,
		UpdatedAt: sql.NullTime{
			Time:  e.UpdatedAt,
			Valid: !e.UpdatedAt.IsZero(),
		},
		DeletedAt: sql.NullTime{
			Time:  e.DeletedAt,
			Valid: !e.DeletedAt.IsZero(),
		},
		Sharding: gen.NullRatelimitOverridesSharding{
			RatelimitOverridesSharding: gen.RatelimitOverridesSharding("edge"),
			Valid:                      false,
		},
	}

	return m
}
