package hydra

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/metrics"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/retry"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"go.opentelemetry.io/otel/attribute"

	// MySQL driver
	_ "github.com/go-sql-driver/mysql"
)

// Config holds the configuration for creating a new Engine instance.
//
// All fields except Store are optional and will use sensible defaults
// if not provided.
type Config struct {
	// DSN is the database connection string for MySQL.
	// This field is required and cannot be empty.
	// The engine will create an SQLC store from this connection.
	DSN string

	// Namespace provides tenant isolation for workflows. All workflows
	// created by this engine will be scoped to this namespace.
	// Defaults to "default" if not specified.
	Namespace string

	// Clock provides time-related operations for testing and scheduling.
	// Defaults to a real clock implementation if not specified.
	Clock clock.Clock

	// Logger handles structured logging for the engine operations.
	// Defaults to a no-op logger if not specified.
	Logger logging.Logger

	// Marshaller handles serialization of workflow payloads and step results.
	// Defaults to JSON marshalling if not specified.
	Marshaller Marshaller
}

// NewConfig creates a default config with sensible defaults.
//
// The returned config uses:
// - "default" namespace
// - Real clock implementation
// - All other fields will be set to their defaults when passed to New()
func NewConfig() Config {
	return Config{
		DSN:        "",
		Namespace:  "default",
		Clock:      clock.New(),
		Logger:     nil,
		Marshaller: nil,
	}
}

// Engine is the core workflow orchestration engine that manages workflow
// lifecycle, coordination, and execution.
//
// The engine is responsible for:
// - Starting new workflows and managing their state
// - Coordinating workflow execution across multiple workers
// - Handling cron-based scheduled workflows
// - Providing namespace isolation for multi-tenant deployments
// - Recording metrics and managing observability
//
// Engine instances are thread-safe and can be shared across multiple
// workers and goroutines.
type Engine struct {
	db           *sql.DB
	namespace    string
	cronHandlers map[string]CronHandler
	clock        clock.Clock
	logger       logging.Logger
	marshaller   Marshaller
}

// New creates a new Engine instance with the provided configuration.
//
// The engine will validate the configuration and apply defaults for
// any missing optional fields. The Store field is required and the
// function will panic if it is nil.
//
// Example:
//
//	engine := hydra.New(hydra.Config{
//	    Store:     gormStore,
//	    Namespace: "production",
//	    Logger:    logger,
//	})
func New(config Config) (*Engine, error) {

	err := assert.All(
		assert.NotEmpty(config.DSN),
		assert.NotNil(config.Clock),
		assert.NotEmpty(config.Namespace),
		assert.NotNil(config.Logger),
		assert.NotNil(config.Marshaller),
	)
	if err != nil {
		return nil, err
	}

	var db *sql.DB
	err = retry.New(
		retry.Attempts(10),
		retry.Backoff(func(n int) time.Duration {
			return time.Duration(n) * time.Second
		}),
	).Do(func() error {
		var openErr error
		db, openErr = sql.Open("mysql", config.DSN)
		if openErr != nil {
			config.Logger.Info("mysql not ready yet, retrying...", "error", openErr.Error())
		}
		return openErr

	})

	if err != nil {
		return nil, fmt.Errorf("hydra: failed to open database connection: %w", err)
	}

	err = retry.New(
		retry.Attempts(10),
		retry.Backoff(func(n int) time.Duration {
			return time.Duration(n) * time.Second
		}),
	).Do(func() error {
		return db.Ping()
	})
	// Test the connection
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("hydra: failed to ping database: %v", err)
	}

	return &Engine{
		db:           db,
		namespace:    config.Namespace,
		cronHandlers: make(map[string]CronHandler),
		clock:        config.Clock,
		logger:       config.Logger,
		marshaller:   config.Marshaller,
	}, nil
}

// GetNamespace returns the namespace for this engine instance.
//
// This method is primarily used by workers and internal components
// to scope database operations to the correct tenant namespace.
func (e *Engine) GetNamespace() string {
	return e.namespace
}

// GetDB returns the database connection for direct query usage
func (e *Engine) GetDB() *sql.DB {
	return e.db
}

// RegisterCron registers a cron job with the given schedule and handler.
//
// The cronSpec follows standard cron syntax (e.g., "0 0 * * *" for daily at midnight).
// The name must be unique within this engine's namespace. The handler will be
// called according to the schedule.
//
// Example:
//
//	err := engine.RegisterCron("0 */6 * * *", "cleanup-task", func(ctx context.Context) error {
//	    return performCleanup(ctx)
//	})
//
// Returns an error if a cron job with the same name is already registered.
func (e *Engine) RegisterCron(cronSpec, name string, handler CronHandler) error {
	if _, exists := e.cronHandlers[name]; exists {
		return fmt.Errorf("cron %q is already registered", name)
	}

	e.cronHandlers[name] = handler

	// Use new Query pattern instead of store abstraction
	return store.Query.CreateCronJob(context.Background(), e.db, store.CreateCronJobParams{
		ID:           uid.New(uid.CronJobPrefix),
		Name:         name,
		CronSpec:     cronSpec,
		Namespace:    e.namespace,
		WorkflowName: sql.NullString{String: "", Valid: false}, // Empty since this uses a handler, not a workflow
		Enabled:      true,
		CreatedAt:    e.clock.Now().UnixMilli(),
		UpdatedAt:    e.clock.Now().UnixMilli(),
		LastRunAt:    sql.NullInt64{Int64: 0, Valid: false},
		NextRunAt:    calculateNextRun(cronSpec, e.clock.Now()),
	})
}

// StartWorkflow starts a new workflow execution with the given name and payload.
//
// This method creates a new workflow execution record in the database and makes
// it available for workers to pick up and execute. The workflow will be queued
// in a pending state until a worker acquires a lease and begins execution.
//
// Parameters:
// - ctx: Context for the operation, which may include cancellation and timeouts
// - workflowName: Must match the Name() method of a registered workflow type
// - payload: The input data for the workflow, which will be serialized and stored
// - opts: Optional configuration for retry behavior, timeouts, and trigger metadata
//
// Returns:
// - executionID: A unique identifier for this workflow execution
// - error: Any error that occurred during workflow creation
//
// The payload will be marshalled using the engine's configured marshaller (JSON by default)
// and must be serializable. The workflow will be executed with the configured retry
// policy and timeout settings.
//
// Example:
//
//	executionID, err := engine.StartWorkflow(ctx, "order-processing", &OrderRequest{
//	    CustomerID: "cust_123",
//	    Items:      []Item{{SKU: "item_456", Quantity: 2}},
//	}, hydra.WithMaxAttempts(5), hydra.WithTimeout(30*time.Minute))
//
// Metrics recorded:
// - hydra_workflows_started_total (counter)
// - hydra_workflows_queued (gauge)
// - hydra_payload_size_bytes (histogram)
func (e *Engine) StartWorkflow(ctx context.Context, workflowName string, payload any, opts ...WorkflowOption) (string, error) {
	// Start tracing span for workflow creation
	ctx, span := tracing.Start(ctx, "hydra.engine.StartWorkflow")
	defer span.End()

	executionID := uid.New(uid.WorkflowPrefix)

	span.SetAttributes(
		attribute.String("hydra.workflow.name", workflowName),
		attribute.String("hydra.execution.id", executionID),
		attribute.String("hydra.namespace", e.namespace),
	)

	config := &WorkflowConfig{
		MaxAttempts:     3, // Default to 3 attempts total (1 initial + 2 retries)
		TimeoutDuration: 1 * time.Hour,
		RetryBackoff:    1 * time.Second,
		TriggerType:     store.WorkflowExecutionsTriggerTypeApi, // Default trigger type
		TriggerSource:   nil,
	}
	for _, opt := range opts {
		opt(config)
	}

	span.SetAttributes(
		attribute.String("hydra.trigger.type", string(config.TriggerType)),
	)

	data, err := e.marshaller.Marshal(payload)
	if err != nil {
		metrics.SerializationErrorsTotal.WithLabelValues(e.namespace, workflowName, "input").Inc()
		tracing.RecordError(span, err)
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Record payload size
	metrics.RecordPayloadSize(e.namespace, workflowName, "input", len(data))

	// Extract trace ID and span ID from span context for workflow correlation
	traceID := ""
	spanID := ""
	if spanContext := span.SpanContext(); spanContext.IsValid() {
		traceID = spanContext.TraceID().String()
		spanID = spanContext.SpanID().String()
	}

	// Use new Query pattern instead of store abstraction
	err = store.Query.CreateWorkflow(ctx, e.db, store.CreateWorkflowParams{
		ID:                executionID,
		WorkflowName:      workflowName,
		Status:            store.WorkflowExecutionsStatusPending,
		InputData:         data,
		OutputData:        []byte{},
		ErrorMessage:      sql.NullString{String: "", Valid: false},
		CreatedAt:         e.clock.Now().UnixMilli(),
		StartedAt:         sql.NullInt64{Int64: 0, Valid: false},
		CompletedAt:       sql.NullInt64{Int64: 0, Valid: false},
		MaxAttempts:       config.MaxAttempts,
		RemainingAttempts: config.MaxAttempts, // Start with full attempts available
		NextRetryAt:       sql.NullInt64{Int64: 0, Valid: false},
		Namespace:         e.namespace,
		TriggerType:       store.NullWorkflowExecutionsTriggerType{WorkflowExecutionsTriggerType: store.WorkflowExecutionsTriggerTypeApi, Valid: false}, // Trigger type conversion not implemented
		TriggerSource:     sql.NullString{String: "", Valid: false},
		SleepUntil:        sql.NullInt64{Int64: 0, Valid: false},
		TraceID:           sql.NullString{String: traceID, Valid: traceID != ""},
		SpanID:            sql.NullString{String: spanID, Valid: spanID != ""},
	})
	if err != nil {
		metrics.RecordError(e.namespace, "engine", "workflow_creation_failed")
		tracing.RecordError(span, err)
		return "", fmt.Errorf("failed to create workflow: %w", err)
	}

	// Record workflow started
	triggerTypeStr := string(config.TriggerType)
	metrics.WorkflowsStartedTotal.WithLabelValues(e.namespace, workflowName, triggerTypeStr).Inc()
	metrics.WorkflowsQueued.WithLabelValues(e.namespace, "pending").Inc()

	span.SetAttributes(attribute.String("hydra.workflow.status", "created"))

	return executionID, nil
}
