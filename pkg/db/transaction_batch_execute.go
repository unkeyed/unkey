package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql/metrics"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
)

const transactionBatchMarker = "unkey_transaction_committed"

var errTransactionBatchNotConfirmed = errors.New("transaction batch commit was not confirmed")

// executeTransactionBatch runs existing sqlc statements atomically in one
// MySQL protocol exchange. The terminal marker is positive proof that COMMIT
// completed. A missing marker is an unknown outcome and must not be retried.
func (r *Replica) executeTransactionBatch(
	ctx context.Context,
	operation string,
	statements []transactionBatchStatement,
) error {
	ctx, span := tracing.Start(ctx, operation)
	defer span.End()
	span.SetAttributes(attribute.String("mode", r.mode))

	query, args := r.transactionBatchQuery(ctx, operation, statements)

	conn, err := r.db.Conn(ctx)
	if err != nil {
		recordBatchMetrics(r.mode, time.Duration(0), err)
		tracing.RecordErrorUnless(span, err)
		return err
	}
	defer func() { _ = conn.Close() }()

	start := time.Now()
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		cleanupTransactionBatchConnection(conn)
		recordBatchMetrics(r.mode, time.Since(start), err)
		tracing.RecordErrorUnless(span, err)
		return err
	}

	committed := false
	for {
		resultSetHasMarker := false
		columns, columnsErr := rows.Columns()
		if columnsErr != nil {
			break
		}
		for rows.Next() {
			if len(columns) != 1 {
				continue
			}
			var value any
			if scanErr := rows.Scan(&value); scanErr != nil {
				continue
			}
			switch marker := value.(type) {
			case string:
				resultSetHasMarker = marker == transactionBatchMarker
			case []byte:
				resultSetHasMarker = string(marker) == transactionBatchMarker
			}
		}
		committed = resultSetHasMarker
		if !rows.NextResultSet() {
			break
		}
	}
	rowsErr := rows.Err()
	closeErr := rows.Close()

	if committed {
		// The mutation is durable once the marker arrives. A later drain error only
		// makes this physical connection unsafe to reuse.
		if rowsErr != nil || closeErr != nil {
			discardConnection(conn)
			logger.Warn("discarding database connection after committed transaction batch", "operation", operation, "rows_error", rowsErr, "close_error", closeErr)
		}
		recordBatchMetrics(r.mode, time.Since(start), nil)
		return nil
	}

	cleanupTransactionBatchConnection(conn)
	if rowsErr != nil {
		err = rowsErr
	} else if closeErr != nil {
		err = closeErr
	} else {
		err = errTransactionBatchNotConfirmed
	}
	recordBatchMetrics(r.mode, time.Since(start), err)
	tracing.RecordErrorUnless(span, err)
	return err
}

func (r *Replica) transactionBatchQuery(
	ctx context.Context,
	operation string,
	statements []transactionBatchStatement,
) (string, []any) {
	queries := make([]string, 0, len(statements)+3)
	queries = append(queries, "-- name: "+operation+"Start :exec\nSTART TRANSACTION")
	args := make([]any, 0)
	for _, statement := range statements {
		query, statementArgs := renderTransactionBatchStatement(statement)
		queries = append(queries, query)
		args = append(args, statementArgs...)
	}
	queries = append(queries,
		"-- name: "+operation+"Commit :exec\nCOMMIT",
		"-- name: "+operation+"Marker :one\nSELECT '"+transactionBatchMarker+"'",
	)
	for i := range queries {
		queries[i] = r.annotate(ctx, queries[i])
	}
	return strings.Join(queries, ";\n") + ";", args
}

func renderTransactionBatchStatement(statement transactionBatchStatement) (string, []any) {
	var query strings.Builder
	args := make([]any, 0, len(statement.args))
	argument := 0
	quote := byte(0)
	lineComment := false
	blockComment := false
	for i := 0; i < len(statement.query); i++ {
		char := statement.query[i]
		next := byte(0)
		if i+1 < len(statement.query) {
			next = statement.query[i+1]
		}

		switch {
		case lineComment:
			query.WriteByte(char)
			if char == '\n' {
				lineComment = false
			}
		case blockComment:
			query.WriteByte(char)
			if char == '*' && next == '/' {
				query.WriteByte(next)
				i++
				blockComment = false
			}
		case quote != 0:
			query.WriteByte(char)
			if char == '\\' && next != 0 {
				query.WriteByte(next)
				i++
				continue
			}
			if char == quote {
				if next == quote {
					query.WriteByte(next)
					i++
				} else {
					quote = 0
				}
			}
		case char == '-' && next == '-':
			query.WriteByte(char)
			query.WriteByte(next)
			i++
			lineComment = true
		case char == '#':
			query.WriteByte(char)
			lineComment = true
		case char == '/' && next == '*':
			query.WriteByte(char)
			query.WriteByte(next)
			i++
			blockComment = true
		case char == '\'' || char == '"' || char == '`':
			query.WriteByte(char)
			quote = char
		case char == '?':
			if argument >= len(statement.args) {
				panic("transaction batch statement has more placeholders than arguments")
			}
			arg := statement.args[argument]
			argument++
			if arg.result != nil {
				query.WriteString(arg.result.reference())
			} else {
				query.WriteByte('?')
				args = append(args, arg.value)
			}
		default:
			query.WriteByte(char)
		}
	}
	if argument != len(statement.args) {
		panic(fmt.Sprintf("transaction batch statement has %d placeholders and %d arguments", argument, len(statement.args)))
	}

	rendered := query.String()
	if statement.result != nil {
		headerEnd := strings.IndexByte(rendered, '\n')
		if headerEnd < 0 || !strings.HasPrefix(rendered, "-- name:") {
			panic("transaction batch result statement is missing its sqlc header")
		}
		rendered = rendered[:headerEnd+1] + "SET " + statement.result.reference() + " = (\n" + rendered[headerEnd+1:] + "\n)"
	}
	return rendered, args
}

func cleanupTransactionBatchConnection(conn *sql.Conn) {
	rollbackCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = conn.ExecContext(rollbackCtx, "ROLLBACK")
	discardConnection(conn)
}

func discardConnection(conn *sql.Conn) {
	_ = conn.Raw(func(any) error {
		return driver.ErrBadConn
	})
}

func recordBatchMetrics(mode string, duration time.Duration, err error) {
	status := statusSuccess
	if err != nil {
		status = statusError
	}
	metrics.DatabaseOperationsLatency.WithLabelValues(mode, "batch", status).Observe(duration.Seconds())
	metrics.DatabaseOperationsTotal.WithLabelValues(mode, "batch", status).Inc()
}
