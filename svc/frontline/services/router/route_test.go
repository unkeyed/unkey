package router

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
)

// mockQuerier implements db.Querier for unit tests. Only the methods used by
// the router are wired up; the rest panic so we notice immediately if the
// router starts calling something unexpected.
type mockQuerier struct {
	db.Querier
	findRouteByFQDN func(ctx context.Context, fqdn string) (db.FindFrontlineRouteByFQDNRow, error)
	findSentinels   func(ctx context.Context, envID string) ([]db.FindHealthyRoutableSentinelsByEnvironmentIDRow, error)
	findInstances   func(ctx context.Context, deploymentID string) ([]db.Instance, error)
}

func (m *mockQuerier) FindFrontlineRouteByFQDN(ctx context.Context, fqdn string) (db.FindFrontlineRouteByFQDNRow, error) {
	return m.findRouteByFQDN(ctx, fqdn)
}

func (m *mockQuerier) FindHealthyRoutableSentinelsByEnvironmentID(ctx context.Context, envID string) ([]db.FindHealthyRoutableSentinelsByEnvironmentIDRow, error) {
	return m.findSentinels(ctx, envID)
}

func (m *mockQuerier) FindInstancesByDeploymentID(ctx context.Context, deploymentID string) ([]db.Instance, error) {
	return m.findInstances(ctx, deploymentID)
}

// newTestRouter creates a router service with real caches (so SWR calls
// refreshFromOrigin on miss) and the given mock DB.
func newTestRouter(t *testing.T, mock *mockQuerier, portalAddr string) *service {
	t.Helper()

	clk := clock.New()
	cacheConfig := func(resource string) cache.Config[string, db.FindFrontlineRouteByFQDNRow] {
		return cache.Config[string, db.FindFrontlineRouteByFQDNRow]{
			Fresh:    time.Minute,
			Stale:    5 * time.Minute,
			MaxSize:  100,
			Resource: resource,
			Clock:    clk,
		}
	}

	routeCache, err := cache.New(cacheConfig("test_frontline_routes"))
	require.NoError(t, err)

	sentinelCache, err := cache.New(cache.Config[string, []db.FindHealthyRoutableSentinelsByEnvironmentIDRow]{
		Fresh:    time.Minute,
		Stale:    5 * time.Minute,
		MaxSize:  100,
		Resource: "test_sentinels",
		Clock:    clk,
	})
	require.NoError(t, err)

	instanceCache, err := cache.New(cache.Config[string, []db.Instance]{
		Fresh:    time.Minute,
		Stale:    5 * time.Minute,
		MaxSize:  100,
		Resource: "test_instances",
		Clock:    clk,
	})
	require.NoError(t, err)

	svc, err := New(Config{
		Platform:               "dev",
		Region:                 "local",
		DB:                     mock,
		FrontlineRouteCache:    routeCache,
		SentinelsByEnvironment: sentinelCache,
		InstancesByDeployment:  instanceCache,
		PortalAddr:             portalAddr,
	})
	require.NoError(t, err)
	return svc
}

func TestRoute_PortalRouteForwardsToPortalService(t *testing.T) {
	t.Parallel()

	mock := &mockQuerier{
		findRouteByFQDN: func(_ context.Context, _ string) (db.FindFrontlineRouteByFQDNRow, error) {
			return db.FindFrontlineRouteByFQDNRow{
				RouteType:      db.FrontlineRoutesRouteTypePortal,
				PortalConfigID: sql.NullString{String: "pcfg_test123", Valid: true},
				PathPrefix:     sql.NullString{String: "/portal", Valid: true},
			}, nil
		},
	}

	svc := newTestRouter(t, mock, "portal:3000")
	decision, err := svc.Route(context.Background(), "acme.unkey.com")

	require.NoError(t, err)
	require.Equal(t, DestinationPortal, decision.Destination)
	require.Equal(t, "portal:3000", decision.Address)
	require.Equal(t, "/portal", decision.PathPrefix)
	require.Empty(t, decision.DeploymentID, "portal routes should not have a deployment ID")
}

func TestRoute_DeploymentRouteContinuesToUseSentinel(t *testing.T) {
	t.Parallel()

	mock := &mockQuerier{
		findRouteByFQDN: func(_ context.Context, _ string) (db.FindFrontlineRouteByFQDNRow, error) {
			return db.FindFrontlineRouteByFQDNRow{
				RouteType:     db.FrontlineRoutesRouteTypeDeployment,
				EnvironmentID: sql.NullString{String: "env_abc", Valid: true},
				DeploymentID:  sql.NullString{String: "dep_xyz", Valid: true},
			}, nil
		},
		findSentinels: func(_ context.Context, _ string) ([]db.FindHealthyRoutableSentinelsByEnvironmentIDRow, error) {
			return []db.FindHealthyRoutableSentinelsByEnvironmentIDRow{
				{
					K8sAddress:     "sentinel-0.sentinel:8080",
					RegionName:     "local",
					RegionPlatform: "dev",
				},
			}, nil
		},
		findInstances: func(_ context.Context, _ string) ([]db.Instance, error) {
			return []db.Instance{
				{Status: db.InstancesStatusRunning},
			}, nil
		},
	}

	svc := newTestRouter(t, mock, "portal:3000")
	decision, err := svc.Route(context.Background(), "myapp.unkey.app")

	require.NoError(t, err)
	require.Equal(t, DestinationLocalSentinel, decision.Destination)
	require.Equal(t, "dep_xyz", decision.DeploymentID)
	require.Equal(t, "sentinel-0.sentinel:8080", decision.Address)
	require.Empty(t, decision.PathPrefix, "deployment routes should not have a path prefix")
}

func TestRoute_UnconfiguredSubdomainReturns404(t *testing.T) {
	t.Parallel()

	mock := &mockQuerier{
		findRouteByFQDN: func(_ context.Context, _ string) (db.FindFrontlineRouteByFQDNRow, error) {
			return db.FindFrontlineRouteByFQDNRow{}, sql.ErrNoRows
		},
	}

	svc := newTestRouter(t, mock, "portal:3000")
	_, err := svc.Route(context.Background(), "nonexistent.unkey.com")

	require.Error(t, err)
	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Frontline.Routing.ConfigNotFound.URN(), urn)
}

func TestRoute_PortalRouteWithNoPortalAddrReturns503(t *testing.T) {
	t.Parallel()

	mock := &mockQuerier{
		findRouteByFQDN: func(_ context.Context, _ string) (db.FindFrontlineRouteByFQDNRow, error) {
			return db.FindFrontlineRouteByFQDNRow{
				RouteType:      db.FrontlineRoutesRouteTypePortal,
				PortalConfigID: sql.NullString{String: "pcfg_test123", Valid: true},
				PathPrefix:     sql.NullString{String: "/portal", Valid: true},
			}, nil
		},
	}

	// Empty portal address — portal service not configured.
	svc := newTestRouter(t, mock, "")
	_, err := svc.Route(context.Background(), "acme.unkey.com")

	require.Error(t, err)
	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Frontline.Proxy.ServiceUnavailable.URN(), urn)
}

func TestRoute_PortalRouteWithEmptyPathPrefix(t *testing.T) {
	t.Parallel()

	mock := &mockQuerier{
		findRouteByFQDN: func(_ context.Context, _ string) (db.FindFrontlineRouteByFQDNRow, error) {
			return db.FindFrontlineRouteByFQDNRow{
				RouteType:      db.FrontlineRoutesRouteTypePortal,
				PortalConfigID: sql.NullString{String: "pcfg_test123", Valid: true},
				PathPrefix:     sql.NullString{String: "", Valid: false},
			}, nil
		},
	}

	svc := newTestRouter(t, mock, "portal:3000")
	decision, err := svc.Route(context.Background(), "acme.unkey.com")

	require.NoError(t, err)
	require.Equal(t, DestinationPortal, decision.Destination)
	require.Empty(t, decision.PathPrefix)
}
