package observability

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// BuildMetrics provides instrumentation for build operations
type BuildMetrics struct {
	// Counters
	buildsTotal        metric.Int64Counter
	buildErrorsTotal   metric.Int64Counter
	buildCancellations metric.Int64Counter

	// Histograms
	buildDuration    metric.Float64Histogram
	pullDuration     metric.Float64Histogram
	extractDuration  metric.Float64Histogram
	optimizeDuration metric.Float64Histogram

	// Gauges
	activeBuilds metric.Int64UpDownCounter
	queuedBuilds metric.Int64UpDownCounter

	// Size metrics
	imageSizeBytes   metric.Int64Histogram
	rootfsSizeBytes  metric.Int64Histogram
	compressionRatio metric.Float64Histogram

	// Resource usage
	buildMemoryUsage metric.Int64Histogram
	buildDiskUsage   metric.Int64Histogram
	buildCPUUsage    metric.Float64Histogram

	// Build step counters
	buildStepsTotal   metric.Int64Counter
	buildStepErrors   metric.Int64Counter
	buildStepDuration metric.Float64Histogram

	// Base asset initialization metrics
	baseAssetInitRetries  metric.Int64Counter
	baseAssetInitFailures metric.Int64Counter

	highCardinalityEnabled bool
	logger                 *slog.Logger
}

// NewBuildMetrics creates a new BuildMetrics instance
func NewBuildMetrics(logger *slog.Logger, highCardinalityEnabled bool) (*BuildMetrics, error) {
	meter := otel.Meter("builderd")

	metrics := &BuildMetrics{ //nolint:exhaustruct // Metric fields are initialized individually below after error checking
		highCardinalityEnabled: highCardinalityEnabled,
		logger:                 logger,
	}

	var err error

	// Build counters
	metrics.buildsTotal, err = meter.Int64Counter(
		"builderd_builds_total",
		metric.WithDescription("Total number of builds started"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	metrics.buildErrorsTotal, err = meter.Int64Counter(
		"builderd_build_errors_total",
		metric.WithDescription("Total number of build failures"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	metrics.buildCancellations, err = meter.Int64Counter(
		"builderd_build_cancellations_total",
		metric.WithDescription("Total number of build cancellations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Duration histograms
	metrics.buildDuration, err = meter.Float64Histogram(
		"builderd_build_duration_seconds",
		metric.WithDescription("Time taken to complete builds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			1, 5, 10, 30, 60, 120, 300, 600, 900, 1800, 3600,
		),
	)
	if err != nil {
		return nil, err
	}

	metrics.pullDuration, err = meter.Float64Histogram(
		"builderd_pull_duration_seconds",
		metric.WithDescription("Time taken to pull images/sources"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			1, 5, 10, 30, 60, 120, 300, 600,
		),
	)
	if err != nil {
		return nil, err
	}

	metrics.extractDuration, err = meter.Float64Histogram(
		"builderd_extract_duration_seconds",
		metric.WithDescription("Time taken to extract image layers"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.1, 0.5, 1, 5, 10, 30, 60, 120,
		),
	)
	if err != nil {
		return nil, err
	}

	metrics.optimizeDuration, err = meter.Float64Histogram(
		"builderd_optimize_duration_seconds",
		metric.WithDescription("Time taken to optimize rootfs"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.1, 0.5, 1, 5, 10, 30, 60,
		),
	)
	if err != nil {
		return nil, err
	}

	// Gauges
	metrics.activeBuilds, err = meter.Int64UpDownCounter(
		"builderd_active_builds",
		metric.WithDescription("Number of currently active builds"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	metrics.queuedBuilds, err = meter.Int64UpDownCounter(
		"builderd_queued_builds",
		metric.WithDescription("Number of queued builds"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Size metrics
	metrics.imageSizeBytes, err = meter.Int64Histogram(
		"builderd_image_size_bytes",
		metric.WithDescription("Size of source images in bytes"),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(
			1<<20, 10<<20, 50<<20, 100<<20, 500<<20, // 1MB to 500MB
			1<<30, 2<<30, 5<<30, 10<<30, // 1GB to 10GB
		),
	)
	if err != nil {
		return nil, err
	}

	metrics.rootfsSizeBytes, err = meter.Int64Histogram(
		"builderd_rootfs_size_bytes",
		metric.WithDescription("Size of generated rootfs in bytes"),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(
			1<<20, 10<<20, 50<<20, 100<<20, 500<<20, // 1MB to 500MB
			1<<30, 2<<30, 5<<30, // 1GB to 5GB
		),
	)
	if err != nil {
		return nil, err
	}

	metrics.compressionRatio, err = meter.Float64Histogram(
		"builderd_compression_ratio",
		metric.WithDescription("Compression ratio (original/final size)"),
		metric.WithUnit("1"),
		metric.WithExplicitBucketBoundaries(
			0.1, 0.2, 0.5, 0.7, 0.8, 0.9, 1.0, 1.2, 1.5, 2.0,
		),
	)
	if err != nil {
		return nil, err
	}

	// Resource usage
	metrics.buildMemoryUsage, err = meter.Int64Histogram(
		"builderd_build_memory_usage_bytes",
		metric.WithDescription("Peak memory usage during builds"),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(
			100<<20, 500<<20, // 100MB, 500MB
			1<<30, 2<<30, 4<<30, 8<<30, // 1GB, 2GB, 4GB, 8GB
		),
	)
	if err != nil {
		return nil, err
	}

	metrics.buildDiskUsage, err = meter.Int64Histogram(
		"builderd_build_disk_usage_bytes",
		metric.WithDescription("Peak disk usage during builds"),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(
			100<<20, 500<<20, // 100MB, 500MB
			1<<30, 5<<30, 10<<30, 50<<30, // 1GB, 5GB, 10GB, 50GB
		),
	)
	if err != nil {
		return nil, err
	}

	metrics.buildCPUUsage, err = meter.Float64Histogram(
		"builderd_build_cpu_usage_cores",
		metric.WithDescription("CPU cores utilized during builds"),
		metric.WithUnit("1"),
		metric.WithExplicitBucketBoundaries(
			0.1, 0.5, 1.0, 2.0, 4.0, 8.0, 16.0,
		),
	)
	if err != nil {
		return nil, err
	}

	// Build step metrics
	metrics.buildStepsTotal, err = meter.Int64Counter(
		"builderd_build_steps_total",
		metric.WithDescription("Total number of build steps executed"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	metrics.buildStepErrors, err = meter.Int64Counter(
		"builderd_build_step_errors_total",
		metric.WithDescription("Total number of build step errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	metrics.buildStepDuration, err = meter.Float64Histogram(
		"builderd_build_step_duration_seconds",
		metric.WithDescription("Duration of individual build steps"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.1, 0.5, 1, 5, 10, 30, 60, 120, 300,
		),
	)
	if err != nil {
		return nil, err
	}

	// Base asset initialization metrics
	metrics.baseAssetInitRetries, err = meter.Int64Counter(
		"builderd_base_asset_init_retries_total",
		metric.WithDescription("Total number of base asset initialization retries"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	metrics.baseAssetInitFailures, err = meter.Int64Counter(
		"builderd_base_asset_init_failures_total",
		metric.WithDescription("Total number of base asset initialization final failures"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	logger.Info("build metrics initialized",
		slog.Bool("high_cardinality_enabled", highCardinalityEnabled),
	)

	return metrics, nil
}

// RecordBuildStart records the start of a build
func (m *BuildMetrics) RecordBuildStart(ctx context.Context, buildType, sourceType, tenantTier string) {
	attrs := []attribute.KeyValue{
		attribute.String("build_type", buildType),
		attribute.String("source_type", sourceType),
		attribute.String("tenant_tier", tenantTier),
	}

	m.buildsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.activeBuilds.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordBuildComplete records the completion of a build
func (m *BuildMetrics) RecordBuildComplete(ctx context.Context, buildType, sourceType string, duration time.Duration, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("build_type", buildType),
		attribute.String("source_type", sourceType),
		attribute.String("status", func() string {
			if success {
				return "success"
			}
			return "failure"
		}()),
	}

	m.buildDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	m.activeBuilds.Add(ctx, -1, metric.WithAttributes(attrs...))

	if !success {
		m.buildErrorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordBuildCancellation records a build cancellation
func (m *BuildMetrics) RecordBuildCancellation(ctx context.Context, buildType, sourceType string) {
	attrs := []attribute.KeyValue{
		attribute.String("build_type", buildType),
		attribute.String("source_type", sourceType),
	}

	m.buildCancellations.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.activeBuilds.Add(ctx, -1, metric.WithAttributes(attrs...))
}

// RecordPullDuration records the time taken to pull source
func (m *BuildMetrics) RecordPullDuration(ctx context.Context, sourceType string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("source_type", sourceType),
	}

	m.pullDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordExtractDuration records the time taken to extract
func (m *BuildMetrics) RecordExtractDuration(ctx context.Context, sourceType string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("source_type", sourceType),
	}

	m.extractDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordOptimizeDuration records the time taken to optimize
func (m *BuildMetrics) RecordOptimizeDuration(ctx context.Context, duration time.Duration) {
	m.optimizeDuration.Record(ctx, duration.Seconds())
}

// RecordImageSize records the size of source images
func (m *BuildMetrics) RecordImageSize(ctx context.Context, sourceType string, sizeBytes int64) {
	attrs := []attribute.KeyValue{
		attribute.String("source_type", sourceType),
	}

	m.imageSizeBytes.Record(ctx, sizeBytes, metric.WithAttributes(attrs...))
}

// RecordRootfsSize records the size of generated rootfs
func (m *BuildMetrics) RecordRootfsSize(ctx context.Context, sizeBytes int64) {
	m.rootfsSizeBytes.Record(ctx, sizeBytes)
}

// RecordCompressionRatio records the compression ratio achieved
func (m *BuildMetrics) RecordCompressionRatio(ctx context.Context, ratio float64) {
	m.compressionRatio.Record(ctx, ratio)
}

// RecordResourceUsage records peak resource usage during build
func (m *BuildMetrics) RecordResourceUsage(ctx context.Context, memoryBytes, diskBytes int64, cpuCores float64) {
	m.buildMemoryUsage.Record(ctx, memoryBytes)
	m.buildDiskUsage.Record(ctx, diskBytes)
	m.buildCPUUsage.Record(ctx, cpuCores)
}

// RecordQueuedBuild records a build being queued
func (m *BuildMetrics) RecordQueuedBuild(ctx context.Context) {
	m.queuedBuilds.Add(ctx, 1)
}

// RecordDequeuedBuild records a build being dequeued
func (m *BuildMetrics) RecordDequeuedBuild(ctx context.Context) {
	m.queuedBuilds.Add(ctx, -1)
}

// RecordBuildStepStart records the start of a build step
func (m *BuildMetrics) RecordBuildStepStart(ctx context.Context, stepName, sourceType string) {
	attrs := []attribute.KeyValue{
		attribute.String("step", stepName),
		attribute.String("source_type", sourceType),
	}

	m.buildStepsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordBuildStepComplete records the completion of a build step
func (m *BuildMetrics) RecordBuildStepComplete(ctx context.Context, stepName, sourceType string, duration time.Duration, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("step", stepName),
		attribute.String("source_type", sourceType),
		attribute.String("status", func() string {
			if success {
				return "success"
			}
			return "failure"
		}()),
	}

	m.buildStepDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if !success {
		m.buildStepErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordBaseAssetInitRetry records a retry attempt for base asset initialization
func (m *BuildMetrics) RecordBaseAssetInitRetry(ctx context.Context, attempt int, reason string) {
	attrs := []attribute.KeyValue{
		attribute.Int("attempt", attempt),
		attribute.String("reason", reason),
	}

	m.baseAssetInitRetries.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordBaseAssetInitFailure records a final failure of base asset initialization after all retries
func (m *BuildMetrics) RecordBaseAssetInitFailure(ctx context.Context, totalAttempts int, finalError string) {
	attrs := []attribute.KeyValue{
		attribute.Int("total_attempts", totalAttempts),
		attribute.String("final_error", finalError),
	}

	m.baseAssetInitFailures.Add(ctx, 1, metric.WithAttributes(attrs...))
}
