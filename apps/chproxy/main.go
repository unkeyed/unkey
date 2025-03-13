package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
)

const (
	LOG_TICKER_SAMPLE_RATE = 10 // Only log every 10th ticker flush
)

var (
	telemetry *TelemetryConfig
)

type Batch struct {
	Rows   []string
	Params url.Values
}

func persist(ctx context.Context, batch *Batch, config *Config) error {
	ctx, span := telemetry.Tracer.Start(ctx, "persist_batch")
	defer span.End()

	if len(batch.Rows) == 0 {
		return nil
	}

	telemetry.Metrics.BatchCounter.Add(ctx, 1)
	telemetry.Metrics.RowCounter.Add(ctx, int64(len(batch.Rows)))

	span.SetAttributes(
		attribute.Int("rows", len(batch.Rows)),
		attribute.String("query", batch.Params.Get("query")),
	)

	u, err := url.Parse(config.ClickhouseURL)
	if err != nil {
		telemetry.Metrics.ErrorCounter.Add(ctx, 1)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	u.RawQuery = batch.Params.Encode()

	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), strings.NewReader(strings.Join(batch.Rows, "\n")))
	if err != nil {
		telemetry.Metrics.ErrorCounter.Add(ctx, 1)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	req.Header.Add("Content-Type", "text/plain")
	username := u.User.Username()
	password, ok := u.User.Password()
	if !ok {
		err := fmt.Errorf("password not set")
		telemetry.Metrics.ErrorCounter.Add(ctx, 1)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	req.SetBasicAuth(username, password)

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		telemetry.Metrics.ErrorCounter.Add(ctx, 1)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		config.Logger.Info("rows persisted",
			"count", len(batch.Rows),
			"query", batch.Params.Get("query"))
	} else {
		telemetry.Metrics.ErrorCounter.Add(ctx, 1)
		body, err := io.ReadAll(res.Body)
		if err != nil {
			span.RecordError(err)
			return err
		}

		errorMsg := string(body)
		span.SetStatus(codes.Error, errorMsg)
		span.RecordError(fmt.Errorf("HTTP %d: %s", res.StatusCode, errorMsg))

		config.Logger.Error("unable to persist batch",
			"response", errorMsg,
			"status_code", res.StatusCode,
			"query", batch.Params.Get("query"))
	}
	return nil
}

func main() {
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cleanup func(context.Context) error
	telemetry, cleanup, err = setupTelemetry(ctx, config)
	if err != nil {
		log.Fatalf("failed to setup telemetry: %v", err)
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
		"max_buffer_size", config.MaxBufferSize,
		"max_batch_size", config.MaxBatchSize,
		"flush_interval", config.FlushInterval.String())

	requiredAuthorization := "Basic " + base64.StdEncoding.EncodeToString([]byte(config.BasicAuth))

	buffer := make(chan *Batch, config.MaxBufferSize)
	// blocks until we've persisted everything and the process may stop
	done := make(chan bool)

	go func() {
		buffered := 0
		batchesByParams := make(map[string]*Batch)
		ticker := time.NewTicker(config.FlushInterval)
		tickerCount := 0

		flushAndReset := func(ctx context.Context, reason string) {
			ctx, span := telemetry.Tracer.Start(ctx, "flush_batches")
			defer span.End()

			startTime := time.Now()

			// Record metrics
			telemetry.Metrics.FlushCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("reason", reason)))

			span.SetAttributes(
				attribute.Int("batch_count", len(batchesByParams)),
				attribute.Int("buffered_rows", buffered),
				attribute.String("reason", reason),
			)

			// We'll sample logs for ticker-based flushes
			shouldLog := true
			if reason == "ticker" {
				tickerCount++
				// Only log every LOG_TICKER_SAMPLE_RATE times
				shouldLog = (tickerCount%LOG_TICKER_SAMPLE_RATE == 0)
			}

			// Only log if we should based on sampling
			if shouldLog {
				config.Logger.Info("flushing batches",
					"reason", reason,
					"batch_count", len(batchesByParams),
					"buffered_rows", buffered)
			}

			for _, batch := range batchesByParams {
				err := persist(ctx, batch, config)
				if err != nil {
					// Always log errors regardless of sampling
					config.Logger.Error("error flushing batch",
						"error", err.Error(),
						"query", batch.Params.Get("query"))
				}
			}

			duration := time.Since(startTime).Seconds()
			telemetry.Metrics.FlushDuration.Record(ctx, duration,
				metric.WithAttributes(attribute.String("reason", reason)))

			span.SetAttributes(attribute.Float64("duration_seconds", duration))

			buffered = 0
			SetBufferSize(0)
			batchesByParams = make(map[string]*Batch)
			ticker.Reset(config.FlushInterval)
		}

		for {
			select {
			case b, ok := <-buffer:
				if !ok {
					config.Logger.Info("buffer channel closed, flushing remaining batches")
					flushAndReset(ctx, "shutdown")
					done <- true
					return
				}

				params := b.Params.Encode()
				batch, ok := batchesByParams[params]
				if !ok {
					batchesByParams[params] = b
					config.Logger.Debug("new batch type received",
						"query", b.Params.Get("query"))
				} else {
					batch.Rows = append(batch.Rows, b.Rows...)
				}

				buffered += len(b.Rows)
				SetBufferSize(int64(buffered))

				if buffered >= config.MaxBatchSize {
					config.Logger.Info("flushing due to max batch size",
						"buffered_rows", buffered,
						"max_size", config.MaxBatchSize)
					flushAndReset(ctx, "max_size")
				}
			case <-ticker.C:
				config.Logger.Debug("flushing due to max batch size",
					"buffered_rows", buffered,
					"max_size", config.MaxBatchSize)
				flushAndReset(ctx, "ticker")
			}
		}
	}()

	http.HandleFunc("/v1/liveness", func(w http.ResponseWriter, r *http.Request) {
		_, span := telemetry.Tracer.Start(r.Context(), "liveness_check")
		defer span.End()

		w.Write([]byte("ok"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx, span := telemetry.Tracer.Start(r.Context(), "handle_request")
		defer span.End()

		span.SetAttributes(
			attribute.String("method", r.Method),
			attribute.String("path", r.URL.Path),
			attribute.String("remote_addr", r.RemoteAddr),
		)

		if r.Header.Get("Authorization") != requiredAuthorization {
			telemetry.Metrics.ErrorCounter.Add(ctx, 1)
			log.Println("invalid authorization header, expected", requiredAuthorization, r.Header.Get("Authorization"))
			config.Logger.Error("invalid authorization header",
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent())

			span.SetStatus(codes.Error, "unauthorized")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		query := r.URL.Query().Get("query")
		span.SetAttributes(attribute.String("query", query))

		if query == "" || !strings.HasPrefix(strings.ToLower(query), "insert into") {
			telemetry.Metrics.ErrorCounter.Add(ctx, 1)
			config.Logger.Warn("Invalid query",
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

		span.SetAttributes(attribute.Int("row_count", len(rows)))

		config.Logger.Debug("received insert request",
			"row_count", len(rows),
			"table", strings.Split(query, " ")[2])

		buffer <- &Batch{
			Params: params,
			Rows:   rows,
		}

		w.Write([]byte("ok"))
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
