// unkey/go/pkg/db/replica.go

package db

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
	"go.opentelemetry.io/otel/attribute"
)

// Replica wraps a standard SQL database connection and implements the gen.DBTX interface
// to enable interaction with the generated database code.
type Replica struct {
	mode         string
	db           *sql.DB // Underlying database connection
	mu           sync.RWMutex
	dsn          string
	logger       logging.Logger
	retryCount   int
	maxRetries   int
	reconnecting atomic.Bool
	hostname     string
}

// Ensure Replica implements the gen.DBTX interface
var _ DBTX = (*Replica)(nil)

// isUnhealthyTabletError checks if the error indicates an unhealthy tablet
func isUnhealthyTabletError(err error) bool {
	return strings.Contains(err.Error(), "no healthy tablet available")
}

// logReplicaHostname queries and logs the current replica hostname
func (r *Replica) logReplicaHostname(ctx context.Context) {
	var hostname string
	row := r.db.QueryRowContext(ctx, "SELECT @@hostname")
	if err := row.Scan(&hostname); err != nil {
		r.logger.Warn("failed to query replica hostname", "error", err.Error())
		return
	}

	r.hostname = hostname
	r.logger.Info("connected to replica", "hostname", hostname, "mode", r.mode)
}

// reconnect attempts to reconnect to the database
func (r *Replica) reconnect(ctx context.Context) error {
	// Check if already reconnecting to prevent concurrent reconnection attempts
	if !r.reconnecting.CompareAndSwap(false, true) {
		return nil
	}

	defer r.reconnecting.Store(false)

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if we've exceeded max retries
	if r.retryCount >= r.maxRetries {
		r.logger.Warn("max reconnection attempts reached, keeping existing connection",
			"mode", r.mode,
			"retries", r.retryCount,
			"maxRetries", r.maxRetries)
		return nil
	}

	r.retryCount++

	r.logger.Info("attempting to reconnect to database",
		"mode", r.mode,
		"attempt", r.retryCount,
		"maxRetries", r.maxRetries)

	// Close old connection
	if err := r.db.Close(); err != nil {
		r.logger.Warn("error closing old connection", "error", err.Error())
	}

	// Open new connection
	newDB, err := sql.Open("mysql", r.dsn)
	if err != nil {
		r.logger.Error("failed to reconnect", "error", err.Error(), "mode", r.mode)
		return err
	}

	// Test the connection
	if err := newDB.PingContext(ctx); err != nil {
		r.logger.Error("failed to ping new connection", "error", err.Error(), "mode", r.mode)
		newDB.Close()
		return err
	}

	// Replace the connection
	r.db = newDB

	// Reset retry count on successful reconnection
	r.retryCount = 0

	r.logger.Info("successfully reconnected to database", "mode", r.mode)

	// Log the new replica hostname
	r.logReplicaHostname(ctx)

	return nil
}

// handleQueryError checks if an error requires reconnection and attempts it
func (r *Replica) handleQueryError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	if isUnhealthyTabletError(err) {
		r.logger.Warn("detected unhealthy tablet error", "error", err.Error(), "mode", r.mode, "hostname", r.hostname)
		// Attempt reconnection (non-blocking)
		go r.reconnect(ctx)
	}

	return err
}

// ExecContext executes a SQL statement and returns a result summary.
// It's used for INSERT, UPDATE, DELETE statements that don't return rows.
func (r *Replica) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	ctx, span := tracing.Start(ctx, "ExecContext")
	defer span.End()
	span.SetAttributes(
		attribute.String("mode", r.mode),
		attribute.String("query", query),
	)

	// Acquire read lock to prevent reconnection during query
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Track metrics
	start := time.Now()
	result, err := r.db.ExecContext(ctx, query, args...)

	// Record latency and operation count
	duration := time.Since(start).Seconds()
	status := statusSuccess
	if err != nil {
		status = statusError
		tracing.RecordError(span, err)
		r.handleQueryError(ctx, err)
	}

	metrics.DatabaseOperationsLatency.WithLabelValues(r.mode, "exec", status).Observe(duration)
	metrics.DatabaseOperationsTotal.WithLabelValues(r.mode, "exec", status).Inc()

	return result, err
}

// PrepareContext prepares a SQL statement for later execution.
func (r *Replica) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	ctx, span := tracing.Start(ctx, "PrepareContext")
	defer span.End()
	span.SetAttributes(
		attribute.String("mode", r.mode),
		attribute.String("query", query),
	)

	// Acquire read lock to prevent reconnection during query
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Track metrics
	start := time.Now()
	//nolint:sqlclosecheck // Rows returned to caller, who must close them
	stmt, err := r.db.PrepareContext(ctx, query)

	// Record latency and operation count
	duration := time.Since(start).Seconds()
	status := statusSuccess
	if err != nil {
		status = statusError
		tracing.RecordError(span, err)
		r.handleQueryError(ctx, err)
	}

	metrics.DatabaseOperationsLatency.WithLabelValues(r.mode, "prepare", status).Observe(duration)
	metrics.DatabaseOperationsTotal.WithLabelValues(r.mode, "prepare", status).Inc()

	return stmt, err
}

// QueryContext executes a SQL query that returns rows.
func (r *Replica) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	ctx, span := tracing.Start(ctx, "QueryContext")
	defer span.End()
	span.SetAttributes(
		attribute.String("mode", r.mode),
		attribute.String("query", query),
	)

	// Acquire read lock to prevent reconnection during query
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Track metrics
	start := time.Now()
	//nolint:sqlclosecheck // Rows returned to caller, who must close them
	rows, err := r.db.QueryContext(ctx, query, args...)

	// Record latency and operation count
	duration := time.Since(start).Seconds()
	status := statusSuccess
	if err != nil {
		status = statusError
		tracing.RecordError(span, err)
		r.handleQueryError(ctx, err)
	}

	metrics.DatabaseOperationsLatency.WithLabelValues(r.mode, "query", status).Observe(duration)
	metrics.DatabaseOperationsTotal.WithLabelValues(r.mode, "query", status).Inc()

	return rows, err
}

// QueryRowContext executes a SQL query that returns a single row.
func (r *Replica) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	ctx, span := tracing.Start(ctx, "QueryRowContext")
	defer span.End()
	span.SetAttributes(
		attribute.String("mode", r.mode),
		attribute.String("query", query),
	)

	// Acquire read lock to prevent reconnection during query
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Track metrics
	start := time.Now()
	row := r.db.QueryRowContext(ctx, query, args...)

	// Record latency and operation count
	duration := time.Since(start).Seconds()
	// QueryRowContext doesn't return an error, but we can still track timing
	status := statusSuccess

	metrics.DatabaseOperationsLatency.WithLabelValues(r.mode, "query_row", status).Observe(duration)
	metrics.DatabaseOperationsTotal.WithLabelValues(r.mode, "query_row", status).Inc()

	return row
}

// Begin starts a transaction and returns it.
// This method provides a way to use the Replica in transaction-based operations.
func (r *Replica) Begin(ctx context.Context) (DBTx, error) {
	ctx, span := tracing.Start(ctx, "Begin")
	defer span.End()
	span.SetAttributes(attribute.String("mode", r.mode))

	// Acquire read lock to prevent reconnection during query
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Track metrics
	start := time.Now()
	tx, err := r.db.BeginTx(ctx, nil)

	// Record latency and operation count
	duration := time.Since(start).Seconds()
	status := statusSuccess
	if err != nil {
		status = statusError
		tracing.RecordError(span, err)
		r.handleQueryError(ctx, err)
	}

	metrics.DatabaseOperationsLatency.WithLabelValues(r.mode, "begin", status).Observe(duration)
	metrics.DatabaseOperationsTotal.WithLabelValues(r.mode, "begin", status).Inc()

	if err != nil {
		return nil, err
	}

	// Wrap the transaction with tracing
	return WrapTxWithContext(tx, r.mode+"_tx", ctx), nil
}
