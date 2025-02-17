package transform

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/entities"
)

func IdentityModelToEntity(m gen.Identity) (entities.Identity, error) {

	identity := entities.Identity{
		ID:          m.ID,
		ExternalID:  m.ExternalID,
		WorkspaceID: m.WorkspaceID,
		CreatedAt:   time.UnixMilli(m.CreatedAt),
		Meta:        map[string]any{},
		UpdatedAt:   time.Time{},
		DeletedAt:   time.Time{},
		Environment: m.Environment,
	}

	err := json.Unmarshal([]byte(m.Meta), &identity.Meta)
	if err != nil {
		return entities.Identity{}, fmt.Errorf("unable to unmarshal meta: %w", err)
	}

	if m.UpdatedAt.Valid {
		identity.UpdatedAt = time.UnixMilli(m.UpdatedAt.Int64)
	}

	return identity, nil
}
