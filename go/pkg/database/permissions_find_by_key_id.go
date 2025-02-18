package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) FindPermissionsByKeyID(ctx context.Context, keyID string) ([]string, error) {
	permissions, err := db.read().FindPermissionsForKey(ctx, gen.FindPermissionsForKeyParams{KeyID: keyID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []string{}, nil
		}
		return nil, fault.Wrap(err, fault.WithTag(fault.DATABASE_ERROR))
	}

	return permissions, nil
}
