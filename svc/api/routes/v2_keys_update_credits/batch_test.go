package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

func TestKeyCreditsBatchStateMatrix(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()
	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})

	tests := []struct {
		name              string
		operation         string
		initial           *int64
		value             sql.NullInt64
		deleted           bool
		missing           bool
		wantRemaining     sql.NullInt64
		wantApplied       bool
		wantDeleted       bool
		wantMissing       bool
		wantUnlimited     bool
		wantOverflow      bool
		wantRefillCleared bool
	}{
		{name: "set finite from finite", operation: "set", initial: int64Pointer(10), value: sql.NullInt64{Int64: 7, Valid: true}, wantRemaining: sql.NullInt64{Int64: 7, Valid: true}, wantApplied: true},
		{name: "set finite from unlimited", operation: "set", value: sql.NullInt64{Int64: 7, Valid: true}, wantRemaining: sql.NullInt64{Int64: 7, Valid: true}, wantApplied: true},
		{name: "set unlimited from finite", operation: "set", initial: int64Pointer(10), wantApplied: true, wantRefillCleared: true},
		{name: "set unlimited from unlimited no-op", operation: "set", wantApplied: true, wantRefillCleared: true},
		{name: "increment finite", operation: "increment", initial: int64Pointer(10), value: sql.NullInt64{Int64: 5, Valid: true}, wantRemaining: sql.NullInt64{Int64: 15, Valid: true}, wantApplied: true},
		{name: "increment zero no-op", operation: "increment", initial: int64Pointer(10), value: sql.NullInt64{Valid: true}, wantRemaining: sql.NullInt64{Int64: 10, Valid: true}, wantApplied: true},
		{name: "increment unlimited", operation: "increment", value: sql.NullInt64{Int64: 5, Valid: true}, wantUnlimited: true},
		{name: "increment overflow", operation: "increment", initial: int64Pointer(math.MaxInt64), value: sql.NullInt64{Int64: 1, Valid: true}, wantRemaining: sql.NullInt64{Int64: math.MaxInt64, Valid: true}, wantOverflow: true},
		{name: "decrement finite", operation: "decrement", initial: int64Pointer(10), value: sql.NullInt64{Int64: 4, Valid: true}, wantRemaining: sql.NullInt64{Int64: 6, Valid: true}, wantApplied: true},
		{name: "decrement clamps underflow", operation: "decrement", initial: int64Pointer(3), value: sql.NullInt64{Int64: 4, Valid: true}, wantRemaining: sql.NullInt64{Valid: true}, wantApplied: true},
		{name: "decrement zero no-op", operation: "decrement", initial: int64Pointer(10), value: sql.NullInt64{Valid: true}, wantRemaining: sql.NullInt64{Int64: 10, Valid: true}, wantApplied: true},
		{name: "decrement unlimited", operation: "decrement", value: sql.NullInt64{Int64: 1, Valid: true}, wantUnlimited: true},
		{name: "set deleted", operation: "set", initial: int64Pointer(10), value: sql.NullInt64{Int64: 7, Valid: true}, deleted: true, wantRemaining: sql.NullInt64{Int64: 10, Valid: true}, wantDeleted: true},
		{name: "increment deleted", operation: "increment", initial: int64Pointer(10), value: sql.NullInt64{Int64: 1, Valid: true}, deleted: true, wantRemaining: sql.NullInt64{Int64: 10, Valid: true}, wantDeleted: true},
		{name: "decrement deleted", operation: "decrement", initial: int64Pointer(10), value: sql.NullInt64{Int64: 1, Valid: true}, deleted: true, wantRemaining: sql.NullInt64{Int64: 10, Valid: true}, wantDeleted: true},
		{name: "set missing", operation: "set", value: sql.NullInt64{Int64: 7, Valid: true}, missing: true, wantMissing: true},
		{name: "increment missing", operation: "increment", value: sql.NullInt64{Int64: 1, Valid: true}, missing: true, wantMissing: true},
		{name: "decrement missing", operation: "decrement", value: sql.NullInt64{Int64: 1, Valid: true}, missing: true, wantMissing: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keyID := uid.New(uid.KeyPrefix)
			if !test.missing {
				refillAmount := int64(20)
				key := h.CreateKey(seed.CreateKeyRequest{
					WorkspaceID:  workspace.ID,
					KeySpaceID:   api.KeyAuthID.String,
					Remaining:    test.initial,
					RefillAmount: &refillAmount,
					Deleted:      test.deleted,
				})
				keyID = key.KeyID
			}

			before, err := db.Query.ListClickhouseOutboxByWorkspace(ctx, h.DB.RO(), workspace.ID)
			require.NoError(t, err)
			result, err := executeCreditBatch(ctx, h.DB.BatchRW(), test.operation, keyID, test.value, validOutbox(workspace.ID, keyID))
			require.NoError(t, err)
			require.Equal(t, test.wantApplied, result.Applied)
			require.Equal(t, test.wantDeleted, result.Deleted)
			require.Equal(t, test.wantMissing, result.Missing)
			require.Equal(t, test.wantUnlimited, result.Unlimited)
			require.Equal(t, test.wantOverflow, result.Overflow)

			after, err := db.Query.ListClickhouseOutboxByWorkspace(ctx, h.DB.RO(), workspace.ID)
			require.NoError(t, err)
			if test.wantApplied {
				require.Len(t, after, len(before)+1)
			} else {
				require.Len(t, after, len(before))
			}
			if test.missing {
				return
			}

			key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyID)
			require.NoError(t, err)
			require.Equal(t, test.wantRemaining, key.RemainingRequests)
			if test.wantApplied {
				remaining := "unlimited"
				if key.RemainingRequests.Valid {
					remaining = fmt.Sprintf("%d", key.RemainingRequests.Int64)
				}
				var payload struct {
					Description string `json:"description"`
				}
				require.NoError(t, json.Unmarshal(after[len(after)-1].Payload, &payload))
				require.Equal(t, fmt.Sprintf("Updated Key %s, set remaining to %s.", keyID, remaining), payload.Description)
			}
			if test.wantRefillCleared {
				require.False(t, key.RefillAmount.Valid)
				require.False(t, key.RefillDay.Valid)
			} else {
				require.Equal(t, sql.NullInt64{Int64: 20, Valid: true}, key.RefillAmount)
			}
		})
	}
}

func TestKeyCreditsBatchRollsBackOnAuditFailure(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()
	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Remaining:   int64Pointer(10),
	})
	outbox := validOutbox(workspace.ID, key.KeyID)
	outbox.Payload = json.RawMessage(`not-json`)

	_, err := h.DB.BatchRW().UpdateKeyCreditsIncrementWithAuditBatch(ctx, db.UpdateKeyCreditsIncrementReturningParams{
		ID:      key.KeyID,
		Credits: sql.NullInt64{Int64: 5, Valid: true},
	}, outbox)
	require.Error(t, err)

	stored, err := db.Query.FindKeyByID(ctx, h.DB.RO(), key.KeyID)
	require.NoError(t, err)
	require.Equal(t, int64(10), stored.RemainingRequests.Int64)
	rows, err := db.Query.ListClickhouseOutboxByWorkspace(ctx, h.DB.RO(), workspace.ID)
	require.NoError(t, err)
	require.Empty(t, rows)
}

func executeCreditBatch(ctx context.Context, replica *db.Replica, operation, keyID string, value sql.NullInt64, outbox db.InsertClickhouseOutboxForCreditUpdateParams) (db.KeyCreditsBatchResult, error) {
	switch operation {
	case "set":
		clearRefill := int64(0)
		if !value.Valid {
			clearRefill = 1
		}
		return replica.UpdateKeyCreditsSetWithAuditBatch(ctx, db.UpdateKeyCreditsSetParams{
			ID: keyID, Credits: value, ClearRefillAmount: clearRefill, ClearRefillDay: clearRefill,
		}, outbox)
	case "increment":
		return replica.UpdateKeyCreditsIncrementWithAuditBatch(ctx, db.UpdateKeyCreditsIncrementReturningParams{ID: keyID, Credits: value}, outbox)
	case "decrement":
		return replica.UpdateKeyCreditsDecrementWithAuditBatch(ctx, db.UpdateKeyCreditsDecrementReturningParams{ID: keyID, Credits: value}, outbox)
	default:
		return db.KeyCreditsBatchResult{}, fmt.Errorf("unknown operation %q", operation)
	}
}

func validOutbox(workspaceID, keyID string) db.InsertClickhouseOutboxForCreditUpdateParams {
	return db.InsertClickhouseOutboxForCreditUpdateParams{
		Version:     "audit_log.v1",
		WorkspaceID: workspaceID,
		EventID:     uid.New(uid.AuditLogPrefix),
		Payload:     json.RawMessage(`{"description":"pending"}`),
		KeyID:       keyID,
		CreatedAt:   time.Now().UnixMilli(),
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}
