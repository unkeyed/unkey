package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
	"go.opentelemetry.io/otel/attribute"
)

const (
	statusSuccess = "success"
	statusError   = "error"
)

// WrapTxWithContext wraps a standard sql.Tx with our DBTx interface for tracing, using the provided context
func WrapTxWithContext(tx *sql.Tx, mode string, ctx context.Context) DBTx {
	return &TracedTx{
		tx:   tx,
		mode: mode,
		ctx:  ctx,
	}
}

// TracedTx wraps a sql.Tx to add tracing to all database operations within a transaction
type TracedTx struct {
	tx   *sql.Tx
	mode string
	ctx  context.Context // Store the context for commit/rollback tracing
}

// Ensure TracedTx implements the DBTx interface
var _ DBTx = (*TracedTx)(nil)

// ExecContext executes a SQL statement within the transaction with tracing
func (t *TracedTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	ctx, span := tracing.Start(ctx, "Tx.ExecContext")
	defer span.End()
	span.SetAttributes(
		attribute.String("mode", t.mode),
		attribute.String("query", query),
	)

	start := time.Now()
	result, err := t.tx.ExecContext(ctx, query, args...)

	duration := time.Since(start).Seconds()
	status := statusSuccess
	if err != nil {
		status = statusError
	}

	metrics.DatabaseOperationsLatency.WithLabelValues(t.mode, "exec", status).Observe(duration)
	metrics.DatabaseOperationsTotal.WithLabelValues(t.mode, "exec", status).Inc()

	return result, err
}

// PrepareContext prepares a SQL statement within the transaction with tracing
func (t *TracedTx) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	ctx, span := tracing.Start(ctx, "Tx.PrepareContext")
	defer span.End()
	span.SetAttributes(
		attribute.String("mode", t.mode),
		attribute.String("query", query),
	)

	start := time.Now()
	//nolint:sqlclosecheck // Rows returned to caller, who must close them
	stmt, err := t.tx.PrepareContext(ctx, query)

	duration := time.Since(start).Seconds()
	status := statusSuccess
	if err != nil {
		status = statusError
	}

	metrics.DatabaseOperationsLatency.WithLabelValues(t.mode, "prepare", status).Observe(duration)
	metrics.DatabaseOperationsTotal.WithLabelValues(t.mode, "prepare", status).Inc()

	return stmt, err // nolint:sqlclosecheck
}

// QueryContext executes a SQL query within the transaction with tracing
func (t *TracedTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	ctx, span := tracing.Start(ctx, "Tx.QueryContext")
	defer span.End()
	span.SetAttributes(
		attribute.String("mode", t.mode),
		attribute.String("query", query),
	)

	start := time.Now()
	//nolint:sqlclosecheck // Rows returned to caller, who must close them
	rows, err := t.tx.QueryContext(ctx, query, args...)

	duration := time.Since(start).Seconds()
	status := statusSuccess
	if err != nil {
		status = statusError
	}

	metrics.DatabaseOperationsLatency.WithLabelValues(t.mode, "query", status).Observe(duration)
	metrics.DatabaseOperationsTotal.WithLabelValues(t.mode, "query", status).Inc()

	return rows, err // nolint:sqlclosecheck
}

// QueryRowContext executes a SQL query that returns a single row within the transaction with tracing
func (t *TracedTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	ctx, span := tracing.Start(ctx, "Tx.QueryRowContext")
	defer span.End()
	span.SetAttributes(
		attribute.String("mode", t.mode),
		attribute.String("query", query),
	)

	start := time.Now()
	row := t.tx.QueryRowContext(ctx, query, args...)

	duration := time.Since(start).Seconds()
	status := statusSuccess

	metrics.DatabaseOperationsLatency.WithLabelValues(t.mode, "query_row", status).Observe(duration)
	metrics.DatabaseOperationsTotal.WithLabelValues(t.mode, "query_row", status).Inc()

	return row
}

// Commit commits the transaction with tracing
func (t *TracedTx) Commit() error {
	_, span := tracing.Start(t.ctx, "Tx.Commit")
	defer span.End()
	span.SetAttributes(attribute.String("mode", t.mode))

	start := time.Now()
	err := t.tx.Commit()

	duration := time.Since(start).Seconds()
	status := statusSuccess
	if err != nil {
		status = statusError
	}

	metrics.DatabaseOperationsLatency.WithLabelValues(t.mode, "commit", status).Observe(duration)
	metrics.DatabaseOperationsTotal.WithLabelValues(t.mode, "commit", status).Inc()

	return err
}

// Rollback rolls back the transaction with tracing
func (t *TracedTx) Rollback() error {
	_, span := tracing.Start(t.ctx, "Tx.Rollback")
	defer span.End()
	span.SetAttributes(attribute.String("mode", t.mode))

	start := time.Now()
	err := t.tx.Rollback()

	duration := time.Since(start).Seconds()
	status := statusSuccess
	if err != nil {
		status = statusError
	}

	metrics.DatabaseOperationsLatency.WithLabelValues(t.mode, "rollback", status).Observe(duration)
	metrics.DatabaseOperationsTotal.WithLabelValues(t.mode, "rollback", status).Inc()

	return err
}
