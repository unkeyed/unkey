package runner

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// RunFunc is a long-running goroutine that should return nil on clean exit
// or a non-nil error if it fails and should trigger shutdown.
// It must respect ctx cancellation for graceful stopping.
type RunFunc func(ctx context.Context) error

// ShutdownFunc is a context-aware cleanup handler.
type ShutdownFunc func(ctx context.Context) error

// CloseFunc is a simple cleanup handler.
type CloseFunc func() error

// Runner manages long-running goroutines and cleanup handlers.
// It starts all registered goroutines, waits for OS signals or errors,
// then triggers cleanup in reverse registration order.
type Runner struct {
	mu           sync.Mutex
	tasks        []RunFunc
	cleanups     []ShutdownFunc
	running      bool
	shuttingDown bool
}

// New creates a new Runner.
func New() *Runner {
	return &Runner{
		mu:           sync.Mutex{},
		tasks:        nil,
		cleanups:     nil,
		running:      false,
		shuttingDown: false,
	}
}

// Go registers a long-running goroutine to be started when Run is called.
// The function should return nil on clean exit or an error to trigger shutdown.
// It must respect ctx cancellation for graceful stopping.
func (r *Runner) Go(fn RunFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running || r.shuttingDown {
		return
	}
	r.tasks = append(r.tasks, fn)
}

// Defer adds cleanup functions to be called during shutdown.
// They are called in reverse order of registration.
func (r *Runner) Defer(fns ...CloseFunc) {
	r.Register(fns...)
}

// Register is an alias for Defer for compatibility with other interfaces.
func (r *Runner) Register(fns ...CloseFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.shuttingDown {
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
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.shuttingDown {
		return
	}
	r.cleanups = append(r.cleanups, fns...)
}

// RegisterCtx is an alias for DeferCtx for compatibility with otel.Registrar interface.
func (r *Runner) RegisterCtx(fns ...func(ctx context.Context) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.shuttingDown {
		return
	}
	for _, fn := range fns {
		r.cleanups = append(r.cleanups, fn)
	}
}

// RunConfig configures the Run behavior.
type RunConfig struct {
	Timeout time.Duration
	Signals []os.Signal
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

// Run starts all registered goroutines and blocks until:
// - An OS signal is received, OR
// - The parent context is canceled, OR
// - Any goroutine returns a non-nil error
//
// It then cancels all goroutines, runs cleanup handlers in reverse order,
// and returns any errors.
func (r *Runner) Run(ctx context.Context, opts ...RunOption) error {
	cfg := RunConfig{
		Timeout: 30 * time.Second,
		Signals: []os.Signal{syscall.SIGINT, syscall.SIGTERM},
	}
	for _, o := range opts {
		o(&cfg)
	}

	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return errors.New("runner is already running")
	}
	r.running = true
	tasks := make([]RunFunc, len(r.tasks))
	copy(tasks, r.tasks)
	r.mu.Unlock()

	sigCtx, stop := signal.NotifyContext(ctx, cfg.Signals...)
	defer stop()

	runCtx, cancel := context.WithCancel(sigCtx)
	defer cancel()

	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	for _, task := range tasks {
		wg.Add(1)
		go func(fn RunFunc) {
			defer wg.Done()
			if err := fn(runCtx); err != nil && !errors.Is(err, context.Canceled) {
				select {
				case errCh <- err:
				default:
				}
			}
		}(task)
	}

	var cause error
	select {
	case <-sigCtx.Done():
	case cause = <-errCh:
	}

	cancel()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancelShutdown()

	// Wait for all tasks to complete before we shutdown, as those tasks might still rely on shared resources.
	wg.Wait()
	shutdownErrs := r.shutdown(shutdownCtx)

	if cause != nil && len(shutdownErrs) > 0 {
		return errors.Join(append([]error{cause}, shutdownErrs...)...)
	}
	if cause != nil {
		return cause
	}
	if len(shutdownErrs) > 0 {
		return errors.Join(shutdownErrs...)
	}
	return nil
}

func (r *Runner) shutdown(ctx context.Context) []error {
	r.mu.Lock()
	if r.shuttingDown {
		r.mu.Unlock()
		return nil
	}
	r.shuttingDown = true
	cleanups := make([]ShutdownFunc, len(r.cleanups))
	copy(cleanups, r.cleanups)
	r.mu.Unlock()

	var errs []error
	for i := len(cleanups) - 1; i >= 0; i-- {
		if err := cleanups[i](ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
