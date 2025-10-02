# Clickhouse Query Rewriter

Secure SQL query rewriting for ClickHouse with workspace isolation, table aliases, virtual column resolution, and automatic row limiting.

## Features

- Workspace Isolation: Automatically injects `workspace_id` filter into all queries
- Table Aliases: Map user-friendly table names to actual ClickHouse tables
- Virtual Columns: Resolve `apiId` and `externalId` via custom resolver functions
- Row Limiting: Automatically cap or add LIMIT clause to queries
- Security: Blocks INSERT/UPDATE/DELETE, system tables, dangerous functions
- SQL Injection Protection: Comprehensive protection against injection attacks

## Quick Start

```go
import (
    "context"
    chquery "github.com/unkeyed/unkey/go/pkg/clickhouse/query"
)

rewriter := chquery.New(chquery.Config{
    WorkspaceID: "ws_abc123",
    Limit:       10000,
    TableAliases: map[string]string{
        "key_verifications": "default.key_verifications_raw_v2",
    },
    AllowedTables: []string{
        "default.key_verifications_raw_v2",
    },
})

userQuery := "SELECT COUNT(*) FROM key_verifications WHERE outcome = 'VALID'"
safeQuery, err := rewriter.Rewrite(context.Background(), userQuery)
// Result: SELECT COUNT(*) FROM default.key_verifications_raw_v2
//         WHERE workspace_id = 'ws_abc123' AND outcome = 'VALID'
//         LIMIT 10000
```

## Virtual Columns

Virtual columns allow users to query with friendly identifiers that get resolved to internal IDs via database lookups.

```go
rewriter := chquery.New(chquery.Config{
    WorkspaceID: "ws_abc123",
    Limit:       10000,
    TableAliases: map[string]string{
        "key_verifications": "default.key_verifications_raw_v2",
    },
    AllowedTables: []string{
        "default.key_verifications_raw_v2",
    },
    VirtualColumns: map[string]chquery.VirtualColumn{
        "apiId": {
            ActualColumn: "key_space_id",
            Resolver: func(ctx context.Context, apiIDs []string) (map[string]string, error) {
                // Batch lookup: apiId -> key_space_id
                results, err := db.Query.FindKeyAuthsByIds(ctx, db.RO(), apiIDs)
                if err != nil {
                    return nil, err
                }

                lookup := make(map[string]string)
                for _, result := range results {
                    lookup[result.ApiID] = result.KeyAuthID
                }
                return lookup, nil
            },
        },
        "externalId": {
            ActualColumn: "identity_id",
            Resolver: func(ctx context.Context, externalIDs []string) (map[string]string, error) {
                // Batch lookup: externalId -> identity_id
                identities, err := db.Query.FindIdentities(ctx, db.RO(), db.FindIdentitiesParams{
                    WorkspaceID: workspaceID,
                    Identities:  externalIDs,
                })
                if err != nil {
                    return nil, err
                }

                lookup := make(map[string]string)
                for _, identity := range identities {
                    lookup[identity.ExternalID] = identity.ID
                }
                return lookup, nil
            },
        },
    },
})

// Single value
userQuery := "SELECT * FROM key_verifications WHERE apiId = 'api_123'"
safeQuery, err := rewriter.Rewrite(ctx, userQuery)
// Result: SELECT * FROM default.key_verifications_raw_v2
//         WHERE workspace_id = 'ws_abc123' AND key_space_id = 'keyauth_xyz'
//         LIMIT 10000

// Multiple values with IN clause
userQuery := "SELECT * FROM key_verifications WHERE apiId IN ('api_123', 'api_456')"
safeQuery, err := rewriter.Rewrite(ctx, userQuery)
// Result: SELECT * FROM default.key_verifications_raw_v2
//         WHERE workspace_id = 'ws_abc123' AND key_space_id IN ('keyauth_xyz', 'keyauth_abc')
//         LIMIT 10000
```

## Row Limiting

The rewriter automatically enforces row limits to prevent large result sets:

```go
rewriter := chquery.New(chquery.Config{
    WorkspaceID: "ws_abc123",
    Limit:       1000, // Max 1000 rows
    // ... other config
})

// Query without LIMIT - adds one
"SELECT * FROM key_verifications"
// -> "... LIMIT 1000"

// Query with LIMIT higher than max - caps it
"SELECT * FROM key_verifications LIMIT 5000"
// -> "... LIMIT 1000"

// Query with LIMIT lower than max - preserves it
"SELECT * FROM key_verifications LIMIT 10"
// -> "... LIMIT 10"

// Set Limit to 0 to disable enforcement
rewriter := chquery.New(chquery.Config{
    Limit: 0, // No limit enforcement
    // ...
})
```

## API Handler Example

```go
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
    auth, _, err := h.Keys.GetRootKey(ctx, s)
    if err != nil {
        return err
    }

    req, err := zen.BindBody[Request](s)
    if err != nil {
        return err
    }

    rewriter := chquery.New(chquery.Config{
        WorkspaceID: auth.AuthorizedWorkspaceID,
        Limit:       10000,
        TableAliases: map[string]string{
            "key_verifications":            "default.key_verifications_raw_v2",
            "key_verifications_per_minute": "default.key_verifications_per_minute_v2",
            "key_verifications_per_hour":   "default.key_verifications_per_hour_v2",
            "key_verifications_per_day":    "default.key_verifications_per_day_v2",
        },
        AllowedTables: []string{
            "default.key_verifications_raw_v2",
            "default.key_verifications_per_minute_v2",
            "default.key_verifications_per_hour_v2",
            "default.key_verifications_per_day_v2",
        },
        VirtualColumns: map[string]chquery.VirtualColumn{
            "apiId": {
                ActualColumn: "key_space_id",
                Resolver: func(ctx context.Context, apiIDs []string) (map[string]string, error) {
                    results, err := db.Query.FindKeyAuthsByIds(ctx, h.DB.RO(), apiIDs)
                    if err != nil {
                        return nil, err
                    }
                    lookup := make(map[string]string)
                    for _, result := range results {
                        lookup[result.ApiID] = result.KeyAuthID
                    }
                    return lookup, nil
                },
            },
            "externalId": {
                ActualColumn: "identity_id",
                Resolver: func(ctx context.Context, externalIDs []string) (map[string]string, error) {
                    identities, err := db.Query.FindIdentities(ctx, h.DB.RO(), db.FindIdentitiesParams{
                        WorkspaceID: auth.AuthorizedWorkspaceID,
                        Identities:  externalIDs,
                    })
                    if err != nil {
                        return nil, err
                    }
                    lookup := make(map[string]string)
                    for _, identity := range identities {
                        lookup[identity.ExternalID] = identity.ID
                    }
                    return lookup, nil
                },
            },
        },
    })

    // Single call handles everything: extraction, resolution, rewriting
    safeQuery, err := rewriter.Rewrite(ctx, req.Query)
    if err != nil {
        return err
    }

    // Execute query
    verifications, err := h.ClickHouse.QueryToMaps(ctx, safeQuery)
    if err != nil {
        return err
    }

    return s.JSON(http.StatusOK, Response{
        Data: ResponseData{
            Verifications: verifications,
        },
    })
}
```

## Security

### Workspace Isolation

Every query automatically gets workspace filtering:

```sql
-- User query:
SELECT * FROM key_verifications

-- Executed query:
SELECT * FROM default.key_verifications_raw_v2
WHERE workspace_id = 'ws_abc123'
LIMIT 10000
```

Even if users try to bypass it, they can't:

```sql
-- User query:
SELECT * FROM key_verifications WHERE workspace_id = 'ws_attacker'

-- Executed query:
SELECT * FROM default.key_verifications_raw_v2
WHERE workspace_id = 'ws_victim' AND workspace_id = 'ws_attacker'
LIMIT 10000

-- Returns empty (impossible condition)
```

### Blocked Operations

- INSERT, UPDATE, DELETE, DROP, ALTER
- System tables: `system.*`, `information_schema.*`
- Dangerous functions: `file()`, `url()`, `executable()`, `system()`, `shell()`
- Stacked queries: Semicolons only allowed within quoted strings

### Virtual Column Security

Virtual column resolvers are called with batched IDs to prevent N+1 queries and should validate workspace ownership:

```go
Resolver: func(ctx context.Context, apiIDs []string) (map[string]string, error) {
    // Batch lookup prevents N+1
    results, err := db.Query.FindKeyAuthsByIds(ctx, db.RO(), apiIDs)
    if err != nil {
        return nil, err
    }

    lookup := make(map[string]string)
    for _, result := range results {
        lookup[result.ApiID] = result.KeyAuthID
    }

    // Verify all IDs were found (prevents silent failures)
    for _, apiID := range apiIDs {
        if _, found := lookup[apiID]; !found {
            return nil, fmt.Errorf("api %s not found", apiID)
        }
    }

    return lookup, nil
}
```

## Testing

Run all tests:
```bash
go test ./pkg/clickhouse/query/... -v
```

Test coverage includes:
- Basic query rewriting with workspace isolation
- Virtual column extraction and resolution
- SQL injection protection
- Workspace isolation bypass attempts
- Row limit enforcement
- Complex queries with AND/OR/NOT/IN operators
- Aggregate functions and GROUP BY/HAVING
- Case sensitivity variations
- Quote escaping
- Stacked query prevention
