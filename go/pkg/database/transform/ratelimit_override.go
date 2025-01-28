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
		Duration:    time.Millisecond * time.Duration(m.Duration),
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

func RatelimitOverrideEntityToModel(e entities.RatelimitOverride) (gen.RatelimitOverride, error) {
	m := gen.RatelimitOverride{
		ID:          e.ID,
		WorkspaceID: e.WorkspaceID,
		NamespaceID: e.NamespaceID,
		Identifier:  e.Identifier,
		Limit:       e.Limit,
		// nolint:gosec // an int32 still gives us > 24 days of duration
		Duration: int32(e.Duration.Milliseconds()),
		Async: sql.NullBool{
			Bool:  false,
			Valid: false,
		},
		CreatedAt: e.CreatedAt,
		UpdatedAt: sql.NullTime{
			Time:  e.UpdatedAt,
			Valid: !e.UpdatedAt.IsZero(),
		},
		Sharding: gen.NullRatelimitOverridesSharding{
			RatelimitOverridesSharding: "edge",
			Valid:                      false,
		},
		DeletedAt: sql.NullTime{
			Time:  e.DeletedAt,
			Valid: !e.DeletedAt.IsZero(),
		},
	}
	return m, nil
}

func RatelimitOverrideEntityToInsertParams(e entities.RatelimitOverride) (gen.InsertOverrideParams, error) {
	p := gen.InsertOverrideParams{
		ID:          e.ID,
		WorkspaceID: e.WorkspaceID,
		NamespaceID: e.NamespaceID,
		Identifier:  e.Identifier,
		Limit:       e.Limit,
		// nolint:gosec // an int32 still gives us > 24 days of duration
		Duration: int32(e.Duration.Milliseconds()),
	}
	return p, nil
}
