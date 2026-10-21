package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestTightenFreeTierApiThrottle(t *testing.T) {
	database := openTestDatabase(t)

	const throttle = 5
	unset := sql.NullInt32{Int32: 0, Valid: false}
	limit := func(v int32) sql.NullInt32 { return sql.NullInt32{Int32: v, Valid: true} }

	cases := []struct {
		name    string
		tier    string
		current sql.NullInt32
		want    sql.NullInt32
	}{
		{"free without throttle is throttled", "Free", unset, limit(throttle)},
		{"free with looser throttle is lowered", "Free", limit(1_000), limit(throttle)},
		{"free with stricter throttle keeps it", "Free", limit(2), limit(2)},
		{"free blocked keeps zero", "Free", limit(0), limit(0)},
		{"paid tier is untouched", "Pro", unset, unset},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := NewQueries(beginRollbackTx(t, database))
			workspaceID := uid.New(uid.WorkspacePrefix)
			insertTestWorkspace(t, q, workspaceID, uid.DNS1035())
			require.NoError(t, q.InsertWorkspaceBilling(t.Context(), InsertWorkspaceBillingParams{
				WorkspaceID: workspaceID,
				CreatedAt:   1,
			}))
			require.NoError(t, q.UpdateWorkspaceBillingTier(t.Context(), UpdateWorkspaceBillingTierParams{
				Tier:        sql.NullString{String: tc.tier, Valid: true},
				UpdatedAt:   sql.NullInt64{Int64: 2, Valid: true},
				WorkspaceID: workspaceID,
			}))
			require.NoError(t, q.UpsertLimit(t.Context(), UpsertLimitParams{
				WorkspaceID:                           workspaceID,
				ApiBillableOperationsCountMaxPerMonth: 150_000,
				ApiRequestsCountMaxPerMinute:          tc.current,
				LogsRetentionDaysMax:                  7,
				LogsAuditRetentionDaysMax:             30,
				TeamEnabled:                           false,
				CpuCoresMax:                           10,
				CpuCoresMaxPerInstance:                2,
				MemoryMibMax:                          20_480,
				MemoryMibMaxPerInstance:               4_096,
				StorageMibMax:                         51_200,
				StorageMibMaxPerInstance:              10_240,
				BuildsConcurrentMax:                   1,
				CustomDomainsMax:                      0,
				AutoscalingReplicasMax:                0,
			}))

			require.NoError(t, q.TightenFreeTierApiThrottle(t.Context(), TightenFreeTierApiThrottleParams{
				RequestsPerMinute: limit(throttle),
				WorkspaceID:       workspaceID,
			}))

			got, err := q.FindLimitsByWorkspaceID(t.Context(), workspaceID)
			require.NoError(t, err)
			require.Equal(t, tc.want, got.ApiRequestsCountMaxPerMinute)
		})
	}
}
