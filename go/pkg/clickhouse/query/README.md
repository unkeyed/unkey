# Clickhouse Query Rewriter

Secure SQL query rewriting for Clickhouse with workspace isolation, table aliases, and virtual column resolution.

## Features

- ✅ **Workspace Isolation**: Automatically injects `workspace_id` filter into all queries
- ✅ **Table Aliases**: Map user-friendly table names to actual Clickhouse tables
- ✅ **Virtual Columns**: Handle `apiId` and `externalId` which require database lookups
- ✅ **Security**: Blocks INSERT/UPDATE/DELETE, system tables, dangerous functions
- ✅ **SQL Injection Protection**: Comprehensive tests against injection attacks

## Usage

### Basic Query Rewriting

```go
import "github.com/unkeyed/unkey/go/pkg/clickhouse/query"

rewriter := query.New(query.Config{
    WorkspaceID: "ws_abc123",
    TableAliases: map[string]string{
        "key_verifications": "default.key_verifications_v1",
    },
    AllowedTables: []string{
        "default.key_verifications_v1",
    },
})

// User writes friendly query
userQuery := "SELECT COUNT(*) FROM key_verifications WHERE valid = true"

// Rewrite to safe query
safeQuery, err := rewriter.Rewrite(userQuery)
// Result: SELECT COUNT(*) FROM `default.key_verifications_v1`
//         WHERE workspace_id = 'ws_abc123' AND valid = true
```

### Virtual Columns (apiId, externalId)

When users query with `apiId` or `externalId`, these need to be resolved via database lookups. Supports both `=` comparisons and `IN ()` clauses:

```go
rewriter := query.New(query.Config{
    WorkspaceID: "ws_abc123",
    TableAliases: map[string]string{
        "key_verifications": "default.key_verifications_v1",
    },
    AllowedTables: []string{
        "default.key_verifications_v1",
    },
    VirtualColumns: map[string]string{
        "apiId":      "key_space_id",
        "externalId": "identity_id",
    },
})

// Example 1: Single value with = comparison
userQuery := "SELECT * FROM key_verifications WHERE apiId = 'api_123'"

// Step 1: Extract virtual columns that need resolution
virtualCols, err := rewriter.ExtractVirtualColumns(userQuery)
// virtualCols[0] = {
//     VirtualColumn: "apiId",
//     Value: "api_123",
//     ActualColumn: "key_space_id",
// }

// Step 2: Lookup the actual values (your database code)
for i := range virtualCols {
    if virtualCols[i].VirtualColumn == "apiId" {
        api, err := db.GetAPIByID(virtualCols[i].Value)
        virtualCols[i].ActualValue = api.KeyAuthID
    }
    if virtualCols[i].VirtualColumn == "externalId" {
        identity, err := db.GetIdentityByExternalID(workspaceID, virtualCols[i].Value)
        virtualCols[i].ActualValue = identity.ID
    }
}

// Step 3: Rewrite query with resolved values
safeQuery, err := rewriter.RewriteWithVirtualColumns(userQuery, virtualCols)
// Result: SELECT * FROM `default.key_verifications_v1`
//         WHERE workspace_id = 'ws_abc123' AND key_space_id = 'keyauth_xyz'

// Example 2: Multiple values with IN clause
userQuery := "SELECT * FROM key_verifications WHERE apiId IN ('api_123', 'api_456')"

// Step 1: Extract virtual columns
virtualCols, err := rewriter.ExtractVirtualColumns(userQuery)
// virtualCols[0] = {
//     VirtualColumn: "apiId",
//     Values: []string{"api_123", "api_456"},
//     ActualColumn: "key_space_id",
// }

// Step 2: Lookup the actual values for each apiId
for i := range virtualCols {
    if virtualCols[i].VirtualColumn == "apiId" && len(virtualCols[i].Values) > 0 {
        var actualValues []string
        for _, apiID := range virtualCols[i].Values {
            api, err := db.GetAPIByID(apiID)
            actualValues = append(actualValues, api.KeyAuthID)
        }
        virtualCols[i].ActualValues = actualValues
    }
}

// Step 3: Rewrite query with resolved values
safeQuery, err := rewriter.RewriteWithVirtualColumns(userQuery, virtualCols)
// Result: SELECT * FROM `default.key_verifications_v1`
//         WHERE workspace_id = 'ws_abc123' AND key_space_id IN ('keyauth_xyz', 'keyauth_abc')
```

## V2 API Endpoint Example

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

    rewriter := query.New(query.Config{
        WorkspaceID: workspaceID,
        TableAliases: map[string]string{
            "key_verifications": "default.key_verifications_v1",
            "ratelimits":        "default.ratelimits_v1",
        },
        AllowedTables: []string{
            "default.key_verifications_v1",
            "default.ratelimits_v1",
        },
        VirtualColumns: map[string]string{
            "apiId":      "key_space_id",
            "externalId": "identity_id",
        },
    })

    // Extract and resolve virtual columns
    virtualCols, err := rewriter.ExtractVirtualColumns(req.Query)
    if err != nil {
        return c.JSON(400, gin.H{"error": err.Error()})
    }

    for i := range virtualCols {
        switch virtualCols[i].VirtualColumn {
        case "apiId":
            // Handle both single value (=) and multiple values (IN)
            if len(virtualCols[i].Values) > 0 {
                // IN clause - lookup multiple values
                var actualValues []string
                for _, apiID := range virtualCols[i].Values {
                    api, err := db.GetAPI(apiID)
                    if err != nil {
                        return c.JSON(404, gin.H{"error": fmt.Sprintf("API %s not found", apiID)})
                    }
                    actualValues = append(actualValues, api.KeyAuthID)
                }
                virtualCols[i].ActualValues = actualValues
            } else {
                // = comparison - lookup single value
                api, err := db.GetAPI(virtualCols[i].Value)
                if err != nil {
                    return c.JSON(404, gin.H{"error": "API not found"})
                }
                virtualCols[i].ActualValue = api.KeyAuthID
            }

        case "externalId":
            // Handle both single value (=) and multiple values (IN)
            if len(virtualCols[i].Values) > 0 {
                // IN clause - lookup multiple values
                var actualValues []string
                for _, externalID := range virtualCols[i].Values {
                    identity, err := db.GetIdentity(workspaceID, externalID)
                    if err != nil {
                        return c.JSON(404, gin.H{"error": fmt.Sprintf("Identity %s not found", externalID)})
                    }
                    actualValues = append(actualValues, identity.ID)
                }
                virtualCols[i].ActualValues = actualValues
            } else {
                // = comparison - lookup single value
                identity, err := db.GetIdentity(workspaceID, virtualCols[i].Value)
                if err != nil {
                    return c.JSON(404, gin.H{"error": "Identity not found"})
                }
                virtualCols[i].ActualValue = identity.ID
            }
        }
    }

    // Rewrite query
    safeQuery, err := rewriter.RewriteWithVirtualColumns(req.Query, virtualCols)
    if err != nil {
        return c.JSON(400, gin.H{"error": err.Error()})
    }

    // Execute against Clickhouse
    rows, err := clickhouse.Query(safeQuery)
    if err != nil {
        return c.JSON(500, gin.H{"error": "query failed"})
    }

    return c.JSON(200, gin.H{"data": rows})
}
```

## Security Features

### Workspace Isolation

Every query automatically gets `workspace_id = 'ws_xxx'` injected:

```sql
-- User writes:
SELECT * FROM key_verifications

-- Executed as:
SELECT * FROM default.key_verifications_v1
WHERE workspace_id = 'ws_abc123'
```

### Injection Protection

Even if users try to bypass workspace filtering, they can't:

```sql
-- User writes:
SELECT * FROM key_verifications WHERE workspace_id = 'ws_attacker'

-- Executed as:
SELECT * FROM default.key_verifications_v1
WHERE workspace_id = 'ws_victim' AND workspace_id = 'ws_attacker'

-- Returns empty (impossible condition) ✅
```

### Blocked Operations

- INSERT, UPDATE, DELETE, DROP
- System tables (`system.*`, `information_schema.*`)
- Dangerous functions (`file()`, `url()`, `executable()`)
- Stacked queries (`; DROP TABLE`)

## Testing

```bash
go test ./pkg/clickhouse/query/... -v
```

Tests cover:
- Basic rewriting
- Virtual column extraction and replacement
- SQL injection attacks
- Workspace isolation bypass attempts
- Case sensitivity variations
- Quote escaping
