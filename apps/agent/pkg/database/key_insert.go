package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
)

func (db *database) InsertKey(ctx context.Context, key *authenticationv1.Key) error {
	params, err := transformKeyEntitytoInsertKeyParams(key)
	if err != nil {
		return fmt.Errorf("unable to transform key entity to InsertKeyParams")
	}
	return db.write().InsertKey(ctx, params)
}

func transformKeyEntitytoInsertKeyParams(key *authenticationv1.Key) (gen.InsertKeyParams, error) {
	params := gen.InsertKeyParams{
		ID:             key.KeyId,
		Hash:           key.Hash,
		Start:          key.Start,
		OwnerID:        sql.NullString{String: key.GetOwnerId(), Valid: key.OwnerId != nil},
		CreatedAt:      time.UnixMilli(key.CreatedAt),
		Expires:        sql.NullTime{Time: time.UnixMilli(key.GetExpires()), Valid: key.GetExpires() != 0},
		WorkspaceID:    key.WorkspaceId,
		ForWorkspaceID: sql.NullString{String: key.GetForWorkspaceId(), Valid: key.ForWorkspaceId != nil},
		Name:           sql.NullString{String: key.GetName(), Valid: key.Name != nil},
		KeyAuthID:      key.KeyAuthId,
	}

	if key.Meta != nil {
		params.Meta = sql.NullString{String: key.GetMeta(), Valid: true}
	}

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

	}

	return params, nil

}
