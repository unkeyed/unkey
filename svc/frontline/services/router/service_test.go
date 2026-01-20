package router

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

func newTestCache[K comparable, V any](t *testing.T) cache.Cache[K, V] {
	t.Helper()
	clk := clock.New()
	c, err := cache.New(cache.Config[K, V]{
		Fresh:    time.Minute,
		Stale:    5 * time.Minute,
		Logger:   logging.NewNoop(),
		MaxSize:  1000,
		Resource: "test",
		Clock:    clk,
	})
	require.NoError(t, err)
	return c
}

func TestSelectSentinel_LocalRegionWithRunningInstances(t *testing.T) {
	ctx := context.Background()

	runningRegionsCache := newTestCache[string, []string](t)
	runningRegionsCache.Set(ctx, "deploy-1", []string{"us-east-1.aws", "us-west-2.aws"})

	svc := &service{
		logger:                           logging.NewNoop(),
		region:                           "us-east-1.aws",
		clock:                            clock.New(),
		runningInstanceRegionsByDeployID: runningRegionsCache,
	}

	route := &db.FrontlineRoute{DeploymentID: "deploy-1"}
	sentinels := []db.Sentinel{
		{Region: "us-east-1.aws", Health: db.SentinelsHealthHealthy, K8sAddress: "sentinel-1.local"},
		{Region: "us-west-2.aws", Health: db.SentinelsHealthHealthy, K8sAddress: "sentinel-2.local"},
	}

	decision, err := svc.SelectSentinel(ctx, route, sentinels)
	require.NoError(t, err)
	require.NotNil(t, decision.LocalSentinel)
	require.Equal(t, "us-east-1.aws", decision.LocalSentinel.Region)
	require.Empty(t, decision.NearestNLBRegion)
}

func TestSelectSentinel_LocalRegionHealthyButNoRunningInstances(t *testing.T) {
	ctx := context.Background()

	runningRegionsCache := newTestCache[string, []string](t)
	runningRegionsCache.Set(ctx, "deploy-1", []string{"us-west-2.aws"})

	svc := &service{
		logger:                           logging.NewNoop(),
		region:                           "us-east-1.aws",
		clock:                            clock.New(),
		runningInstanceRegionsByDeployID: runningRegionsCache,
	}

	route := &db.FrontlineRoute{DeploymentID: "deploy-1"}
	sentinels := []db.Sentinel{
		{Region: "us-east-1.aws", Health: db.SentinelsHealthHealthy, K8sAddress: "sentinel-1.local"},
		{Region: "us-west-2.aws", Health: db.SentinelsHealthHealthy, K8sAddress: "sentinel-2.local"},
	}

	decision, err := svc.SelectSentinel(ctx, route, sentinels)
	require.NoError(t, err)
	require.Nil(t, decision.LocalSentinel)
	require.Equal(t, "us-west-2.aws", decision.NearestNLBRegion)
}

func TestSelectSentinel_NoRunningInstancesAnywhere(t *testing.T) {
	ctx := context.Background()

	runningRegionsCache := newTestCache[string, []string](t)
	runningRegionsCache.Set(ctx, "deploy-1", []string{})

	svc := &service{
		logger:                           logging.NewNoop(),
		region:                           "us-east-1.aws",
		clock:                            clock.New(),
		runningInstanceRegionsByDeployID: runningRegionsCache,
	}

	route := &db.FrontlineRoute{DeploymentID: "deploy-1"}
	sentinels := []db.Sentinel{
		{Region: "us-east-1.aws", Health: db.SentinelsHealthHealthy, K8sAddress: "sentinel-1.local"},
		{Region: "us-west-2.aws", Health: db.SentinelsHealthHealthy, K8sAddress: "sentinel-2.local"},
	}

	decision, err := svc.SelectSentinel(ctx, route, sentinels)
	require.Error(t, err)
	require.Nil(t, decision)
	require.Contains(t, err.Error(), "no regions with running instances")
}

func TestSelectSentinel_NoHealthySentinels(t *testing.T) {
	ctx := context.Background()

	runningRegionsCache := newTestCache[string, []string](t)
	runningRegionsCache.Set(ctx, "deploy-1", []string{"us-east-1.aws"})

	svc := &service{
		logger:                           logging.NewNoop(),
		region:                           "us-east-1.aws",
		clock:                            clock.New(),
		runningInstanceRegionsByDeployID: runningRegionsCache,
	}

	route := &db.FrontlineRoute{DeploymentID: "deploy-1"}
	sentinels := []db.Sentinel{
		{Region: "us-east-1.aws", Health: db.SentinelsHealthUnhealthy, K8sAddress: "sentinel-1.local"},
		{Region: "us-west-2.aws", Health: db.SentinelsHealthUnhealthy, K8sAddress: "sentinel-2.local"},
	}

	decision, err := svc.SelectSentinel(ctx, route, sentinels)
	require.Error(t, err)
	require.Nil(t, decision)
	require.Contains(t, err.Error(), "no healthy sentinels")
}

func TestSelectSentinel_ProximityBasedSelection(t *testing.T) {
	ctx := context.Background()

	runningRegionsCache := newTestCache[string, []string](t)
	runningRegionsCache.Set(ctx, "deploy-1", []string{"eu-west-1.aws", "ap-southeast-1.aws"})

	svc := &service{
		logger:                           logging.NewNoop(),
		region:                           "us-east-1.aws",
		clock:                            clock.New(),
		runningInstanceRegionsByDeployID: runningRegionsCache,
	}

	route := &db.FrontlineRoute{DeploymentID: "deploy-1"}
	sentinels := []db.Sentinel{
		{Region: "eu-west-1.aws", Health: db.SentinelsHealthHealthy, K8sAddress: "sentinel-eu.local"},
		{Region: "ap-southeast-1.aws", Health: db.SentinelsHealthHealthy, K8sAddress: "sentinel-ap.local"},
	}

	decision, err := svc.SelectSentinel(ctx, route, sentinels)
	require.NoError(t, err)
	require.Nil(t, decision.LocalSentinel)
	require.Equal(t, "eu-west-1.aws", decision.NearestNLBRegion)
}

func TestSelectSentinel_HealthySentinelButNoInstancesInThatRegion(t *testing.T) {
	ctx := context.Background()

	runningRegionsCache := newTestCache[string, []string](t)
	runningRegionsCache.Set(ctx, "deploy-1", []string{"eu-central-1.aws"})

	svc := &service{
		logger:                           logging.NewNoop(),
		region:                           "us-east-1.aws",
		clock:                            clock.New(),
		runningInstanceRegionsByDeployID: runningRegionsCache,
	}

	route := &db.FrontlineRoute{DeploymentID: "deploy-1"}
	sentinels := []db.Sentinel{
		{Region: "us-east-1.aws", Health: db.SentinelsHealthHealthy, K8sAddress: "sentinel-us.local"},
		{Region: "eu-central-1.aws", Health: db.SentinelsHealthHealthy, K8sAddress: "sentinel-eu.local"},
	}

	decision, err := svc.SelectSentinel(ctx, route, sentinels)
	require.NoError(t, err)
	require.Nil(t, decision.LocalSentinel)
	require.Equal(t, "eu-central-1.aws", decision.NearestNLBRegion)
}
