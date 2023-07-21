package database

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/apps/api/pkg/entities"
	"go.uber.org/zap"
)

func (db *database) UpdateKey(ctx context.Context, key entities.Key) error {
	m, err := keyEntityToModel(key)
	if err != nil {
		return fmt.Errorf("unable to convert key")
	}

	db.logger.Info("db Update key", zap.Any("m", m))

	const sqlstr = `UPDATE unkey.keys SET ` +
		`api_id = ?, hash = ?, start = ?, owner_id = ?, meta = ?, created_at = ?, expires = ?, ratelimit_type = ?, ratelimit_limit = ?, ratelimit_refill_rate = ?, ratelimit_refill_interval = ?, workspace_id = ?, for_workspace_id = ?, name = ?, remaining_requests = ? ` +
		`WHERE id = ?`
	_, err = db.write().ExecContext(ctx, sqlstr, m.APIID, m.Hash, m.Start, m.OwnerID, m.Meta, m.CreatedAt, m.Expires, m.RatelimitType, m.RatelimitLimit, m.RatelimitRefillRate, m.RatelimitRefillInterval, m.WorkspaceID, m.ForWorkspaceID, m.Name, m.RemainingRequests, m.ID)
	if err != nil {
		return fmt.Errorf("unable to update key, %w", err)
	}
	return nil
}
