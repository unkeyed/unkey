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
// Goroutines registered with Go() start immediately.
// Run() blocks until OS signals or errors, then triggers cleanup.
type Runner struct {
	mu           sync.Mutex
	cleanups     []ShutdownFunc
	running      bool
	shuttingDown bool
	health       *healthState

	wg     sync.WaitGroup
	errCh  chan error
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new Runner.
func New() *Runner {
	ctx, cancel := context.WithCancel(context.Background())
	return &Runner{
		mu:           sync.Mutex{},
		cleanups:     nil,
		running:      false,
		shuttingDown: false,
		health:       newHealthState(),
		wg:           sync.WaitGroup{},
		errCh:        make(chan error, 1),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Go starts a long-running goroutine immediately.
// The function should return nil on clean exit or an error to trigger shutdown.
// It must respect ctx cancellation for graceful stopping.
func (r *Runner) Go(fn RunFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.shuttingDown {
		return
	}

	r.wg.Go(func() {
		err := fn(r.ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
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
	if r.running {
		r.mu.Unlock()
		return errors.New("runner is already running")
	}
	r.running = true
	r.health.started.Store(true)
	if cfg.ReadinessTimeout > 0 {
		r.health.checkTimeout = cfg.ReadinessTimeout
	}
	r.mu.Unlock()

	sigCtx, stop := signal.NotifyContext(ctx, cfg.Signals...)
	defer stop()

	var cause error
	select {
	case <-sigCtx.Done():
	case cause = <-r.errCh:
	}

	r.cancel()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancelShutdown()

	shutdownErrs := r.shutdown(shutdownCtx)

	r.wg.Wait()

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
	r.health.shuttingDown.Store(true)
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
