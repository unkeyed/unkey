// unkey/go/pkg/db/replica.go

package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
	"go.opentelemetry.io/otel/attribute"
)

// Replica wraps a standard SQL database connection and implements the gen.DBTX interface
// to enable interaction with the generated database code.
type Replica struct {
	mode string
	db   *sql.DB // Underlying database connection
}

// Ensure Replica implements the gen.DBTX interface
var _ DBTX = (*Replica)(nil)

// ExecContext executes a SQL statement and returns a result summary.
// It's used for INSERT, UPDATE, DELETE statements that don't return rows.
func (r *Replica) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	ctx, span := tracing.Start(ctx, "ExecContext")
	defer span.End()
	span.SetAttributes(
		attribute.String("mode", r.mode),
		attribute.String("query", query),
	)

	// Track metrics
	start := time.Now()
	result, err := r.db.ExecContext(ctx, query, args...)

	// Record latency and operation count
	duration := time.Since(start).Seconds()
	status := statusSuccess
	if err != nil {
		status = statusError
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

	// Track metrics
	start := time.Now()
	//nolint:sqlclosecheck // Rows returned to caller, who must close them
	stmt, err := r.db.PrepareContext(ctx, query)

	// Record latency and operation count
	duration := time.Since(start).Seconds()
	status := statusSuccess
	if err != nil {
		status = statusError
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

	// Track metrics
	start := time.Now()
	//nolint:sqlclosecheck // Rows returned to caller, who must close them
	rows, err := r.db.QueryContext(ctx, query, args...)

	// Record latency and operation count
	duration := time.Since(start).Seconds()
	status := statusSuccess
	if err != nil {
		status = statusError
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

	// Track metrics
	start := time.Now()
	tx, err := r.db.BeginTx(ctx, nil)

	// Record latency and operation count
	duration := time.Since(start).Seconds()
	status := statusSuccess
	if err != nil {
		status = statusError
	}

	metrics.DatabaseOperationsLatency.WithLabelValues(r.mode, "begin", status).Observe(duration)
	metrics.DatabaseOperationsTotal.WithLabelValues(r.mode, "begin", status).Inc()

	if err != nil {
		return nil, err
	}

	// Wrap the transaction with tracing
	return WrapTxWithContext(tx, r.mode+"_tx", ctx), nil
}
