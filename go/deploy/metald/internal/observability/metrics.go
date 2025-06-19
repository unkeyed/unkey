package observability

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// VMMetrics tracks VM-related operations using OpenTelemetry counters
type VMMetrics struct {
	logger                 *slog.Logger
	meter                  metric.Meter
	highCardinalityEnabled bool

	// VM lifecycle counters
	vmCreateRequests   metric.Int64Counter
	vmCreateSuccess    metric.Int64Counter
	vmCreateFailures   metric.Int64Counter
	vmBootRequests     metric.Int64Counter
	vmBootSuccess      metric.Int64Counter
	vmBootFailures     metric.Int64Counter
	vmShutdownRequests metric.Int64Counter
	vmShutdownSuccess  metric.Int64Counter
	vmShutdownFailures metric.Int64Counter
	vmDeleteRequests   metric.Int64Counter
	vmDeleteSuccess    metric.Int64Counter
	vmDeleteFailures   metric.Int64Counter

	// VM state operation counters
	vmPauseRequests  metric.Int64Counter
	vmPauseSuccess   metric.Int64Counter
	vmPauseFailures  metric.Int64Counter
	vmResumeRequests metric.Int64Counter
	vmResumeSuccess  metric.Int64Counter
	vmResumeFailures metric.Int64Counter
	vmRebootRequests metric.Int64Counter
	vmRebootSuccess  metric.Int64Counter
	vmRebootFailures metric.Int64Counter

	// VM information counters
	vmInfoRequests    metric.Int64Counter
	vmListRequests    metric.Int64Counter
	vmMetricsRequests metric.Int64Counter

	// Process management counters
	processCreateRequests metric.Int64Counter
	processCreateSuccess  metric.Int64Counter
	processCreateFailures metric.Int64Counter
	processTerminations   metric.Int64Counter
	processCleanups       metric.Int64Counter

	// Jailer-specific counters
	jailerStartRequests metric.Int64Counter
	jailerStartSuccess  metric.Int64Counter
	jailerStartFailures metric.Int64Counter

	// Duration histograms for operation timing
	vmCreateDuration   metric.Float64Histogram
	vmBootDuration     metric.Float64Histogram
	vmShutdownDuration metric.Float64Histogram
	vmDeleteDuration   metric.Float64Histogram
}

// NewVMMetrics creates a new VM metrics instance
func NewVMMetrics(logger *slog.Logger, highCardinalityEnabled bool) (*VMMetrics, error) {
	meter := otel.Meter("unkey.metald.vm.operations")

	vm := &VMMetrics{ //nolint:exhaustruct // Metric fields are initialized below with error handling
		logger:                 logger.With("component", "vm_metrics"),
		meter:                  meter,
		highCardinalityEnabled: highCardinalityEnabled,
	}

	var err error

	// VM lifecycle counters
	if vm.vmCreateRequests, err = meter.Int64Counter(
		"unkey_metald_vm_create_requests_total",
		metric.WithDescription("Total number of VM create requests"),
	); err != nil {
		return nil, err
	}

	if vm.vmCreateSuccess, err = meter.Int64Counter(
		"unkey_metald_vm_create_success_total",
		metric.WithDescription("Total number of successful VM creates"),
	); err != nil {
		return nil, err
	}

	if vm.vmCreateFailures, err = meter.Int64Counter(
		"unkey_metald_vm_create_failures_total",
		metric.WithDescription("Total number of failed VM creates"),
	); err != nil {
		return nil, err
	}

	if vm.vmBootRequests, err = meter.Int64Counter(
		"unkey_metald_vm_boot_requests_total",
		metric.WithDescription("Total number of VM boot requests"),
	); err != nil {
		return nil, err
	}

	if vm.vmBootSuccess, err = meter.Int64Counter(
		"unkey_metald_vm_boot_success_total",
		metric.WithDescription("Total number of successful VM boots"),
	); err != nil {
		return nil, err
	}

	if vm.vmBootFailures, err = meter.Int64Counter(
		"unkey_metald_vm_boot_failures_total",
		metric.WithDescription("Total number of failed VM boots"),
	); err != nil {
		return nil, err
	}

	if vm.vmShutdownRequests, err = meter.Int64Counter(
		"unkey_metald_vm_shutdown_requests_total",
		metric.WithDescription("Total number of VM shutdown requests"),
	); err != nil {
		return nil, err
	}

	if vm.vmShutdownSuccess, err = meter.Int64Counter(
		"unkey_metald_vm_shutdown_success_total",
		metric.WithDescription("Total number of successful VM shutdowns"),
	); err != nil {
		return nil, err
	}

	if vm.vmShutdownFailures, err = meter.Int64Counter(
		"unkey_metald_vm_shutdown_failures_total",
		metric.WithDescription("Total number of failed VM shutdowns"),
	); err != nil {
		return nil, err
	}

	if vm.vmDeleteRequests, err = meter.Int64Counter(
		"unkey_metald_vm_delete_requests_total",
		metric.WithDescription("Total number of VM delete requests"),
	); err != nil {
		return nil, err
	}

	if vm.vmDeleteSuccess, err = meter.Int64Counter(
		"unkey_metald_vm_delete_success_total",
		metric.WithDescription("Total number of successful VM deletes"),
	); err != nil {
		return nil, err
	}

	if vm.vmDeleteFailures, err = meter.Int64Counter(
		"unkey_metald_vm_delete_failures_total",
		metric.WithDescription("Total number of failed VM deletes"),
	); err != nil {
		return nil, err
	}

	// VM state operation counters
	if vm.vmPauseRequests, err = meter.Int64Counter(
		"unkey_metald_vm_pause_requests_total",
		metric.WithDescription("Total number of VM pause requests"),
	); err != nil {
		return nil, err
	}

	if vm.vmPauseSuccess, err = meter.Int64Counter(
		"unkey_metald_vm_pause_success_total",
		metric.WithDescription("Total number of successful VM pauses"),
	); err != nil {
		return nil, err
	}

	if vm.vmPauseFailures, err = meter.Int64Counter(
		"unkey_metald_vm_pause_failures_total",
		metric.WithDescription("Total number of failed VM pauses"),
	); err != nil {
		return nil, err
	}

	if vm.vmResumeRequests, err = meter.Int64Counter(
		"unkey_metald_vm_resume_requests_total",
		metric.WithDescription("Total number of VM resume requests"),
	); err != nil {
		return nil, err
	}

	if vm.vmResumeSuccess, err = meter.Int64Counter(
		"unkey_metald_vm_resume_success_total",
		metric.WithDescription("Total number of successful VM resumes"),
	); err != nil {
		return nil, err
	}

	if vm.vmResumeFailures, err = meter.Int64Counter(
		"unkey_metald_vm_resume_failures_total",
		metric.WithDescription("Total number of failed VM resumes"),
	); err != nil {
		return nil, err
	}

	if vm.vmRebootRequests, err = meter.Int64Counter(
		"unkey_metald_vm_reboot_requests_total",
		metric.WithDescription("Total number of VM reboot requests"),
	); err != nil {
		return nil, err
	}

	if vm.vmRebootSuccess, err = meter.Int64Counter(
		"unkey_metald_vm_reboot_success_total",
		metric.WithDescription("Total number of successful VM reboots"),
	); err != nil {
		return nil, err
	}

	if vm.vmRebootFailures, err = meter.Int64Counter(
		"unkey_metald_vm_reboot_failures_total",
		metric.WithDescription("Total number of failed VM reboots"),
	); err != nil {
		return nil, err
	}

	// VM information counters
	if vm.vmInfoRequests, err = meter.Int64Counter(
		"unkey_metald_vm_info_requests_total",
		metric.WithDescription("Total number of VM info requests"),
	); err != nil {
		return nil, err
	}

	if vm.vmListRequests, err = meter.Int64Counter(
		"unkey_metald_vm_list_requests_total",
		metric.WithDescription("Total number of VM list requests"),
	); err != nil {
		return nil, err
	}

	if vm.vmMetricsRequests, err = meter.Int64Counter(
		"unkey_metald_vm_metrics_requests_total",
		metric.WithDescription("Total number of VM metrics requests"),
	); err != nil {
		return nil, err
	}

	// Process management counters
	if vm.processCreateRequests, err = meter.Int64Counter(
		"unkey_metald_process_create_requests_total",
		metric.WithDescription("Total number of process create requests"),
	); err != nil {
		return nil, err
	}

	if vm.processCreateSuccess, err = meter.Int64Counter(
		"unkey_metald_process_create_success_total",
		metric.WithDescription("Total number of successful process creates"),
	); err != nil {
		return nil, err
	}

	if vm.processCreateFailures, err = meter.Int64Counter(
		"unkey_metald_process_create_failures_total",
		metric.WithDescription("Total number of failed process creates"),
	); err != nil {
		return nil, err
	}

	if vm.processTerminations, err = meter.Int64Counter(
		"unkey_metald_process_terminations_total",
		metric.WithDescription("Total number of process terminations"),
	); err != nil {
		return nil, err
	}

	if vm.processCleanups, err = meter.Int64Counter(
		"unkey_metald_process_cleanups_total",
		metric.WithDescription("Total number of process cleanups"),
	); err != nil {
		return nil, err
	}

	// Jailer-specific counters
	if vm.jailerStartRequests, err = meter.Int64Counter(
		"unkey_metald_jailer_start_requests_total",
		metric.WithDescription("Total number of jailer start requests"),
	); err != nil {
		return nil, err
	}

	if vm.jailerStartSuccess, err = meter.Int64Counter(
		"unkey_metald_jailer_start_success_total",
		metric.WithDescription("Total number of successful jailer starts"),
	); err != nil {
		return nil, err
	}

	if vm.jailerStartFailures, err = meter.Int64Counter(
		"unkey_metald_jailer_start_failures_total",
		metric.WithDescription("Total number of failed jailer starts"),
	); err != nil {
		return nil, err
	}

	// Duration histograms
	if vm.vmCreateDuration, err = meter.Float64Histogram(
		"unkey_metald_vm_create_duration_seconds",
		metric.WithDescription("VM create operation duration"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, err
	}

	if vm.vmBootDuration, err = meter.Float64Histogram(
		"unkey_metald_vm_boot_duration_seconds",
		metric.WithDescription("VM boot operation duration"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, err
	}

	if vm.vmShutdownDuration, err = meter.Float64Histogram(
		"unkey_metald_vm_shutdown_duration_seconds",
		metric.WithDescription("VM shutdown operation duration"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, err
	}

	if vm.vmDeleteDuration, err = meter.Float64Histogram(
		"unkey_metald_vm_delete_duration_seconds",
		metric.WithDescription("VM delete operation duration"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, err
	}

	vm.logger.Info("VM metrics initialized")
	return vm, nil
}

// VM lifecycle metric methods
func (vm *VMMetrics) RecordVMCreateRequest(ctx context.Context, backend string) {
	vm.vmCreateRequests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
	))
}

func (vm *VMMetrics) RecordVMCreateSuccess(ctx context.Context, vmID string, backend string, duration time.Duration) {
	// Success counters don't include VM ID to avoid high cardinality
	vm.vmCreateSuccess.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
	))
	vm.vmCreateDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("backend", backend),
	))
}

func (vm *VMMetrics) RecordVMCreateFailure(ctx context.Context, backend string, errorType string) {
	vm.vmCreateFailures.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
		attribute.String("error_type", errorType),
	))
}

func (vm *VMMetrics) RecordVMBootRequest(ctx context.Context, vmID string, backend string) {
	// Request counters don't include VM ID to avoid high cardinality
	vm.vmBootRequests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
	))
}

func (vm *VMMetrics) RecordVMBootSuccess(ctx context.Context, vmID string, backend string, duration time.Duration) {
	// Success counters don't include VM ID to avoid high cardinality
	vm.vmBootSuccess.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
	))
	vm.vmBootDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("backend", backend),
	))
}

func (vm *VMMetrics) RecordVMBootFailure(ctx context.Context, vmID string, backend string, errorType string) {
	// Failure metrics only include backend and error type, not VM ID
	vm.vmBootFailures.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
		attribute.String("error_type", errorType),
	))
}

func (vm *VMMetrics) RecordVMShutdownRequest(ctx context.Context, vmID string, backend string, force bool) {
	// Request counters don't include VM ID to avoid high cardinality
	vm.vmShutdownRequests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
		attribute.Bool("force", force),
	))
}

func (vm *VMMetrics) RecordVMShutdownSuccess(ctx context.Context, vmID string, backend string, force bool, duration time.Duration) {
	// Success counters don't include VM ID to avoid high cardinality
	vm.vmShutdownSuccess.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
		attribute.Bool("force", force),
	))
	vm.vmShutdownDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("backend", backend),
		attribute.Bool("force", force),
	))
}

func (vm *VMMetrics) RecordVMShutdownFailure(ctx context.Context, vmID string, backend string, force bool, errorType string) {
	// Failure metrics only include backend, force flag, and error type
	vm.vmShutdownFailures.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
		attribute.Bool("force", force),
		attribute.String("error_type", errorType),
	))
}

func (vm *VMMetrics) RecordVMDeleteRequest(ctx context.Context, vmID string, backend string) {
	// Request counters don't include VM ID to avoid high cardinality
	vm.vmDeleteRequests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
	))
}

func (vm *VMMetrics) RecordVMDeleteSuccess(ctx context.Context, vmID string, backend string, duration time.Duration) {
	// Success counters don't include VM ID to avoid high cardinality
	vm.vmDeleteSuccess.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
	))
	vm.vmDeleteDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("backend", backend),
	))
}

func (vm *VMMetrics) RecordVMDeleteFailure(ctx context.Context, vmID string, backend string, errorType string) {
	// Failure metrics only include backend and error type
	vm.vmDeleteFailures.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
		attribute.String("error_type", errorType),
	))
}

// VM state operation metric methods
func (vm *VMMetrics) RecordVMPauseRequest(ctx context.Context, vmID string, backend string) {
	// Request counters don't include VM ID to avoid high cardinality
	vm.vmPauseRequests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
	))
}

func (vm *VMMetrics) RecordVMPauseSuccess(ctx context.Context, vmID string, backend string) {
	// Success counters don't include VM ID to avoid high cardinality
	vm.vmPauseSuccess.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
	))
}

func (vm *VMMetrics) RecordVMPauseFailure(ctx context.Context, vmID string, backend string, errorType string) {
	// Failure metrics only include backend and error type
	vm.vmPauseFailures.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
		attribute.String("error_type", errorType),
	))
}

func (vm *VMMetrics) RecordVMResumeRequest(ctx context.Context, vmID string, backend string) {
	// Request counters don't include VM ID to avoid high cardinality
	vm.vmResumeRequests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
	))
}

func (vm *VMMetrics) RecordVMResumeSuccess(ctx context.Context, vmID string, backend string) {
	// Success counters don't include VM ID to avoid high cardinality
	vm.vmResumeSuccess.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
	))
}

func (vm *VMMetrics) RecordVMResumeFailure(ctx context.Context, vmID string, backend string, errorType string) {
	// Failure metrics only include backend and error type
	vm.vmResumeFailures.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
		attribute.String("error_type", errorType),
	))
}

func (vm *VMMetrics) RecordVMRebootRequest(ctx context.Context, vmID string, backend string) {
	// Request counters don't include VM ID to avoid high cardinality
	vm.vmRebootRequests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
	))
}

func (vm *VMMetrics) RecordVMRebootSuccess(ctx context.Context, vmID string, backend string) {
	// Success counters don't include VM ID to avoid high cardinality
	vm.vmRebootSuccess.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
	))
}

func (vm *VMMetrics) RecordVMRebootFailure(ctx context.Context, vmID string, backend string, errorType string) {
	// Failure metrics only include backend and error type
	vm.vmRebootFailures.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
		attribute.String("error_type", errorType),
	))
}

// VM information metric methods
func (vm *VMMetrics) RecordVMInfoRequest(ctx context.Context, vmID string, backend string) {
	// Info request counters don't include VM ID to avoid high cardinality
	vm.vmInfoRequests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
	))
}

func (vm *VMMetrics) RecordVMListRequest(ctx context.Context, backend string) {
	vm.vmListRequests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
	))
}

func (vm *VMMetrics) RecordVMMetricsRequest(ctx context.Context, vmID string, backend string) {
	// Metrics request counters don't include VM ID to avoid high cardinality
	vm.vmMetricsRequests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("backend", backend),
	))
}

// Process management metric methods
func (vm *VMMetrics) RecordProcessCreateRequest(ctx context.Context, vmID string, useJailer bool) {
	var attrs []attribute.KeyValue
	if vm.highCardinalityEnabled {
		attrs = append(attrs, attribute.String("vm_id", vmID))
	}
	attrs = append(attrs, attribute.Bool("use_jailer", useJailer))
	vm.processCreateRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (vm *VMMetrics) RecordProcessCreateSuccess(ctx context.Context, vmID string, processID string, useJailer bool) {
	var attrs []attribute.KeyValue
	if vm.highCardinalityEnabled {
		attrs = append(attrs,
			attribute.String("vm_id", vmID),
			attribute.String("process_id", processID),
		)
	}
	attrs = append(attrs, attribute.Bool("use_jailer", useJailer))
	vm.processCreateSuccess.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (vm *VMMetrics) RecordProcessCreateFailure(ctx context.Context, vmID string, useJailer bool, errorType string) {
	var attrs []attribute.KeyValue
	if vm.highCardinalityEnabled {
		attrs = append(attrs, attribute.String("vm_id", vmID))
	}
	attrs = append(attrs,
		attribute.Bool("use_jailer", useJailer),
		attribute.String("error_type", errorType),
	)
	vm.processCreateFailures.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (vm *VMMetrics) RecordProcessTermination(ctx context.Context, vmID string, processID string, exitCode int) {
	var attrs []attribute.KeyValue
	if vm.highCardinalityEnabled {
		attrs = append(attrs,
			attribute.String("vm_id", vmID),
			attribute.String("process_id", processID),
		)
	}
	attrs = append(attrs, attribute.Int("exit_code", exitCode))
	vm.processTerminations.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (vm *VMMetrics) RecordProcessCleanup(ctx context.Context, vmID string, processID string) {
	var attrs []attribute.KeyValue
	if vm.highCardinalityEnabled {
		attrs = append(attrs,
			attribute.String("vm_id", vmID),
			attribute.String("process_id", processID),
		)
	}
	vm.processCleanups.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// Jailer-specific metric methods
func (vm *VMMetrics) RecordJailerStartRequest(ctx context.Context, vmID string, jailerID string) {
	var attrs []attribute.KeyValue
	if vm.highCardinalityEnabled {
		attrs = append(attrs,
			attribute.String("vm_id", vmID),
			attribute.String("jailer_id", jailerID),
		)
	}
	vm.jailerStartRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (vm *VMMetrics) RecordJailerStartSuccess(ctx context.Context, vmID string, jailerID string) {
	var attrs []attribute.KeyValue
	if vm.highCardinalityEnabled {
		attrs = append(attrs,
			attribute.String("vm_id", vmID),
			attribute.String("jailer_id", jailerID),
		)
	}
	vm.jailerStartSuccess.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (vm *VMMetrics) RecordJailerStartFailure(ctx context.Context, vmID string, jailerID string, errorType string) {
	var attrs []attribute.KeyValue
	if vm.highCardinalityEnabled {
		attrs = append(attrs,
			attribute.String("vm_id", vmID),
			attribute.String("jailer_id", jailerID),
		)
	}
	attrs = append(attrs, attribute.String("error_type", errorType))
	vm.jailerStartFailures.Add(ctx, 1, metric.WithAttributes(attrs...))
}
