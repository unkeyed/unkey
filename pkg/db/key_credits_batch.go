package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
)

const (
	keyCreditsDeleted   int64 = -1
	keyCreditsUnlimited int64 = -2
	keyCreditsOverflow  int64 = -3
)

type KeyCreditsBatchResult struct {
	Applied           bool
	Missing           bool
	Deleted           bool
	Unlimited         bool
	Overflow          bool
	RemainingRequests sql.NullInt64
}

func (r *Replica) UpdateKeyCreditsIncrementWithAuditBatch(
	ctx context.Context,
	update UpdateKeyCreditsIncrementReturningParams,
	outbox InsertClickhouseOutboxForCreditUpdateParams,
) (KeyCreditsBatchResult, error) {
	return r.executeKeyCreditsBatch(ctx, "UpdateKeyCreditsIncrementWithAuditBatch", updateKeyCreditsIncrementReturningTransactionBatchStatement(update), outbox, false)
}

func (r *Replica) UpdateKeyCreditsDecrementWithAuditBatch(
	ctx context.Context,
	update UpdateKeyCreditsDecrementReturningParams,
	outbox InsertClickhouseOutboxForCreditUpdateParams,
) (KeyCreditsBatchResult, error) {
	return r.executeKeyCreditsBatch(ctx, "UpdateKeyCreditsDecrementWithAuditBatch", updateKeyCreditsDecrementReturningTransactionBatchStatement(update), outbox, false)
}

func (r *Replica) UpdateKeyCreditsSetWithAuditBatch(
	ctx context.Context,
	update UpdateKeyCreditsSetParams,
	outbox InsertClickhouseOutboxForCreditUpdateParams,
) (KeyCreditsBatchResult, error) {
	return r.executeKeyCreditsBatch(ctx, "UpdateKeyCreditsSetWithAuditBatch", updateKeyCreditsSetTransactionBatchStatement(update), outbox, !update.Credits.Valid)
}

// executeKeyCreditsBatch reads the guarded update result from its MySQL OK
// packet. The mutation, audit outbox write, and commit use one protocol exchange.
func (r *Replica) executeKeyCreditsBatch(
	ctx context.Context,
	operation string,
	update transactionBatchStatement,
	outbox InsertClickhouseOutboxForCreditUpdateParams,
	nullBalance bool,
) (result KeyCreditsBatchResult, err error) {
	ctx, span := tracing.Start(ctx, operation)
	defer span.End()
	span.SetAttributes(attribute.String("mode", r.mode))

	outboxStatement := insertClickhouseOutboxForCreditUpdateTransactionBatchStatement(outbox)
	updateQuery, updateArgs := renderTransactionBatchStatement(update)
	outboxQuery, outboxArgs := renderTransactionBatchStatement(outboxStatement)
	queries := []string{
		"-- name: " + operation + "Start :exec\nSTART TRANSACTION",
		updateQuery,
		outboxQuery,
		"-- name: " + operation + "Commit :exec\nCOMMIT",
		"-- name: " + operation + "Marker :exec\nDO 1",
	}
	for i := range queries {
		queries[i] = r.annotate(ctx, queries[i])
	}
	args := append(updateArgs, outboxArgs...)

	conn, err := r.db.Conn(ctx)
	if err != nil {
		recordBatchMetrics(r.mode, 0, err)
		tracing.RecordErrorUnless(span, err)
		return result, err
	}
	defer func() { _ = conn.Close() }()

	start := time.Now()
	rowsAffected, lastInsertIDs, err := executeMultiStatement(ctx, conn, strings.Join(queries, ";\n")+";", args)
	if err != nil {
		cleanupTransactionBatchConnection(conn)
		recordBatchMetrics(r.mode, time.Since(start), err)
		tracing.RecordErrorUnless(span, err)
		return result, err
	}

	if len(rowsAffected) != len(queries) || len(lastInsertIDs) != len(queries) {
		err = fmt.Errorf("credit batch returned %d row counts and %d insert IDs for %d statements", len(rowsAffected), len(lastInsertIDs), len(queries))
		cleanupTransactionBatchConnection(conn)
		recordBatchMetrics(r.mode, time.Since(start), err)
		tracing.RecordErrorUnless(span, err)
		return result, err
	}

	updateRows := rowsAffected[1]
	outboxRows := rowsAffected[2]
	state := lastInsertIDs[1]
	switch {
	case updateRows == 0:
		result.Missing = true
	case updateRows != 1:
		err = fmt.Errorf("credit batch updated unexpected row count: %d", updateRows)
	case state == keyCreditsDeleted:
		result.Deleted = true
	case state == keyCreditsUnlimited:
		result.Unlimited = true
	case state == keyCreditsOverflow:
		result.Overflow = true
	case state < 0:
		err = fmt.Errorf("credit batch returned unknown state: %d", state)
	default:
		result.Applied = true
		result.RemainingRequests = sql.NullInt64{Int64: state, Valid: !nullBalance}
	}
	if err == nil {
		expectedOutboxRows := int64(0)
		if result.Applied {
			expectedOutboxRows = 1
		}
		if outboxRows != expectedOutboxRows {
			err = fmt.Errorf("credit batch inserted %d audit rows, expected %d", outboxRows, expectedOutboxRows)
		}
	}
	if err != nil {
		discardConnection(conn)
		recordBatchMetrics(r.mode, time.Since(start), err)
		tracing.RecordErrorUnless(span, err)
		return result, err
	}

	recordBatchMetrics(r.mode, time.Since(start), nil)
	return result, nil
}

func executeMultiStatement(ctx context.Context, conn *sql.Conn, query string, args []any) ([]int64, []int64, error) {
	namedArgs := make([]driver.NamedValue, len(args))
	for i, arg := range args {
		namedArgs[i] = driver.NamedValue{Name: "", Ordinal: i + 1, Value: arg}
	}

	var rowsAffected, lastInsertIDs []int64
	err := conn.Raw(func(driverConn any) error {
		checker, hasChecker := driverConn.(driver.NamedValueChecker)
		for i := range namedArgs {
			var convertErr error
			if hasChecker {
				convertErr = checker.CheckNamedValue(&namedArgs[i])
			} else {
				namedArgs[i].Value, convertErr = driver.DefaultParameterConverter.ConvertValue(namedArgs[i].Value)
			}
			if convertErr != nil {
				return fmt.Errorf("convert batch argument %d: %w", i, convertErr)
			}
		}
		execer, ok := driverConn.(driver.ExecerContext)
		if !ok {
			return errors.New("database driver does not support context-aware execution")
		}
		driverResult, execErr := execer.ExecContext(ctx, query, namedArgs)
		if execErr != nil {
			return execErr
		}
		result, resultOK := driverResult.(mysql.Result)
		if !resultOK {
			return errors.New("database driver did not return multi-statement results")
		}
		rowsAffected = append(rowsAffected, result.AllRowsAffected()...)
		lastInsertIDs = append(lastInsertIDs, result.AllLastInsertIds()...)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return rowsAffected, lastInsertIDs, nil
}
