package deployment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"
)

type DeploymentEngine struct {
	db           *sql.DB
	logger       *slog.Logger
	publisher    EventPublisher
	stepRegistry map[DeploymentStep]StepExecutor
	workers      int
	shutdown     chan struct{}
	wg           sync.WaitGroup
	mu           sync.RWMutex
	running      bool
}

type EngineConfig struct {
	DB           *sql.DB
	Logger       *slog.Logger
	Publisher    EventPublisher
	Workers      int
	PollInterval time.Duration
}

type StepExecutionContext struct {
	Deployment   *Deployment
	StepResult   *StepResult
	Attempt      int
	Logger       *slog.Logger
	StartTime    time.Time
}

type RetryPolicy struct {
	MaxAttempts        int
	InitialBackoff     time.Duration
	MaxBackoff         time.Duration
	BackoffMultiplier  float64
	RetryableErrorCodes []string
}

type ErrorCategory string

const (
	ErrorCategoryNetwork     ErrorCategory = "network"
	ErrorCategoryAuth        ErrorCategory = "auth"
	ErrorCategoryConfig      ErrorCategory = "config"
	ErrorCategoryResource    ErrorCategory = "resource"
	ErrorCategoryTimeout     ErrorCategory = "timeout"
	ErrorCategoryUser        ErrorCategory = "user"
	ErrorCategorySystem      ErrorCategory = "system"
)

type CategorizedError struct {
	Category    ErrorCategory `json:"category"`
	Code        string        `json:"code"`
	Message     string        `json:"message"`
	Retryable   bool         `json:"retryable"`
	Fatal       bool         `json:"fatal"`
	Underlying  error        `json:"-"`
}

func (e *CategorizedError) Error() string {
	return fmt.Sprintf("[%s:%s] %s", e.Category, e.Code, e.Message)
}

func NewDeploymentEngine(config EngineConfig) *DeploymentEngine {
	if config.Workers <= 0 {
		config.Workers = 1
	}

	return &DeploymentEngine{
		db:           config.DB,
		logger:       config.Logger,
		publisher:    config.Publisher,
		stepRegistry: make(map[DeploymentStep]StepExecutor),
		workers:      config.Workers,
		shutdown:     make(chan struct{}),
		running:      false,
	}
}

func (e *DeploymentEngine) RegisterStep(step DeploymentStep, executor StepExecutor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stepRegistry[step] = executor
}

func (e *DeploymentEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("engine already running")
	}
	e.running = true
	e.mu.Unlock()

	e.logger.Info("Starting deployment engine", "workers", e.workers)

	// Start worker goroutines
	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go e.worker(ctx, i)
	}

	// Start cleanup goroutine
	e.wg.Add(1)
	go e.cleanup(ctx)

	return nil
}

func (e *DeploymentEngine) Stop(ctx context.Context) error {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = false
	e.mu.Unlock()

	e.logger.Info("Stopping deployment engine")
	close(e.shutdown)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		e.logger.Info("All workers stopped gracefully")
	case <-ctx.Done():
		e.logger.Warn("Timeout waiting for workers to stop")
		return ctx.Err()
	}

	return nil
}

func (e *DeploymentEngine) worker(ctx context.Context, workerID int) {
	defer e.wg.Done()

	logger := e.logger.With("worker_id", workerID)
	logger.Info("Worker started")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.shutdown:
			logger.Info("Worker shutting down")
			return
		case <-ctx.Done():
			logger.Info("Worker context cancelled")
			return
		case <-ticker.C:
			if err := e.processNextDeployment(ctx, logger); err != nil {
				logger.Error("Error processing deployment", "error", err)
			}
		}
	}
}

func (e *DeploymentEngine) cleanup(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-e.shutdown:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.cleanupStaleDeployments(ctx)
		}
	}
}

func (e *DeploymentEngine) processNextDeployment(ctx context.Context, logger *slog.Logger) error {
	// Get next pending deployment with row locking
	deployment, err := e.getNextPendingDeployment(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // No work to do
		}
		return fmt.Errorf("failed to get next deployment: %w", err)
	}

	if deployment == nil {
		return nil
	}

	deploymentLogger := logger.With(
		"deployment_id", deployment.ID,
		"customer_id", deployment.CustomerID,
		"project_id", deployment.ProjectID,
