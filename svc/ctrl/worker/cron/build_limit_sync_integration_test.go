package cron_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestRunBuildLimitSync_Integration covers the states Restate's rules can be
// in when a tick runs: none, already correct, a wrong default, a workspace
// rule the database calls for, and one it does not. The repeated-tick case is
// the one that matters for the design, because the
// handler writes unconditionally and relies on Restate leaving an identical
// write alone.
func TestRunBuildLimitSync_Integration(t *testing.T) {
	h := harness.New(t)
	admin := restateadmin.New(restateadmin.Config{BaseURL: h.RestateAdmin, APIKey: ""})
	client := hydrav1.NewCronServiceIngressClient(h.Restate, "build-limit-sync")

	empty, err := admin.ListRules(h.Ctx)
	require.NoError(t, err)
	require.Empty(t, empty, "no rules exist before the first tick")

	_, err = client.RunBuildLimitSync().Request(h.Ctx, &hydrav1.RunBuildLimitSyncRequest{})
	require.NoError(t, err)

	written, err := admin.ListRules(h.Ctx)
	require.NoError(t, err)
	require.Len(t, written, 1)
	require.Equal(t, "builds/*", written[0].Pattern)
	require.Equal(t, uint32(1), written[0].Concurrency)
	require.False(t, written[0].Disabled)

	_, err = client.RunBuildLimitSync().Request(h.Ctx, &hydrav1.RunBuildLimitSyncRequest{})
	require.NoError(t, err)

	// A version bump wakes every build queued at that level, so a tick that
	// rewrites the same limits must not move it
	unchanged, err := admin.ListRules(h.Ctx)
	require.NoError(t, err)
	require.Equal(t, written, unchanged, "rewriting identical limits must not touch the rule")

	err = admin.UpsertRules(h.Ctx, []restateadmin.RuleUpsert{
		{Pattern: "builds/*", Concurrency: 7, Description: "raised by hand"},
	})
	require.NoError(t, err)

	_, err = client.RunBuildLimitSync().Request(h.Ctx, &hydrav1.RunBuildLimitSyncRequest{})
	require.NoError(t, err)

	healed, err := admin.ListRules(h.Ctx)
	require.NoError(t, err)
	require.Len(t, healed, 1)
	require.Equal(t, uint32(1), healed[0].Concurrency, "the database is the source of truth, not a hand-set rule")
	require.Greater(t, healed[0].Version, written[0].Version, "a real limit change must move the version")

	raised := h.Seed.CreateWorkspace(h.Ctx)
	limits := db.UpsertLimitParams{
		WorkspaceID:                           raised.ID,
		ApiBillableOperationsCountMaxPerMonth: 1_000_000,
		ApiRequestsCountMaxPerMinute:          sql.NullInt32{Valid: false, Int32: 0},
		LogsRetentionDaysMax:                  7,
		LogsAuditRetentionDaysMax:             30,
		TeamEnabled:                           false,
		CpuCoresMax:                           10,
		CpuCoresMaxPerInstance:                2,
		MemoryMibMax:                          20_480,
		MemoryMibMaxPerInstance:               4_096,
		StorageMibMax:                         51_200,
		StorageMibMaxPerInstance:              10_240,
		BuildsConcurrentMax:                   3,
		CustomDomainsMax:                      0,
		AutoscalingReplicasMax:                0,
	}
	require.NoError(t, h.DB.UpsertLimit(h.Ctx, limits))
	err = admin.UpsertRules(h.Ctx, []restateadmin.RuleUpsert{
		{Pattern: "builds/ws_KEBAP", Concurrency: 9, Description: "set by hand"},
	})
	require.NoError(t, err)

	_, err = client.RunBuildLimitSync().Request(h.Ctx, &hydrav1.RunBuildLimitSyncRequest{})
	require.NoError(t, err)

	byPattern := func(rules []restateadmin.Rule) map[string]restateadmin.Rule {
		out := make(map[string]restateadmin.Rule, len(rules))
		for _, rule := range rules {
			out[rule.Pattern] = rule
		}
		return out
	}
	withWorkspace, err := admin.ListRules(h.Ctx)
	require.NoError(t, err)
	rules := byPattern(withWorkspace)
	require.Len(t, rules, 2, "the default rule plus one rule for the raised workspace, nothing else")
	require.Equal(t, uint32(3), rules["builds/"+raised.ID].Concurrency)
	require.Equal(t, uint32(1), rules["builds/*"].Concurrency)
	require.NotContains(t, rules, "builds/ws_KEBAP", "a workspace rule the database does not call for is deleted")

	limits.BuildsConcurrentMax = 1
	require.NoError(t, h.DB.UpsertLimit(h.Ctx, limits))

	_, err = client.RunBuildLimitSync().Request(h.Ctx, &hydrav1.RunBuildLimitSyncRequest{})
	require.NoError(t, err)

	backToDefault, err := admin.ListRules(h.Ctx)
	require.NoError(t, err)
	require.Equal(t, healed, backToDefault, "a workspace back at the default loses its rule and the default rule is untouched")
}
