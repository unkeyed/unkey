package database

import (
	"context"
	"fmt"
	"github.com/chronark/unkey/apps/api/pkg/database/models"
	"github.com/chronark/unkey/apps/api/pkg/entities"
)

func (db *Database) ListKeysByApiId(ctx context.Context, apiId string, limit int, offset int) ([]entities.Key, error) {
	ctx, span := db.tracer.Start(ctx, "db.listKeysByApiId")
	defer span.End()

	const query = `SELECT ` +
		`id, api_id, hash, start, owner_id, meta, created_at, expires, ratelimit_type, ratelimit_limit, ratelimit_refill_rate, ratelimit_refill_interval, workspace_id, for_workspace_id ` +
		`FROM unkey.keys ` +
		`WHERE api_id = ? ORDER BY created_at ASC LIMIT ? OFFSET ?`

	rows, err := db.read().Query(query, apiId, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("unable to count keys from db: %w", err)
	}
	defer rows.Close()

	keys := []entities.Key{}
	for rows.Next() {

		k := &models.Key{}
		err := rows.Scan(&k.ID, &k.APIID, &k.Hash, &k.Start, &k.OwnerID, &k.Meta, &k.CreatedAt, &k.Expires, &k.RatelimitType, &k.RatelimitLimit, &k.RatelimitRefillRate, &k.RatelimitRefillInterval, &k.WorkspaceID, &k.ForWorkspaceID)
		if err != nil {
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}

		key, err := keyModelToEntity(k)
		if err != nil {
			return nil, fmt.Errorf("unable to convert key: %w", err)
		}
		keys = append(keys, key)
	}

	return keys, nil

}
