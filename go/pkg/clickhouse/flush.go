package clickhouse

import (
	"context"
	"fmt"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// flush writes a batch of rows to the specified ClickHouse table.
// It handles the preparation of the batch, appending each row as a struct,
// and finally sending the batch to ClickHouse.
//
// This function is used internally by the batch processors to efficiently
// insert data in batches rather than individual rows.
//
// Parameters:
//   - ctx: Context for the operation, allowing for cancellation and timeouts
//   - conn: The ClickHouse connection to use
//   - table: The name of the destination table
//   - rows: A slice of structs representing the rows to insert
//
// Returns an error if any part of the batch operation fails.
func flush[T any](ctx context.Context, conn ch.Conn, table string, rows []T) error {
	batch, err := conn.PrepareBatch(
		ctx,
		fmt.Sprintf("INSERT INTO %s", table),
		driver.WithReleaseConnection(),
	)
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("preparing batch failed", ""))
	}
	for _, row := range rows {
		err = batch.AppendStruct(&row)
		if err != nil {
			return fault.Wrap(err, fault.WithDesc("appending struct to batch failed", ""))
		}
	}
	err = batch.Send()
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("committing batch failed", ""))
	}
	return nil
}
