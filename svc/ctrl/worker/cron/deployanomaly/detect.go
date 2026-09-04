package deployanomaly

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	bucketDurationMillis   = int64(5 * 60 * 1_000)
	maximumBaselineBuckets = int64(288)
)

//go:embed thresholds.json
var thresholdData []byte

type thresholdFile struct {
	SigmaK                  float64                 `json:"sigmaK"`
	SensitivitySigmaOffsets sensitivitySigmaOffsets `json:"sensitivitySigmaOffsets"`
	MinimumLifetimeBuckets  int64                   `json:"minimumLifetimeBuckets"`
	MinimumStddevRatio      float64                 `json:"minimumStddevRatio"`
	StddevFloors            map[Metric]float64      `json:"stddevFloors"`
	BaselineMinimums        map[Metric]int64        `json:"baselineMinimums"`
	ActivityFloors          ActivityFloors          `json:"activityFloors"`
	RequestDrop             RequestDropRule         `json:"requestDrop"`
	Catastrophic            CatastrophicRules       `json:"catastrophic"`
	Recovery                RecoveryThresholds      `json:"recovery"`
	MaxOpenDurationSeconds  int64                   `json:"maxOpenDurationSeconds"`
}

type sensitivitySigmaOffsets struct {
	Low  float64 `json:"low"`
	High float64 `json:"high"`
}

var productionThresholds = loadThresholds()

func loadThresholds() thresholdFile {
	var thresholds thresholdFile
	if err := json.Unmarshal(thresholdData, &thresholds); err != nil {
		panic(fmt.Sprintf("parse embedded deploy anomaly thresholds: %s", err))
	}
	return thresholds
}

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
		return productionThresholds.SigmaK + productionThresholds.SensitivitySigmaOffsets.Low
	case SensitivityNormal:
		return productionThresholds.SigmaK
	case SensitivityHigh:
		return productionThresholds.SigmaK + productionThresholds.SensitivitySigmaOffsets.High
	default:
		return productionThresholds.SigmaK
	}
}

// StddevFloors prevents a flat or near-flat baseline from producing an
// unusably small sigma threshold.
type StddevFloors struct {
	Error5xxRatio float64
	Error4xxRatio float64
	Requests      float64
	EgressBytes   float64
	CPUSeconds    float64
}

// ActivityFloors suppresses alerts whose absolute activity is too small to be
// actionable.
type ActivityFloors struct {
	ErrorExcessFailures float64 `json:"errorExcessFailures"`
	Requests            float64 `json:"requests"`
	EgressBytes         float64 `json:"egressBytes"`
	CPUSeconds          float64 `json:"cpuSeconds"`
	MemoryUtilization   float64 `json:"memoryUtilization"`
	OOMKilled           float64 `json:"oomKilled"`
	CrashLoop           float64 `json:"crashLoop"`
}

// RequestDropRule configures the robust request-loss detector. Recent activity
// counts the last 12 complete buckets whose traffic met ActivityPerBucket.
type RequestDropRule struct {
	RecentLevelFraction  float64 `json:"recentLevelFraction"`
	ActivityPerBucket    float64 `json:"activityPerBucket"`
	MinimumActiveBuckets int64   `json:"minimumActiveBuckets"`
	MinimumAbsoluteLoss  float64 `json:"minimumAbsoluteLoss"`
}

// CatastrophicRules bypasses interval confirmation for severe failures.
type CatastrophicRules struct {
	Error5xxRatio    float64 `json:"error5xxRatio"`
	Error5xxFailures float64 `json:"error5xxFailures"`
}

// RecoveryThresholds adds hysteresis to open-alert resolution.
type RecoveryThresholds struct {
	MemoryUtilization  float64 `json:"memoryUtilization"`
	SigmaReduction     float64 `json:"sigmaReduction"`
	ConsecutiveWindows int     `json:"consecutiveWindows"`
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
	SigmaK             float64
	MinimumStddevRatio float64
	StddevFloors       StddevFloors
	ActivityFloors     ActivityFloors
	BaselineMinimums   BaselineMinimums
	RequestDrop        RequestDropRule
	Catastrophic       CatastrophicRules
	Recovery           RecoveryThresholds
	// MaxOpenDuration closes a sustained level shift after the rolling baseline
	// has fully adapted, even when the opening snapshot never recovers.
	MaxOpenDuration time.Duration
}

// DefaultConfig returns the production defaults for a sensitivity.
func DefaultConfig(sensitivity Sensitivity) Config {
	return Config{
		SigmaK:             sensitivity.SigmaK(),
		MinimumStddevRatio: productionThresholds.MinimumStddevRatio,
		StddevFloors: StddevFloors{
			Error5xxRatio: productionThresholds.StddevFloors[MetricError5xx],
			Error4xxRatio: productionThresholds.StddevFloors[MetricError4xx],
			Requests:      productionThresholds.StddevFloors[MetricRequests],
			EgressBytes:   productionThresholds.StddevFloors[MetricEgressBytes],
			CPUSeconds:    productionThresholds.StddevFloors[MetricCPUSeconds],
		},
		ActivityFloors: productionThresholds.ActivityFloors,
		BaselineMinimums: BaselineMinimums{
			Error5xx:     productionThresholds.BaselineMinimums[MetricError5xx],
			Error4xx:     productionThresholds.BaselineMinimums[MetricError4xx],
			Requests:     productionThresholds.BaselineMinimums[MetricRequests],
			RequestsDrop: productionThresholds.BaselineMinimums[MetricRequestsDrop],
			EgressBytes:  productionThresholds.BaselineMinimums[MetricEgressBytes],
			CPUSeconds:   productionThresholds.BaselineMinimums[MetricCPUSeconds],
		},
		RequestDrop:     productionThresholds.RequestDrop,
		Catastrophic:    productionThresholds.Catastrophic,
		Recovery:        productionThresholds.Recovery,
		MaxOpenDuration: time.Duration(productionThresholds.MaxOpenDurationSeconds) * time.Second,
	}
}

// BaselineWindowBuckets returns the aligned 5-minute buckets in the app's
// lifetime inside the 24-hour lookback.
func BaselineWindowBuckets(windowStart, appCreatedAt int64) int64 {
	lifetimeStart := AppLifetimeStart(windowStart, appCreatedAt)
	if lifetimeStart >= windowStart {
		return 0
	}
	return (windowStart - lifetimeStart) / bucketDurationMillis
}

// AppLifetimeStart aligns app creation to a detector bucket and caps it at the
// 24-hour baseline boundary.
func AppLifetimeStart(windowStart, appCreatedAt int64) int64 {
	lookbackStart := windowStart - maximumBaselineBuckets*bucketDurationMillis
	alignedCreatedAt := appCreatedAt / bucketDurationMillis * bucketDurationMillis
	return max(lookbackStart, alignedCreatedAt)
}

// RecentRequestStats returns the median and active-bucket count for the 12
// aligned recent request values supplied by ClickHouse.
func RecentRequestStats(requests []float64, activityFloor float64) (float64, int64) {
	if len(requests) == 0 {
		return 0, 0
	}
	ordered := append([]float64(nil), requests...)
	sort.Float64s(ordered)
	middle := len(ordered) / 2
	median := ordered[middle]
	if len(ordered)%2 == 0 {
		median = (ordered[middle-1] + ordered[middle]) / 2
	}
	active := int64(0)
	for _, value := range requests {
		if value >= activityFloor {
			active++
		}
	}
	return median, active
}

// Input describes one closed 5-minute window and its trailing baseline.
// BaselineMean and BaselineStddev cover only ObservedBaselineBuckets, the
// non-empty buckets returned by ClickHouse. BaselineWindowBuckets is the number
// of elapsed buckets represented by the baseline, including missing buckets.
// Detect pads missing buckets with zero for count-like spike metrics because no
// aggregate row means no traffic. A full trailing 24-hour window has 288
// buckets. Error metrics use request-weighted ratios instead.
type Input struct {
	Metric Metric

	Current              float64
	Maximum              float64
	RequestsInWindow     float64
	RecentMedianRequests float64
	RecentActiveBuckets  int64

	BaselineMean            float64
	BaselineStddev          float64
	ObservedBaselineBuckets int64
	BaselineWindowBuckets   int64
	LifetimeStart           int64

	PreviousCandidate bool
}

// Result describes how Detect classified a window. For sigma and error metrics,
// BaselineStddev is the effective deviation after the applicable padding and
// configured variance guards.
type Result struct {
	Outcome Outcome

	Observed       float64
	BaselineMean   float64
	BaselineStddev float64
	ThresholdValue float64
	SigmaK         float64
	RawCount       float64
	Requests       float64
	ExpectedCount  float64
	Catastrophic   bool
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
		return detectThreshold(input.Current, cfg.ActivityFloors.MemoryUtilization, false, "memory utilization reached threshold")
	case MetricOOMKilled:
		return detectThreshold(input.Current, cfg.ActivityFloors.OOMKilled, true, "OOM kill observed")
	case MetricCrashLoop:
		return detectThreshold(input.Current, cfg.ActivityFloors.CrashLoop, true, "crash loop observed")
	case MetricRequestsDrop:
		return detectRequestsDrop(input, cfg)
	case MetricError5xx, MetricError4xx:
		return detectError(input, cfg)
	case MetricRequests, MetricEgressBytes, MetricCPUSeconds:
		return detectSigma(input, cfg)
	default:
		return Result{
			Outcome:        OutcomeNone,
			Observed:       input.Current,
			BaselineMean:   0,
			BaselineStddev: 0,
			ThresholdValue: 0,
			SigmaK:         0,
			RawCount:       0,
			Requests:       0,
			ExpectedCount:  0,
			Catastrophic:   false,
			Reason:         "unsupported metric",
		}
	}
}

func detectThreshold(observed, threshold float64, catastrophic bool, reason string) Result {
	result := Result{
		Outcome:        OutcomeNone,
		Observed:       observed,
		BaselineMean:   0,
		BaselineStddev: 0,
		ThresholdValue: threshold,
		SigmaK:         0,
		RawCount:       0,
		Requests:       0,
		ExpectedCount:  0,
		Catastrophic:   false,
		Reason:         "threshold not reached",
	}
	if observed >= threshold {
		result.Outcome = OutcomeAnomaly
		result.Catastrophic = catastrophic
		result.Reason = reason
	}
	return result
}

func detectSigma(input Input, cfg Config) Result {
	mean, stddev := effectiveBaseline(input, cfg)
	threshold := mean + cfg.SigmaK*stddev
	result := Result{
		Outcome:        OutcomeNone,
		Observed:       input.Current,
		BaselineMean:   mean,
		BaselineStddev: stddev,
		ThresholdValue: threshold,
		SigmaK:         cfg.SigmaK,
		RawCount:       0,
		Requests:       0,
		ExpectedCount:  0,
		Catastrophic:   false,
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

	result.Outcome = OutcomeAnomaly
	result.Reason = "current value exceeded sigma threshold"
	return result
}

func detectError(input Input, cfg Config) Result {
	ratio := 0.0
	if input.RequestsInWindow > 0 {
		ratio = input.Current / input.RequestsInWindow
	}
	stddev := max(input.BaselineStddev, input.BaselineMean*cfg.MinimumStddevRatio, stddevFloor(input.Metric, cfg.StddevFloors))
	threshold := input.BaselineMean + cfg.SigmaK*stddev
	expected := input.BaselineMean * input.RequestsInWindow
	excess := input.Current - expected
	result := Result{
		Outcome:        OutcomeNone,
		Observed:       ratio,
		BaselineMean:   input.BaselineMean,
		BaselineStddev: stddev,
		ThresholdValue: threshold,
		SigmaK:         cfg.SigmaK,
		RawCount:       input.Current,
		Requests:       input.RequestsInWindow,
		ExpectedCount:  expected,
		Catastrophic:   false,
		Reason:         "error ratio did not exceed sigma threshold",
	}

	if input.Metric == MetricError5xx && ratio >= cfg.Catastrophic.Error5xxRatio && input.Current >= cfg.Catastrophic.Error5xxFailures {
		result.Outcome = OutcomeAnomaly
		result.Catastrophic = true
		result.Reason = "catastrophic 5xx error rate"
		return result
	}
	minimum := minimumBaselineBuckets(input.Metric, cfg.BaselineMinimums)
	if input.ObservedBaselineBuckets < minimum {
		result.Outcome = OutcomeInsufficient
		result.Reason = fmt.Sprintf("baseline has fewer than %d buckets", minimum)
		return result
	}
	if ratio <= threshold {
		return result
	}
	if excess < cfg.ActivityFloors.ErrorExcessFailures {
		result.Reason = "excess failures are below the activity floor"
		return result
	}
	if !input.PreviousCandidate {
		result.Outcome = OutcomeCandidate
		result.Reason = "error spike needs a second consecutive window"
		return result
	}

	result.Outcome = OutcomeAnomaly
	result.Reason = "error spike confirmed in consecutive windows"
	return result
}

func detectRequestsDrop(input Input, cfg Config) Result {
	threshold := input.RecentMedianRequests * cfg.RequestDrop.RecentLevelFraction
	loss := input.RecentMedianRequests - input.Current
	result := Result{
		Outcome:        OutcomeNone,
		Observed:       input.Current,
		BaselineMean:   input.RecentMedianRequests,
		BaselineStddev: 0,
		ThresholdValue: threshold,
		SigmaK:         0,
		RawCount:       0,
		Requests:       0,
		ExpectedCount:  0,
		Catastrophic:   false,
		Reason:         "current requests did not fall below the recent-level threshold",
	}

	minimum := minimumBaselineBuckets(input.Metric, cfg.BaselineMinimums)
	if input.ObservedBaselineBuckets < minimum {
		result.Outcome = OutcomeInsufficient
		result.Reason = fmt.Sprintf("baseline has fewer than %d buckets", minimum)
		return result
	}
	if input.RecentActiveBuckets < cfg.RequestDrop.MinimumActiveBuckets {
		result.Reason = "recent traffic is too intermittent for request drop detection"
		return result
	}
	if input.Current >= threshold {
		return result
	}
	if loss < cfg.RequestDrop.MinimumAbsoluteLoss {
		result.Reason = "absolute request loss is below the activity floor"
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

func effectiveBaseline(input Input, cfg Config) (float64, float64) {
	mean, stddev := paddedBaseline(input)
	return mean, max(stddev, mean*cfg.MinimumStddevRatio, stddevFloor(input.Metric, cfg.StddevFloors))
}

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

func stddevFloor(metric Metric, floors StddevFloors) float64 {
	switch metric {
	case MetricError5xx:
		return floors.Error5xxRatio
	case MetricError4xx:
		return floors.Error4xxRatio
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

// Recovered reports whether a complete current window is inside the recovery
// band captured when an alert opened. Rolling baselines are intentionally not
// used because a sustained incident would eventually teach them that the
// regression is normal.
func Recovered(input Input, snapshot Result, cfg Config) bool {
	switch input.Metric {
	case MetricError5xx, MetricError4xx:
		ratio := 0.0
		if input.RequestsInWindow > 0 {
			ratio = input.Current / input.RequestsInWindow
		}
		return ratio <= snapshot.BaselineMean+max(0, snapshot.SigmaK-cfg.Recovery.SigmaReduction)*snapshot.BaselineStddev
	case MetricRequests, MetricEgressBytes, MetricCPUSeconds:
		return input.Current <= snapshot.BaselineMean+max(0, snapshot.SigmaK-cfg.Recovery.SigmaReduction)*snapshot.BaselineStddev
	case MetricRequestsDrop:
		return input.Current >= snapshot.ThresholdValue
	case MetricMemoryUtilization:
		return input.Current < cfg.Recovery.MemoryUtilization
	case MetricOOMKilled, MetricCrashLoop:
		return input.Current < snapshot.ThresholdValue
	default:
		return false
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

func meetsActivityFloor(input Input, floors ActivityFloors) bool {
	switch input.Metric {
	case MetricRequests:
		return input.Current >= floors.Requests
	case MetricEgressBytes:
		return input.Current >= floors.EgressBytes
	case MetricCPUSeconds:
		return input.Current >= floors.CPUSeconds
	case MetricError5xx, MetricError4xx, MetricRequestsDrop, MetricMemoryUtilization, MetricOOMKilled, MetricCrashLoop:
		return false
	default:
		return false
	}
}

// ShouldResolve reports whether an open anomaly has stayed quiet long enough
// to resolve without flapping on one or two normal windows.
func ShouldResolve(consecutiveQuietWindows, minimumConsecutiveWindows int) bool {
	return consecutiveQuietWindows >= minimumConsecutiveWindows
}
