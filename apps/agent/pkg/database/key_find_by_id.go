package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"errors"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	keysv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/keys/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

func (db *database) FindKeyById(ctx context.Context, keyId string) (*keysv1.Key, bool, error) {

	model, err := db.read().FindKeyById(ctx, keyId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("unable to find key: %w", err)
	}

	api, err := transformKeyModelToEntity(model)
	if err != nil {
		return nil, true, fmt.Errorf("unable to transform key model to entity: %w", err)
	}
	return api, true, nil
}

func transformKeyModelToEntity(m gen.Key) (*keysv1.Key, error) {

	key := &keysv1.Key{
		Id:          m.ID,
		KeyAuthId:   m.KeyAuthID,
		WorkspaceId: m.WorkspaceID,
		Hash:        m.Hash,
		Start:       m.Start,
		CreatedAt:   m.CreatedAt.UnixMilli(),
	}

	if m.Name.Valid {
		key.Name = util.Pointer(m.Name.String)
	}

	if m.OwnerID.Valid {
		key.OwnerId = util.Pointer(m.OwnerID.String)
	}

	if m.Meta.Valid {
		err := json.Unmarshal([]byte(m.Meta.String), &key.Meta)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal meta '%s': %w", m.Meta.String, err)
		}
	}
	if m.Expires.Valid {
		key.Expires = util.Pointer(m.Expires.Time.UnixMilli())
	}
	if m.RatelimitType.Valid {
		key.Ratelimit = &keysv1.Ratelimit{
			Limit:          m.RatelimitLimit.Int32,
			RefillRate:     m.RatelimitRefillRate.Int32,
			RefillInterval: m.RatelimitRefillInterval.Int32,
		}
		switch m.RatelimitType.String {
		case "fast":
			key.Ratelimit.Type = keysv1.RatelimitType_RATELIMIT_TYPE_FAST
		case "consistent":
			key.Ratelimit.Type = keysv1.RatelimitType_RATELIMIT_TYPE_CONSISTENT
		}

	}
	if m.ForWorkspaceID.Valid {
		key.ForWorkspaceId = util.Pointer(m.ForWorkspaceID.String)
	}
	if m.RemainingRequests.Valid {
		remaining := m.RemainingRequests.Int32
		key.Remaining = &remaining
	}

	return key, nil
}
