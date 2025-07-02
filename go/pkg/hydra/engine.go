package hydra

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/metrics"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// Config holds the configuration for creating a new Engine instance.
//
// All fields except Store are optional and will use sensible defaults
// if not provided.
type Config struct {
	// Store is the persistence layer for workflow state and metadata.
	// This field is required and cannot be nil.
	Store store.Store

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
		Store:      nil,
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
	store        store.Store
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
func New(config Config) *Engine {
	if config.Store == nil {
		panic("hydra: config.Store cannot be nil")
	}

	namespace := config.Namespace
	if namespace == "" {
		namespace = "default"
	}

	clk := config.Clock
	if clk == nil {
		clk = clock.New() // Default to real clock
	}

	logger := config.Logger
	if logger == nil {
		logger = logging.NewNoop() // Default logger
	}

	marshaller := config.Marshaller
	if marshaller == nil {
		marshaller = NewJSONMarshaller() // Default to JSON marshaller
	}

	return &Engine{
		store:        config.Store,
		namespace:    namespace,
		cronHandlers: make(map[string]CronHandler),
		clock:        clk,
		logger:       logger,
		marshaller:   marshaller,
	}
}

// NewWithStore creates a new Engine with the provided store and default config.
//
// This is a convenience function for creating an engine with minimal configuration.
// Other configuration options will use their default values.
//
// Deprecated: Use New(Config{...}) for more explicit configuration.
func NewWithStore(st store.Store, namespace string, clk clock.Clock) *Engine {
	return New(Config{
		Store:      st,
		Namespace:  namespace,
		Clock:      clk,
		Logger:     nil,
		Marshaller: nil,
	})
}

// GetNamespace returns the namespace for this engine instance.
//
// This method is primarily used by workers and internal components
// to scope database operations to the correct tenant namespace.
func (e *Engine) GetNamespace() string {
	return e.namespace
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

	cronJob := &store.CronJob{
		ID:           uid.New(uid.CronJobPrefix),
		Name:         name,
		CronSpec:     cronSpec,
		Namespace:    e.namespace,
		WorkflowName: "", // Empty since this uses a handler, not a workflow
		Enabled:      true,
		CreatedAt:    e.clock.Now().UnixMilli(),
		UpdatedAt:    e.clock.Now().UnixMilli(),
		LastRunAt:    nil,
		NextRunAt:    calculateNextRun(cronSpec, e.clock.Now()),
	}

	return e.store.UpsertCronJob(context.Background(), cronJob)
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

	executionID := uid.New("wf")

	config := &WorkflowConfig{
		MaxAttempts:     3, // Default to 3 attempts total (1 initial + 2 retries)
		TimeoutDuration: 1 * time.Hour,
		RetryBackoff:    1 * time.Second,
		TriggerType:     TriggerTypeAPI, // Default trigger type
		TriggerSource:   nil,
	}
	for _, opt := range opts {
		opt(config)
	}

	data, err := e.marshaller.Marshal(payload)
	if err != nil {
		metrics.SerializationErrorsTotal.WithLabelValues(e.namespace, workflowName, "input").Inc()
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Record payload size
	metrics.RecordPayloadSize(e.namespace, workflowName, "input", len(data))

	workflow := &store.WorkflowExecution{
		ID:                executionID,
		WorkflowName:      workflowName,
		Status:            store.WorkflowStatusPending,
		InputData:         data,
		OutputData:        nil,
		ErrorMessage:      "",
		Namespace:         e.namespace,
		MaxAttempts:       config.MaxAttempts,
		RemainingAttempts: config.MaxAttempts, // Start with full attempts available
		CreatedAt:         e.clock.Now().UnixMilli(),
		StartedAt:         nil,
		CompletedAt:       nil,
		NextRetryAt:       nil,
		SleepUntil:        nil,
		TriggerType:       config.TriggerType,
		TriggerSource:     config.TriggerSource,
		TraceID:           "",
	}

	err = e.store.CreateWorkflow(ctx, workflow)
	if err != nil {
		metrics.RecordError(e.namespace, "engine", "workflow_creation_failed")
		return "", fmt.Errorf("failed to create workflow: %w", err)
	}

	// Record workflow started
	triggerTypeStr := string(config.TriggerType)
	metrics.WorkflowsStartedTotal.WithLabelValues(e.namespace, workflowName, triggerTypeStr).Inc()
	metrics.WorkflowsQueued.WithLabelValues(e.namespace, "pending").Inc()

	return workflow.ID, nil
}

// GetStore returns the underlying store (for testing purposes)
func (e *Engine) GetStore() store.Store {
	return e.store
}
