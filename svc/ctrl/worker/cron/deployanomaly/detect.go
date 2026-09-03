package deployanomaly

import (
	"fmt"
	"math"
)

const (
	bucketDurationMillis   = int64(5 * 60 * 1_000)
	maximumBaselineBuckets = int64(288)
)

// Metric identifies a Deploy anomaly signal. These values are also persisted
// in MySQL and must remain wire-compatible with the alert_events metric enum.
type Metric string

const (
	MetricError5xx          Metric = "error_5xx"
	MetricError4xx          Metric = "error_4xx"
	MetricRequests          Metric = "requests"
	MetricRequestsDrop      Metric = "requests_drop"
	MetricEgressBytes       Metric = "egress_bytes"
	MetricCPUSeconds        Metric = "cpu_seconds"
	MetricMemoryUtilization Metric = "memory_utilization"
	MetricOOMKilled         Metric = "oom_killed"
	MetricCrashLoop         Metric = "crash_loop"
)

// Outcome is the classification of one closed metric window.
type Outcome string

const (
	OutcomeNone         Outcome = "none"
	OutcomeInsufficient Outcome = "insufficient"
	OutcomeCandidate    Outcome = "candidate"
	OutcomeAnomaly      Outcome = "anomaly"
)

// Sensitivity controls how many effective standard deviations a sigma metric
// must exceed. Higher sensitivity uses a lower sigma multiplier.
type Sensitivity string

const (
	SensitivityLow    Sensitivity = "low"
	SensitivityNormal Sensitivity = "normal"
	SensitivityHigh   Sensitivity = "high"
)

// SigmaK returns the sigma multiplier for the sensitivity. An empty or unknown
// value uses normal sensitivity so configuration drift does not increase noise.
func (s Sensitivity) SigmaK() float64 {
	switch s {
	case SensitivityLow:
		return 5
	case SensitivityNormal:
		return 4
	case SensitivityHigh:
		return 3
	default:
		return 4
	}
}

// StddevFloors prevents a flat or near-flat baseline from producing an
// unusably small sigma threshold.
type StddevFloors struct {
	Errors      float64
	Requests    float64
	EgressBytes float64
	CPUSeconds  float64
}

// ActivityFloors suppresses alerts whose absolute activity is too small to be
// actionable. Error metrics must meet both ErrorCount and ErrorRequests.
type ActivityFloors struct {
	ErrorCount    float64
	ErrorRequests float64
	Requests      float64
	// RequestsDropBaseline applies to the padded baseline rather than the
	// current window. Quiet apps commonly have zero requests, so only a drop
	// from sustained activity indicates a likely outage.
	RequestsDropBaseline float64
	EgressBytes          float64
	CPUSeconds           float64
	MemoryUtilization    float64
	OOMKilled            float64
	CrashLoop            float64
}

// BaselineMinimums defines the observed history required before each sigma
// metric can alert. Request drops wait longer because short-lived test traffic
// followed by silence is not evidence of an outage.
type BaselineMinimums struct {
	Error5xx     int64
	Error4xx     int64
	Requests     int64
	RequestsDrop int64
	EgressBytes  int64
	CPUSeconds   int64
}

// Config holds tunable detection thresholds. Start with [DefaultConfig], then
// override fields for workspace-specific alert settings.
type Config struct {
	SigmaK           float64
	StddevFloors     StddevFloors
	ActivityFloors   ActivityFloors
	BaselineMinimums BaselineMinimums
}

// DefaultConfig returns the production defaults for a sensitivity. The
// standard-deviation floors are 2 errors, 20 requests, 1 MiB egress, and 1 CPU
// second. Activity floors are 10 errors with 100 requests, 200 current requests
// for spikes, 200 baseline requests for drops, 100 MiB egress, 60 CPU seconds,
// 90% memory, and one OOM or crash-loop event.
func DefaultConfig(sensitivity Sensitivity) Config {
	return Config{
		SigmaK: sensitivity.SigmaK(),
		StddevFloors: StddevFloors{
			Errors:      2,
			Requests:    20,
			EgressBytes: 1 << 20,
			CPUSeconds:  1,
		},
		ActivityFloors: ActivityFloors{
			ErrorCount:           10,
			ErrorRequests:        100,
			Requests:             200,
			RequestsDropBaseline: 200,
			EgressBytes:          100 << 20,
			CPUSeconds:           60,
			MemoryUtilization:    0.90,
			OOMKilled:            1,
			CrashLoop:            1,
		},
		BaselineMinimums: BaselineMinimums{
			Error5xx:     12,
			Error4xx:     12,
			Requests:     12,
			RequestsDrop: 72,
			EgressBytes:  12,
			CPUSeconds:   12,
		},
	}
}

// BaselineWindowBuckets returns the number of 5-minute buckets in the app's
// observed lifetime inside the 24-hour lookback. Padding only this span avoids
// treating the time before a new app emitted its first bucket as zero traffic.
func BaselineWindowBuckets(windowStart, firstBucketTime int64) int64 {
	if firstBucketTime <= 0 || firstBucketTime >= windowStart {
		return 0
	}
	return min(maximumBaselineBuckets, (windowStart-firstBucketTime)/bucketDurationMillis)
}

// Input describes one closed 5-minute window and its trailing baseline.
// BaselineMean and BaselineStddev cover only ObservedBaselineBuckets, the
// non-empty buckets returned by ClickHouse. BaselineWindowBuckets is the number
// of elapsed buckets represented by the baseline, including missing buckets.
// Detect pads missing buckets with zero for sigma metrics because no aggregate
// row means no traffic. A full trailing 24-hour window has 288 buckets.
type Input struct {
	Metric Metric

	Current          float64
	RequestsInWindow float64

	BaselineMean            float64
	BaselineStddev          float64
	ObservedBaselineBuckets int64
	BaselineWindowBuckets   int64

	PreviousCandidate bool
}

// Result describes how Detect classified a window. BaselineStddev is the
// effective deviation after zero-padding and the configured variance guards.
type Result struct {
	Outcome Outcome

	Observed       float64
	BaselineMean   float64
	BaselineStddev float64
	Threshold      float64
	SigmaK         float64
	Reason         string
}

// Detect classifies one closed metric window. Error spikes and request drops
// need two consecutive anomalous windows. A single empty request window often
// reflects ingest lag through Frontline's buffer and the per-minute and
// per-5-minute materialized views, rather than a workload outage. Other usage
// and threshold metrics fire from one window.
func Detect(input Input, cfg Config) Result {
	switch input.Metric {
	case MetricMemoryUtilization:
		return detectThreshold(input.Current, cfg.ActivityFloors.MemoryUtilization, "memory utilization reached threshold")
	case MetricOOMKilled:
		return detectThreshold(input.Current, cfg.ActivityFloors.OOMKilled, "OOM kill observed")
	case MetricCrashLoop:
		return detectThreshold(input.Current, cfg.ActivityFloors.CrashLoop, "crash loop observed")
	case MetricRequestsDrop:
		return detectRequestsDrop(input, cfg)
	case MetricError5xx, MetricError4xx, MetricRequests, MetricEgressBytes, MetricCPUSeconds:
		return detectSigma(input, cfg)
	default:
		return Result{
			Outcome:        OutcomeNone,
			Observed:       input.Current,
			BaselineMean:   0,
			BaselineStddev: 0,
			Threshold:      0,
			SigmaK:         0,
			Reason:         "unsupported metric",
		}
	}
}

// detectThreshold applies the direct threshold used by memory and instance
// lifecycle signals, which do not need a historical baseline.
func detectThreshold(observed, threshold float64, reason string) Result {
	result := Result{
		Outcome:        OutcomeNone,
		Observed:       observed,
		BaselineMean:   0,
		BaselineStddev: 0,
		Threshold:      threshold,
		SigmaK:         0,
		Reason:         "threshold not reached",
	}
	if observed >= threshold {
		result.Outcome = OutcomeAnomaly
		result.Reason = reason
	}
	return result
}

// detectSigma applies zero-padding, variance guards, activity floors, and the
// strict sigma comparison for count-like metrics.
func detectSigma(input Input, cfg Config) Result {
	mean, stddev := effectiveBaseline(input, cfg)
	threshold := mean + cfg.SigmaK*stddev
	result := Result{
		Outcome:        OutcomeNone,
		Observed:       input.Current,
		BaselineMean:   mean,
		BaselineStddev: stddev,
		Threshold:      threshold,
		SigmaK:         cfg.SigmaK,
		Reason:         "current value did not exceed sigma threshold",
	}

	minimum := minimumBaselineBuckets(input.Metric, cfg.BaselineMinimums)
	if input.ObservedBaselineBuckets < minimum {
		result.Outcome = OutcomeInsufficient
		result.Reason = fmt.Sprintf("baseline has fewer than %d buckets", minimum)
		return result
	}
	if !meetsActivityFloor(input, cfg.ActivityFloors) {
		result.Reason = "current activity is below the metric floor"
		return result
	}
	if input.Current <= threshold {
		return result
	}

	if input.Metric == MetricError5xx || input.Metric == MetricError4xx {
		if !input.PreviousCandidate {
			result.Outcome = OutcomeCandidate
			result.Reason = "error spike needs a second consecutive window"
			return result
		}
		result.Outcome = OutcomeAnomaly
		result.Reason = "error spike confirmed in consecutive windows"
		return result
	}

	result.Outcome = OutcomeAnomaly
	result.Reason = "current value exceeded sigma threshold"
	return result
}

// detectRequestsDrop applies the lower sigma bound and confirms it across two
// windows before alerting. The baseline activity floor prevents normal zero
// traffic on quiet apps from becoming an outage signal.
func detectRequestsDrop(input Input, cfg Config) Result {
	mean, stddev := effectiveBaseline(input, cfg)
	threshold := max(0, mean-cfg.SigmaK*stddev)
	result := Result{
		Outcome:        OutcomeNone,
		Observed:       input.Current,
		BaselineMean:   mean,
		BaselineStddev: stddev,
		Threshold:      threshold,
		SigmaK:         cfg.SigmaK,
		Reason:         "current value did not fall below sigma threshold",
	}

	minimum := minimumBaselineBuckets(input.Metric, cfg.BaselineMinimums)
	if input.ObservedBaselineBuckets < minimum {
		result.Outcome = OutcomeInsufficient
		result.Reason = fmt.Sprintf("baseline has fewer than %d buckets", minimum)
		return result
	}
	if mean < cfg.ActivityFloors.RequestsDropBaseline {
		result.Reason = "baseline activity is below the request drop floor"
		return result
	}
	if input.Current >= threshold {
		return result
	}
	if !input.PreviousCandidate {
		result.Outcome = OutcomeCandidate
		result.Reason = "request drop needs a second consecutive window"
		return result
	}

	result.Outcome = OutcomeAnomaly
	result.Reason = "request drop confirmed in consecutive windows"
	return result
}

// effectiveBaseline returns the zero-padded mean and guarded population
// standard deviation shared by upper and lower sigma rules.
func effectiveBaseline(input Input, cfg Config) (float64, float64) {
	mean, stddev := paddedBaseline(input)
	return mean, max(stddev, mean*0.1, stddevFloor(input.Metric, cfg.StddevFloors))
}

// paddedBaseline reconstructs population moments after adding zero-valued
// missing buckets without requiring ClickHouse to return every bucket to Go.
func paddedBaseline(input Input) (float64, float64) {
	if input.ObservedBaselineBuckets == 0 || input.BaselineWindowBuckets == 0 {
		return 0, 0
	}

	observed := float64(input.ObservedBaselineBuckets)
	window := float64(input.BaselineWindowBuckets)
	mean := input.BaselineMean * observed / window
	secondMoment := (input.BaselineStddev*input.BaselineStddev + input.BaselineMean*input.BaselineMean) * observed / window
	variance := max(0, secondMoment-mean*mean)
	return mean, math.Sqrt(variance)
}

// stddevFloor returns the absolute deviation guard for a sigma metric.
func stddevFloor(metric Metric, floors StddevFloors) float64 {
	switch metric {
	case MetricError5xx, MetricError4xx:
		return floors.Errors
	case MetricRequests, MetricRequestsDrop:
		return floors.Requests
	case MetricEgressBytes:
		return floors.EgressBytes
	case MetricCPUSeconds:
		return floors.CPUSeconds
	case MetricMemoryUtilization, MetricOOMKilled, MetricCrashLoop:
		return 0
	default:
		return 0
	}
}

func minimumBaselineBuckets(metric Metric, minimums BaselineMinimums) int64 {
	switch metric {
	case MetricError5xx:
		return minimums.Error5xx
	case MetricError4xx:
		return minimums.Error4xx
	case MetricRequests:
		return minimums.Requests
	case MetricRequestsDrop:
		return minimums.RequestsDrop
	case MetricEgressBytes:
		return minimums.EgressBytes
	case MetricCPUSeconds:
		return minimums.CPUSeconds
	case MetricMemoryUtilization, MetricOOMKilled, MetricCrashLoop:
		return 0
	default:
		return 0
	}
}

// meetsActivityFloor reports whether the window has enough absolute activity
// to make a sigma crossing actionable.
func meetsActivityFloor(input Input, floors ActivityFloors) bool {
	switch input.Metric {
	case MetricError5xx, MetricError4xx:
		return input.Current >= floors.ErrorCount && input.RequestsInWindow >= floors.ErrorRequests
	case MetricRequests:
		return input.Current >= floors.Requests
	case MetricEgressBytes:
		return input.Current >= floors.EgressBytes
	case MetricCPUSeconds:
		return input.Current >= floors.CPUSeconds
	case MetricRequestsDrop, MetricMemoryUtilization, MetricOOMKilled, MetricCrashLoop:
		return false
	default:
		return false
	}
}

// ShouldResolve reports whether an open anomaly has stayed quiet long enough
// to resolve without flapping on one or two normal windows.
func ShouldResolve(consecutiveQuietWindows int) bool {
	return consecutiveQuietWindows >= 3
}
