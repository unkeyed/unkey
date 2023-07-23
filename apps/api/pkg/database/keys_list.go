package database

import (
	"context"
	"fmt"
	"github.com/unkeyed/unkey/apps/api/pkg/database/models"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
)

func (db *database) ListKeysByKeyAuthId(ctx context.Context, keyAuthId string, limit int, offset int, ownerId string) ([]entities.Key, error) {

	query := `SELECT ` +
		`id, hash, start, owner_id, meta, created_at, expires, ratelimit_type, ratelimit_limit, ratelimit_refill_rate, ratelimit_refill_interval, workspace_id, for_workspace_id, key_auth_id ` +
		`FROM unkey.keys ` +
		`WHERE key_auth_id = ?`
	if ownerId != "" {
		query += " AND owner_id = ?"
	}

	query += ` ORDER BY created_at ASC LIMIT ? OFFSET ?`

	args := make([]any, 0)
	args = append(args, keyAuthId)

	if ownerId != "" {
		args = append(args, ownerId)
	}
	args = append(args, limit, offset)

	rows, err := db.read().Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("unable to list keys from db: %w", err)
	}
	defer rows.Close()

	keys := []entities.Key{}
	for rows.Next() {

		k := &models.Key{}
		err := rows.Scan(&k.ID, &k.Hash, &k.Start, &k.OwnerID, &k.Meta, &k.CreatedAt, &k.Expires, &k.RatelimitType, &k.RatelimitLimit, &k.RatelimitRefillRate, &k.RatelimitRefillInterval, &k.WorkspaceID, &k.ForWorkspaceID, &k.KeyAuthID)
		if err != nil {
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}

		e, err := keyModelToEntity(k)
		if err != nil {
			return nil, fmt.Errorf("unable to convert key: %w", err)
		}

		keys = append(keys, e)
	}

	return keys, nil

}
