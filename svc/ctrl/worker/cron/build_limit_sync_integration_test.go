package cron_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
)

// TestRunBuildLimitSync_Integration covers the three states Restate's rules
// can be in when a tick runs: none, already correct, and a wrong limit.
// The repeated-tick case is the one that matters for the design, because the
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
}
