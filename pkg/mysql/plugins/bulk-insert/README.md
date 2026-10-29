# SQLC Bulk Insert Plugin

A plugin for [sqlc](https://github.com/sqlc-dev/sqlc) that automatically generates bulk insert functions for existing insert queries.

## Features

- **Efficient bulk inserts**: Generates functions that execute a single SQL query with multiple VALUES clauses instead of multiple individual INSERT statements
- **ON DUPLICATE KEY UPDATE support**: Preserves MySQL's ON DUPLICATE KEY UPDATE clauses in bulk operations
- **Type-safe**: Uses the same parameter structs as the original insert functions
- **Configurable**: Supports sqlc's `emit_methods_with_db_argument` option
- **Zero runtime overhead**: All SQL parsing and code generation happens at compile time

## How it works

For each existing insert function like `InsertKey`, the plugin generates a corresponding bulk function `BulkInsertKey` that:

1. Takes a slice of parameter structs instead of a single struct
2. Builds a single SQL query with multiple VALUES clauses
3. Executes the query in one database roundtrip

### Example

Original function:
```go
func (q *Queries) InsertKey(ctx context.Context, db DBTX, arg InsertKeyParams) error
```

Generated bulk function:
```go
func (q *Queries) BulkInsertKey(ctx context.Context, db DBTX, args []InsertKeyParams) error
```

## Configuration

The plugin is configured in your `sqlc.json`:

```json
{
  "plugins": [
    {
      "name": "bulk-insert",
      "process": {
        "cmd": "./plugins/dist/bulk-insert"
      }
    }
  ]
}
```

## Generated Code Example

For an INSERT query like:
```sql
INSERT INTO keys (id, name, created_at) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE name = VALUES(name)
```

The plugin generates two files:

### Individual Function (`bulk_insert_key.go`)
```go
const bulkInsertKey = `INSERT INTO keys (id, name, created_at) VALUES %s ON DUPLICATE KEY UPDATE name = VALUES(name)`

func (q *Queries) BulkInsertKey(ctx context.Context, db DBTX, args []InsertKeyParams) error {
    if len(args) == 0 {
        return nil
    }

    // Build the bulk insert query
    valueClauses := make([]string, len(args))
    for i := range args {
        valueClauses[i] = "(?, ?, ?)"
    }

    bulkQuery := fmt.Sprintf(bulkInsertKey, strings.Join(valueClauses, ", "))

    // Collect all arguments
    var allArgs []interface{}
    for _, arg := range args {
        allArgs = append(allArgs, arg.ID)
        allArgs = append(allArgs, arg.Name)
        allArgs = append(allArgs, arg.CreatedAt)
    }

    // Execute the bulk insert
    _, err := db.ExecContext(ctx, bulkQuery, allArgs...)
    return err
}
```

### Interface Definition (`bulk_querier.go`)
```go
// BulkQuerier contains bulk insert methods.
type BulkQuerier interface {
    BulkInsertKey(ctx context.Context, db DBTX, args []InsertKeyParams) error
    BulkInsertUser(ctx context.Context, db DBTX, args []InsertUserParams) error
    // ... other bulk insert methods
}

// Ensure BulkQueries implements BulkQuerier
var _ BulkQuerier = (*BulkQueries)(nil)
```

## Usage

The plugin generates both individual bulk functions and a `BulkQuerier` interface for type safety.

### Direct Method Usage
```go
// Instead of multiple individual inserts
for _, key := range keys {
    err := q.InsertKey(ctx, db, key)
    if err != nil {
        return err
    }
}

// Use a single bulk insert
err := q.BulkInsertKey(ctx, db, keys)
if err != nil {
    return err
}
```

### Interface Usage
```go
// Use the BulkQuerier interface for type safety
var bulkQuerier BulkQuerier = queries
err := bulkQuerier.BulkInsertKey(ctx, db, keys)
if err != nil {
    return err
}
```

### Custom Interface
```go
// Combine with the main Querier interface
type MyQuerier interface {
    Querier
    BulkQuerier
}

// Now you have access to both regular and bulk operations
var myQuerier MyQuerier = queries
```

## Performance Benefits

- **Reduced database roundtrips**: Single query instead of N queries
- **Lower network overhead**: One network request instead of N requests
- **Better transaction performance**: All inserts happen atomically
- **Improved throughput**: Especially beneficial for large datasets

## Bulk updates

Add singular `Update...` query names to `bulk_updates` in the plugin's
`codegen.options` to generate typed bulk updates:

```json
{
  "emit_methods_with_db_argument": false,
  "bulk_updates": ["UpdateVaultEncryptedKey"]
}
```

Keep the source query as a single-row `:execrows` update:

```sql
-- name: UpdateVaultEncryptedKey :execrows
UPDATE encrypted_keys
SET encrypted = sqlc.arg(ciphertext),
    encryption_key_id = sqlc.arg(key_id)
WHERE pk = sqlc.arg(pk)
  AND workspace_id = sqlc.arg(keyring);
```

The plugin generates this method on `BulkQueries`:

```go
func (q *BulkQueries) UpdateVaultEncryptedKeys(ctx context.Context, args []UpdateVaultEncryptedKeyParams) (int64, error)
```

Each call executes one parameterized `UPDATE` with `CASE` expressions. It returns
the affected-row count without inserting missing rows. An empty slice returns
zero without a query. If predicates overlap, the first matching entry wins.
Callers must bound the slice size to fit database statement and parameter limits.

Bulk updates support one table, `column = parameter` assignments, and equality
predicates joined by `AND`. Use distinct `sqlc.arg` names for all parameters.
The generator rejects unsupported SQL and updates to predicate columns, because
MySQL evaluates assignments from left to right. It does not generate updates
unless their names appear in `bulk_updates`.

## Limitations

- Automatic query detection only covers INSERT queries that have parameters
- Requires MySQL for ON DUPLICATE KEY UPDATE support
- Generated functions maintain the same transaction semantics as individual inserts
