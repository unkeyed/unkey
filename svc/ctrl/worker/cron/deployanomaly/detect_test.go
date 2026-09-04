package deployanomaly

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDetectSigma(t *testing.T) {
	cfg := DefaultConfig(SensitivityNormal)
	tests := []struct {
		name          string
		input         Input
		wantOutcome   Outcome
		wantMean      float64
		wantStddev    float64
		wantThreshold float64
	}{
		{
			name: "flat request baseline spikes",
			input: Input{
				Metric: MetricRequests, Current: 1_401, BaselineMean: 1_000,
				ObservedBaselineBuckets: 288, BaselineWindowBuckets: 288,
			},
			wantOutcome: OutcomeAnomaly, wantMean: 1_000, wantStddev: 100, wantThreshold: 1_400,
		},
		{
			name: "value equal to threshold stays quiet",
			input: Input{
				Metric: MetricRequests, Current: 1_400, BaselineMean: 1_000,
				ObservedBaselineBuckets: 288, BaselineWindowBuckets: 288,
			},
			wantOutcome: OutcomeNone, wantMean: 1_000, wantStddev: 100, wantThreshold: 1_400,
		},
		{
			name: "cpu spike below activity floor stays quiet",
			input: Input{
				Metric: MetricCPUSeconds, Current: 50,
				ObservedBaselineBuckets: 288, BaselineWindowBuckets: 288,
			},
			wantOutcome: OutcomeNone, wantStddev: 1, wantThreshold: 4,
		},
		{
			name: "short history is insufficient",
			input: Input{
				Metric: MetricRequests, Current: 1_000,
				ObservedBaselineBuckets: 11, BaselineWindowBuckets: 288,
			},
			wantOutcome: OutcomeInsufficient, wantStddev: 20, wantThreshold: 80,
		},
		{
			name: "missing buckets are padded with zero",
			input: Input{
				Metric: MetricCPUSeconds, Current: 61, BaselineMean: 10,
				ObservedBaselineBuckets: 12, BaselineWindowBuckets: 24,
			},
			wantOutcome: OutcomeAnomaly, wantMean: 5, wantStddev: 5, wantThreshold: 25,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Detect(test.input, cfg)
			require.Equal(t, test.wantOutcome, result.Outcome)
			require.InDelta(t, test.wantMean, result.BaselineMean, 1e-9)
			require.InDelta(t, test.wantStddev, result.BaselineStddev, 1e-9)
			require.InDelta(t, test.wantThreshold, result.ThresholdValue, 1e-9)
		})
	}
}

func TestDetectErrors(t *testing.T) {
	cfg := DefaultConfig(SensitivityNormal)
	tests := []struct {
		name             string
		input            Input
		wantOutcome      Outcome
		wantObserved     float64
		wantThreshold    float64
		wantCatastrophic bool
	}{
		{
			name: "traffic growth without rate growth stays quiet",
			input: Input{
				Metric: MetricError5xx, Current: 100, RequestsInWindow: 10_000,
				BaselineMean: 0.01, BaselineStddev: 0.001, ObservedBaselineBuckets: 288,
			},
			wantOutcome: OutcomeNone, wantObserved: 0.01, wantThreshold: 0.05,
		},
		{
			name: "moderate 5xx spike becomes candidate",
			input: Input{
				Metric: MetricError5xx, Current: 40, RequestsInWindow: 100,
				BaselineMean: 0.10, ObservedBaselineBuckets: 288,
			},
			wantOutcome: OutcomeCandidate, wantObserved: 0.40, wantThreshold: 0.14,
		},
		{
			name: "moderate 5xx spike confirms",
			input: Input{
				Metric: MetricError5xx, Current: 40, RequestsInWindow: 100,
				BaselineMean: 0.10, ObservedBaselineBuckets: 288, PreviousCandidate: true,
			},
			wantOutcome: OutcomeAnomaly, wantObserved: 0.40, wantThreshold: 0.14,
		},
		{
			name: "99 of 99 failures are catastrophic",
			input: Input{
				Metric: MetricError5xx, Current: 99, RequestsInWindow: 99,
				BaselineMean: 0.01, ObservedBaselineBuckets: 288,
			},
			wantOutcome: OutcomeAnomaly, wantObserved: 1, wantThreshold: 0.05, wantCatastrophic: true,
		},
		{
			name: "catastrophic 5xx bypasses history and sigma",
			input: Input{
				Metric: MetricError5xx, Current: 50, RequestsInWindow: 100,
				BaselineMean: 0.90, ObservedBaselineBuckets: 0,
			},
			wantOutcome: OutcomeAnomaly, wantObserved: 0.50, wantThreshold: 1.26, wantCatastrophic: true,
		},
		{
			name: "4xx anomaly is detected",
			input: Input{
				Metric: MetricError4xx, Current: 40, RequestsInWindow: 100,
				BaselineMean: 0.10, ObservedBaselineBuckets: 288, PreviousCandidate: true,
			},
			wantOutcome: OutcomeAnomaly, wantObserved: 0.40, wantThreshold: 0.14,
		},
		{
			name: "small excess failure count stays quiet",
			input: Input{
				Metric: MetricError5xx, Current: 10, RequestsInWindow: 20,
				BaselineMean: 0.10, ObservedBaselineBuckets: 288,
			},
			wantOutcome: OutcomeNone, wantObserved: 0.50, wantThreshold: 0.14,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Detect(test.input, cfg)
			require.Equal(t, test.wantOutcome, result.Outcome)
			require.InDelta(t, test.wantObserved, result.Observed, 1e-9)
			require.InDelta(t, test.wantThreshold, result.ThresholdValue, 1e-9)
			require.Equal(t, test.wantCatastrophic, result.Catastrophic)
			require.Equal(t, test.input.Current, result.RawCount)
			require.Equal(t, test.input.RequestsInWindow, result.Requests)
		})
	}
}

func TestDetectRequestsDrop(t *testing.T) {
	cfg := DefaultConfig(SensitivityNormal)
	tests := []struct {
		name          string
		input         Input
		wantOutcome   Outcome
		wantThreshold float64
	}{
		{
			name: "500 requests to zero confirms anomaly",
			input: Input{
				Metric: MetricRequestsDrop, Current: 0, RecentMedianRequests: 500, RecentActiveBuckets: 12,
				ObservedBaselineBuckets: 288, PreviousCandidate: true,
			},
			wantOutcome: OutcomeAnomaly, wantThreshold: 125,
		},
		{
			name: "first low window becomes candidate",
			input: Input{
				Metric: MetricRequestsDrop, Current: 0, RecentMedianRequests: 500, RecentActiveBuckets: 12,
				ObservedBaselineBuckets: 288,
			},
			wantOutcome: OutcomeCandidate, wantThreshold: 125,
		},
		{
			name: "bursty alternating traffic stays quiet",
			input: Input{
				Metric: MetricRequestsDrop, Current: 0, RecentMedianRequests: 500, RecentActiveBuckets: 6,
				ObservedBaselineBuckets: 288,
			},
			wantOutcome: OutcomeNone, wantThreshold: 125,
		},
		{
			name: "drop above quarter level stays quiet",
			input: Input{
				Metric: MetricRequestsDrop, Current: 130, RecentMedianRequests: 500, RecentActiveBuckets: 12,
				ObservedBaselineBuckets: 288,
			},
			wantOutcome: OutcomeNone, wantThreshold: 125,
		},
		{
			name: "absolute loss below floor stays quiet",
			input: Input{
				Metric: MetricRequestsDrop, Current: 50, RecentMedianRequests: 240, RecentActiveBuckets: 12,
				ObservedBaselineBuckets: 288,
			},
			wantOutcome: OutcomeNone, wantThreshold: 60,
		},
		{
			name: "less than six hours history is insufficient",
			input: Input{
				Metric: MetricRequestsDrop, Current: 0, RecentMedianRequests: 500, RecentActiveBuckets: 12,
				ObservedBaselineBuckets: 71,
			},
			wantOutcome: OutcomeInsufficient, wantThreshold: 125,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Detect(test.input, cfg)
			require.Equal(t, test.wantOutcome, result.Outcome)
			require.Equal(t, test.input.Current, result.Observed)
			require.Equal(t, test.input.RecentMedianRequests, result.BaselineMean)
			require.InDelta(t, test.wantThreshold, result.ThresholdValue, 1e-9)
			require.Zero(t, result.SigmaK)
		})
	}
}

func TestRecentRequestStats(t *testing.T) {
	median, active := RecentRequestStats([]float64{0, 1_000, 0, 1_000, 0, 1_000, 0, 1_000, 0, 1_000, 0, 1_000}, 200)
	require.Equal(t, 500.0, median)
	require.Equal(t, int64(6), active)
}

func TestDetectThresholdMetrics(t *testing.T) {
	cfg := DefaultConfig(SensitivityNormal)
	tests := []struct {
		name             string
		metric           Metric
		current          float64
		want             Outcome
		threshold        float64
		wantCatastrophic bool
	}{
		{name: "memory at threshold", metric: MetricMemoryUtilization, current: 0.90, want: OutcomeAnomaly, threshold: 0.90},
		{name: "memory below threshold", metric: MetricMemoryUtilization, current: 0.89, want: OutcomeNone, threshold: 0.90},
		{name: "oom kill", metric: MetricOOMKilled, current: 1, want: OutcomeAnomaly, threshold: 1, wantCatastrophic: true},
		{name: "crash loop", metric: MetricCrashLoop, current: 1, want: OutcomeAnomaly, threshold: 1, wantCatastrophic: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Detect(Input{Metric: test.metric, Current: test.current}, cfg)
			require.Equal(t, test.want, result.Outcome)
			require.Equal(t, test.threshold, result.ThresholdValue)
			require.Equal(t, test.wantCatastrophic, result.Catastrophic)
		})
	}
}

func TestRecoveredUsesOpeningSnapshot(t *testing.T) {
	cfg := DefaultConfig(SensitivityNormal)
	snapshot := Result{BaselineMean: 10, BaselineStddev: 2, ThresholdValue: 18, SigmaK: 4}

	regression := Input{
		Metric: MetricCPUSeconds, Current: 20,
		BaselineMean: 100, BaselineStddev: 1, ObservedBaselineBuckets: 288, BaselineWindowBuckets: 288,
	}
	require.False(t, Recovered(regression, snapshot, cfg), "inflated rolling baseline must not resolve")

	recovered := regression
	recovered.Current = 16
	require.True(t, Recovered(recovered, snapshot, cfg))

	require.True(t, Recovered(Input{Metric: MetricMemoryUtilization, Current: 0.84}, snapshot, cfg))
	require.False(t, Recovered(Input{Metric: MetricMemoryUtilization, Current: 0.85}, snapshot, cfg))
	require.True(t, Recovered(Input{Metric: MetricRequestsDrop, Current: 125}, Result{ThresholdValue: 125}, cfg))
}

func TestCatastrophicError5xxDoesNotRecoverInsideNoisyBand(t *testing.T) {
	cfg := DefaultConfig(SensitivityNormal)
	input := Input{
		Metric: MetricError5xx, Current: 600, RequestsInWindow: 1_000,
		BaselineMean: 0.2, BaselineStddev: 0.15, ObservedBaselineBuckets: 288,
	}

	snapshot := Detect(input, cfg)
	require.Equal(t, OutcomeAnomaly, snapshot.Outcome)
	require.True(t, snapshot.Catastrophic)
	require.False(t, Recovered(input, snapshot, cfg))

	input.Current = 490
	require.True(t, Recovered(input, snapshot, cfg))
}

func TestSustainedLevelShiftExpiresAfterMaxOpenDuration(t *testing.T) {
	cfg := DefaultConfig(SensitivityNormal)
	firedAt := int64(5 * time.Minute / time.Millisecond)
	snapshot := Result{BaselineMean: 100, BaselineStddev: 10, ThresholdValue: 140, SigmaK: 4}
	sustained := Input{Metric: MetricRequests, Current: 200}
	require.False(t, Recovered(sustained, snapshot, cfg))

	lastWindowBeforeExpiry := firedAt + int64((cfg.MaxOpenDuration-5*time.Minute)/time.Millisecond)
	require.False(t, maxOpenDurationReached(firedAt, lastWindowBeforeExpiry, cfg.MaxOpenDuration))
	require.True(t, maxOpenDurationReached(firedAt, firedAt+int64(cfg.MaxOpenDuration/time.Millisecond), cfg.MaxOpenDuration))
}

func TestDetectNewAppSteadyTraffic(t *testing.T) {
	cfg := DefaultConfig(SensitivityNormal)
	windowStart := int64(13 * 5 * 60 * 1_000)
	appCreatedAt := windowStart - 12*bucketDurationMillis
	result := Detect(Input{
		Metric: MetricRequests, Current: 1_000, BaselineMean: 1_000,
		ObservedBaselineBuckets: 12,
		BaselineWindowBuckets:   BaselineWindowBuckets(windowStart, appCreatedAt),
	}, cfg)

	require.Equal(t, OutcomeNone, result.Outcome)
	require.Equal(t, 1_000.0, result.BaselineMean)
	require.Equal(t, 1_400.0, result.ThresholdValue)
}

func TestBaselineWindowBuckets(t *testing.T) {
	const windowStart = int64(2_000_000_000)
	tests := []struct {
		name         string
		appCreatedAt int64
		want         int64
	}{
		{name: "new app lifetime aligns creation", appCreatedAt: windowStart - 12*bucketDurationMillis + time.Minute.Milliseconds(), want: 12},
		{name: "full lookback", appCreatedAt: windowStart - 288*bucketDurationMillis, want: 288},
		{name: "old sparse app gets full lifetime", appCreatedAt: windowStart - 400*bucketDurationMillis, want: 288},
		{name: "current bucket", appCreatedAt: windowStart, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, BaselineWindowBuckets(windowStart, test.appCreatedAt))
		})
	}
}

func TestSensitivitySigmaK(t *testing.T) {
	require.Equal(t, 5.0, SensitivityLow.SigmaK())
	require.Equal(t, 4.0, SensitivityNormal.SigmaK())
	require.Equal(t, 3.0, SensitivityHigh.SigmaK())
	require.Equal(t, 4.0, Sensitivity("unknown").SigmaK())
}

func TestDefaultConfigMatchesSharedThresholds(t *testing.T) {
	cfg := DefaultConfig(SensitivityNormal)
	require.Equal(t, productionThresholds.SigmaK, cfg.SigmaK)
	require.Equal(t, productionThresholds.MinimumStddevRatio, cfg.MinimumStddevRatio)
	require.Equal(t, productionThresholds.StddevFloors[MetricError5xx], cfg.StddevFloors.Error5xxRatio)
	require.Equal(t, productionThresholds.StddevFloors[MetricError4xx], cfg.StddevFloors.Error4xxRatio)
	require.Equal(t, productionThresholds.StddevFloors[MetricRequests], cfg.StddevFloors.Requests)
	require.Equal(t, productionThresholds.StddevFloors[MetricEgressBytes], cfg.StddevFloors.EgressBytes)
	require.Equal(t, productionThresholds.StddevFloors[MetricCPUSeconds], cfg.StddevFloors.CPUSeconds)
	require.Equal(t, productionThresholds.ActivityFloors, cfg.ActivityFloors)
	require.Equal(t, productionThresholds.RequestDrop, cfg.RequestDrop)
	require.Equal(t, productionThresholds.Catastrophic, cfg.Catastrophic)
	require.Equal(t, productionThresholds.Recovery, cfg.Recovery)
	require.Equal(t, time.Duration(productionThresholds.MaxOpenDurationSeconds)*time.Second, cfg.MaxOpenDuration)
	require.False(t, ShouldResolve(cfg.Recovery.ConsecutiveWindows-1, cfg.Recovery.ConsecutiveWindows))
	require.True(t, ShouldResolve(cfg.Recovery.ConsecutiveWindows, cfg.Recovery.ConsecutiveWindows))
}

func TestParseThresholdsRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		old     string
		replace string
		wantErr string
	}{
		{name: "unknown field", old: `"sigmaK": 4`, replace: `"sigmaK": 4, "unknown": 1`, wantErr: "unknown field"},
		{name: "zero sigma", old: `"sigmaK": 4`, replace: `"sigmaK": 0`, wantErr: "sigmaK must be"},
		{name: "invalid sensitivity offset", old: `"low": 1`, replace: `"low": -1`, wantErr: "sensitivitySigmaOffsets.low"},
		{
			name: "overflowed sensitivity sigma",
			old: `"sigmaK": 4,
  "sensitivitySigmaOffsets": {
    "low": 1`,
			replace: `"sigmaK": 9e307,
  "sensitivitySigmaOffsets": {
    "low": 9e307`,
			wantErr: "sigmaK plus sensitivitySigmaOffsets.low",
		},
		{name: "ratio above one", old: `"minimumStddevRatio": 0.1`, replace: `"minimumStddevRatio": 1.1`, wantErr: "minimumStddevRatio"},
		{name: "missing stddev floor", old: ",\n    \"cpu_seconds\": 1", replace: "", wantErr: "stddevFloors must contain"},
		{name: "zero baseline minimum", old: `"requests_drop": 72`, replace: `"requests_drop": 0`, wantErr: "baselineMinimums.requests_drop"},
		{name: "zero activity floor", old: `"cpuSeconds": 60`, replace: `"cpuSeconds": 0`, wantErr: "activityFloors.cpuSeconds"},
		{name: "invalid drop ratio", old: `"recentLevelFraction": 0.25`, replace: `"recentLevelFraction": 0`, wantErr: "requestDrop.recentLevelFraction"},
		{name: "recovery memory reaches activity floor", old: `"memoryUtilization": 0.85`, replace: `"memoryUtilization": 0.9`, wantErr: "recovery.memoryUtilization"},
		{name: "recovery reduction reaches sensitivity sigma", old: `"sigmaReduction": 1`, replace: `"sigmaReduction": 3`, wantErr: "recovery.sigmaReduction"},
		{name: "invalid recovery windows", old: `"consecutiveWindows": 3`, replace: `"consecutiveWindows": 0`, wantErr: "recovery.consecutiveWindows"},
		{name: "invalid max duration", old: `"maxOpenDurationSeconds": 86400`, replace: `"maxOpenDurationSeconds": 0`, wantErr: "maxOpenDurationSeconds"},
		{name: "max duration exceeds policy", old: `"maxOpenDurationSeconds": 86400`, replace: `"maxOpenDurationSeconds": 2592001`, wantErr: "must not exceed 30 days"},
		{name: "max duration overflows time duration", old: `"maxOpenDurationSeconds": 86400`, replace: `"maxOpenDurationSeconds": 9223372037`, wantErr: "must fit time.Duration"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := strings.Replace(string(thresholdData), test.old, test.replace, 1)
			require.NotEqual(t, string(thresholdData), data, "test mutation must match embedded JSON")
			_, err := parseThresholds([]byte(data))
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestParseThresholdsRejectsTrailingJSONValue(t *testing.T) {
	_, err := parseThresholds(append(append([]byte(nil), thresholdData...), []byte("\n{}")...))
	require.ErrorContains(t, err, "multiple JSON values")
}

func TestParseThresholdsAcceptsEmbeddedConfiguration(t *testing.T) {
	_, err := parseThresholds(thresholdData)
	require.NoError(t, err)
}

func TestDetectSigmaIsScaleInvariant(t *testing.T) {
	cfg := Config{SigmaK: 4}
	base := Input{
		Metric: MetricCPUSeconds, Current: 19, BaselineMean: 10, BaselineStddev: 2,
		ObservedBaselineBuckets: 288, BaselineWindowBuckets: 288,
	}
	want := Detect(base, cfg).Outcome
	require.Equal(t, OutcomeAnomaly, want)

	for _, scale := range []float64{0.25, 2, 10, 1_000} {
		scaled := base
		scaled.Current *= scale
		scaled.BaselineMean *= scale
		scaled.BaselineStddev *= scale
		require.Equal(t, want, Detect(scaled, cfg).Outcome, "scale %v", scale)
	}
}
