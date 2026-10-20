package ratelimit

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/gcra"
	gcradb "github.com/unkeyed/unkey/pkg/gcra/db"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestGCRAName_IsUnambiguousAndBounded(t *testing.T) {
	base := RatelimitRequest{
		WorkspaceID: uid.New(uid.WorkspacePrefix),
		Namespace:   "a:1:b",
		Identifier:  "c",
		Duration:    time.Minute,
		Limit:       10,
		Cost:        1,
		Time:        time.Time{},
		UseGCRA:     true,
	}
	shifted := base
	shifted.Namespace = "a"
	shifted.Identifier = "b:1:c"
	require.NotEqual(t, gcraName(base), gcraName(shifted))

	otherLimit := base
	otherLimit.Limit = 20
	require.Equal(t, gcraName(base), gcraName(otherLimit), "a limit change must keep what was spent")

	otherDuration := base
	otherDuration.Duration = time.Hour
	require.NotEqual(t, gcraName(base), gcraName(otherDuration))

	long := base
	long.Identifier = strings.Repeat("x", gcra.NameMaxLength)
	require.True(t, strings.HasPrefix(gcraName(long), "rlh:"))
	require.LessOrEqual(t, len(gcraName(long)), gcra.NameMaxLength)
}

func TestRatelimit_GCRASharesStateBetweenServices(t *testing.T) {
	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock(time.Now().Round(0))
	newService := func() *service {
		svc, err := New(Config{Clock: clk, Counter: counter.NewMemory(), DB: env.db, Region: "test-region"})
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, svc.Close()) })
		return svc
	}
	a, b := newService(), newService()
	ctx := context.Background()
	req := RatelimitRequest{
		WorkspaceID: uid.New(uid.WorkspacePrefix),
		Namespace:   uid.New(uid.RatelimitNamespacePrefix),
		Identifier:  uid.New(uid.TestPrefix),
		Limit:       30,
		Duration:    100 * time.Second,
		Cost:        1,
		Time:        time.Time{},
		UseGCRA:     true,
	}

	for i := range 3 {
		resp, err := a.Ratelimit(ctx, req)
		require.NoError(t, err)
		require.True(t, resp.Success)
		require.Equal(t, int64(2-i), resp.Remaining)
		require.Equal(t, int64(i+1), resp.Current)
	}
	start := clk.Now()
	clk.Tick(time.Second)
	require.Eventually(t, func() bool {
		row, err := gcradb.Query.ReadPeers(ctx, env.db.RW(), gcradb.ReadPeersParams{Name: gcraName(req), WriterID: ""})
		require.NoError(t, err)
		return row.SpentNs == 3*3333333334
	}, 10*time.Second, 50*time.Millisecond)

	denied, err := b.Ratelimit(ctx, req)
	require.NoError(t, err)
	require.False(t, denied.Success)
	require.Equal(t, start.Add(10*time.Second+2), denied.Reset)

	regional := req
	regional.UseGCRA = false
	allowed, err := b.Ratelimit(ctx, regional)
	require.NoError(t, err)
	require.True(t, allowed.Success, "a request without UseGCRA keeps the regional counters")

	other := req
	other.Identifier = uid.New(uid.TestPrefix)
	responses, err := b.RatelimitMany(ctx, []RatelimitRequest{other, req})
	require.NoError(t, err)
	require.True(t, responses[0].Success)
	require.Equal(t, int64(3), responses[0].Remaining, "nothing was spent, so its own cost is reported back")
	require.False(t, responses[1].Success)
}
