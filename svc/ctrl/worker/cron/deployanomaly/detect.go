package deployanomaly

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"time"
)

const (
	bucketDurationMillis   = int64(5 * 60 * 1_000)
	maximumBaselineBuckets = int64(288)
	maximumOpenDuration    = 30 * 24 * time.Hour
)

//go:embed thresholds.json
var thresholdData []byte

type thresholdFile struct {
	SigmaK                  float64                 `json:"sigmaK"`
	SensitivitySigmaOffsets sensitivitySigmaOffsets `json:"sensitivitySigmaOffsets"`
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
	thresholds, err := parseThresholds(thresholdData)
	if err != nil {
		panic(fmt.Sprintf("invalid embedded deploy anomaly thresholds: %s", err))
	}
	return thresholds
}

func parseThresholds(data []byte) (thresholdFile, error) {
	var thresholds thresholdFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&thresholds); err != nil {
		return thresholdFile{}, fmt.Errorf("decode thresholds: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return thresholdFile{}, fmt.Errorf("decode thresholds: multiple JSON values")
		}
		return thresholdFile{}, fmt.Errorf("decode trailing thresholds data: %w", err)
	}
	if err := validateThresholds(thresholds); err != nil {
		return thresholdFile{}, err
	}
	return thresholds, nil
}

func validateThresholds(thresholds thresholdFile) error {
	if err := positiveThreshold("sigmaK", thresholds.SigmaK); err != nil {
		return err
	}
	if err := positiveThreshold("sensitivitySigmaOffsets.low", thresholds.SensitivitySigmaOffsets.Low); err != nil {
		return err
	}
	if !finite(thresholds.SensitivitySigmaOffsets.High) || thresholds.SensitivitySigmaOffsets.High >= 0 {
		return fmt.Errorf("sensitivitySigmaOffsets.high must be finite and negative")
	}
	lowSigma := thresholds.SigmaK + thresholds.SensitivitySigmaOffsets.Low
	highSigma := thresholds.SigmaK + thresholds.SensitivitySigmaOffsets.High
	if !finite(lowSigma) || lowSigma <= 0 {
		return fmt.Errorf("sigmaK plus sensitivitySigmaOffsets.low must be finite and positive")
	}
	if !finite(highSigma) || highSigma <= 0 {
		return fmt.Errorf("sigmaK plus sensitivitySigmaOffsets.high must be finite and positive")
	}
	if err := ratioThreshold("minimumStddevRatio", thresholds.MinimumStddevRatio); err != nil {
		return err
	}

	stddevMetrics := []Metric{MetricError5xx, MetricError4xx, MetricRequests, MetricEgressBytes, MetricCPUSeconds}
	if len(thresholds.StddevFloors) != len(stddevMetrics) {
		return fmt.Errorf("stddevFloors must contain exactly %d metrics", len(stddevMetrics))
	}
	for _, metric := range stddevMetrics {
		if err := positiveThreshold("stddevFloors."+string(metric), thresholds.StddevFloors[metric]); err != nil {
			return err
		}
	}

	baselineMetrics := append(append([]Metric(nil), stddevMetrics...), MetricRequestsDrop)
	if len(thresholds.BaselineMinimums) != len(baselineMetrics) {
		return fmt.Errorf("baselineMinimums must contain exactly %d metrics", len(baselineMetrics))
	}
	for _, metric := range baselineMetrics {
		if thresholds.BaselineMinimums[metric] <= 0 || thresholds.BaselineMinimums[metric] > maximumBaselineBuckets {
			return fmt.Errorf("baselineMinimums.%s must be between 1 and %d", metric, maximumBaselineBuckets)
		}
	}

	positiveValues := []struct {
		name  string
		value float64
	}{
		{name: "activityFloors.errorExcessFailures", value: thresholds.ActivityFloors.ErrorExcessFailures},
		{name: "activityFloors.requests", value: thresholds.ActivityFloors.Requests},
		{name: "activityFloors.egressBytes", value: thresholds.ActivityFloors.EgressBytes},
		{name: "activityFloors.cpuSeconds", value: thresholds.ActivityFloors.CPUSeconds},
		{name: "activityFloors.oomKilled", value: thresholds.ActivityFloors.OOMKilled},
		{name: "activityFloors.crashLoop", value: thresholds.ActivityFloors.CrashLoop},
		{name: "requestDrop.activityPerBucket", value: thresholds.RequestDrop.ActivityPerBucket},
		{name: "requestDrop.minimumAbsoluteLoss", value: thresholds.RequestDrop.MinimumAbsoluteLoss},
		{name: "catastrophic.error5xxFailures", value: thresholds.Catastrophic.Error5xxFailures},
		{name: "recovery.sigmaReduction", value: thresholds.Recovery.SigmaReduction},
	}
	for _, item := range positiveValues {
		if err := positiveThreshold(item.name, item.value); err != nil {
			return err
		}
	}

	ratioValues := []struct {
		name  string
		value float64
	}{
		{name: "activityFloors.memoryUtilization", value: thresholds.ActivityFloors.MemoryUtilization},
		{name: "requestDrop.recentLevelFraction", value: thresholds.RequestDrop.RecentLevelFraction},
		{name: "catastrophic.error5xxRatio", value: thresholds.Catastrophic.Error5xxRatio},
		{name: "recovery.memoryUtilization", value: thresholds.Recovery.MemoryUtilization},
	}
	for _, item := range ratioValues {
		if err := ratioThreshold(item.name, item.value); err != nil {
			return err
		}
	}
	if thresholds.Recovery.MemoryUtilization >= thresholds.ActivityFloors.MemoryUtilization {
		return fmt.Errorf("recovery.memoryUtilization must be less than activityFloors.memoryUtilization")
	}

	if thresholds.RequestDrop.MinimumActiveBuckets <= 0 || thresholds.RequestDrop.MinimumActiveBuckets > 12 {
		return fmt.Errorf("requestDrop.minimumActiveBuckets must be between 1 and 12")
	}
	if thresholds.Recovery.SigmaReduction >= min(lowSigma, highSigma) {
		return fmt.Errorf("recovery.sigmaReduction must be less than the smallest sensitivity sigma")
	}
	if thresholds.Recovery.ConsecutiveWindows <= 0 {
		return fmt.Errorf("recovery.consecutiveWindows must be positive")
	}
	if thresholds.MaxOpenDurationSeconds <= 0 {
		return fmt.Errorf("maxOpenDurationSeconds must be positive")
	}
	if thresholds.MaxOpenDurationSeconds > math.MaxInt64/int64(time.Second) {
		return fmt.Errorf("maxOpenDurationSeconds must fit time.Duration")
	}
	if time.Duration(thresholds.MaxOpenDurationSeconds)*time.Second > maximumOpenDuration {
		return fmt.Errorf("maxOpenDurationSeconds must not exceed 30 days")
	}
	return nil
}

func positiveThreshold(name string, value float64) error {
	if !finite(value) || value <= 0 {
		return fmt.Errorf("%s must be finite and positive", name)
	}
	return nil
}

func ratioThreshold(name string, value float64) error {
	if !finite(value) || value <= 0 || value > 1 {
		return fmt.Errorf("%s must be finite, greater than 0, and at most 1", name)
	}
	return nil
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
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

	if catastrophicError5xx(input, ratio, cfg.Catastrophic) {
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

func catastrophicError5xx(input Input, ratio float64, rules CatastrophicRules) bool {
	return input.Metric == MetricError5xx &&
		ratio >= rules.Error5xxRatio &&
		input.Current >= rules.Error5xxFailures
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
		if catastrophicError5xx(input, ratio, cfg.Catastrophic) {
			return false
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
