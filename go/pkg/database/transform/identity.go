package transform

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/entities"
)

func KeyModelToEntity(m gen.Key) (entities.Key, error) {

	key := entities.Key{
		ID:             m.ID,
		KeySpaceID:     m.KeyAuthID,
		WorkspaceID:    m.WorkspaceID,
		Hash:           m.Hash,
		Start:          m.Start,
		CreatedAt:      m.CreatedAt,
		ForWorkspaceID: "",
		Name:           "",
		Enabled:        m.Enabled,
		IdentityID:     "",
		Meta:           map[string]any{},
		UpdatedAt:      time.Time{},
		DeletedAt:      time.Time{},
		Environment:    "",
		Expires:        time.Time{},
		Identity:       nil,
		Permissions:    []string{},
	}

	if m.Name.Valid {
		key.Name = m.Name.String
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

	if m.ForWorkspaceID.Valid {
		key.ForWorkspaceID = m.ForWorkspaceID.String
	}
	if m.IdentityID.Valid {
		key.IdentityID = m.IdentityID.String
	}

	if m.UpdatedAtM.Valid {
		key.UpdatedAt = time.UnixMilli(m.UpdatedAtM.Int64)
	}

	if m.DeletedAtM.Valid {
		key.DeletedAt = time.UnixMilli(m.DeletedAtM.Int64)
	}
	if m.Environment.Valid {
		key.Environment = m.Environment.String
	}

	return key, nil
}
