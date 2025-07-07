// Package db provides bulk database operations for the Unkey platform.
// This file contains utilities for performing bulk inserts that are not
// supported by sqlc code generation, allowing efficient batch operations
// on large datasets while maintaining type safety and error handling.
//
// Bulk operations are essential for performance when dealing with many
// records, reducing round-trips to the database and improving throughput
// for batch processing scenarios common in API key management systems.
package db

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// BulkInsert executes a bulk insert operation using a provided SQL query template
// and a slice of argument sets. Since sqlc does not support bulk insert generation,
// this function provides a way to perform batch inserts with proper placeholder
// expansion and error handling.
//
// The function takes a SQL query template with placeholder values and expands
// it to accommodate multiple rows of data. The query parameter should contain
// a single VALUES clause that will be replicated for each argument set.
//
// The args parameter must be a slice of []interface{} where each element contains
// the values for one row in the same order as the placeholders in the query.
// Each row must have the exact same number of values matching the query placeholders.
//
// Query template format:
//   - Use MySQL-style ? placeholders for parameters
//   - Include a single VALUES clause that matches one argument set
//   - Support for INSERT with ON DUPLICATE KEY UPDATE clauses
//   - Compatible with any INSERT variant (INSERT, INSERT IGNORE, REPLACE, etc.)
//
// The function handles:
//   - Automatic placeholder expansion for multiple rows
//   - Type-safe argument passing through generics
//   - Context cancellation and timeout propagation
//   - Consistent error handling with database operation patterns
//
// Performance considerations:
//   - Bulk inserts are significantly faster than individual INSERT statements
//   - MySQL has limits on query size and parameter count (max_allowed_packet)
//   - Consider batching very large datasets to avoid hitting database limits
//   - Use transactions when bulk operations are part of larger atomic operations
//
// The ctx parameter provides cancellation and timeout control for the operation.
// The db parameter must implement the DBTX interface, supporting both direct
// database connections and transaction contexts for atomic operations.
//
// BulkInsert returns nil on successful execution or an error if the operation
// fails. Database errors are returned directly without additional wrapping,
// allowing callers to handle specific error conditions as needed.
//
// Common usage patterns:
//   - Batch API key creation during workspace initialization
//   - Bulk permission assignments for role-based access control
//   - Mass import operations from external systems
//   - Periodic data synchronization between services
//
// Example basic bulk insert:
//
//	keys := []db.InsertKeyParams{
//		{ID: "key1", KeyAuthID: "auth1", Hash: "hash1"},
//		{ID: "key2", KeyAuthID: "auth2", Hash: "hash2"},
//		{ID: "key3", KeyAuthID: "auth3", Hash: "hash3"},
//	}
//
//	query := "INSERT INTO keys (id, key_auth_id, hash) VALUES (?, ?, ?)"
//	err := db.BulkInsert(ctx, database.RW(), query, keys)
//	if err != nil {
//		return fmt.Errorf("failed to bulk insert keys: %w", err)
//	}
//
// Example with ON DUPLICATE KEY UPDATE:
//
//	permissions := []PermissionParams{
//		{KeyID: "key1", Permission: "read"},
//		{KeyID: "key2", Permission: "write"},
//	}
//
//	query := `INSERT INTO key_permissions (key_id, permission) VALUES (?, ?)
//	          ON DUPLICATE KEY UPDATE permission = VALUES(permission)`
//	err := db.BulkInsert(ctx, tx, query, permissions)
//	if err != nil {
//		return fmt.Errorf("failed to upsert permissions: %w", err)
//	}
//
// Example within transaction:
//
//	err := db.Tx(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) error {
//		// Create workspace
//		workspace, err := db.Query.InsertWorkspace(ctx, tx, workspaceParams)
//		if err != nil {
//			return err
//		}
//
//		// Bulk insert initial API keys
//		query := "INSERT INTO keys (id, workspace_id, name) VALUES (?, ?, ?)"
//		err = db.BulkInsert(ctx, tx, query, initialKeys)
//		if err != nil {
//			return fmt.Errorf("failed to create initial keys: %w", err)
//		}
//
//		return nil
//	})
//
// Limitations and considerations:
//   - The query template must match the structure of each argument set exactly
//   - No validation is performed on the SQL query syntax or parameter count
//   - Very large batches may hit database limits on query size or parameters
//   - The function assumes all argument sets have the same structure and type
//   - Error messages may not clearly indicate which specific row caused failures
//
// Anti-patterns to avoid:
//   - Using BulkInsert for single-row operations (use generated sqlc functions)
//   - Mixing different parameter structures in the same args slice
//   - Including multiple VALUES clauses in the query template
//   - Ignoring context cancellation in long-running bulk operations
//
// For very large datasets, consider:
//   - Splitting into smaller batches to avoid memory pressure
//   - Using transactions to ensure consistency across batches
//   - Implementing retry logic for transient database errors
//   - Monitoring query execution time and database resource usage
//
// See [DBTX] for available database interfaces and [Tx] for transaction
// utilities when bulk operations need atomic guarantees.
func BulkInsert[T any](ctx context.Context, db DBTX, query string, args []T) error {
	if len(args) == 0 {
		return nil
	}

	valuesIndex := strings.Index(strings.ToUpper(query), "VALUES")
	if valuesIndex == -1 {
		return fmt.Errorf("bulk insert query must contain VALUES clause")
	}

	beforeValues := query[:valuesIndex+6]
	afterValues := ""

	valueStart := strings.Index(query[valuesIndex:], "(")
	if valueStart == -1 {
		return fmt.Errorf("bulk insert query must contain parenthesized VALUES clause")
	}

	valueStart += valuesIndex
	parenCount := 0
	valueEnd := valueStart

	// Find the matching closing parenthesis
	for i := valueStart; i < len(query); i++ {
		if query[i] == '(' {
			parenCount++
		} else if query[i] == ')' {
			parenCount--
			if parenCount == 0 {
				valueEnd = i + 1
				break
			}
		}
	}

	if parenCount != 0 {
		return fmt.Errorf("bulk insert query has unmatched parentheses in VALUES clause")
	}

	valuesClause := query[valueStart:valueEnd]
	if valueEnd < len(query) {
		afterValues = query[valueEnd:]
	}

	// Build the expanded query with multiple VALUES clauses
	valuesClauses := make([]string, len(args))
	for i := range args {
		valuesClauses[i] = valuesClause
	}

	expandedQuery := beforeValues + " " + strings.Join(valuesClauses, ", ") + afterValues

	flatArgs := make([]any, 0)
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		for i := 0; i < v.NumField(); i++ {
			flatArgs = append(flatArgs, v.Field(i).Interface())
		}
	}

	_, err := db.ExecContext(ctx, expandedQuery, flatArgs...)
	return err
}
