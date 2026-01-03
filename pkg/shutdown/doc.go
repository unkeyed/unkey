// Package shutdown provides a way to manage graceful shutdown of resources.
//
// This package implements a simple system to register cleanup functions that will
// be executed in reverse order of registration when the program needs to terminate.
// It's particularly useful for closing resources like database connections, network
// listeners, or file handles in a controlled manner.
//
// Basic usage:
//
//	// Create a new shutdown manager
//	shutdowns := shutdown.New()
//
//	// Register resources that need cleanup
//	shutdowns.Register(func() error {
//	    return db.Close()
//	})
//
//	// Register context-aware cleanup functions
//	shutdowns.RegisterCtx(func(ctx context.Context) error {
//	    return server.ShutdownWithContext(ctx)
//	})
//
//	// When it's time to shut down, call:
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	if err := shutdowns.Shutdown(ctx); err != nil {
//	    log.Printf("Error during shutdown: %v", err)
//	}
package shutdown
