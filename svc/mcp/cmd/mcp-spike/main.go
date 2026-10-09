// Command mcp-spike serves the WorkOS AuthKit staging spike.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/svc/mcp"
)

func main() {
	cfg, err := mcp.ConfigFromEnv(os.Getenv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mcp-spike: %s\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		Level:       slog.LevelInfo,
		ReplaceAttr: nil,
	}))
	handler, err := mcp.NewHandler(cfg, logger)
	if err != nil {
		logger.Error("mcp spike configuration rejected", "error", err.Error())
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	logger.Info("mcp spike listening",
		"addr", cfg.ListenAddr,
		"issuer", cfg.Issuer,
		"public_base_url", cfg.PublicBaseURL,
		"log_claims", cfg.LogClaims,
	)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("mcp spike stopped", "error", err.Error())
		os.Exit(1)
	}
}
