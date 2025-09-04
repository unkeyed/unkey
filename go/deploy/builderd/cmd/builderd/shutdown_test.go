package main

import (
	"log/slog"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/unkeyed/unkey/go/deploy/builderd/internal/config"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/observability"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/service"
)

// TestShutdownRace tests the graceful shutdown sequence for race conditions
func TestShutdownRace(t *testing.T) {
	// Create a minimal logger for testing
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create minimal config
	cfg := &config.Config{
		Server: config.ServerConfig{
			ShutdownTimeout: 5 * time.Second,
		},
	}

	// Create mock servers
	server := &http.Server{
		Addr: ":0", // Use any available port
	}
	promServer := &http.Server{
		Addr: ":0", // Use any available port
	}

	// Create minimal providers (can be nil for this test)
	var providers *observability.Providers

	// Create minimal builder service (can be nil for this test)
	var builderService *service.BuilderService

	// Shutdown coordination variables
	var shutdownStarted int64
	var shutdownMutex sync.Mutex

	// Test concurrent shutdown attempts to detect race conditions
	const numGoroutines = 10
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Reset shutdown state before each attempt
			atomic.StoreInt64(&shutdownStarted, 0)

			// Call performGracefulShutdown concurrently
			performGracefulShutdown(
				logger.With("goroutine_id", id),
				server,
				promServer,
				providers,
				builderService,
				&shutdownStarted,
				&shutdownMutex,
				cfg.Server.ShutdownTimeout,
			)
		}(i)
	}

	// Wait for all goroutines to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Set a reasonable timeout for the test
	select {
	case <-done:
		t.Log("All shutdown goroutines completed successfully")
	case <-time.After(10 * time.Second):
		t.Fatal("Test timed out - likely a deadlock or race condition")
	}
}

// TestShutdownSequence tests the shutdown sequence with realistic components
func TestShutdownSequence(t *testing.T) {
	// Create a logger for testing
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create config
	cfg := &config.Config{
		Server: config.ServerConfig{
			ShutdownTimeout: 2 * time.Second,
		},
		OpenTelemetry: config.OpenTelemetryConfig{
			Enabled: false, // Disable OTel to avoid external dependencies
		},
	}

	// Create HTTP servers
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    "127.0.0.1:0", // Use any available port
		Handler: mux,
	}

	promMux := http.NewServeMux()
	promMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# Metrics"))
	})

	promServer := &http.Server{
		Addr:    "127.0.0.1:0", // Use any available port
		Handler: promMux,
	}

	// Start servers in background
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("HTTP server error: %v", err)
		}
	}()

	go func() {
		if err := promServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("Prometheus server error: %v", err)
		}
	}()

	// Give servers time to start
	time.Sleep(100 * time.Millisecond)

	// Test graceful shutdown
	var shutdownStarted int64
	var shutdownMutex sync.Mutex

	start := time.Now()
	performGracefulShutdown(
		logger,
		server,
		promServer,
		nil, // No OTel providers
		nil, // No builder service
		&shutdownStarted,
		&shutdownMutex,
		cfg.Server.ShutdownTimeout,
	)
	duration := time.Since(start)

	t.Logf("Shutdown completed in %v", duration)

	// Verify shutdown completed within reasonable time
	if duration > 5*time.Second {
		t.Errorf("Shutdown took too long: %v", duration)
	}
}

// TestShutdownTimeout tests that shutdown respects the configured timeout
func TestShutdownTimeout(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create a server that will hang during shutdown
	hangingServer := &http.Server{
		Addr: "127.0.0.1:0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate long-running request
			time.Sleep(10 * time.Second)
			w.WriteHeader(http.StatusOK)
		}),
	}

	// Very short timeout to test timeout behavior
	shortTimeout := 100 * time.Millisecond

	var shutdownStarted int64
	var shutdownMutex sync.Mutex

	start := time.Now()
	performGracefulShutdown(
		logger,
		hangingServer,
		nil, // No prom server
		nil, // No OTel providers
		nil, // No builder service
		&shutdownStarted,
		&shutdownMutex,
		shortTimeout,
	)
	duration := time.Since(start)

	t.Logf("Shutdown with timeout completed in %v", duration)

	// Should complete within a reasonable time even with hanging components
	if duration > 5*time.Second {
		t.Errorf("Shutdown took too long even with timeout: %v", duration)
	}
}
