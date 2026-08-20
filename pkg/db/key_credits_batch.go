package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type KeyCreditsBatchResult struct {
	Applied           bool
	KeyID             sql.NullString
	DeletedAtM        sql.NullInt64
	RemainingRequests sql.NullInt64
	RefillAmount      sql.NullInt64
	RefillDay         sql.NullInt16
}

func (r *Replica) UpdateKeyCreditsIncrementWithAuditBatch(
	ctx context.Context,
	update UpdateKeyCreditsIncrementReturningParams,
	outbox InsertClickhouseOutboxForCreditUpdateParams,
) (KeyCreditsBatchResult, error) {
	return r.executeKeyCreditsBatch(ctx, "UpdateKeyCreditsIncrementWithAuditBatch", updateKeyCreditsIncrementReturningTransactionBatchStatement(update), outbox)
}

func (r *Replica) UpdateKeyCreditsDecrementWithAuditBatch(
	ctx context.Context,
	update UpdateKeyCreditsDecrementReturningParams,
	outbox InsertClickhouseOutboxForCreditUpdateParams,
) (KeyCreditsBatchResult, error) {
	return r.executeKeyCreditsBatch(ctx, "UpdateKeyCreditsDecrementWithAuditBatch", updateKeyCreditsDecrementReturningTransactionBatchStatement(update), outbox)
}

func (r *Replica) UpdateKeyCreditsSetWithAuditBatch(
	ctx context.Context,
	update UpdateKeyCreditsSetParams,
	outbox InsertClickhouseOutboxForCreditUpdateParams,
) (KeyCreditsBatchResult, error) {
	return r.executeKeyCreditsBatch(ctx, "UpdateKeyCreditsSetWithAuditBatch", updateKeyCreditsSetTransactionBatchStatement(update), outbox)
}

// executeKeyCreditsBatch returns the in-transaction credit state from a
// diagnostic result set before consuming the terminal commit marker. This is
// one MySQL protocol exchange, including the mutation and audit outbox write.
func (r *Replica) executeKeyCreditsBatch(
	ctx context.Context,
	operation string,
	update transactionBatchStatement,
	outbox InsertClickhouseOutboxForCreditUpdateParams,
) (result KeyCreditsBatchResult, err error) {
	ctx, span := tracing.Start(ctx, operation)
	defer span.End()
	span.SetAttributes(attribute.String("mode", r.mode))

	outboxStatement := insertClickhouseOutboxForCreditUpdateTransactionBatchStatement(outbox)
	queries := []string{
		"-- name: " + operation + "Start :exec\nSTART TRANSACTION",
		update.query,
		outboxStatement.query,
		findKeyCreditsBatchResult,
		"-- name: " + operation + "Commit :exec\nCOMMIT",
		"-- name: " + operation + "Marker :one\nSELECT '" + transactionBatchMarker + "'",
	}
	for i := range queries {
		queries[i] = r.annotate(ctx, queries[i])
	}
	args := append(update.args, outboxStatement.args...)
	args = append(args, outbox.KeyID)

	conn, err := r.db.Conn(ctx)
	if err != nil {
		recordBatchMetrics(r.mode, 0, err)
		tracing.RecordErrorUnless(span, err)
		return result, err
	}
	defer func() { _ = conn.Close() }()

	start := time.Now()
	rows, err := conn.QueryContext(ctx, strings.Join(queries, ";\n")+";", args...)
	if err != nil {
		cleanupTransactionBatchConnection(conn)
		recordBatchMetrics(r.mode, time.Since(start), err)
		tracing.RecordErrorUnless(span, err)
		return result, err
	}

	var diagnosticErr error
	if !rows.Next() {
		diagnosticErr = errors.New("credit batch returned no diagnostic row")
	} else {
		var outboxRows int64
		diagnosticErr = rows.Scan(
			&outboxRows,
			&result.KeyID,
			&result.DeletedAtM,
			&result.RemainingRequests,
			&result.RefillAmount,
			&result.RefillDay,
		)
		result.Applied = outboxRows == 1
		if diagnosticErr == nil && outboxRows != 0 && outboxRows != 1 {
			diagnosticErr = fmt.Errorf("credit batch inserted unexpected audit row count: %d", outboxRows)
		}
	}
	for rows.Next() {
		diagnosticErr = errors.New("credit batch returned multiple diagnostic rows")
	}

	committed := false
	if rows.NextResultSet() && rows.Next() {
		var marker string
		if scanErr := rows.Scan(&marker); scanErr == nil && marker == transactionBatchMarker {
			committed = true
		}
	}
	for rows.Next() {
	}
	for rows.NextResultSet() {
		for rows.Next() {
		}
	}
	rowsErr := rows.Err()
	closeErr := rows.Close()

	if committed {
		if rowsErr != nil || closeErr != nil {
			discardConnection(conn)
			logger.Warn("discarding database connection after committed credit batch", "operation", operation, "rows_error", rowsErr, "close_error", closeErr)
		}
		if diagnosticErr != nil {
			recordBatchMetrics(r.mode, time.Since(start), diagnosticErr)
			tracing.RecordErrorUnless(span, diagnosticErr)
			return result, diagnosticErr
		}
		recordBatchMetrics(r.mode, time.Since(start), nil)
		return result, nil
	}

	cleanupTransactionBatchConnection(conn)
	switch {
	case rowsErr != nil:
		err = rowsErr
	case closeErr != nil:
		err = closeErr
	default:
		err = errTransactionBatchNotConfirmed
	}
	recordBatchMetrics(r.mode, time.Since(start), err)
	tracing.RecordErrorUnless(span, err)
	return result, err
}
