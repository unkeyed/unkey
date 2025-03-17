package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	maxBufferSize int = 50000
	maxBatchSize  int = 10000
)

var (
	telemetry  *TelemetryConfig
	inFlight   sync.WaitGroup // incoming requests
	httpClient *http.Client   // shared http client
)

func main() {
	config, err := LoadConfig()
	if err != nil {
		config.Logger.Error("failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	httpClient = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        25,
			MaxIdleConnsPerHost: 25,
			IdleConnTimeout:     60 * time.Second,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cleanup func(context.Context) error
	telemetry, cleanup, err = setupTelemetry(ctx, config)
	if err != nil {
		config.Logger.Error("failed to setup telemetry", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := cleanup(cleanupCtx); err != nil {
			log.Printf("failed to cleanup telemetry: %v", err)
		}
	}()

	if telemetry != nil && telemetry.LogHandler != nil {
		config.Logger = slog.New(telemetry.LogHandler)

		slog.SetDefault(config.Logger)
	}

	config.Logger.Info(fmt.Sprintf("%s starting", config.ServiceName),
		"flush_interval", config.FlushInterval.String())

	requiredAuthorization := "Basic " + base64.StdEncoding.EncodeToString([]byte(config.BasicAuth))

	buffer := make(chan *Batch, maxBufferSize)

	// blocks until we've persisted everything and the process may stop
	done := startBufferProcessor(ctx, buffer, config, telemetry)

	http.HandleFunc("/v1/liveness", func(w http.ResponseWriter, r *http.Request) {
		_, span := telemetry.Tracer.Start(r.Context(), "liveness_check")
		defer span.End()

		span.SetAttributes(
			attribute.String("method", r.Method),
			attribute.String("path", r.URL.Path),
		)

		w.Write([]byte("ok"))
		span.SetStatus(codes.Ok, "")
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		inFlight.Add(1)
		defer inFlight.Done()

		ctx, span := telemetry.Tracer.Start(r.Context(), "handle_request")
		defer span.End()

		telemetry.Metrics.RequestCounter.Add(ctx, 1)

		if r.Header.Get("Authorization") != requiredAuthorization {
			telemetry.Metrics.ErrorCounter.Add(ctx, 1)
			config.Logger.Error("invalid authorization header",
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent())

			span.RecordError(err)
			span.SetStatus(codes.Error, "unauthorized")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		span.SetAttributes(
			attribute.String("method", r.Method),
			attribute.String("path", r.URL.Path),
			attribute.String("remote_addr", r.RemoteAddr),
		)

		query := r.URL.Query().Get("query")
		span.SetAttributes(attribute.String("query", query))

		if query == "" || !strings.HasPrefix(strings.ToLower(query), "insert into") {
			telemetry.Metrics.ErrorCounter.Add(ctx, 1)
			config.Logger.Warn("invalid query",
				"query", query,
				"remote_addr", r.RemoteAddr)

			span.SetStatus(codes.Error, "wrong query")
			http.Error(w, "wrong query", http.StatusBadRequest)
			return
		}

		params := r.URL.Query()
		params.Del("query_id")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			telemetry.Metrics.ErrorCounter.Add(ctx, 1)
			config.Logger.Error("failed to read request body",
				"error", err.Error(),
				"remote_addr", r.RemoteAddr)

			span.RecordError(err)
			span.SetStatus(codes.Error, "cannot read body")
			http.Error(w, "cannot read body", http.StatusInternalServerError)
			return
		}
		rows := strings.Split(string(body), "\n")

		config.Logger.Debug("received insert request",
			"row_count", len(rows),
			"table", strings.Split(query, " ")[2])

		buffer <- &Batch{
			Params: params,
			Rows:   rows,
			Table:  strings.Split(query, " ")[2],
		}

		w.Write([]byte("ok"))
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(
			attribute.Int("row_count", len(rows)),
			attribute.String("table", strings.Split(query, " ")[2]),
		)
	})

	// Setup signal handling
	signalCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start HTTP server in a goroutine
	server := &http.Server{Addr: fmt.Sprintf(":%s", config.ListenerPort)}
	go func() {
		config.Logger.Info("server listening", "port", config.ListenerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			config.Logger.Error("failed to start server", "error", err.Error())
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-signalCtx.Done()
	config.Logger.Info("shutdown signal received")

	// Create a timeout context for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Start a goroutine to track in-flight requests
	shutdownComplete := make(chan struct{})
	go func() {
		inFlight.Wait()
		close(shutdownComplete)
	}()

	// Attempt graceful shutdown
	config.Logger.Info("shutting down server")
	if err := server.Shutdown(shutdownCtx); err != nil {
		config.Logger.Error("server shutdown error", "error", err.Error())
	}

	// Close the buffer channel and wait for processing to finish
	close(buffer)
	<-done
	config.Logger.Info("graceful shutdown complete")
}
