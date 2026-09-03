package deployanomaly

import (
	"testing"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
)

func TestParseWindowStart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		want    int64
		wantErr bool
	}{
		{name: "prefixed", key: "deploy-anomaly-1800", want: 1_800_000},
		{name: "bare rolling compatibility", key: "1800", want: 1_800_000},
		{name: "not aligned", key: "deploy-anomaly-1801", wantErr: true},
		{name: "not numeric", key: "deploy-anomaly-nope", wantErr: true},
		{name: "empty", key: "deploy-anomaly-", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseWindowStart(test.key)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestShardKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{name: "valid", key: ShardKey(1_800_000, 3, 16)},
		{name: "unaligned window", key: "1800001/3/16", wantErr: true},
		{name: "shard outside count", key: "1800000/16/16", wantErr: true},
		{name: "zero shard count", key: "1800000/0/0", wantErr: true},
		{name: "missing part", key: "1800000/0", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			window, shard, count, err := ParseShardKey(test.key)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, int64(1_800_000), window)
			require.Equal(t, uint64(3), shard)
			require.Equal(t, uint64(16), count)
		})
	}
}

func TestNewHandlerDefaultsShardCount(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(HandlerConfig{Heartbeat: healthcheck.NewNoop()})
	require.NoError(t, err)
	require.Equal(t, defaultShardCount, handler.shardCount)
}

func TestMetricDataState(t *testing.T) {
	t.Parallel()

	require.Equal(t,
		hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_INCOMPLETE,
		metricDataState(true, false),
	)
	require.Equal(t,
		hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_ZERO_COMPLETE,
		metricDataState(false, true),
	)
	require.Equal(t,
		hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_PRESENT,
		metricDataState(true, true),
	)
}

func TestActionableGroups(t *testing.T) {
	t.Parallel()

	windowStart := int64(2_000_000)
	windowEnd := windowStart + windowDurationMillis
	spike := anomalyGroup{"ws-b", "p", "app", "env"}
	open := anomalyGroup{"ws-a", "p", "app", "env"}
	incomplete := anomalyGroup{"ws-c", "p", "app", "env"}
	groups := map[anomalyGroup]groupWindow{
		spike: {
			request: &clickhouse.RequestAnomalyWindow{
				RequestsCurrent: 1_000, RequestsBaselineMean: 100,
				RequestsBaselineStddev: 0, BaselineBuckets: 12,
				FirstBucketTime:      windowStart - 12*windowDurationMillis,
				CurrentBucketPresent: true,
			},
		},
		open:       {forced: true},
		incomplete: {request: &clickhouse.RequestAnomalyWindow{RequestsCurrent: 1_000, CurrentBucketPresent: true}},
	}

	got := actionableGroups(groups, windowStart, windowEnd, clickhouse.AnomalySourceWatermarks{
		Requests: windowEnd, Resources: 0,
	})
	require.Equal(t, []anomalyGroup{open, spike}, got)
}

func TestProductionAndRequestDropSuppression(t *testing.T) {
	t.Parallel()

	require.True(t, isProduction(mysqltype.EnvironmentKindProduction))
	require.False(t, isProduction(mysqltype.EnvironmentKindPreview))
	require.True(t, requestDropSuppressed(&hydrav1.EvaluateDeployAnomalyRequest{}))
	require.True(t, requestDropSuppressed(&hydrav1.EvaluateDeployAnomalyRequest{
		DeploymentId: "dep", DeploymentDesiredState: "stopped",
	}))
	require.False(t, requestDropSuppressed(&hydrav1.EvaluateDeployAnomalyRequest{
		DeploymentId: "dep", DeploymentDesiredState: "running",
	}))
}
