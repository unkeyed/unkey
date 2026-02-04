package seed

import (
	"context"
	"math/rand"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/uid"
)

// ClickHouseSeeder provides methods to seed test data in ClickHouse.
type ClickHouseSeeder struct {
	t    *testing.T
	conn ch.Conn
}

// NewClickHouseSeeder creates a new ClickHouseSeeder instance.
func NewClickHouseSeeder(t *testing.T, conn ch.Conn) *ClickHouseSeeder {
	return &ClickHouseSeeder{t: t, conn: conn}
}

// InsertVerifications inserts key verification records for a workspace.
func (s *ClickHouseSeeder) InsertVerifications(ctx context.Context, workspaceID string, count int, timestamp time.Time, outcome string) {
	const batchSize = 10_000
	for i := 0; i < count; i += batchSize {
		batchCount := min(batchSize, count-i)
		verifications := make([]schema.KeyVerification, batchCount)
		for j := range batchCount {
			verifications[j] = schema.KeyVerification{
				RequestID:    uid.New(uid.RequestPrefix),
				Time:         timestamp.Add(time.Duration(i+j) * time.Millisecond).UnixMilli(),
				WorkspaceID:  workspaceID,
				KeySpaceID:   uid.New(uid.KeySpacePrefix),
				IdentityID:   "",
				ExternalID:   "",
				KeyID:        uid.New(uid.KeyPrefix),
				Region:       "us-east-1",
				Outcome:      outcome,
				Tags:         []string{},
				SpentCredits: 0,
				Latency:      rand.Float64() * 100, //nolint:gosec
			}
		}

		batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO default.key_verifications_raw_v2")
		require.NoError(s.t, err)

		for _, v := range verifications {
			err = batch.AppendStruct(&v)
			require.NoError(s.t, err)
		}

		err = batch.Send()
		require.NoError(s.t, err)
	}
}

// InsertRatelimits inserts ratelimit records for a workspace.
func (s *ClickHouseSeeder) InsertRatelimits(ctx context.Context, workspaceID string, count int, timestamp time.Time, passed bool) {
	const batchSize = 10_000
	var remaining uint64 = 50
	if !passed {
		remaining = 0
	}

	for i := 0; i < count; i += batchSize {
		batchCount := min(batchSize, count-i)
		ratelimits := make([]schema.Ratelimit, batchCount)
		for j := range batchCount {
			ratelimits[j] = schema.Ratelimit{
				RequestID:   uid.New(uid.RequestPrefix),
				Time:        timestamp.Add(time.Duration(i+j) * time.Millisecond).UnixMilli(),
				WorkspaceID: workspaceID,
				NamespaceID: uid.New(uid.RatelimitNamespacePrefix),
				Identifier:  uid.New(uid.IdentityPrefix),
				Passed:      passed,
				Latency:     rand.Float64() * 10, //nolint:gosec
				OverrideID:  "",
				Limit:       100,
				Remaining:   remaining,
				ResetAt:     timestamp.Add(time.Minute).UnixMilli(),
			}
		}

		batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO default.ratelimits_raw_v2")
		require.NoError(s.t, err)

		for _, r := range ratelimits {
			err = batch.AppendStruct(&r)
			require.NoError(s.t, err)
		}

		err = batch.Send()
		require.NoError(s.t, err)
	}
}
