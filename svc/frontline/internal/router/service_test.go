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
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/router"
)

type stubQuerier struct {
	db.Querier
	row       db.FindFrontlineRouteByFQDNRow
	instances []db.FindInstancesByDeploymentIDRow
}

func (s stubQuerier) FindFrontlineRouteByFQDN(_ context.Context, _ string) (db.FindFrontlineRouteByFQDNRow, error) {
	return s.row, nil
}

func (s stubQuerier) FindInstancesByDeploymentID(_ context.Context, _ string) ([]db.FindInstancesByDeploymentIDRow, error) {
	return s.instances, nil
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

func TestRoute_ExcludedRegionsAreRequestScoped(t *testing.T) {
	t.Parallel()
	for _, region := range []string{"us-east-1", "us-east-2", "unknown-region"} {
		t.Run(region, func(t *testing.T) {
			t.Parallel()
			instances := make([]db.FindInstancesByDeploymentIDRow, 0, 3)
			for _, name := range []string{"us-east-1", "us-west-2", "eu-north-1"} {
				instances = append(instances, db.FindInstancesByDeploymentIDRow{
					ID: uid.New(uid.InstancePrefix), Status: db.InstancesStatusRunning,
					RegionName: name, RegionPlatform: "aws",
				})
			}
			svc, err := router.New(router.Config{
				Platform: "aws", Region: region,
				DB: stubQuerier{
					row: db.FindFrontlineRouteByFQDNRow{
						DeploymentID: uid.New(uid.DeploymentPrefix), DesiredState: db.DeploymentsDesiredStateRunning,
					},
					instances: instances,
				},
				FrontlineRouteCache:   newCache[db.FindFrontlineRouteByFQDNRow](t, "frontline_routes"),
				InstancesByDeployment: newCache[[]db.FindInstancesByDeploymentIDRow](t, "instances"),
				PolicyCache:           newCache[[]*frontlinev1.Policy](t, "policies"),
			})
			require.NoError(t, err)

			decision, err := svc.Route(t.Context(), "app.example.com", "us-east-1.aws", "us-west-2.aws")
			require.NoError(t, err)
			require.Equal(t, "eu-north-1.aws", decision.RemoteRegionAddress)
			if region == "us-east-1" {
				require.Equal(t, router.DestinationLocalInstance, decision.Destination)
				require.Len(t, decision.LocalInstances, 1)
			} else {
				require.Equal(t, router.DestinationRemoteRegion, decision.Destination)
			}

			decision, err = svc.Route(t.Context(), "app.example.com", "us-east-1.aws", "us-west-2.aws", "eu-north-1.aws")
			if region == "us-east-1" {
				require.NoError(t, err)
				require.Empty(t, decision.RemoteRegionAddress)
				require.Len(t, decision.LocalInstances, 1)
			} else {
				require.Error(t, err)
				code, ok := fault.GetCode(err)
				require.True(t, ok)
				require.Equal(t, codes.Frontline.Routing.NoRunningInstances.URN(), code)
			}

			decision, err = svc.Route(t.Context(), "app.example.com", "eu-north-1.aws")
			require.NoError(t, err)
			require.Contains(t, []string{"us-east-1.aws", "us-west-2.aws"}, decision.RemoteRegionAddress)
		})
	}
}
