package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"errors"

	gen "github.com/unkeyed/unkey/apps/api/gen/database"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
)

func (db *database) FindKeyById(ctx context.Context, keyId string) (entities.Key, bool, error) {

	model, err := db.readReplica.query.FindKeyById(ctx, keyId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.Key{}, false, nil
		}
		return entities.Key{}, false, fmt.Errorf("unable to find key: %w", err)
	}

	api, err := transformKeyModelToEntity(model)
	if err != nil {
		return entities.Key{}, true, fmt.Errorf("unable to transform key model to entity: %w", err)
	}
	return api, true, nil
}

func transformKeyModelToEntity(m gen.Key) (entities.Key, error) {

	key := entities.Key{
		Id:          m.ID,
		KeyAuthId:   m.KeyAuthID,
		WorkspaceId: m.WorkspaceID,
		Hash:        m.Hash,
		Start:       m.Start,
		CreatedAt:   m.CreatedAt,
	}

	if m.Name.Valid {
		key.Name = m.Name.String
	}

	if m.OwnerID.Valid {
		key.OwnerId = m.OwnerID.String
	}

	if m.Meta.Valid {
		err := json.Unmarshal([]byte(m.Meta.String), &key.Meta)
		if err != nil {
			return entities.Key{}, fmt.Errorf("uanble to unmarshal meta: %w", err)
		}
	}
	if m.Expires.Valid {
		key.Expires = m.Expires.Time
	}
	if m.RatelimitType.Valid {
		key.Ratelimit = &entities.Ratelimit{
			Type:           m.RatelimitType.String,
			Limit:          m.RatelimitLimit.Int32,
			RefillRate:     m.RatelimitRefillRate.Int32,
			RefillInterval: m.RatelimitRefillInterval.Int32,
		}
	}
	if m.ForWorkspaceID.Valid {
		key.ForWorkspaceId = m.ForWorkspaceID.String
	}
	if m.RemainingRequests.Valid {
		remaining := m.RemainingRequests.Int32
		key.Remaining = &remaining
	}

	return key, nil
}
