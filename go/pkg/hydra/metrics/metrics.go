package metrics

import (
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/unkeyed/unkey/go/pkg/version"
)

var constLabels = prometheus.Labels{
	"region":  os.Getenv("UNKEY_REGION"),
	"version": version.Version,
}

var workflowLatencyBuckets = []float64{
	0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0,
	10.0, 30.0, 60.0, 120.0, 300.0, 600.0,
}

var stepLatencyBuckets = []float64{
	0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5,
	1.0, 2.5, 5.0, 10.0, 30.0, 60.0,
}

var dbLatencyBuckets = []float64{
	0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1,
	0.2, 0.5, 1.0, 2.0, 5.0,
}

var WorkflowsStartedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "workflows_started_total",
		Help:        "Total number of workflows started",
		ConstLabels: constLabels,
	},
	[]string{"namespace", "workflow_name", "trigger_type"},
)

var WorkflowsCompletedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "workflows_completed_total",
		Help:        "Total number of workflows completed",
		ConstLabels: constLabels,
	},
	[]string{"namespace", "workflow_name", "status"},
)

var WorkflowsRetriedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "workflows_retried_total",
		Help:        "Total number of workflow retry attempts",
		ConstLabels: constLabels,
	},
	[]string{"namespace", "workflow_name", "attempt"},
)

var WorkflowDurationSeconds = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem:   "hydra",
		Name:        "workflow_duration_seconds",
		Help:        "Time taken to complete workflows",
		ConstLabels: constLabels,
		Buckets:     workflowLatencyBuckets,
	},
	[]string{"namespace", "workflow_name", "status"},
)

var WorkflowQueueTimeSeconds = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem:   "hydra",
		Name:        "workflow_queue_time_seconds",
		Help:        "Time workflow spent queued before execution",
		ConstLabels: constLabels,
		Buckets:     workflowLatencyBuckets,
	},
	[]string{"namespace", "workflow_name"},
)

var WorkflowsActive = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem:   "hydra",
		Name:        "workflows_active",
		Help:        "Currently running workflows",
		ConstLabels: constLabels,
	},
	[]string{"namespace", "worker_id"},
)

var WorkflowsQueued = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem:   "hydra",
		Name:        "workflows_queued",
		Help:        "Workflows waiting to be processed",
		ConstLabels: constLabels,
	},
	[]string{"namespace", "status"},
)

var StepsExecutedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "steps_executed_total",
		Help:        "Total number of workflow steps executed",
		ConstLabels: constLabels,
	},
	[]string{"namespace", "workflow_name", "step_name", "status"},
)

var StepsRetriedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "steps_retried_total",
		Help:        "Total number of step retry attempts",
		ConstLabels: constLabels,
	},
	[]string{"namespace", "workflow_name", "step_name"},
)

var StepsCachedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "steps_cached_total",
		Help:        "Steps skipped due to checkpointing",
		ConstLabels: constLabels,
	},
	[]string{"namespace", "workflow_name", "step_name"},
)

var StepDurationSeconds = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem:   "hydra",
		Name:        "step_duration_seconds",
		Help:        "Time taken to execute workflow steps",
		ConstLabels: constLabels,
		Buckets:     stepLatencyBuckets,
	},
	[]string{"namespace", "workflow_name", "step_name", "status"},
)

var WorkerHeartbeatsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "worker_heartbeats_total",
		Help:        "Total number of worker heartbeat operations",
		ConstLabels: constLabels,
	},
	[]string{"worker_id", "namespace", "status"},
)

var LeaseAcquisitionsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "lease_acquisitions_total",
		Help:        "Total number of lease acquisition attempts",
		ConstLabels: constLabels,
	},
	[]string{"worker_id", "resource_type", "status"},
)

var WorkerPollsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "worker_polls_total",
		Help:        "Total number of worker polling operations",
		ConstLabels: constLabels,
	},
	[]string{"worker_id", "namespace", "status"},
)

var WorkerConcurrencyCurrent = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem:   "hydra",
		Name:        "worker_concurrency_current",
		Help:        "Current workflow concurrency per worker",
		ConstLabels: constLabels,
	},
	[]string{"worker_id", "namespace"},
)

var DbOperationsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "db_operations_total",
		Help:        "Total number of database operations",
		ConstLabels: constLabels,
	},
	[]string{"operation", "table", "status"},
)

var DbOperationDurationSeconds = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem:   "hydra",
		Name:        "db_operation_duration_seconds",
		Help:        "Time taken for database operations",
		ConstLabels: constLabels,
		Buckets:     dbLatencyBuckets,
	},
	[]string{"operation", "table", "status"},
)

var DbConnectionsActive = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem:   "hydra",
		Name:        "db_connections_active",
		Help:        "Active database connections",
		ConstLabels: constLabels,
	},
	[]string{"worker_id"},
)

var SleepsStartedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "sleeps_started_total",
		Help:        "Total number of sleep operations initiated",
		ConstLabels: constLabels,
	},
	[]string{"namespace", "workflow_name"},
)

var SleepsResumedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "sleeps_resumed_total",
		Help:        "Total number of workflows resumed from sleep",
		ConstLabels: constLabels,
	},
	[]string{"namespace", "workflow_name"},
)

var CronTriggersTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "cron_triggers_total",
		Help:        "Total number of cron-triggered workflows",
		ConstLabels: constLabels,
	},
	[]string{"namespace", "cron_name", "status"},
)

var SleepDurationSeconds = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem:   "hydra",
		Name:        "sleep_duration_seconds",
		Help:        "Actual sleep durations",
		ConstLabels: constLabels,
		Buckets:     workflowLatencyBuckets,
	},
	[]string{"namespace", "workflow_name"},
)

var CronExecutionLatencySeconds = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem:   "hydra",
		Name:        "cron_execution_latency_seconds",
		Help:        "Delay between scheduled and actual cron execution",
		ConstLabels: constLabels,
		Buckets:     []float64{0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0, 300.0},
	},
	[]string{"namespace", "cron_name"},
)

var WorkflowsSleeping = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem:   "hydra",
		Name:        "workflows_sleeping",
		Help:        "Currently sleeping workflows",
		ConstLabels: constLabels,
	},
	[]string{"namespace"},
)

var ErrorsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "errors_total",
		Help:        "Total number of errors across all components",
		ConstLabels: constLabels,
	},
	[]string{"namespace", "component", "error_type"},
)

var PanicsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "panics_total",
		Help:        "Total number of panic recoveries",
		ConstLabels: constLabels,
	},
	[]string{"worker_id", "component"},
)

var TimeoutsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "timeouts_total",
		Help:        "Total number of operation timeouts",
		ConstLabels: constLabels,
	},
	[]string{"namespace", "operation_type"},
)

var PayloadSizeBytes = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem:   "hydra",
		Name:        "payload_size_bytes",
		Help:        "Size of workflow and step payloads",
		ConstLabels: constLabels,
		Buckets:     []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000},
	},
	[]string{"namespace", "workflow_name", "direction"},
)

var SerializationErrorsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem:   "hydra",
		Name:        "serialization_errors_total",
		Help:        "Total number of payload serialization errors",
		ConstLabels: constLabels,
	},
	[]string{"namespace", "workflow_name", "direction"},
)

func ObserveWorkflowDuration(namespace, workflowName, status string, start time.Time) {
	duration := time.Since(start)
	WorkflowDurationSeconds.WithLabelValues(namespace, workflowName, status).Observe(duration.Seconds())
}

func ObserveStepDuration(namespace, workflowName, stepName, status string, start time.Time) {
	duration := time.Since(start)
	StepDurationSeconds.WithLabelValues(namespace, workflowName, stepName, status).Observe(duration.Seconds())
}

func ObserveDbOperation(operation, table, status string, start time.Time) {
	duration := time.Since(start)
	DbOperationsTotal.WithLabelValues(operation, table, status).Inc()
	DbOperationDurationSeconds.WithLabelValues(operation, table, status).Observe(duration.Seconds())
}

func RecordError(namespace, component, errorType string) {
	ErrorsTotal.WithLabelValues(namespace, component, errorType).Inc()
}

func RecordPayloadSize(namespace, workflowName, direction string, size int) {
	PayloadSizeBytes.WithLabelValues(namespace, workflowName, direction).Observe(float64(size))
}
