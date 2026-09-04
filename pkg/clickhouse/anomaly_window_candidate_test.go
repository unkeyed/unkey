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
			if group >= groupCount/2 {
				requests = 10_000_000_000_000 + int64(random.Intn(100_000))*1_000_000
			}
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

		currentRequests := int64(300 + random.Intn(800))
		if group >= groupCount/2 {
			currentRequests = 10_000_000_000_000 + int64(random.Intn(100_000))*1_000_000
		}
		current := map[int32]int64{200: currentRequests, 404: 5, 500: 5}
		switch group % 4 {
		case 1:
			if group >= groupCount/2 {
				current = map[int32]int64{200: 14_000_000_000_000 + int64(random.Intn(100_000))*1_000_000, 404: 20, 500: 20}
			} else {
				current = map[int32]int64{200: int64(3_000 + random.Intn(3_000)), 404: 20, 500: 20}
			}
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
	detectorConfig.BaselineMinimums = deployanomaly.BaselineMinimums{
		Error5xx: int64(1 + random.Intn(72)), Error4xx: int64(1 + random.Intn(72)),
		Requests: int64(1 + random.Intn(72)), RequestsDrop: int64(1 + random.Intn(72)),
		EgressBytes: int64(1 + random.Intn(72)), CPUSeconds: int64(1 + random.Intn(72)),
	}
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

func TestAnomalyCandidateFilterIncludesInteriorLifetimeThreshold(t *testing.T) {
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
	ids := candidateTestIDs(uid.New("interior"), 0)
	batch, err := conn.PrepareBatch(ctx, `
		INSERT INTO default.frontline_requests_per_5m_v1
			(time, workspace_id, project_id, app_id, environment_id, deployment_id, response_status, count)
	`)
	require.NoError(t, err)
	for bucket := range 100 {
		require.NoError(t, batch.Append(
			windowStart.Add(-time.Duration(100-bucket)*5*time.Minute),
			ids.WorkspaceID, ids.ProjectID, ids.AppID, ids.EnvironmentID, "dep", int32(200), int64(200),
		))
	}
	require.NoError(t, batch.Append(
		windowStart, ids.WorkspaceID, ids.ProjectID, ids.AppID, ids.EnvironmentID, "dep", int32(200), int64(279),
	))
	require.NoError(t, batch.Send())

	detectorConfig := deployanomaly.DefaultConfig(deployanomaly.SensitivityNormal)
	result := deployanomaly.Detect(deployanomaly.Input{
		Metric:                  deployanomaly.MetricRequests,
		Current:                 279,
		BaselineMean:            200,
		ObservedBaselineBuckets: 100,
		BaselineWindowBuckets:   101,
	}, detectorConfig)
	require.Equal(t, deployanomaly.OutcomeAnomaly, result.Outcome)
	require.Less(t, result.ThresholdValue, 279.0)

	filter := candidateTestFilter(detectorConfig)
	rows, err := client.GetRequestAnomalyWindows(ctx, clickhouse.AnomalyWindowsRequest{
		WindowStart: windowStart.UnixMilli(), WorkspaceIDs: []string{ids.WorkspaceID}, CandidateFilter: &filter,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQL candidate filtering must include every possible app lifetime")
	require.Equal(t, ids, requestWindowKey(rows[0]))
}

func TestAnomalyCandidateFilterUsesError5xxBaselineMinimum(t *testing.T) {
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
	ids := candidateTestIDs(uid.New("error-minimum"), 0)
	batch, err := conn.PrepareBatch(ctx, `
		INSERT INTO default.frontline_requests_per_5m_v1
			(time, workspace_id, project_id, app_id, environment_id, deployment_id, response_status, count)
	`)
	require.NoError(t, err)
	for bucket := range 6 {
		timestamp := windowStart.Add(-time.Duration(6-bucket) * 5 * time.Minute)
		require.NoError(t, batch.Append(timestamp, ids.WorkspaceID, ids.ProjectID, ids.AppID, ids.EnvironmentID, "dep", int32(200), int64(99)))
		require.NoError(t, batch.Append(timestamp, ids.WorkspaceID, ids.ProjectID, ids.AppID, ids.EnvironmentID, "dep", int32(500), int64(1)))
	}
	require.NoError(t, batch.Append(windowStart, ids.WorkspaceID, ids.ProjectID, ids.AppID, ids.EnvironmentID, "dep", int32(200), int64(70)))
	require.NoError(t, batch.Append(windowStart, ids.WorkspaceID, ids.ProjectID, ids.AppID, ids.EnvironmentID, "dep", int32(500), int64(30)))
	require.NoError(t, batch.Send())

	detectorConfig := deployanomaly.DefaultConfig(deployanomaly.SensitivityNormal)
	detectorConfig.BaselineMinimums.Error5xx = 6
	result := deployanomaly.Detect(deployanomaly.Input{
		Metric: deployanomaly.MetricError5xx, Current: 30, RequestsInWindow: 100,
		BaselineMean: 0.01, BaselineStddev: 0.01, ObservedBaselineBuckets: 6,
		PreviousCandidate: true,
	}, detectorConfig)
	require.Equal(t, deployanomaly.OutcomeAnomaly, result.Outcome)

	filter := candidateTestFilter(detectorConfig)
	rows, err := client.GetRequestAnomalyWindows(ctx, clickhouse.AnomalyWindowsRequest{
		WindowStart: windowStart.UnixMilli(), WorkspaceIDs: []string{ids.WorkspaceID}, CandidateFilter: &filter,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQL candidate filtering must use the error_5xx baseline minimum")
	require.Equal(t, ids, requestWindowKey(rows[0]))
}

func TestAnomalyCandidateFilterToleratesFloat64Cancellation(t *testing.T) {
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
	ids := candidateTestIDs(uid.New("float64"), 0)
	batch, err := conn.PrepareBatch(ctx, `
		INSERT INTO default.frontline_requests_per_5m_v1
			(time, workspace_id, project_id, app_id, environment_id, deployment_id, response_status, count)
	`)
	require.NoError(t, err)
	for bucket := range 99 {
		requests := int64(10_100_000_000_000)
		if bucket < 34 {
			requests = 10_000_000_000_000
		}
		require.NoError(t, batch.Append(
			windowStart.Add(-time.Duration(99-bucket)*5*time.Minute),
			ids.WorkspaceID, ids.ProjectID, ids.AppID, ids.EnvironmentID, "dep", int32(200), requests,
		))
	}
	const current = int64(13_975_536_123_762)
	require.NoError(t, batch.Append(
		windowStart, ids.WorkspaceID, ids.ProjectID, ids.AppID, ids.EnvironmentID, "dep", int32(200), current,
	))
	require.NoError(t, batch.Send())

	all, err := client.GetRequestAnomalyWindows(ctx, clickhouse.AnomalyWindowsRequest{
		WindowStart: windowStart.UnixMilli(), WorkspaceIDs: []string{ids.WorkspaceID},
	})
	require.NoError(t, err)
	require.Len(t, all, 1)
	detectorConfig := deployanomaly.DefaultConfig(deployanomaly.SensitivityNormal)
	result := deployanomaly.Detect(deployanomaly.Input{
		Metric:                  deployanomaly.MetricRequests,
		Current:                 float64(current),
		BaselineMean:            all[0].RequestsBaselineMean,
		BaselineStddev:          all[0].RequestsBaselineStddev,
		ObservedBaselineBuckets: all[0].BaselineBuckets,
		BaselineWindowBuckets:   100,
	}, detectorConfig)
	require.Equal(t, deployanomaly.OutcomeAnomaly, result.Outcome)
	require.InDelta(t, 13_975_536_123_761.945, result.ThresholdValue, 1)

	filter := candidateTestFilter(detectorConfig)
	rows, err := client.GetRequestAnomalyWindows(ctx, clickhouse.AnomalyWindowsRequest{
		WindowStart: windowStart.UnixMilli(), WorkspaceIDs: []string{ids.WorkspaceID}, CandidateFilter: &filter,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1, "Float64 reconstruction error must not exclude a Go anomaly")
	require.Equal(t, ids, requestWindowKey(rows[0]))
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
		Error5xxRatioStddevFloor: cfg.StddevFloors.Error5xxRatio,
		Error4xxRatioStddevFloor: cfg.StddevFloors.Error4xxRatio,
		RequestsStddevFloor:      cfg.StddevFloors.Requests, EgressBytesStddevFloor: cfg.StddevFloors.EgressBytes,
		CPUSecondsStddevFloor: cfg.StddevFloors.CPUSeconds, ErrorExcessFailures: cfg.ActivityFloors.ErrorExcessFailures,
		RequestsActivity: cfg.ActivityFloors.Requests, EgressBytesActivity: cfg.ActivityFloors.EgressBytes,
		CPUSecondsActivity: cfg.ActivityFloors.CPUSeconds, MemoryUtilizationActivity: cfg.ActivityFloors.MemoryUtilization,
		Error5xxBaselineMinimum: cfg.BaselineMinimums.Error5xx, Error4xxBaselineMinimum: cfg.BaselineMinimums.Error4xx,
		RequestsBaselineMinimum: cfg.BaselineMinimums.Requests, RequestDropBaselineMinimum: cfg.BaselineMinimums.RequestsDrop,
		EgressBytesBaselineMinimum: cfg.BaselineMinimums.EgressBytes, CPUSecondsBaselineMinimum: cfg.BaselineMinimums.CPUSeconds,
		RequestDropFraction: cfg.RequestDrop.RecentLevelFraction, RequestDropActivity: cfg.RequestDrop.ActivityPerBucket,
		RequestDropActiveBuckets: cfg.RequestDrop.MinimumActiveBuckets, RequestDropAbsoluteLoss: cfg.RequestDrop.MinimumAbsoluteLoss,
		Catastrophic5xxRatio: cfg.Catastrophic.Error5xxRatio, Catastrophic5xxFailures: cfg.Catastrophic.Error5xxFailures,
	}
}
