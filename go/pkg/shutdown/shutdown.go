package shutdown

import (
	"context"
	"fmt"
	"sync"
)

// ShutdownCtx is a function type for context-aware shutdown handlers.
// It receives a context that may signal timeout or cancellation, and returns
// an error if the shutdown operation fails.
type ShutdownCtx func(ctx context.Context) error

// Shutdown is a function type for simple shutdown handlers that don't require
// a context. It returns an error if the shutdown operation fails.
type Shutdown func() error

// ShutdownError represents a collection of errors that occurred during shutdown.
// It implements the error interface.
type ShutdownError struct {
	Errors []error
}

// Error implements the error interface for ShutdownError.
func (se *ShutdownError) Error() string {
	if len(se.Errors) == 1 {
		return fmt.Sprintf("shutdown error: %v", se.Errors[0])
	}
	return fmt.Sprintf("shutdown errors (%d): %v", len(se.Errors), se.Errors)
}

// Is implements error interface compatibility with errors.Is
func (se *ShutdownError) Is(target error) bool {
	if _, ok := target.(*ShutdownError); ok {
		return true
	}
	return false
}

// Unwrap returns the list of errors to support errors.As and errors.Is
func (se *ShutdownError) Unwrap() []error {
	return se.Errors
}

// Shutdowns maintains a registry of functions to be called during shutdown.
// This implementation is concurrency-safe and can be safely used from multiple
// goroutines simultaneously.
type Shutdowns struct {
	mu        sync.RWMutex
	functions []ShutdownCtx
	// shuttingDown helps prevent new registrations during shutdown
	shuttingDown bool
}

// New creates and returns a new Shutdowns instance with an empty list of
// shutdown functions.
//
// Example:
//
//	shutdowns := shutdown.New()
func New() *Shutdowns {
	return &Shutdowns{
		mu:           sync.RWMutex{},
		functions:    []ShutdownCtx{},
		shuttingDown: false,
	}
}

// Register adds one or more simple shutdown functions to the registry. The functions will
// be called during the Shutdown phase in reverse order of registration.
//
// This method is concurrency-safe and can be called from multiple goroutines.
// However, attempting to register functions during an ongoing shutdown will have no effect.
//
// Example:
//
//	shutdowns.Register(
//	    func() error { return db.Close() },
//	    func() error { return cache.Close() },
//	)
//
// See also: [Shutdowns.RegisterCtx] for context-aware shutdown functions.
func (s *Shutdowns) Register(fns ...Shutdown) {
	if len(fns) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Don't register new functions if we're already shutting down
	if s.shuttingDown {
		return
	}

	for _, fn := range fns {
		// Create a copy of fn to avoid closure issues
		localFn := fn
		s.functions = append(s.functions, func(context.Context) error {
			return localFn()
		})
	}
}

// RegisterCtx adds one or more context-aware shutdown functions to the registry. The
// functions will receive the context passed to Shutdown and will be called
// in reverse order of registration.
//
// This method is concurrency-safe and can be called from multiple goroutines.
// However, attempting to register functions during an ongoing shutdown will have no effect.
//
// Example:
//
//	shutdowns.RegisterCtx(
//	    func(ctx context.Context) error { return server.ShutdownWithContext(ctx) },
//	    func(ctx context.Context) error { return worker.StopWithContext(ctx) },
//	)
//
// See also: [Shutdowns.Register] for simpler shutdown functions that don't need context.
func (s *Shutdowns) RegisterCtx(fns ...ShutdownCtx) {
	if len(fns) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Don't register new functions if we're already shutting down
	if s.shuttingDown {
		return
	}

	s.functions = append(s.functions, fns...)
}

// Shutdown executes all registered shutdown functions in reverse order
// (last registered, first executed). Each function receives the provided
// context, which can be used to signal timeouts or cancellation.
//
// This method collects all errors that occur during shutdown and returns them
// as a ShutdownError. All registered shutdown functions will be called, even if
// some of them return errors.
//
// This method is concurrency-safe and will prevent race conditions with
// concurrent registrations. Only one shutdown sequence can be active at a time.
// Additional calls to Shutdown while a shutdown is in progress will return
// immediately without errors.
//
// If no shutdown functions are registered, Shutdown returns immediately.
//
// Example with timeout and error handling:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	errs := shutdowns.Shutdown(ctx)
//	for i, e := range errs {
//	  log.Printf("Shutdown error %d: %v", i+1, e)
//	}
//
// Performance note: This method executes shutdown functions sequentially,
// not in parallel, which may impact shutdown time for many slow operations.
func (s *Shutdowns) Shutdown(ctx context.Context) []error {
	// Take a snapshot of the functions to avoid holding the lock during execution
	// and to prevent race conditions with new registrations
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shuttingDown {
		return []error{} // Shutdown is already in progress
	}

	s.shuttingDown = true

	// If no functions are registered, return early
	if len(s.functions) == 0 {
		return []error{}
	}

	var shutdownErrs []error

	// Execute the functions in reverse order
	for i := len(s.functions) - 1; i >= 0; i-- {
		f := s.functions[i]
		if err := f(ctx); err != nil {
			shutdownErrs = append(shutdownErrs, err)
		}
	}

	if len(shutdownErrs) > 0 {
		return shutdownErrs
	}

	return []error{}

}
