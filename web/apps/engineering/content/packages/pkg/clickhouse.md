---
title: clickhouse
description: "provides a client for interacting with ClickHouse databases,"
---

Package clickhouse provides a client for interacting with ClickHouse databases, optimized for high-volume event data storage and analytics.

It implements efficient batch processing for different event types, with support for buffering, automatic retries, and graceful shutdown. The package is designed to handle high-throughput logging scenarios where individual event latency is less critical than overall throughput and reliability.

Key features:

  - Batched writes to minimize network overhead
  - Buffer-based queueing to handle traffic spikes
  - Automatic connection management
  - Graceful shutdown with final flush capability
  - Support for multiple event types with dedicated buffers

Example usage:

	// Create a ClickHouse client
	ch, err := clickhouse.New(clickhouse.Config{
	    URL:    "clickhouse://user:pass@clickhouse.example.com:9000/db?secure=true",
	})
	if err != nil {
	    return fmt.Errorf("failed to create clickhouse client: %w", err)
	}

	// Buffer events for batch processing
	ch.BufferRequest(schema.ApiRequestV1{
	    RequestID:      "req_123",
	    Time:           time.Now().UnixMilli(),
	    WorkspaceID:    "ws_abc",
	    Host:           "api.example.com",
	    Method:         "POST",
	    Path:           "/v1/keys/verify",
	    ResponseStatus: 200,
	})

	// Events are automatically flushed based on batch size and interval
	// When shutting down:
	err = ch.Shutdown(ctx)

## Variables

```go
var (
	_ Bufferer   = (*clickhouse)(nil)
	_ Querier    = (*clickhouse)(nil)
	_ ClickHouse = (*clickhouse)(nil)
)
```

```go
var (
	// validIdentifier matches safe ClickHouse identifiers (usernames, policy names, quota names, profile names)
	// Allows alphanumeric characters and underscores only
	validIdentifier = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

	// validTableName matches safe ClickHouse table names in database.table format
	// Allows alphanumeric characters and underscores in both database and table parts
	validTableName = regexp.MustCompile(`^[a-zA-Z0-9_]+\.[a-zA-Z0-9_]+$`)

	// Table type patterns for retention filter generation
	rawTablePattern      = regexp.MustCompile(`_raw_v\d+$`)
	perMinuteHourPattern = regexp.MustCompile(`_per_minute_v\d+$|_per_hour_v\d+$`)
	perDayMonthPattern   = regexp.MustCompile(`_per_day_v\d+$|_per_month_v\d+$`)
)
```

resourceLimitCodes maps ClickHouse exception codes to error responses
```go
var resourceLimitCodes = map[int32]errorResponse{
	159: {
		code:    codes.User.UnprocessableEntity.QueryExecutionTimeout.URN(),
		message: "Query execution time limit exceeded. Try simplifying your query or reducing the time range.",
	},
	241: {
		code:    codes.User.UnprocessableEntity.QueryMemoryLimitExceeded.URN(),
		message: "Query memory limit exceeded. Try simplifying your query or reducing the result set size.",
	},
	396: {
		code:    codes.User.UnprocessableEntity.QueryExecutionTimeout.URN(),
		message: "Query was cancelled due to resource limits.",
	},
	158: {
		code:    codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(),
		message: "Query attempted to read too many rows. Try adding more filters or reducing the time range.",
	},
	198: {
		code:    codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(),
		message: "Query attempted to read too many rows. Try adding more filters or reducing the time range.",
	},
	202: {
		code:    codes.User.TooManyRequests.QueryQuotaExceeded.URN(),
		message: "Query quota exceeded for the current time window. Please try again later.",
	},
}
```

resourceLimitPatterns maps error message patterns to error responses
```go
var resourceLimitPatterns = map[string]errorResponse{
	"timeout": {
		code:    codes.User.UnprocessableEntity.QueryExecutionTimeout.URN(),
		message: "Query execution time limit exceeded. Try simplifying your query or reducing the time range.",
	},
	"execution time": {
		code:    codes.User.UnprocessableEntity.QueryExecutionTimeout.URN(),
		message: "Query execution time limit exceeded. Try simplifying your query or reducing the time range.",
	},
	"memory": {
		code:    codes.User.UnprocessableEntity.QueryMemoryLimitExceeded.URN(),
		message: "Query memory limit exceeded. Try simplifying your query or reducing the result set size.",
	},
	"too many rows": {
		code:    codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(),
		message: "Query attempted to read too many rows. Try adding more filters or reducing the time range.",
	},
	"limit for rows_to_read": {
		code:    codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(),
		message: "Query attempted to read too many rows. Try adding more filters or reducing the time range.",
	},
	"limit for rows": {
		code:    codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(),
		message: "Query attempted to read too many rows. Try adding more filters or reducing the time range.",
	},
	"quota": {
		code:    codes.User.TooManyRequests.QueryQuotaExceeded.URN(),
		message: "Query quota exceeded for the current time window. Please try again later.",
	},
}
```

ClickHouse exception codes that indicate user query errors
```go
var userErrorCodes = map[int32]bool{
	47:  true,
	60:  true,
	62:  true,
	386: true,
	43:  true,
	352: true,
}
```

Common user error patterns in ClickHouse error messages
```go
var userErrorPatterns = map[string]bool{
	"unknown identifier":          true,
	"unknown expression":          true,
	"unknown function":            true,
	"unknown column":              true,
	"unknown table":               true,
	"missing columns":             true,
	"there is no column":          true,
	"type mismatch":               true,
	"cannot convert":              true,
	"syntax error":                true,
	"expected":                    true,
	"illegal type":                true,
	"ambiguous column":            true,
	"not an aggregate function":   true,
	"division by zero":            true,
	"aggregate function":          true,
	"window function":             true,
	"unknown_identifier":          true,
	"db::exception":               true,
	"maybe you meant":             true,
	"no such column":              true,
	"doesn't exist":               true,
	"does not exist":              true,
	"failed at position":          true,
	"unexpected token":            true,
	"invalid expression":          true,
	"invalid number of arguments": true,
	"wrong number of arguments":   true,
	"cannot parse":                true,
	"unrecognized token":          true,
	"no matching signature":       true,
	"incompatible types":          true,
	"illegal aggregation":         true,
	"cannot find column":          true,
	"not allowed in this context": true,
	"not supported":               true,
	"invalid combination":         true,
	"invalid or illegal":          true,
}
```


## Functions

### func DefaultAllowedTables

```go
func DefaultAllowedTables() []string
```

DefaultAllowedTables returns the default list of tables for analytics access

### func ExtractUserFriendlyError

```go
func ExtractUserFriendlyError(err error) string
```

ExtractUserFriendlyError extracts a user-friendly error message from ClickHouse error. It preserves the key information like unknown identifiers, suggestions, and error context.

### func IsUserQueryError

```go
func IsUserQueryError(err error) bool
```

IsUserQueryError checks if the ClickHouse error is due to a bad query (user error) vs a system/infrastructure error.

Returns true for errors like: - Unknown column/identifier - Type mismatches - Syntax errors - Division by zero

Returns false for errors like: - Connection failures - Timeouts - Infrastructure issues

### func Select

```go
func Select[T any](ctx context.Context, conn ch.Conn, query string, parameters map[string]string) ([]T, error)
```

Select executes a ClickHouse query and unmarshals the results into a slice of type T. It handles parameter substitution and type conversion to provide a safer and more convenient interface than direct driver usage.

The query string can contain named parameters in the format {name} which will be replaced with values from the parameters map. This helps prevent SQL injection by properly escaping parameter values.

Important: The generic type T must be properly annotated with \`ch\` struct tags to be unmarshalled correctly. Each field that corresponds to a ClickHouse column must have a \`ch\` tag specifying the column name, like:

	type UserRecord struct {
		UserID    string `ch:"user_id"`
		Username  string `ch:"username"`
		CreatedAt int64  `ch:"created_at"`
	}

See the schema package for examples of properly annotated structs.

The context can be used to control cancellation and timeout behavior for the query. For long-running queries, consider using context timeouts to prevent indefinite blocking.

Select will return an error if the connection is invalid, the query is malformed, or if the results cannot be unmarshaled into the destination type. Callers should always check the error return value before using the results.

Example:

	type UserRecord struct {
		ID       string `ch:"id"`
		Username string `ch:"username"`
		Created  int64  `ch:"created"`
	}

	// Query users who registered in the last week
	users, err := clickhouse.Select[UserRecord](ctx, conn,
		"SELECT id, username, created FROM users WHERE created > {since}",
		map[string]string{"since": "2023-01-01"})
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	// Process results
	for _, user := range users {
		fmt.Println(user.Username)
	}

For large result sets, consider using LIMIT clauses or pagination in your queries to prevent excessive memory usage. The function loads all results into memory at once.

This function is thread-safe and can be used concurrently with the same connection, as the underlying ClickHouse driver handles connection pooling.

### func WrapClickHouseError

```go
func WrapClickHouseError(err error) error
```

WrapClickHouseError wraps a ClickHouse error with appropriate error codes and user-friendly messages. It detects resource limit violations and other user errors and tags them with specific error codes.


## Types

### type Bufferer

```go
type Bufferer interface {
	// BufferApiRequest adds an API request event to the buffer.
	// These are typically HTTP requests to the API with request and response details.
	BufferApiRequest(schema.ApiRequest)

	// BufferKeyVerification adds a key verification event to the buffer.
	// These represent API key validation operations with their outcomes.
	BufferKeyVerification(schema.KeyVerification)

	// BufferRatelimit adds a ratelimit event to the buffer.
	// These represent API ratelimit operations with their outcome.
	BufferRatelimit(schema.Ratelimit)

	// BufferRatelimit adds a ratelimit event to the buffer.
	// These represent API ratelimit operations with their outcome.
	BufferBuildStep(schema.BuildStepV1)

	// BufferRatelimit adds a ratelimit event to the buffer.
	// These represent API ratelimit operations with their outcome.
	BufferBuildStepLog(schema.BuildStepLogV1)

	// BufferSentinelRequest adds a sentinel request event to the buffer.
	// These represent requests routed through sentinel to deployment instances.
	BufferSentinelRequest(schema.SentinelRequest)
}
```

Bufferer defines the interface for systems that can buffer events for batch processing. It provides methods to add different types of events to their respective buffers.

This interface allows for different implementations, such as a real ClickHouse client or a no-op implementation for testing or development.

### type ClickHouse

```go
type ClickHouse interface {
	Bufferer
	Querier

	// Closes the underlying ClickHouse connection.
	Close() error

	// Ping verifies the connection to the ClickHouse database.
	Ping(ctx context.Context) error
}
```

### type Config

```go
type Config struct {
	// URL is the ClickHouse connection string
	// Format: clickhouse://username:password@host:port/database?param1=value1&...
	URL string
}
```

Config contains the configuration options for the ClickHouse client.

### type Querier

```go
type Querier interface {
	// Conn returns a connection to the ClickHouse database.
	Conn() ch.Conn

	// QueryToMaps executes a query and scans all rows into a slice of maps.
	// Each map represents a row with column names as keys.
	// This is useful for dynamic queries where the schema is not known at compile time.
	QueryToMaps(ctx context.Context, query string, args ...any) ([]map[string]any, error)

	// Exec executes a DDL or DML statement (CREATE, ALTER, DROP, etc.)
	Exec(ctx context.Context, sql string, args ...any) error

	// ConfigureUser creates or updates a ClickHouse user with permissions, quotas, and settings.
	// This is idempotent and can be called multiple times to update configuration.
	ConfigureUser(ctx context.Context, config UserConfig) error

	GetBillableVerifications(ctx context.Context, workspaceID string, year, month int) (int64, error)

	GetBillableRatelimits(ctx context.Context, workspaceID string, year, month int) (int64, error)

	// GetBillableUsageAboveThreshold returns total billable usage for workspaces that exceed a minimum threshold.
	// This pre-filters in ClickHouse rather than returning all workspaces, making it efficient for quota checking.
	// Returns a map from workspace ID to total usage count (only for workspaces >= minUsage).
	GetBillableUsageAboveThreshold(ctx context.Context, year, month int, minUsage int64) (map[string]int64, error)
}
```

### type UserConfig

```go
type UserConfig struct {
	WorkspaceID string
	Username    string
	Password    string

	// Tables to grant SELECT permission on
	AllowedTables []string

	// Quota settings (per window)
	QuotaDurationSeconds      int32
	MaxQueriesPerWindow       int32
	MaxExecutionTimePerWindow int32

	// Per-query limits (settings profile)
	MaxQueryExecutionTime int32
	MaxQueryMemoryBytes   int64
	MaxQueryResultRows    int32

	// Data retention (in days) - read from quotas table
	RetentionDays int32
}
```

UserConfig contains configuration for creating/updating a ClickHouse user

