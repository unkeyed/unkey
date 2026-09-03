package deployanomaly

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectSigma(t *testing.T) {
	defaultConfig := DefaultConfig(SensitivityNormal)

	tests := []struct {
		name          string
		input         Input
		config        Config
		wantOutcome   Outcome
		wantMean      float64
		wantStddev    float64
		wantThreshold float64
		wantReason    string
	}{
		{
			name: "flat baseline and error spike becomes candidate",
			input: Input{
				Metric:                  MetricError5xx,
				Current:                 20,
				RequestsInWindow:        100,
				BaselineMean:            10,
				ObservedBaselineBuckets: 288,
				BaselineWindowBuckets:   288,
			},
			config:        defaultConfig,
			wantOutcome:   OutcomeCandidate,
			wantMean:      10,
			wantStddev:    2,
			wantThreshold: 18,
			wantReason:    "error spike needs a second consecutive window",
		},
		{
			name: "relative guard protects a zero variance baseline",
			input: Input{
				Metric:                  MetricRequests,
				Current:                 1_401,
				BaselineMean:            1_000,
				ObservedBaselineBuckets: 288,
				BaselineWindowBuckets:   288,
			},
			config:        defaultConfig,
			wantOutcome:   OutcomeAnomaly,
			wantMean:      1_000,
			wantStddev:    100,
			wantThreshold: 1_400,
			wantReason:    "current value exceeded sigma threshold",
		},
		{
			name: "value equal to threshold stays quiet",
			input: Input{
				Metric:                  MetricRequests,
				Current:                 1_400,
				BaselineMean:            1_000,
				ObservedBaselineBuckets: 288,
				BaselineWindowBuckets:   288,
			},
			config:        defaultConfig,
			wantOutcome:   OutcomeNone,
			wantMean:      1_000,
			wantStddev:    100,
			wantThreshold: 1_400,
			wantReason:    "current value did not exceed sigma threshold",
		},
		{
			name: "cpu spike below activity floor stays quiet",
			input: Input{
				Metric:                  MetricCPUSeconds,
				Current:                 50,
				ObservedBaselineBuckets: 288,
				BaselineWindowBuckets:   288,
			},
			config:        defaultConfig,
			wantOutcome:   OutcomeNone,
			wantStddev:    1,
			wantThreshold: 4,
			wantReason:    "current activity is below the metric floor",
		},
		{
			name: "error spike below request floor stays quiet",
			input: Input{
				Metric:                  MetricError4xx,
				Current:                 20,
				RequestsInWindow:        99,
				BaselineMean:            10,
				ObservedBaselineBuckets: 288,
				BaselineWindowBuckets:   288,
			},
			config:        defaultConfig,
			wantOutcome:   OutcomeNone,
			wantMean:      10,
			wantStddev:    2,
			wantThreshold: 18,
			wantReason:    "current activity is below the metric floor",
		},
		{
			name: "short history is insufficient",
			input: Input{
				Metric:                  MetricRequests,
				Current:                 1_000,
				ObservedBaselineBuckets: 11,
				BaselineWindowBuckets:   288,
			},
			config:        defaultConfig,
			wantOutcome:   OutcomeInsufficient,
			wantStddev:    20,
			wantThreshold: 80,
			wantReason:    "baseline has fewer than 12 buckets",
		},
		{
			name: "missing buckets are padded with zero",
			input: Input{
				Metric:                  MetricCPUSeconds,
				Current:                 26,
				BaselineMean:            10,
				ObservedBaselineBuckets: 12,
				BaselineWindowBuckets:   24,
			},
			config: Config{
				SigmaK: 4,
			},
			wantOutcome:   OutcomeAnomaly,
			wantMean:      5,
			wantStddev:    5,
			wantThreshold: 25,
			wantReason:    "current value exceeded sigma threshold",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Detect(test.input, test.config)

			require.Equal(t, test.wantOutcome, result.Outcome)
			require.Equal(t, test.input.Current, result.Observed)
			require.InDelta(t, test.wantMean, result.BaselineMean, 1e-9)
			require.InDelta(t, test.wantStddev, result.BaselineStddev, 1e-9)
			require.InDelta(t, test.wantThreshold, result.Threshold, 1e-9)
			require.Equal(t, test.config.SigmaK, result.SigmaK)
			require.Equal(t, test.wantReason, result.Reason)
		})
	}
}

func TestDetectNewAppSteadyTraffic(t *testing.T) {
	cfg := DefaultConfig(SensitivityNormal)
	windowStart := int64(13 * 5 * 60 * 1_000)
	firstBucketTime := windowStart - 12*bucketDurationMillis

	result := Detect(Input{
		Metric:                  MetricRequests,
		Current:                 1_000,
		BaselineMean:            1_000,
		ObservedBaselineBuckets: 12,
		BaselineWindowBuckets:   BaselineWindowBuckets(windowStart, firstBucketTime),
	}, cfg)

	require.Equal(t, OutcomeNone, result.Outcome)
	require.Equal(t, 1_000.0, result.BaselineMean)
	require.Equal(t, 100.0, result.BaselineStddev)
	require.Equal(t, 1_400.0, result.Threshold)
}

func TestBaselineWindowBuckets(t *testing.T) {
	const windowStart = int64(2_000_000_000)

	tests := []struct {
		name            string
		firstBucketTime int64
		want            int64
	}{
		{name: "new app lifetime", firstBucketTime: windowStart - 12*bucketDurationMillis, want: 12},
		{name: "full lookback", firstBucketTime: windowStart - 288*bucketDurationMillis, want: 288},
		{name: "older than lookback", firstBucketTime: windowStart - 400*bucketDurationMillis, want: 288},
		{name: "missing first bucket", firstBucketTime: 0, want: 0},
		{name: "current bucket", firstBucketTime: windowStart, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, BaselineWindowBuckets(windowStart, test.firstBucketTime))
		})
	}
}

func TestDetectConfirmation(t *testing.T) {
	cfg := DefaultConfig(SensitivityNormal)
	base := Input{
		Current:                 20,
		RequestsInWindow:        100,
		BaselineMean:            10,
		ObservedBaselineBuckets: 288,
		BaselineWindowBuckets:   288,
	}

	tests := []struct {
		name              string
		metric            Metric
		previousCandidate bool
		want              Outcome
	}{
		{name: "first 5xx interval", metric: MetricError5xx, want: OutcomeCandidate},
		{name: "confirmed 5xx interval", metric: MetricError5xx, previousCandidate: true, want: OutcomeAnomaly},
		{name: "first 4xx interval", metric: MetricError4xx, want: OutcomeCandidate},
		{name: "confirmed 4xx interval", metric: MetricError4xx, previousCandidate: true, want: OutcomeAnomaly},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := base
			input.Metric = test.metric
			input.PreviousCandidate = test.previousCandidate
			require.Equal(t, test.want, Detect(input, cfg).Outcome)
		})
	}

	usage := Input{
		Metric:                  MetricEgressBytes,
		Current:                 101 << 20,
		ObservedBaselineBuckets: 288,
		BaselineWindowBuckets:   288,
	}
	require.Equal(t, OutcomeAnomaly, Detect(usage, cfg).Outcome)
}

func TestDetectRequestsDrop(t *testing.T) {
	cfg := DefaultConfig(SensitivityNormal)

	tests := []struct {
		name          string
		input         Input
		wantOutcome   Outcome
		wantMean      float64
		wantStddev    float64
		wantThreshold float64
		wantReason    string
	}{
		{
			name: "million requests to zero confirms anomaly",
			input: Input{
				Metric:                  MetricRequestsDrop,
				Current:                 0,
				BaselineMean:            1_000_000,
				ObservedBaselineBuckets: 288,
				BaselineWindowBuckets:   288,
				PreviousCandidate:       true,
			},
			wantOutcome:   OutcomeAnomaly,
			wantMean:      1_000_000,
			wantStddev:    100_000,
			wantThreshold: 600_000,
			wantReason:    "request drop confirmed in consecutive windows",
		},
		{
			name: "baseline below activity floor stays quiet",
			input: Input{
				Metric:                  MetricRequestsDrop,
				Current:                 0,
				BaselineMean:            199,
				ObservedBaselineBuckets: 288,
				BaselineWindowBuckets:   288,
			},
			wantOutcome:   OutcomeNone,
			wantMean:      199,
			wantStddev:    20,
			wantThreshold: 119,
			wantReason:    "baseline activity is below the request drop floor",
		},
		{
			name: "drop within sigma bound stays quiet",
			input: Input{
				Metric:                  MetricRequestsDrop,
				Current:                 700,
				BaselineMean:            1_000,
				BaselineStddev:          100,
				ObservedBaselineBuckets: 288,
				BaselineWindowBuckets:   288,
			},
			wantOutcome:   OutcomeNone,
			wantMean:      1_000,
			wantStddev:    100,
			wantThreshold: 600,
			wantReason:    "current value did not fall below sigma threshold",
		},
		{
			name: "short history is insufficient",
			input: Input{
				Metric:                  MetricRequestsDrop,
				Current:                 0,
				BaselineMean:            1_000_000,
				ObservedBaselineBuckets: 71,
				BaselineWindowBuckets:   71,
			},
			wantOutcome:   OutcomeInsufficient,
			wantMean:      1_000_000,
			wantStddev:    100_000,
			wantThreshold: 600_000,
			wantReason:    "baseline has fewer than 72 buckets",
		},
		{
			name: "first low window becomes candidate",
			input: Input{
				Metric:                  MetricRequestsDrop,
				Current:                 0,
				BaselineMean:            1_000_000,
				ObservedBaselineBuckets: 288,
				BaselineWindowBuckets:   288,
			},
			wantOutcome:   OutcomeCandidate,
			wantMean:      1_000_000,
			wantStddev:    100_000,
			wantThreshold: 600_000,
			wantReason:    "request drop needs a second consecutive window",
		},
		{
			name: "negative lower bound clamps to zero",
			input: Input{
				Metric:                  MetricRequestsDrop,
				Current:                 0,
				BaselineMean:            200,
				BaselineStddev:          100,
				ObservedBaselineBuckets: 288,
				BaselineWindowBuckets:   288,
			},
			wantOutcome:   OutcomeNone,
			wantMean:      200,
			wantStddev:    100,
			wantThreshold: 0,
			wantReason:    "current value did not fall below sigma threshold",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Detect(test.input, cfg)

			require.Equal(t, test.wantOutcome, result.Outcome)
			require.Equal(t, test.input.Current, result.Observed)
			require.InDelta(t, test.wantMean, result.BaselineMean, 1e-9)
			require.InDelta(t, test.wantStddev, result.BaselineStddev, 1e-9)
			require.InDelta(t, test.wantThreshold, result.Threshold, 1e-9)
			require.Equal(t, cfg.SigmaK, result.SigmaK)
			require.Equal(t, test.wantReason, result.Reason)
		})
	}
}

func TestDetectThresholdMetrics(t *testing.T) {
	cfg := DefaultConfig(SensitivityNormal)

	tests := []struct {
		name      string
		metric    Metric
		current   float64
		want      Outcome
		threshold float64
	}{
		{name: "memory at threshold", metric: MetricMemoryUtilization, current: 0.90, want: OutcomeAnomaly, threshold: 0.90},
		{name: "memory below threshold", metric: MetricMemoryUtilization, current: 0.89, want: OutcomeNone, threshold: 0.90},
		{name: "oom kill", metric: MetricOOMKilled, current: 1, want: OutcomeAnomaly, threshold: 1},
		{name: "no oom kill", metric: MetricOOMKilled, current: 0, want: OutcomeNone, threshold: 1},
		{name: "crash loop", metric: MetricCrashLoop, current: 1, want: OutcomeAnomaly, threshold: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Detect(Input{Metric: test.metric, Current: test.current}, cfg)
			require.Equal(t, test.want, result.Outcome)
			require.Equal(t, test.current, result.Observed)
			require.Equal(t, test.threshold, result.Threshold)
			require.Zero(t, result.BaselineMean)
			require.Zero(t, result.BaselineStddev)
			require.Zero(t, result.SigmaK)
		})
	}
}

func TestSensitivitySigmaK(t *testing.T) {
	require.Equal(t, 5.0, SensitivityLow.SigmaK())
	require.Equal(t, 4.0, SensitivityNormal.SigmaK())
	require.Equal(t, 3.0, SensitivityHigh.SigmaK())
	require.Equal(t, 4.0, Sensitivity("unknown").SigmaK())
}

func TestShouldResolve(t *testing.T) {
	require.False(t, ShouldResolve(2))
	require.True(t, ShouldResolve(3))
	require.True(t, ShouldResolve(4))
}

func TestDetectSigmaIsScaleInvariant(t *testing.T) {
	cfg := Config{SigmaK: 4}
	base := Input{
		Metric:                  MetricCPUSeconds,
		Current:                 19,
		BaselineMean:            10,
		BaselineStddev:          2,
		ObservedBaselineBuckets: 288,
		BaselineWindowBuckets:   288,
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

func TestDetectRequestsDropIsScaleInvariant(t *testing.T) {
	cfg := Config{SigmaK: 4}
	base := Input{
		Metric:                  MetricRequestsDrop,
		Current:                 500,
		BaselineMean:            1_000,
		BaselineStddev:          100,
		ObservedBaselineBuckets: 288,
		BaselineWindowBuckets:   288,
		PreviousCandidate:       true,
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
