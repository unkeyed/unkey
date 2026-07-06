package clickhouse_test

import (
	"context"
	"math/rand"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// createIdentityVerifications creates verification events attributed to an
// end-user identity in a specific keyspace, each spending the given number of
// credits.
func createIdentityVerifications(workspaceID, keySpaceID, identityID, externalID string, count int, timestamp time.Time, outcome string, spentCredits int64) []schema.KeyVerification {
	verifications := make([]schema.KeyVerification, count)
	for i := range count {
		verifications[i] = schema.KeyVerification{
			RequestID:    uid.New(uid.RequestPrefix),
			Time:         timestamp.Add(time.Duration(i) * time.Second).UnixMilli(),
			WorkspaceID:  workspaceID,
			KeySpaceID:   keySpaceID,
			IdentityID:   identityID,
			ExternalID:   externalID,
			KeyID:        uid.New(uid.KeyPrefix),
			Region:       "us-east-1",
			Outcome:      outcome,
			Tags:         []string{},
			SpentCredits: spentCredits,
			Latency:      rand.Float64() * 100,
		}
	}
	return verifications
}

func TestGetBillableUsagePerIdentity(t *testing.T) {
	chCfg := containers.ClickHouse(t)
	dsn := chCfg.DSN

	client, err := clickhouse.New(clickhouse.Config{
		URL: dsn,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	err = client.Ping(context.Background())
	require.NoError(t, err)

	opts, err := ch.ParseDSN(dsn)
	require.NoError(t, err)

	conn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	ctx := context.Background()

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	t.Run("per-identity counts and credits across two identities", func(t *testing.T) {
		workspaceID := uid.New(uid.WorkspacePrefix)
		keySpaceID := uid.New(uid.KeySpacePrefix)
		identityA := uid.New(uid.IdentityPrefix)
		identityB := uid.New(uid.IdentityPrefix)

		// Identity A: 100 VALID verifications, 2 credits each.
		// Inserted in two batches so unmerged SummingMergeTree parts must still
		// sum to the correct total at read time.
		batchA1 := createIdentityVerifications(workspaceID, keySpaceID, identityA, "user_a", 50, now, "VALID", 2)
		batchA2 := createIdentityVerifications(workspaceID, keySpaceID, identityA, "user_a", 50, now, "VALID", 2)
		// Identity B: 30 VALID verifications, 1 credit each.
		batchB := createIdentityVerifications(workspaceID, keySpaceID, identityB, "user_b", 30, now, "VALID", 1)

		insertVerifications(t, ctx, conn, batchA1)
		insertVerifications(t, ctx, conn, batchA2)
		insertVerifications(t, ctx, conn, batchB)

		require.Eventually(t, func() bool {
			usage, usageErr := client.GetBillableUsagePerIdentity(ctx, workspaceID, year, month, []string{keySpaceID}, nil)
			if usageErr != nil || len(usage) != 2 {
				return false
			}
			byExternal := map[string]clickhouse.IdentityBillableUsage{}
			for _, u := range usage {
				byExternal[u.ExternalID] = u
			}
			a, okA := byExternal["user_a"]
			b, okB := byExternal["user_b"]
			return okA && okB &&
				a.IdentityID == identityA && a.Verifications == 100 && a.SpentCredits == 200 &&
				b.IdentityID == identityB && b.Verifications == 30 && b.SpentCredits == 30
		}, time.Minute, time.Second)

		// Read idempotency: a second read returns identical quantities.
		usage, err := client.GetBillableUsagePerIdentity(ctx, workspaceID, year, month, []string{keySpaceID}, nil)
		require.NoError(t, err)
		require.Len(t, usage, 2)

		// Scope gate: with the keyspace NOT enabled, the same usage is excluded.
		none, err := client.GetBillableUsagePerIdentity(ctx, workspaceID, year, month, []string{uid.New(uid.KeySpacePrefix)}, nil)
		require.NoError(t, err)
		require.Empty(t, none, "usage on a non-enabled keyspace must be excluded")
	})

	t.Run("only the enabled keyspace is billed when an identity spans two", func(t *testing.T) {
		workspaceID := uid.New(uid.WorkspacePrefix)
		enabledKeyspace := uid.New(uid.KeySpacePrefix)
		otherKeyspace := uid.New(uid.KeySpacePrefix)
		identityID := uid.New(uid.IdentityPrefix)

		// 20 verifications on the enabled keyspace, 35 on the other.
		insertVerifications(t, ctx, conn, createIdentityVerifications(workspaceID, enabledKeyspace, identityID, "user_span", 20, now, "VALID", 0))
		insertVerifications(t, ctx, conn, createIdentityVerifications(workspaceID, otherKeyspace, identityID, "user_span", 35, now, "VALID", 0))

		require.Eventually(t, func() bool {
			usage, usageErr := client.GetBillableUsagePerIdentity(ctx, workspaceID, year, month, []string{enabledKeyspace}, nil)
			if usageErr != nil || len(usage) != 1 {
				return false
			}
			// Only the 20 from the enabled keyspace count, not all 55.
			return usage[0].ExternalID == "user_span" && usage[0].Verifications == 20
		}, time.Minute, time.Second)

		// Enabling both keyspaces sums to 55.
		require.Eventually(t, func() bool {
			usage, usageErr := client.GetBillableUsagePerIdentity(ctx, workspaceID, year, month, []string{enabledKeyspace, otherKeyspace}, nil)
			return usageErr == nil && len(usage) == 1 && usage[0].Verifications == 55
		}, time.Minute, time.Second)
	})

	t.Run("rate-limited outcomes are not billable", func(t *testing.T) {
		workspaceID := uid.New(uid.WorkspacePrefix)
		keySpaceID := uid.New(uid.KeySpacePrefix)
		identityID := uid.New(uid.IdentityPrefix)

		// Only RATE_LIMITED outcomes with no credits spent: nothing billable.
		limited := createIdentityVerifications(workspaceID, keySpaceID, identityID, "user_limited", 25, now, "RATE_LIMITED", 0)
		insertVerifications(t, ctx, conn, limited)

		// Give the MV cascade time to propagate, then confirm absence.
		require.Never(t, func() bool {
			usage, usageErr := client.GetBillableUsagePerIdentity(ctx, workspaceID, year, month, []string{keySpaceID}, nil)
			return usageErr == nil && len(usage) > 0
		}, 5*time.Second, time.Second)
	})

	t.Run("empty external_id is excluded, not attributed to a blank subject", func(t *testing.T) {
		workspaceID := uid.New(uid.WorkspacePrefix)
		keySpaceID := uid.New(uid.KeySpacePrefix)

		// Verifications with no identity attached.
		anonymous := createIdentityVerifications(workspaceID, keySpaceID, "", "", 40, now, "VALID", 1)
		// One attributed identity so we can positively confirm propagation.
		attributed := createIdentityVerifications(workspaceID, keySpaceID, uid.New(uid.IdentityPrefix), "user_x", 10, now, "VALID", 1)

		insertVerifications(t, ctx, conn, anonymous)
		insertVerifications(t, ctx, conn, attributed)

		require.Eventually(t, func() bool {
			usage, usageErr := client.GetBillableUsagePerIdentity(ctx, workspaceID, year, month, []string{keySpaceID}, nil)
			if usageErr != nil {
				return false
			}
			// Only the attributed identity appears; the anonymous 40 are absent.
			return len(usage) == 1 && usage[0].ExternalID == "user_x" && usage[0].Verifications == 10
		}, time.Minute, time.Second)
	})

	t.Run("workspace with no usage returns empty result", func(t *testing.T) {
		usage, err := client.GetBillableUsagePerIdentity(ctx, uid.New(uid.WorkspacePrefix), year, month, []string{uid.New(uid.KeySpacePrefix)}, []string{uid.New(uid.RatelimitNamespacePrefix)})
		require.NoError(t, err)
		require.Empty(t, usage)
	})

	t.Run("empty enabled sets bill nothing even with usage present", func(t *testing.T) {
		workspaceID := uid.New(uid.WorkspacePrefix)
		keySpaceID := uid.New(uid.KeySpacePrefix)
		insertVerifications(t, ctx, conn, createIdentityVerifications(workspaceID, keySpaceID, uid.New(uid.IdentityPrefix), "user_none", 10, now, "VALID", 5))

		// Confirm the usage did land under the enabled keyspace...
		require.Eventually(t, func() bool {
			usage, usageErr := client.GetBillableUsagePerIdentity(ctx, workspaceID, year, month, []string{keySpaceID}, nil)
			return usageErr == nil && len(usage) == 1
		}, time.Minute, time.Second)
		// ...but with nothing enabled, nothing bills.
		empty, err := client.GetBillableUsagePerIdentity(ctx, workspaceID, year, month, nil, nil)
		require.NoError(t, err)
		require.Empty(t, empty)
	})

	t.Run("attributed ratelimits count, bare identifiers stay unattributed", func(t *testing.T) {
		workspaceID := uid.New(uid.WorkspacePrefix)
		namespaceID := uid.New(uid.RatelimitNamespacePrefix)
		identityID := uid.New(uid.IdentityPrefix)

		events := make([]schema.RatelimitV3, 0, 40)
		// 15 passed checks attributed to user_rl.
		events = append(events, createRatelimitsV3(workspaceID, namespaceID, identityID, "user_rl", 15, now, true)...)
		// 5 blocked checks for the same identity: not billable.
		events = append(events, createRatelimitsV3(workspaceID, namespaceID, identityID, "user_rl", 5, now, false)...)
		// 20 passed checks with a bare identifier (no identity match).
		events = append(events, createRatelimitsV3(workspaceID, namespaceID, "", "", 20, now, true)...)

		insertRatelimitsV3(t, ctx, conn, events)

		require.Eventually(t, func() bool {
			usage, usageErr := client.GetBillableUsagePerIdentity(ctx, workspaceID, year, month, nil, []string{namespaceID})
			if usageErr != nil || len(usage) != 1 {
				return false
			}
			u := usage[0]
			return u.ExternalID == "user_rl" && u.IdentityID == identityID &&
				u.RatelimitsPassed == 15 && u.Verifications == 0 && u.SpentCredits == 0
		}, time.Minute, time.Second)

		// Scope gate: a non-enabled namespace excludes the same usage.
		none, err := client.GetBillableUsagePerIdentity(ctx, workspaceID, year, month, nil, []string{uid.New(uid.RatelimitNamespacePrefix)})
		require.NoError(t, err)
		require.Empty(t, none, "usage on a non-enabled namespace must be excluded")

		// The mirror MV forwards v3 rows into the v2 raw table, keeping the
		// workspace-grained billable rollup fed: all 35 passed checks count.
		require.Eventually(t, func() bool {
			count, countErr := client.GetBillableRatelimits(ctx, workspaceID, year, month)
			return countErr == nil && count == 35
		}, time.Minute, time.Second)
	})
}

// createRatelimitsV3 creates identity-attributed ratelimit v3 events in a
// specific namespace. Empty identityID/externalID models a bare identifier that
// matched no identity.
func createRatelimitsV3(workspaceID, namespaceID, identityID, externalID string, count int, timestamp time.Time, passed bool) []schema.RatelimitV3 {
	events := make([]schema.RatelimitV3, count)
	identifier := externalID
	if identifier == "" {
		identifier = uid.New(uid.IdentityPrefix)
	}
	var remaining uint64 = 50
	if !passed {
		remaining = 0
	}
	for i := range count {
		events[i] = schema.RatelimitV3{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        timestamp.Add(time.Duration(i) * time.Second).UnixMilli(),
			WorkspaceID: workspaceID,
			NamespaceID: namespaceID,
			Identifier:  identifier,
			IdentityID:  identityID,
			ExternalID:  externalID,
			Passed:      passed,
			Latency:     rand.Float64() * 10,
			OverrideID:  "",
			Limit:       100,
			Remaining:   remaining,
			ResetAt:     timestamp.Add(time.Minute).UnixMilli(),
			Tokens:      1,
		}
	}
	return events
}

func insertRatelimitsV3(t *testing.T, ctx context.Context, conn ch.Conn, events []schema.RatelimitV3) {
	if len(events) == 0 {
		return
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO default.ratelimits_raw_v3")
	require.NoError(t, err)

	for _, e := range events {
		err = batch.AppendStruct(&e)
		require.NoError(t, err)
	}

	err = batch.Send()
	require.NoError(t, err)
}

// TestBackfillBillableIdentityRollups proves the backfill is idempotent (a
// re-run over an already-populated period does not double-count) and that it
// reconstructs rollup rows from the source when the materialized views never
// captured them.
func TestBackfillBillableIdentityRollups(t *testing.T) {
	chCfg := containers.ClickHouse(t)
	dsn := chCfg.DSN

	client, err := clickhouse.New(clickhouse.Config{URL: dsn})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	require.NoError(t, client.Ping(context.Background()))

	opts, err := ch.ParseDSN(dsn)
	require.NoError(t, err)
	conn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	ctx := context.Background()

	// A closed prior month, distinct from the now-based periods other subtests
	// use so their rows never collide with this workspace's backfill.
	now := time.Now()
	firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	ts := firstOfThisMonth.AddDate(0, 0, -5)
	year, month := ts.Year(), int(ts.Month())

	workspaceID := uid.New(uid.WorkspacePrefix)
	keySpaceID := uid.New(uid.KeySpacePrefix)
	identityID := uid.New(uid.IdentityPrefix)
	enabled := []string{keySpaceID}

	// 40 VALID verifications, 3 credits each -> 40 verifications, 120 credits.
	batch := createIdentityVerifications(workspaceID, keySpaceID, identityID, "user_backfill", 40, ts, "VALID", 3)
	insertVerifications(t, ctx, conn, batch)

	// The MV cascade populates the identity rollup.
	require.Eventually(t, func() bool {
		usage, uErr := client.GetBillableUsagePerIdentity(ctx, workspaceID, year, month, enabled, nil)
		return uErr == nil && len(usage) == 1 &&
			usage[0].Verifications == 40 && usage[0].SpentCredits == 120
	}, time.Minute, time.Second)

	// Idempotency: a backfill over the already-populated period must not double.
	require.NoError(t, client.BackfillBillableIdentityRollups(ctx, year, month))
	usage, err := client.GetBillableUsagePerIdentity(ctx, workspaceID, year, month, enabled, nil)
	require.NoError(t, err)
	require.Len(t, usage, 1)
	require.Equal(t, int64(40), usage[0].Verifications, "backfill must not double-count an already-populated period")
	require.Equal(t, int64(120), usage[0].SpentCredits)

	// Simulate history the MVs never saw: delete the rollup rows directly. The
	// source key_verifications_per_month_v3 still holds the period, so backfill
	// can reconstruct it (the MVs alone never would — they only see new inserts).
	for _, table := range []string{
		"billable_verifications_per_identity_per_month_v2",
		"billable_credits_per_identity_per_month_v2",
	} {
		require.NoError(t, conn.Exec(ctx,
			"ALTER TABLE default."+table+" DELETE WHERE workspace_id = {ws:String} AND year = {y:Int32} AND month = {m:Int32} SETTINGS mutations_sync = 2",
			ch.Named("ws", workspaceID), ch.Named("y", year), ch.Named("m", month),
		))
	}
	require.Eventually(t, func() bool {
		u, uErr := client.GetBillableUsagePerIdentity(ctx, workspaceID, year, month, enabled, nil)
		return uErr == nil && len(u) == 0
	}, 30*time.Second, time.Second)

	// Backfill rebuilds the rollup from the retained source.
	require.NoError(t, client.BackfillBillableIdentityRollups(ctx, year, month))
	usage, err = client.GetBillableUsagePerIdentity(ctx, workspaceID, year, month, enabled, nil)
	require.NoError(t, err)
	require.Len(t, usage, 1)
	require.Equal(t, identityID, usage[0].IdentityID)
	require.Equal(t, int64(40), usage[0].Verifications, "backfill reconstructs verifications from the source")
	require.Equal(t, int64(120), usage[0].SpentCredits, "backfill reconstructs credits from the source")

	// The open (current) month and any future month are refused: backfilling a
	// still-accruing period would rebuild from a moving source.
	require.Error(t, client.BackfillBillableIdentityRollups(ctx, now.Year(), int(now.Month())),
		"backfill must refuse the current (open) month")
	require.Error(t, client.BackfillBillableIdentityRollups(ctx, now.Year()+1, 1),
		"backfill must refuse a future month")
}
