package database

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) InsertKey(ctx context.Context, key entities.Key) error {
	meta, err := json.Marshal(key.Meta)
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("failed to marshal key meta", ""))
	}

	remaining := int32(0)
	if key.RemainingRequests != nil {
		// nolint:gosec
		remaining = int32(*key.RemainingRequests)
	}

	identityID := ""
	if key.Identity != nil {
		// nolint:gosec
		identityID = key.Identity.ID
	}

	params := gen.InsertKeyParams{
		ID:          key.ID,
		KeyringID:   key.KeyringID,
		Hash:        key.Hash,
		Start:       key.Start,
		WorkspaceID: key.WorkspaceID,
		ForWorkspaceID: sql.NullString{
			String: key.ForWorkspaceID,
			Valid:  key.ForWorkspaceID != "",
		},
		Name: sql.NullString{
			String: key.Name,
			Valid:  key.Name != "",
		},
		IdentityID: sql.NullString{
			String: identityID,
			Valid:  identityID != "",
		},
		Meta: sql.NullString{
			String: string(meta),
			Valid:  true,
		},
		Expires: sql.NullTime{
			Time:  key.Expires,
			Valid: !key.Expires.IsZero(),
		},
		Enabled: key.Enabled,
		RemainingRequests: sql.NullInt32{
			Int32: remaining,
			Valid: key.RemainingRequests != nil,
		},
		RatelimitAsync: sql.NullBool{
			Bool:  false,
			Valid: false,
		},
		RatelimitLimit: sql.NullInt32{
			Int32: 0,
			Valid: false,
		},
		RatelimitDuration: sql.NullInt64{
			Int64: 0,
			Valid: false,
		},
		Environment: sql.NullString{
			String: key.Environment,
			Valid:  key.Environment != "",
		},
		CreatedAt: db.clock.Now(),
	}

	err = db.write().InsertKey(ctx, params)
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("failed to insert key", ""))
	}

	return nil
}
