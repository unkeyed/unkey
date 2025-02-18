package transform

import (
	"time"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/entities"
)

func RatelimitNamespaceModelToEntity(m gen.RatelimitNamespace) (entities.RatelimitNamespace, error) {
	namespace := entities.RatelimitNamespace{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		Name:        m.Name,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   time.Time{},
		DeletedAt:   time.Time{},
	}

	if m.UpdatedAt.Valid {
		namespace.UpdatedAt = m.UpdatedAt.Time
	}

	if m.DeletedAt.Valid {
		namespace.DeletedAt = m.DeletedAt.Time
	}

	return namespace, nil
}

func RatelimitNamespaceEntityToInsertParams(e entities.RatelimitNamespace) gen.InsertRatelimitNamespaceParams {
	return gen.InsertRatelimitNamespaceParams{
		ID:          e.ID,
		WorkspaceID: e.WorkspaceID,
		Name:        e.Name,
	}
}
