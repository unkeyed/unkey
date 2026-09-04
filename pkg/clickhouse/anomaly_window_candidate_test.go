package clickhouse_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deployanomaly"
)

func TestAnomalyCandidateFilterSuperset(t *testing.T) {
	t.Parallel()

	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	opts, err := ch.ParseDSN(cfg.DSN)
	require.NoError(t, err)
	conn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	ctx := t.Context()
	windowStart := time.Now().UTC().Truncate(5 * time.Minute).Add(-5 * time.Minute)
	const groupCount = 96
	scope := uid.New("candidate")
	random := rand.New(rand.NewSource(42))
	workspaceIDs := make([]string, groupCount)
	lifetimeBuckets := make(map[string]int64, groupCount)
	batch, err := conn.PrepareBatch(ctx, `
		INSERT INTO default.frontline_requests_per_5m_v1
			(time, workspace_id, project_id, app_id, environment_id, deployment_id, response_status, count)
	`)
	require.NoError(t, err)
	for group := range groupCount {
		ids := candidateTestIDs(scope, group)
		workspaceIDs[group] = ids.WorkspaceID
		lifetimeBuckets[ids.WorkspaceID] = int64(72 + random.Intn(217))
		for bucket := range 72 {
			timestamp := windowStart.Add(-time.Duration(72-bucket) * 5 * time.Minute)
			requests := int64(300 + random.Intn(800))
			error4xx := int64(random.Intn(20))
			error5xx := int64(random.Intn(10))
			if group == 2 {
				requests = 100
				error4xx = 0
				error5xx = 90
			}
			for status, count := range map[int32]int64{200: requests - error4xx - error5xx, 404: error4xx, 500: error5xx} {
				require.NoError(t, batch.Append(timestamp, ids.WorkspaceID, ids.ProjectID, ids.AppID, ids.EnvironmentID, "dep", status, count))
			}
		}

		current := map[int32]int64{200: int64(300 + random.Intn(800)), 404: 5, 500: 5}
		switch group % 4 {
		case 1:
			current = map[int32]int64{200: int64(3_000 + random.Intn(3_000)), 404: 20, 500: 20}
		case 2:
			current = map[int32]int64{500: 99}
		case 3:
			continue
		}
		for status, count := range current {
			require.NoError(t, batch.Append(windowStart, ids.WorkspaceID, ids.ProjectID, ids.AppID, ids.EnvironmentID, "dep", status, count))
		}
	}
	require.NoError(t, batch.Send())

	all, err := client.GetRequestAnomalyWindows(ctx, clickhouse.AnomalyWindowsRequest{
		WindowStart: windowStart.UnixMilli(), WorkspaceIDs: workspaceIDs,
	})
	require.NoError(t, err)
	require.Len(t, all, groupCount)
	detectorConfig := deployanomaly.DefaultConfig(deployanomaly.SensitivityNormal)
	want := make(map[clickhouse.AnomalyGroupKey]struct{})
	for _, row := range all {
		windowBuckets := lifetimeBuckets[row.WorkspaceID]
		inputs := []deployanomaly.Input{
			{
				Metric: deployanomaly.MetricError5xx, Current: row.Error5xxCurrent,
				RequestsInWindow: row.RequestsCurrent, BaselineMean: row.Error5xxBaselineMean,
				BaselineStddev: row.Error5xxBaselineStddev, ObservedBaselineBuckets: row.BaselineBuckets,
				BaselineWindowBuckets: windowBuckets,
			},
			{
				Metric: deployanomaly.MetricError4xx, Current: row.Error4xxCurrent,
				RequestsInWindow: row.RequestsCurrent, BaselineMean: row.Error4xxBaselineMean,
				BaselineStddev: row.Error4xxBaselineStddev, ObservedBaselineBuckets: row.BaselineBuckets,
				BaselineWindowBuckets: windowBuckets,
			},
			{
				Metric: deployanomaly.MetricRequests, Current: row.RequestsCurrent,
				BaselineMean: row.RequestsBaselineMean, BaselineStddev: row.RequestsBaselineStddev,
				ObservedBaselineBuckets: row.BaselineBuckets, BaselineWindowBuckets: windowBuckets,
			},
		}
		median, active := deployanomaly.RecentRequestStats(row.RecentRequests, detectorConfig.RequestDrop.ActivityPerBucket)
		inputs = append(inputs, deployanomaly.Input{
			Metric: deployanomaly.MetricRequestsDrop, Current: row.RequestsCurrent,
			RecentMedianRequests: median, RecentActiveBuckets: active,
			ObservedBaselineBuckets: row.BaselineBuckets, BaselineWindowBuckets: windowBuckets,
		})
		for _, input := range inputs {
			result := deployanomaly.Detect(input, detectorConfig)
			if result.Outcome == deployanomaly.OutcomeCandidate || result.Outcome == deployanomaly.OutcomeAnomaly {
				want[requestWindowKey(row)] = struct{}{}
			}
		}
	}

	filter := candidateTestFilter(detectorConfig)
	filtered, err := client.GetRequestAnomalyWindows(ctx, clickhouse.AnomalyWindowsRequest{
		WindowStart: windowStart.UnixMilli(), WorkspaceIDs: workspaceIDs, CandidateFilter: &filter,
	})
	require.NoError(t, err)
	got := make(map[clickhouse.AnomalyGroupKey]struct{}, len(filtered))
	for _, row := range filtered {
		got[requestWindowKey(row)] = struct{}{}
	}
	for key := range want {
		require.Contains(t, got, key)
	}
	require.NotEmpty(t, want)

	explicit := requestWindowKey(all[0])
	explicitRows, err := client.GetRequestAnomalyWindows(ctx, clickhouse.AnomalyWindowsRequest{
		WindowStart: windowStart.UnixMilli(), CandidateFilter: &filter,
		SkipFleet: true, GroupKeys: []clickhouse.AnomalyGroupKey{explicit},
	})
	require.NoError(t, err)
	require.Len(t, explicitRows, 1)
	require.Equal(t, explicit, requestWindowKey(explicitRows[0]))
}

func candidateTestIDs(scope string, group int) clickhouse.AnomalyGroupKey {
	suffix := fmt.Sprintf("%s-%03d", scope, group)
	return clickhouse.AnomalyGroupKey{
		WorkspaceID:   "ws-" + suffix,
		ProjectID:     "prj-" + suffix,
		AppID:         "app-" + suffix,
		EnvironmentID: "env-" + suffix,
	}
}

func requestWindowKey(row clickhouse.RequestAnomalyWindow) clickhouse.AnomalyGroupKey {
	return clickhouse.AnomalyGroupKey{
		WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID,
		AppID: row.AppID, EnvironmentID: row.EnvironmentID,
	}
}

func candidateTestFilter(cfg deployanomaly.Config) clickhouse.AnomalyCandidateFilter {
	return clickhouse.AnomalyCandidateFilter{
		SigmaK: cfg.SigmaK, MinimumStddevRatio: cfg.MinimumStddevRatio,
		ErrorRatioStddevFloor: cfg.StddevFloors.ErrorRatio,
		RequestsStddevFloor:   cfg.StddevFloors.Requests, EgressBytesStddevFloor: cfg.StddevFloors.EgressBytes,
		CPUSecondsStddevFloor: cfg.StddevFloors.CPUSeconds, ErrorExcessFailures: cfg.ActivityFloors.ErrorExcessFailures,
		RequestsActivity: cfg.ActivityFloors.Requests, EgressBytesActivity: cfg.ActivityFloors.EgressBytes,
		CPUSecondsActivity: cfg.ActivityFloors.CPUSeconds, MemoryUtilizationActivity: cfg.ActivityFloors.MemoryUtilization,
		BaselineMinimum: cfg.BaselineMinimums.Requests, RequestDropBaseline: cfg.BaselineMinimums.RequestsDrop,
		RequestDropFraction: cfg.RequestDrop.RecentLevelFraction, RequestDropActivity: cfg.RequestDrop.ActivityPerBucket,
		RequestDropActiveBuckets: cfg.RequestDrop.MinimumActiveBuckets, RequestDropAbsoluteLoss: cfg.RequestDrop.MinimumAbsoluteLoss,
		Catastrophic5xxRatio: cfg.Catastrophic.Error5xxRatio, Catastrophic5xxFailures: cfg.Catastrophic.Error5xxFailures,
	}
}
