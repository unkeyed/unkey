package seed

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deployanomaly"
)

func TestGenerateAlertSeedDataset(t *testing.T) {
	target := alertSeedTarget{
		workspaceID:   "ws_test",
		projectID:     "proj_test",
		appID:         "app_test",
		environmentID: "env_test",
		deploymentID:  "d_test",
	}
	dataset := generateAlertSeedDataset(target, time.Date(2026, time.September, 3, 12, 34, 0, 0, time.UTC))

	require.Len(t, dataset.alerts, 9)
	require.NotEmpty(t, dataset.requests)
	require.NotEmpty(t, dataset.checkpoints)
	require.Len(t, dataset.events, 7)
	expectedHistoryBuckets := int((7*24*time.Hour)/alertBucketSize) + 1
	require.Len(t, dataset.checkpoints, expectedHistoryBuckets*5)
	requestBuckets := make(map[int64]struct{}, expectedHistoryBuckets)
	for _, request := range dataset.requests {
		bucket := time.UnixMilli(request.Time).Truncate(alertBucketSize).UnixMilli()
		requestBuckets[bucket] = struct{}{}
	}
	require.Len(t, requestBuckets, expectedHistoryBuckets)
	require.LessOrEqual(
		t,
		dataset.requests[0].Time,
		time.Date(2026, time.August, 27, 12, 34, 0, 0, time.UTC).UnixMilli(),
	)

	metrics := make(map[seedAlertMetric]struct{}, len(dataset.alerts))
	resolutionMessages := make(map[string]struct{}, 3)
	open, resolved := 0, 0
	config := deployanomaly.DefaultConfig(deployanomaly.SensitivityNormal)
	for _, alert := range dataset.alerts {
		metrics[alert.metric] = struct{}{}
		switch alert.metric {
		case seedMetricMemoryUtilization:
			require.Zero(t, alert.baselineMean)
			require.Zero(t, alert.baselineStddev)
			require.Zero(t, alert.thresholdSigma)
			require.Greater(t, alert.observedValue, config.ActivityFloors.MemoryUtilization)
		case seedMetricOOMKilled:
			require.Zero(t, alert.baselineMean)
			require.Zero(t, alert.baselineStddev)
			require.Zero(t, alert.thresholdSigma)
			require.Greater(t, alert.observedValue, config.ActivityFloors.OOMKilled)
		case seedMetricCrashLoop:
			require.Zero(t, alert.baselineMean)
			require.Zero(t, alert.baselineStddev)
			require.Zero(t, alert.thresholdSigma)
			require.Greater(t, alert.observedValue, config.ActivityFloors.CrashLoop)
		case seedMetricRequestsDrop:
			require.Zero(t, alert.baselineStddev)
			require.Zero(t, alert.thresholdSigma)
			require.Less(t, alert.observedValue, config.RequestDrop.RecentLevelFraction*alert.baselineMean)
		case seedMetricError5xx:
			require.GreaterOrEqual(t, alert.observedValue, 0.0)
			require.LessOrEqual(t, alert.observedValue, 1.0)
			require.GreaterOrEqual(t, alert.baselineStddev, config.StddevFloors.Error5xxRatio)
			require.Greater(t, alert.observedValue, alert.baselineMean+alert.thresholdSigma*alert.baselineStddev)
		case seedMetricError4xx:
			require.GreaterOrEqual(t, alert.observedValue, 0.0)
			require.LessOrEqual(t, alert.observedValue, 1.0)
			require.GreaterOrEqual(t, alert.baselineStddev, config.StddevFloors.Error4xxRatio)
			require.Greater(t, alert.observedValue, alert.baselineMean+alert.thresholdSigma*alert.baselineStddev)
		case seedMetricRequests:
			require.GreaterOrEqual(t, alert.baselineStddev, config.StddevFloors.Requests)
			require.Greater(t, alert.observedValue, alert.baselineMean+alert.thresholdSigma*alert.baselineStddev)
			require.Greater(t, alert.observedValue/alert.baselineMean, 1.0)
		case seedMetricEgressBytes:
			require.Equal(t, config.StddevFloors.EgressBytes, alert.baselineStddev)
			require.Greater(t, alert.observedValue, alert.baselineMean+alert.thresholdSigma*alert.baselineStddev)
			require.Greater(t, alert.observedValue/alert.baselineMean, 1.0)
		case seedMetricCPUSeconds:
			require.GreaterOrEqual(t, alert.baselineStddev, config.StddevFloors.CPUSeconds)
			require.Greater(t, alert.observedValue, alert.baselineMean+alert.thresholdSigma*alert.baselineStddev)
			require.Greater(t, alert.observedValue/alert.baselineMean, 1.0)
		}
		require.InDelta(t, alert.observedValue, sourceObservedValue(dataset, alert), 0.000001)
		if alert.status == seedAlertStatusOpen {
			open++
			require.False(t, alert.resolvedAt.Valid)
			require.False(t, alert.resolutionMessage.Valid)
		} else {
			resolved++
			require.True(t, alert.resolvedAt.Valid)
			require.NotEmpty(t, alert.resolutionMessage.String)
			require.Contains(t, []string{
				"Metric returned to baseline for 3 consecutive windows",
				"Deployment stopped",
				"Baseline adapted after 24 hours",
			}, alert.resolutionMessage.String)
			resolutionMessages[alert.resolutionMessage.String] = struct{}{}
		}
	}

	require.Len(t, metrics, 9)
	require.Equal(t, 5, open)
	require.Equal(t, 4, resolved)
	require.Len(t, resolutionMessages, 3)
}

func sourceObservedValue(dataset alertSeedDataset, alert alertSeedRow) float64 {
	switch alert.metric {
	case seedMetricError5xx, seedMetricError4xx, seedMetricRequests, seedMetricRequestsDrop:
		var matching, total float64
		for _, request := range dataset.requests {
			if request.Time < alert.windowStart || request.Time >= alert.windowEnd {
				continue
			}
			total++
			if alert.metric == seedMetricRequests || alert.metric == seedMetricRequestsDrop ||
				(alert.metric == seedMetricError5xx && request.ResponseStatus >= 500 && request.ResponseStatus < 600) ||
				(alert.metric == seedMetricError4xx && request.ResponseStatus >= 400 && request.ResponseStatus < 500) {
				matching++
			}
		}
		if alert.metric == seedMetricError5xx || alert.metric == seedMetricError4xx {
			return matching / total
		}
		return matching
	case seedMetricEgressBytes, seedMetricCPUSeconds:
		var minimum, maximum int64
		found := false
		for _, checkpoint := range dataset.checkpoints {
			if checkpoint.Ts < alert.windowStart || checkpoint.Ts >= alert.windowEnd {
				continue
			}
			value := checkpoint.NetworkEgressPublicBytes
			if alert.metric == seedMetricCPUSeconds {
				value = checkpoint.CPUUsageUsec
			}
			if !found || value < minimum {
				minimum = value
			}
			if !found || value > maximum {
				maximum = value
			}
			found = true
		}
		value := float64(maximum - minimum)
		if alert.metric == seedMetricCPUSeconds {
			value /= 1_000_000
		}
		return value
	case seedMetricMemoryUtilization:
		type memoryUsage struct {
			memoryBytes    int64
			allocatedBytes int64
		}
		instances := make(map[string]memoryUsage)
		for _, checkpoint := range dataset.checkpoints {
			if checkpoint.Ts < alert.windowStart || checkpoint.Ts >= alert.windowEnd {
				continue
			}
			usage := instances[checkpoint.ContainerUID]
			usage.memoryBytes = max(usage.memoryBytes, checkpoint.MemoryBytes)
			usage.allocatedBytes = max(usage.allocatedBytes, checkpoint.MemoryAllocatedBytes)
			instances[checkpoint.ContainerUID] = usage
		}
		var average float64
		for _, usage := range instances {
			average += float64(usage.memoryBytes) / float64(usage.allocatedBytes)
		}
		return average / float64(len(instances))
	case seedMetricOOMKilled, seedMetricCrashLoop:
		var count float64
		for _, event := range dataset.events {
			if event.Time < alert.windowStart || event.Time >= alert.windowEnd {
				continue
			}
			if (alert.metric == seedMetricOOMKilled && event.EventKind == "terminated" && event.Reason == "OOMKilled") ||
				(alert.metric == seedMetricCrashLoop && event.EventKind == "waiting" && event.Reason == "CrashLoopBackOff") {
				count++
			}
		}
		return count
	}
	return 0
}
