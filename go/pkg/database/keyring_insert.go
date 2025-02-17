package database

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) InsertKeyring(ctx context.Context, keyring entities.Keyring) error {
	params := gen.InsertKeyringParams{
		ID:                 keyring.ID,
		WorkspaceID:        keyring.WorkspaceID,
		StoreEncryptedKeys: keyring.StoreEncryptedKeys,
		DefaultPrefix: sql.NullString{
			String: keyring.DefaultPrefix,
			Valid:  keyring.DefaultPrefix != "",
		},
		DefaultBytes: sql.NullInt32{
			// nolint:gosec
			Int32: int32(keyring.DefaultBytes),
			Valid: keyring.DefaultBytes != 0,
		},
		CreatedAt: sql.NullTime{
			Time:  db.clock.Now(),
			Valid: true,
		},
		CreatedAtM: db.clock.Now().UnixMilli(),
	}

	err := db.write().InsertKeyring(ctx, params)
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("failed to insert key ring", ""))
	}

	return nil
}
