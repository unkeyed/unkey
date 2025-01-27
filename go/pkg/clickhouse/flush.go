package clickhouse

import (
	"context"
	"fmt"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func flush[T any](ctx context.Context, conn ch.Conn, table string, rows []T) error {
	batch, err := conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s", table))
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
