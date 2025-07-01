package hydra

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// Config holds the configuration for creating a new Engine
type Config struct {
	Store store.Store

	Namespace string

	Clock clock.Clock

	Logger logging.Logger

	Marshaller Marshaller
}

// NewConfig creates a default config with sensible defaults
func NewConfig() Config {
	return Config{
		Namespace: "default",
		Clock:     clock.New(),
	}
}

// Engine is the core workflow orchestration engine
type Engine struct {
	store        store.Store
	namespace    string
	cronHandlers map[string]CronHandler
	clock        clock.Clock
	logger       logging.Logger
	marshaller   Marshaller
}

// New creates a new Engine instance with the provided configuration
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

// NewWithStore creates a new Engine with the provided store and default config
func NewWithStore(st store.Store, namespace string, clk clock.Clock) *Engine {
	return New(Config{
		Store:     st,
		Namespace: namespace,
		Clock:     clk,
	})
}

// RegisterCron registers a cron job with the given schedule and handler
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
		NextRunAt:    calculateNextRun(cronSpec, e.clock.Now()),
	}

	return e.store.UpsertCronJob(context.Background(), cronJob)
}

// StartWorkflow starts a new workflow execution with the given name and payload
func (e *Engine) StartWorkflow(ctx context.Context, workflowName string, payload any, opts ...WorkflowOption) (string, error) {

	executionID := uid.New("wf")

	config := &WorkflowConfig{
		MaxAttempts:     3, // Default to 3 attempts total (1 initial + 2 retries)
		TimeoutDuration: 1 * time.Hour,
		RetryBackoff:    1 * time.Second,
	}
	for _, opt := range opts {
		opt(config)
	}

	data, err := e.marshaller.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	workflow := &store.WorkflowExecution{
		ID:                executionID,
		WorkflowName:      workflowName,
		Status:            store.WorkflowStatusPending,
		InputData:         data,
		Namespace:         e.namespace,
		MaxAttempts:       config.MaxAttempts,
		RemainingAttempts: config.MaxAttempts, // Start with full attempts available
		CreatedAt:         e.clock.Now().UnixMilli(),
	}

	err = e.store.CreateWorkflow(ctx, workflow)
	if err != nil {
		return "", fmt.Errorf("failed to create workflow: %w", err)
	}

	return workflow.ID, nil
}

// GetStore returns the underlying store (for testing purposes)
func (e *Engine) GetStore() store.Store {
	return e.store
}

// generateWorkerID generates a unique worker ID using the uid package
func generateWorkerID() string {
	return uid.New(uid.WorkerPrefix)
}
