package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
)

func (db *database) UpdateKey(ctx context.Context, key *authenticationv1.Key) error {
	params, err := transformKeyEntitytoUpdateKeyParams(key)
	if err != nil {
		return fmt.Errorf("unable to transform key entity to UpdateKeyParams")
	}
	return db.write().UpdateKey(ctx, params)
}

func transformKeyEntitytoUpdateKeyParams(key *authenticationv1.Key) (gen.UpdateKeyParams, error) {
	params := gen.UpdateKeyParams{
		ID:        key.KeyId,
		Hash:      key.Hash,
		Start:     key.Start,
		OwnerID:   sql.NullString{String: key.GetOwnerId(), Valid: key.OwnerId != nil},
		CreatedAt: time.UnixMilli(key.CreatedAt),
		Name:      sql.NullString{String: key.GetName(), Valid: key.Name != nil},
	}
	if key.Expires != nil {
		params.Expires = sql.NullTime{Time: time.UnixMilli(key.GetExpires()), Valid: key.Expires != nil}
	}

	metaJson, err := json.Marshal(key.Meta)
	if err != nil {
		return gen.UpdateKeyParams{}, fmt.Errorf("unable to marshal meta: %w", err)
	}
	params.Meta = sql.NullString{String: string(metaJson), Valid: true}

	if key.Ratelimit != nil {
		switch key.Ratelimit.Type {
		case authenticationv1.RatelimitType_RATELIMIT_TYPE_FAST:
			params.RatelimitType = sql.NullString{String: "fast", Valid: true}
		case authenticationv1.RatelimitType_RATELIMIT_TYPE_CONSISTENT:
			params.RatelimitType = sql.NullString{String: "consistent", Valid: true}
		}

		params.RatelimitLimit = sql.NullInt32{Int32: int32(key.Ratelimit.Limit), Valid: true}
		params.RatelimitRefillRate = sql.NullInt32{Int32: key.Ratelimit.RefillRate, Valid: true}
		params.RatelimitRefillInterval = sql.NullInt32{Int32: key.Ratelimit.RefillInterval, Valid: true}
	}

	if key.Remaining != nil {
		params.RemainingRequests = sql.NullInt32{Int32: key.GetRemaining(), Valid: true}
	} else {
		params.RemainingRequests = sql.NullInt32{}
	}

	return params, nil

}
