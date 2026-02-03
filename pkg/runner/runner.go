package runner

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// RunFunc is a long-running goroutine that should return nil on clean exit
// or a non-nil error if it fails and should trigger shutdown.
// It must respect ctx cancellation for graceful stopping.
type RunFunc func(ctx context.Context) error

// ShutdownFunc is a context-aware cleanup handler.
type ShutdownFunc func(ctx context.Context) error

// CloseFunc is a simple cleanup handler.
type CloseFunc func() error

type runnerState uint8

const (
	runnerStateIdle runnerState = iota
	runnerStateRunning
	runnerStateShuttingDown
)

// Runner manages long-running goroutines and cleanup handlers.
// Goroutines registered with Go() start immediately.
// Run() blocks until OS signals or errors, then triggers cleanup.
type Runner struct {
	mu       sync.Mutex
	cleanups []ShutdownFunc
	state    runnerState
	health   *healthState
	logger   logging.Logger

	wg     sync.WaitGroup
	errCh  chan error
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new Runner.
func New(logger logging.Logger) *Runner {

	ctx, cancel := context.WithCancel(context.Background())
	return &Runner{
		mu:       sync.Mutex{},
		cleanups: []ShutdownFunc{},
		state:    runnerStateIdle,
		health:   newHealthState(),
		logger:   logger.With(slog.String("component", "runner")),
		wg:       sync.WaitGroup{},
		errCh:    make(chan error, 1),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// SetLogger replaces the logger used by the runner.
func (r *Runner) SetLogger(logger logging.Logger) {
	if logger == nil {
		panic("runner: logger is required")
	}

	r.mu.Lock()
	r.logger = logger.With(slog.String("component", "runner"))
	r.mu.Unlock()
}

// Recover logs any panic that occurs in the calling goroutine.
func (r *Runner) Recover() {
	if recovered := recover(); recovered != nil {
		r.logger.Error("panic",
			"panic", recovered,
			"stack", string(debug.Stack()),
		)
	}
}

// Go starts a long-running goroutine immediately.
// The function should return nil on clean exit or an error to trigger shutdown.
// It must respect ctx cancellation for graceful stopping.
func (r *Runner) Go(fn RunFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state == runnerStateShuttingDown {
		return
	}

	r.wg.Go(func() {
		err := fn(r.ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			r.logger.Error("runner task failed", "error", err)
			select {
			case r.errCh <- err:
			default:
			}
		}
	})

}

// Defer adds cleanup functions to be called during shutdown.
// They are called in reverse order of registration.
func (r *Runner) Defer(fns ...CloseFunc) {
	if len(fns) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state == runnerStateShuttingDown {
		return
	}
	for _, fn := range fns {
		localFn := fn
		r.cleanups = append(r.cleanups, func(context.Context) error {
			return localFn()
		})
	}
}

// DeferCtx adds context-aware cleanup functions to be called during shutdown.
// They are called in reverse order of registration.
func (r *Runner) DeferCtx(fns ...ShutdownFunc) {
	if len(fns) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state == runnerStateShuttingDown {
		return
	}
	r.cleanups = append(r.cleanups, fns...)
}

// RegisterCtx is an alias for DeferCtx for compatibility with otel.Registrar interface.
func (r *Runner) RegisterCtx(fns ...func(ctx context.Context) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state == runnerStateShuttingDown {
		return
	}
	for _, fn := range fns {
		r.cleanups = append(r.cleanups, fn)
	}
}

// RunConfig configures the Run behavior.
type RunConfig struct {
	Timeout          time.Duration
	Signals          []os.Signal
	ReadinessTimeout time.Duration
}

// RunOption configures Run.
type RunOption func(*RunConfig)

// WithTimeout sets the shutdown timeout (default 30s).
func WithTimeout(d time.Duration) RunOption {
	return func(c *RunConfig) {
		c.Timeout = d
	}
}

// WithSignals sets the OS signals to listen for (default SIGINT, SIGTERM).
func WithSignals(sigs ...os.Signal) RunOption {
	return func(c *RunConfig) {
		c.Signals = sigs
	}
}

// Wait blocks until:
// - An OS signal is received, OR
// - The parent context is canceled, OR
// - Any goroutine returns a non-nil error
//
// It then cancels all goroutines, runs cleanup handlers in reverse order,
// and returns any errors.
func (r *Runner) Wait(ctx context.Context, opts ...RunOption) error {
	cfg := RunConfig{
		Timeout:          30 * time.Second,
		Signals:          []os.Signal{syscall.SIGINT, syscall.SIGTERM},
		ReadinessTimeout: 0,
	}
	for _, o := range opts {
		o(&cfg)
	}

	r.mu.Lock()
	if r.state == runnerStateRunning {
		r.mu.Unlock()
		r.logger.Error("runner already running")
		return errors.New("runner is already running")
	}
	r.state = runnerStateRunning
	r.health.started.Store(true)
	if cfg.ReadinessTimeout > 0 {
		r.health.checkTimeout = cfg.ReadinessTimeout
	}
	r.mu.Unlock()

	r.logger.Info("runner wait started",
		"timeout", cfg.Timeout,
		"readiness_timeout", cfg.ReadinessTimeout,
	)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, cfg.Signals...)
	defer signal.Stop(sigCh)

	var cause error
	reason := ""
	select {
	case sig := <-sigCh:
		reason = "signal"
		r.logger.Info("runner shutdown signal received", "signal", sig.String())
	case <-ctx.Done():
		reason = "context"
		r.logger.Info("runner context canceled", "error", ctx.Err())
	case cause = <-r.errCh:
		reason = "error"
	}

	if cause != nil {
		r.logger.Error("runner shutdown triggered by task error", "error", cause)
	}

	r.logger.Info("runner shutting down", "reason", reason)

	r.cancel()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancelShutdown()

	shutdownErrs := r.shutdown(shutdownCtx)

	r.wg.Wait()

	if cause != nil && len(shutdownErrs) > 0 {
		finalErr := errors.Join(append([]error{cause}, shutdownErrs...)...)
		r.logger.Error("runner shutdown failed", "error", finalErr)
		return finalErr
	}
	if cause != nil {
		r.logger.Error("runner shutdown failed", "error", cause)
		return cause
	}
	if len(shutdownErrs) > 0 {
		finalErr := errors.Join(shutdownErrs...)
		r.logger.Error("runner shutdown failed", "error", finalErr)
		return finalErr
	}

	r.logger.Info("runner shutdown complete")
	return nil
}

func (r *Runner) shutdown(ctx context.Context) []error {
	r.mu.Lock()
	if r.state == runnerStateShuttingDown {
		r.mu.Unlock()
		return nil
	}
	r.state = runnerStateShuttingDown
	r.health.shuttingDown.Store(true)
	cleanups := make([]ShutdownFunc, len(r.cleanups))
	copy(cleanups, r.cleanups)
	r.mu.Unlock()

	var errs []error
	for i := len(cleanups) - 1; i >= 0; i-- {
		if err := cleanups[i](ctx); err != nil {
			r.logger.Error("runner cleanup failed", "error", err)
			errs = append(errs, err)
		}
	}
	return errs
}
