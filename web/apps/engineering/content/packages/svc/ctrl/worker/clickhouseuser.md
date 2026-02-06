---
title: clickhouseuser
description: "implements ClickHouse user provisioning workflows"
---

Package clickhouseuser implements ClickHouse user provisioning workflows.

This package manages the complete lifecycle of ClickHouse users for workspace analytics access. It creates users with appropriate permissions, quotas, and row-level security (RLS) policies that restrict data access to the owning workspace with time-based retention filters.

### Why Restate

ClickHouse user provisioning involves multiple systems: MySQL for credential storage, Vault for password encryption, and ClickHouse for user creation. Any of these operations can fail independently, leaving the system in an inconsistent state. Restate's durable execution ensures that either all steps complete successfully or the workflow can be safely retried.

The virtual object model keys workflows by workspace\_id, preventing concurrent provisioning attempts for the same workspace which could result in password mismatches between MySQL and ClickHouse.

### Key Types

\[Service] is the main entry point implementing hydrav1.ClickhouseUserServiceServer. Configure it via \[Config] and create instances with \[New]. The service exposes \[Service.ConfigureUser] for creating or updating ClickHouse users.

### User Configuration

For each workspace, the service creates:

  - A ClickHouse user with SHA256 password authentication
  - SELECT permissions on analytics tables (key verifications)
  - Row-level security policies restricting access to workspace data
  - Time-based retention filters based on the workspace's quota settings
  - Query quotas (queries per window, execution time limits)
  - Settings profile (per-query limits for execution time, memory, result rows)

### Security

Passwords are generated using crypto/rand and encrypted via the Vault API before storage in MySQL. The encryption uses the workspace\_id as the keyring identifier, ensuring passwords can only be decrypted for their owning workspace.

Row-level security policies use ClickHouse's native RLS feature to enforce workspace isolation at the database level, preventing any possibility of cross-workspace data access even with valid credentials.

### Admin User

This service requires a dedicated ClickHouse admin user with minimal permissions:

  - CREATE/ALTER/DROP USER, QUOTA, ROW POLICY, SETTINGS PROFILE
  - GRANT OPTION on analytics tables for granting SELECT to workspace users

This follows the principle of least privilege - the admin user cannot read any analytics data itself, only manage user access controls.

### Idempotency

\[Service.ConfigureUser] is idempotent. Calling it multiple times for the same workspace will preserve the existing password while updating quotas and reapplying permissions. This allows safe retries and quota updates without breaking existing integrations using the workspace's ClickHouse credentials.

## Constants

```go
const (
	// defaultQuotaDurationSeconds defines the quota window used for rate limits.
	defaultQuotaDurationSeconds = 3600
	// defaultMaxQueriesPerWindow caps queries per quota window.
	defaultMaxQueriesPerWindow = 1000
	// defaultMaxExecutionTimePerWindow caps total query runtime per window.
	defaultMaxExecutionTimePerWindow = 1800
	// defaultMaxQueryExecutionTime caps runtime per query to avoid long-running scans.
	defaultMaxQueryExecutionTime = 30
	// defaultMaxQueryMemoryBytes caps memory per query to avoid exhausting the cluster.
	defaultMaxQueryMemoryBytes = 1000000000
	// defaultMaxQueryResultRows caps result size to prevent accidental full exports.
	defaultMaxQueryResultRows = 10000000
	// passwordLength defines the generated ClickHouse password length.
	passwordLength = 64
)
```

```go
const (
	upper   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lower   = "abcdefghijklmnopqrstuvwxyz"
	digits  = "0123456789"
	special = "-_.~" // RFC 3986 unreserved chars only - safe in DSN userinfo
	all     = upper + lower + digits + special
)
```


## Variables

```go
var _ hydrav1.ClickhouseUserServiceServer = (*Service)(nil)
```


## Functions


## Types

### type Config

```go
type Config struct {
	// DB provides database access for storing encrypted credentials.
	DB db.Database

	// Vault encrypts passwords before database storage. Passwords are encrypted using
	// the workspace ID as the keyring identifier.
	Vault vaultv1connect.VaultServiceClient

	// Clickhouse is the admin connection for creating users and managing permissions.
	// Must be connected as a user with CREATE/ALTER/DROP permissions for USER, QUOTA,
	// ROW POLICY, and SETTINGS PROFILE, plus GRANT OPTION on analytics tables.
	Clickhouse clickhouse.ClickHouse
}
```

Config holds configuration for creating a \[Service] instance.

### type Service

```go
type Service struct {
	hydrav1.UnimplementedClickhouseUserServiceServer
	db         db.Database
	vault      vaultv1connect.VaultServiceClient
	clickhouse clickhouse.ClickHouse
}
```

Service orchestrates ClickHouse user provisioning for workspaces.

Service implements hydrav1.ClickhouseUserServiceServer with \[Service.ConfigureUser] as the primary handler for creating and updating ClickHouse users. It coordinates between MySQL (credential storage), Vault (password encryption), and ClickHouse (user creation with permissions).

Not safe for concurrent use on the same workspace. Concurrency control is handled by Restate's virtual object model which keys handlers by workspace\_id.

#### func New

```go
func New(cfg Config) *Service
```

New creates a \[Service] with the given configuration. The returned service is ready to handle user provisioning requests.

#### func (Service) ConfigureUser

```go
func (s *Service) ConfigureUser(
	ctx restate.ObjectContext,
	req *hydrav1.ConfigureUserRequest,
) (*hydrav1.ConfigureUserResponse, error)
```

ConfigureUser creates or updates a ClickHouse user for a workspace.

For new workspaces, it generates credentials, encrypts them with Vault, and persists the encrypted values before provisioning ClickHouse. For existing workspaces, it preserves credentials while updating quota settings. The flow is idempotent and safe to retry after partial failures.

