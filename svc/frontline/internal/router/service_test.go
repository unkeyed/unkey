package router_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/router"
)

// stubQuerier embeds db.Querier so the test only implements the one method Route
// reaches. Any other call hits the nil embedded interface and panics, which is
// the point: a suspended deployment must short-circuit before getInstances or
// getPolicies run.
type stubQuerier struct {
	db.Querier
	row db.FindFrontlineRouteByFQDNRow
}

func (s stubQuerier) FindFrontlineRouteByFQDN(_ context.Context, _ string) (db.FindFrontlineRouteByFQDNRow, error) {
	return s.row, nil
}

// No instances: the running path falls through to selectDestination, which
// returns NoRunningInstances on an empty set.
func (s stubQuerier) FindInstancesByDeploymentID(_ context.Context, _ string) ([]db.FindInstancesByDeploymentIDRow, error) {
	return nil, nil
}

func newCache[V any](t *testing.T, resource string) cache.Cache[string, V] {
	t.Helper()
	c, err := cache.New(cache.Config[string, V]{
		Fresh:    time.Minute,
		Stale:    time.Minute,
		MaxSize:  10,
		Resource: resource,
		Clock:    clock.New(),
	})
	require.NoError(t, err)
	return c
}

// TestRoute_OfflineDeploymentRoutesByReason verifies the offline branching by
// reason and precedence: a spend-suspended workspace returns SpendLimitReached
// (402) up front, even while its deployment's desired_state is still "running"
// and the stop is mid-propagation; a stopped deployment that is not
// spend-suspended returns DeploymentOffline (503); a missing billing row
// (LEFT JOIN NULL) is not a spend suspension, so it falls to offline.
func TestRoute_OfflineDeploymentRoutesByReason(t *testing.T) {
	cases := []struct {
		name           string
		desiredState   db.DeploymentsDesiredState
		spendSuspended sql.NullBool
		wantURN        codes.URN
	}{
		{"stopped_offline", db.DeploymentsDesiredStateStopped, sql.NullBool{Valid: true, Bool: false}, codes.Frontline.Routing.DeploymentOffline.URN()},
		{"suspended_while_running", db.DeploymentsDesiredStateRunning, sql.NullBool{Valid: true, Bool: true}, codes.Frontline.Routing.SpendLimitReached.URN()},
		{"suspended_while_stopped", db.DeploymentsDesiredStateStopped, sql.NullBool{Valid: true, Bool: true}, codes.Frontline.Routing.SpendLimitReached.URN()},
		{"no_billing_row", db.DeploymentsDesiredStateStopped, sql.NullBool{}, codes.Frontline.Routing.DeploymentOffline.URN()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, err := router.New(router.Config{
				Platform: "test",
				Region:   "test",
				DB: stubQuerier{row: db.FindFrontlineRouteByFQDNRow{
					EnvironmentID:    "env_1",
					DeploymentID:     "dep_1",
					SentinelConfig:   []byte("{}"),
					UpstreamProtocol: db.DeploymentsUpstreamProtocolHttp1,
					DesiredState:     tc.desiredState,
					SpendSuspended:   tc.spendSuspended,
				}},
				FrontlineRouteCache:   newCache[db.FindFrontlineRouteByFQDNRow](t, "frontline_routes"),
				InstancesByDeployment: newCache[[]db.FindInstancesByDeploymentIDRow](t, "instances"),
				PolicyCache:           newCache[[]*frontlinev1.Policy](t, "policies"),
			})
			require.NoError(t, err)

			_, err = svc.Route(context.Background(), "app.example.com")
			require.Error(t, err)

			urn, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, tc.wantURN, urn)
		})
	}
}

// TestRoute_RunningDeploymentPassesGuard verifies the offline guard does not
// fire for a running deployment: it advances past findRoute to instance
// selection, which (with no instances seeded) returns the transient
// NoRunningInstances rather than the offline DeploymentOffline.
func TestRoute_RunningDeploymentPassesGuard(t *testing.T) {
	svc, err := router.New(router.Config{
		Platform: "test",
		Region:   "test",
		DB: stubQuerier{row: db.FindFrontlineRouteByFQDNRow{
			EnvironmentID:    "env_1",
			DeploymentID:     "dep_1",
			SentinelConfig:   []byte("{}"),
			UpstreamProtocol: db.DeploymentsUpstreamProtocolHttp1,
			DesiredState:     db.DeploymentsDesiredStateRunning,
		}},
		FrontlineRouteCache:   newCache[db.FindFrontlineRouteByFQDNRow](t, "frontline_routes"),
		InstancesByDeployment: newCache[[]db.FindInstancesByDeploymentIDRow](t, "instances"),
		PolicyCache:           newCache[[]*frontlinev1.Policy](t, "policies"),
	})
	require.NoError(t, err)

	_, err = svc.Route(context.Background(), "app.example.com")
	require.Error(t, err)

	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Frontline.Routing.NoRunningInstances.URN(), urn,
		"running deployment must pass the guard and fail on instance selection, not as suspended")
}
