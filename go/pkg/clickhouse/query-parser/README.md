# ClickHouse Query Parser

Secure SQL query parsing and rewriting for ClickHouse with workspace isolation, table aliases, and virtual column resolution.

## Features

- **Workspace Isolation**: Automatically injects `workspace_id` filter into all queries
- **Table Aliases**: Map user-friendly table names to actual ClickHouse tables
- **Virtual Columns**: Resolve user-friendly IDs (e.g., `apiId`, `externalId`) to internal IDs via custom resolvers
- **Virtual Column Aliases**: Support multiple names for the same virtual column (e.g., `apiId` and `api_id`)
- **All Operators Supported**: Works with `=`, `!=`, `<`, `>`, `<=`, `>=`, `IN`, and more
- **Security**: Only allows SELECT queries, blocks system tables, whitelists safe functions
- **Limit Enforcement**: Caps or adds LIMIT clauses to prevent resource exhaustion

## Usage

### Basic Query Rewriting

```go
import "github.com/unkeyed/unkey/go/pkg/clickhouse/query-parser"

parser := queryparser.NewParser(queryparser.Config{
    WorkspaceID: "ws_abc123",
    TableAliases: map[string]string{
        "verifications": "default.key_verifications_v1",
    },
    AllowedTables: []string{
        "default.key_verifications_v1",
    },
    Limit: 1000, // Max rows to return
})

// User writes friendly query
userQuery := "SELECT COUNT(*) FROM verifications WHERE outcome = 'VALID'"

// Parse and rewrite to safe query
safeQuery, err := parser.Parse(ctx, userQuery)
// Result: SELECT COUNT(*) FROM default.key_verifications_v1
//         WHERE workspace_id = 'ws_abc123' AND outcome = 'VALID'
//         LIMIT 1000
```

### Virtual Columns with Resolver Functions

Virtual columns allow users to query with friendly IDs that get automatically resolved to internal IDs:

```go
// Define a resolver function
apiIdResolver := func(ctx context.Context, virtualIDs []string) (map[string]string, error) {
    resolved := make(map[string]string)
    for _, apiID := range virtualIDs {
        api, err := db.GetAPIByID(ctx, apiID)
        if err != nil {
            return nil, err
        }
        resolved[apiID] = api.KeyAuthID // Map virtual ID to actual ID
    }
    return resolved, nil
}

parser := queryparser.NewParser(queryparser.Config{
    WorkspaceID: "ws_abc123",
    AllowedTables: []string{
        "default.key_verifications_v1",
    },
    VirtualColumns: map[string]queryparser.VirtualColumn{
        "apiId": {
            ActualColumn: "key_space_id",
            Aliases:      []string{"api_id"}, // Both apiId and api_id work
            Resolver:     apiIdResolver,
        },
    },
})

// Works with = operator
userQuery := "SELECT * FROM default.key_verifications_v1 WHERE apiId = 'api_123'"
safeQuery, err := parser.Parse(ctx, userQuery)
// Result: SELECT * FROM default.key_verifications_v1
//         WHERE workspace_id = 'ws_abc123' AND key_space_id = 'keyauth_xyz'

// Works with IN operator
userQuery = "SELECT * FROM default.key_verifications_v1 WHERE apiId IN ('api_1', 'api_2')"
safeQuery, err = parser.Parse(ctx, userQuery)
// Result: SELECT * FROM default.key_verifications_v1
//         WHERE workspace_id = 'ws_abc123' AND key_space_id IN ('keyauth_abc', 'keyauth_def')

// Works with all comparison operators
userQuery = "SELECT * FROM default.key_verifications_v1 WHERE apiId != 'api_123'"
safeQuery, err = parser.Parse(ctx, userQuery)
```

### Multiple Virtual Columns

```go
apiIdResolver := func(ctx context.Context, virtualIDs []string) (map[string]string, error) {
    resolved := make(map[string]string)
    for _, id := range virtualIDs {
        api, err := db.GetAPIByID(ctx, id)
        if err != nil {
            return nil, err
        }
        resolved[id] = api.KeyAuthID
    }
    return resolved, nil
}

identityResolver := func(ctx context.Context, virtualIDs []string) (map[string]string, error) {
    resolved := make(map[string]string)
    for _, externalID := range virtualIDs {
        identity, err := db.GetIdentityByExternalID(ctx, workspaceID, externalID)
        if err != nil {
            return nil, err
        }
        resolved[externalID] = identity.ID
    }
    return resolved, nil
}

parser := queryparser.NewParser(queryparser.Config{
    WorkspaceID: "ws_abc123",
    AllowedTables: []string{
        "default.key_verifications_v1",
    },
    VirtualColumns: map[string]queryparser.VirtualColumn{
        "apiId": {
            ActualColumn: "key_space_id",
            Aliases:      []string{"api_id"},
            Resolver:     apiIdResolver,
        },
        "externalId": {
            ActualColumn: "identity_id",
            Aliases:      []string{"external_id"},
            Resolver:     identityResolver,
        },
    },
})

userQuery := "SELECT * FROM default.key_verifications_v1 WHERE apiId = 'api_123' AND externalId = 'user_456'"
safeQuery, err := parser.Parse(ctx, userQuery)
// Both virtual columns are resolved automatically
```

## API Endpoint Example

```go
// POST /v2/analytics/query
func HandleAnalyticsQuery(c *gin.Context) {
    var req struct {
        Query string `json:"query"`
    }
    if err := c.BindJSON(&req); err != nil {
        return c.JSON(400, gin.H{"error": "invalid request"})
    }

    workspaceID := getWorkspaceFromAuth(c)

    // Create resolvers
    apiIdResolver := func(ctx context.Context, virtualIDs []string) (map[string]string, error) {
        resolved := make(map[string]string)
        for _, id := range virtualIDs {
            api, err := db.GetAPI(ctx, id)
            if err != nil {
                return nil, fault.Wrap(err, fault.Public(fmt.Sprintf("API %s not found", id)))
            }
            resolved[id] = api.KeyAuthID
        }
        return resolved, nil
    }

    identityResolver := func(ctx context.Context, virtualIDs []string) (map[string]string, error) {
        resolved := make(map[string]string)
        for _, externalID := range virtualIDs {
            identity, err := db.GetIdentity(ctx, workspaceID, externalID)
            if err != nil {
                return nil, fault.Wrap(err, fault.Public(fmt.Sprintf("Identity %s not found", externalID)))
            }
            resolved[externalID] = identity.ID
        }
        return resolved, nil
    }

    parser := queryparser.NewParser(queryparser.Config{
        WorkspaceID: workspaceID,
        TableAliases: map[string]string{
            "verifications": "default.key_verifications_v1",
            "ratelimits":    "default.ratelimits_v1",
        },
        AllowedTables: []string{
            "default.key_verifications_v1",
            "default.ratelimits_v1",
        },
        VirtualColumns: map[string]queryparser.VirtualColumn{
            "apiId": {
                ActualColumn: "key_space_id",
                Aliases:      []string{"api_id"},
                Resolver:     apiIdResolver,
            },
            "externalId": {
                ActualColumn: "identity_id",
                Aliases:      []string{"external_id"},
                Resolver:     identityResolver,
            },
        },
        Limit: 10000,
    })

    // Parse and rewrite query (virtual columns resolved automatically)
    safeQuery, err := parser.Parse(c.Request.Context(), req.Query)
    if err != nil {
        return c.JSON(400, gin.H{"error": err.Error()})
    }

    // Execute against ClickHouse
    rows, err := clickhouse.Query(c.Request.Context(), safeQuery)
    if err != nil {
        return c.JSON(500, gin.H{"error": "query failed"})
    }

    return c.JSON(200, gin.H{"data": rows})
}
```

## Error Codes

The parser returns structured error codes for better error handling and user feedback:

### Parser Validation Errors

- **`err:user:bad_request:invalid_analytics_query`** - Invalid SQL syntax or query structure
- **`err:user:bad_request:invalid_table`** - Table is not allowed or does not exist
- **`err:user:bad_request:invalid_function`** - Function is not in the whitelist
- **`err:user:bad_request:query_not_supported`** - Query type not allowed (only SELECT is supported)

### Resource Limit Errors (from ClickHouse)

These errors are returned when ClickHouse resource limits are exceeded. Use `clickhouse.WrapClickHouseError()` to parse ClickHouse errors:

- **`err:user:bad_request:query_execution_timeout`** - Query exceeded max execution time (default: 30s per query, 1800s per window)
- **`err:user:bad_request:query_memory_limit_exceeded`** - Query exceeded max memory usage (default: 1GB)
- **`err:user:bad_request:query_rows_limit_exceeded`** - Query attempted to read too many rows (default: 10M rows)
- **`err:user:bad_request:query_result_rows_limit_exceeded`** - Query result set too large (default: 10M rows)
- **`err:user:bad_request:query_quota_exceeded`** - Workspace exceeded query quota (default: 1000 queries per hour)

### Example Error Handling

```go
// Parse and rewrite query
safeQuery, err := parser.Parse(ctx, userQuery)
if err != nil {
    code, hasCode := fault.GetCode(err)
    if hasCode {
        switch code {
        case codes.User.BadRequest.InvalidTable.URN():
            return c.JSON(400, gin.H{
                "error": fault.UserFacingMessage(err),
                "code": code,
                "docs": codes.User.BadRequest.InvalidTable.DocsURL(),
            })
        case codes.User.BadRequest.InvalidFunction.URN():
            return c.JSON(400, gin.H{
                "error": fault.UserFacingMessage(err),
                "code": code,
            })
        }
    }

    return c.JSON(400, gin.H{
        "error": fault.UserFacingMessage(err),
    })
}

// Execute query
rows, err := clickhouse.Query(ctx, safeQuery)
if err != nil {
    // Wrap ClickHouse errors with proper error codes
    wrappedErr := clickhouse.WrapClickHouseError(err)

    code, hasCode := fault.GetCode(wrappedErr)
    if hasCode {
        switch code {
        case codes.User.BadRequest.QueryExecutionTimeout.URN():
            return c.JSON(408, gin.H{
                "error": fault.UserFacingMessage(wrappedErr),
                "code": code,
            })
        case codes.User.BadRequest.QueryQuotaExceeded.URN():
            return c.JSON(429, gin.H{
                "error": fault.UserFacingMessage(wrappedErr),
                "code": code,
            })
        case codes.User.BadRequest.QueryMemoryLimitExceeded.URN(),
             codes.User.BadRequest.QueryRowsLimitExceeded.URN(),
             codes.User.BadRequest.QueryResultRowsLimitExceeded.URN():
            return c.JSON(400, gin.H{
                "error": fault.UserFacingMessage(wrappedErr),
                "code": code,
            })
        }
    }

    return c.JSON(500, gin.H{
        "error": fault.UserFacingMessage(wrappedErr),
    })
}
```

## Security Features

### Workspace Isolation

Every query automatically gets `workspace_id = 'ws_xxx'` injected:

```sql
-- User writes:
SELECT * FROM verifications

-- Executed as:
SELECT * FROM default.key_verifications_v1
WHERE workspace_id = 'ws_abc123'
```

### Injection Protection

Even if users try to bypass workspace filtering, they can't:

```sql
-- User writes:
SELECT * FROM verifications WHERE workspace_id = 'ws_attacker'

-- Executed as:
SELECT * FROM default.key_verifications_v1
WHERE workspace_id = 'ws_victim' AND workspace_id = 'ws_attacker'

-- Returns empty (impossible condition)
```

### Blocked Operations

- **Only SELECT allowed**: INSERT, UPDATE, DELETE, DROP are rejected
- **System tables blocked**: `system.*`, `information_schema.*` are not accessible
- **Function whitelist**: Only safe functions are allowed (aggregations, date/time, string, math, etc.)
- **Dangerous functions blocked**: `file()`, `url()`, `executable()`, `system()`, `shell()`, `pipe()`

### Whitelisted Functions

The parser uses a whitelist approach for security. Allowed functions include:

**Aggregate:** count, sum, avg, min, max, any, groupArray, groupUniqArray, uniq, uniqExact

**Date/Time:** now, today, toDate, toDateTime, toStartOfDay, toStartOfWeek, toStartOfMonth, toStartOfYear, toStartOfHour, toStartOfMinute, date_trunc, formatDateTime

**String:** lower, upper, substring, concat, length, trim

**Math:** round, floor, ceil, abs

**Conditional:** if, case, coalesce

**Type Conversion:** toString, toInt32, toInt64, toFloat64

To add more functions, update the `allowedFunctions` map in `validation.go`.

### Limit Enforcement

If a limit is configured, queries are automatically protected:

```sql
-- Config.Limit = 1000

-- User writes:
SELECT * FROM verifications

-- Executed as:
SELECT * FROM default.key_verifications_v1
WHERE workspace_id = 'ws_abc123'
LIMIT 1000

-- User writes:
SELECT * FROM verifications LIMIT 5000

-- Executed as (capped):
SELECT * FROM default.key_verifications_v1
WHERE workspace_id = 'ws_abc123'
LIMIT 1000
```

## Testing

```bash
go test ./pkg/clickhouse/query-parser/... -v
```

Tests cover:

- Basic query rewriting
- Virtual column resolution with all operators (=, !=, <, >, <=, >=, IN)
- Virtual column aliases
- Table aliases and access control
- Workspace isolation
- Limit enforcement
- Function validation (whitelist)
- System table blocking
- Only SELECT queries allowed
- Error codes and user-facing messages

## Architecture

The parser is split into focused modules:

- **types.go** - Core types and configuration
- **parser.go** - Main parsing and orchestration
- **virtual_columns.go** - Virtual column extraction and rewriting
- **tables.go** - Table alias resolution and access control
- **workspace.go** - Workspace filter injection
- **limits.go** - Limit enforcement
- **validation.go** - Function whitelist validation

Each module has corresponding test files with comprehensive coverage.
