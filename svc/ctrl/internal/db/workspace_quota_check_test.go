package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestQuotaCheckTierUpdatedAt(t *testing.T) {
	database := openTestDatabase(t)

	createdAt := time.Date(2026, time.October, 20, 12, 0, 0, 0, time.UTC).UnixMilli()
	later := sql.NullInt64{Int64: createdAt + int64(24*time.Hour/time.Millisecond), Valid: true}
	never := sql.NullInt64{Int64: 0, Valid: false}

	cases := []struct {
		name   string
		mutate func(t *testing.T, q *Queries, workspaceID string)
		want   sql.NullInt64
	}{
		{"created mid period and always free", func(t *testing.T, q *Queries, workspaceID string) {}, never},
		{"compute spend state changed", func(t *testing.T, q *Queries, workspaceID string) {
			require.NoError(t, q.SetWorkspaceDeploySpendSuspended(t.Context(), SetWorkspaceDeploySpendSuspendedParams{
				Suspended: true,
				UpdatedAt: later,
				ID:        workspaceID,
			}))
		}, never},
		{"api tier changed", func(t *testing.T, q *Queries, workspaceID string) {
			require.NoError(t, q.UpdateWorkspaceBillingTier(t.Context(), UpdateWorkspaceBillingTierParams{
				Tier:        sql.NullString{String: "Free", Valid: true},
				UpdatedAt:   later,
				WorkspaceID: workspaceID,
			}))
		}, later},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := NewQueries(beginRollbackTx(t, database))
			workspaceID := uid.New(uid.WorkspacePrefix)
			insertTestWorkspace(t, q, workspaceID, uid.DNS1035())
			require.NoError(t, q.InsertWorkspaceBilling(t.Context(), InsertWorkspaceBillingParams{
				WorkspaceID: workspaceID,
				CreatedAt:   createdAt,
			}))

			tc.mutate(t, q, workspaceID)

			rows, err := q.GetWorkspacesForQuotaCheckByIDs(t.Context(), []string{workspaceID})
			require.NoError(t, err)
			require.Len(t, rows, 1)
			require.Equal(t, tc.want, rows[0].TierUpdatedAt)
		})
	}
}
