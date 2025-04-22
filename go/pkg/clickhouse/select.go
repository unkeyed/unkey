package clickhouse

import (
	"context"

	ch "github.com/ClickHouse/clickhouse-go/v2"
)

// Select executes a ClickHouse query and unmarshals the results into a slice of type T.
// It handles parameter substitution and type conversion to provide a safer and more
// convenient interface than direct driver usage.
//
// The query string can contain named parameters in the format {name} which will be
// replaced with values from the parameters map. This helps prevent SQL injection
// by properly escaping parameter values.
//
// Important: The generic type T must be properly annotated with `ch` struct tags to be
// unmarshalled correctly. Each field that corresponds to a ClickHouse column must have
// a `ch` tag specifying the column name, like:
//
//	type UserRecord struct {
//		UserID    string `ch:"user_id"`
//		Username  string `ch:"username"`
//		CreatedAt int64  `ch:"created_at"`
//	}
//
// See the schema package for examples of properly annotated structs.
//
// The context can be used to control cancellation and timeout behavior for the query.
// For long-running queries, consider using context timeouts to prevent indefinite blocking.
//
// Select will return an error if the connection is invalid, the query is malformed,
// or if the results cannot be unmarshaled into the destination type. Callers should
// always check the error return value before using the results.
//
// Example:
//
//	type UserRecord struct {
//		ID       string `ch:"id"`
//		Username string `ch:"username"`
//		Created  int64  `ch:"created"`
//	}
//
//	// Query users who registered in the last week
//	users, err := clickhouse.Select[UserRecord](ctx, conn,
//		"SELECT id, username, created FROM users WHERE created > {since}",
//		map[string]string{"since": "2023-01-01"})
//	if err != nil {
//		return fmt.Errorf("query failed: %w", err)
//	}
//
//	// Process results
//	for _, user := range users {
//		fmt.Println(user.Username)
//	}
//
// For large result sets, consider using LIMIT clauses or pagination in your queries
// to prevent excessive memory usage. The function loads all results into memory at once.
//
// This function is thread-safe and can be used concurrently with the same connection,
// as the underlying ClickHouse driver handles connection pooling.
func Select[T any](ctx context.Context, conn ch.Conn, query string, parameters map[string]string) ([]T, error) {
	var dest []T
	err := conn.Select(
		ch.Context(ctx, ch.WithParameters(parameters)),
		&dest,
		query,
	)
	if err != nil {
		return nil, err
	}

	return dest, nil
}
